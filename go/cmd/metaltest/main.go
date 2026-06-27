// SPDX-Licence-Identifier: EUPL-1.2

//go:build darwin

// metaltest — smallest pure-Go Metal compute proof: no cgo, no shell-out, drive
// a kernel + mutate a buffer through tmc/apple/metal (purego objc_msgSend).
// Doubles a buffer on the GPU and reads it back. The point isn't the math — it's
// that Go owns the Metal command path directly.
//
// Three things this proves, in order:
//  1. correctness    — compile MSL, dispatch, mutate a buffer, read it back;
//  2. throughput     — serial commit+wait (GPU held-but-idle) vs batched (GPU fed);
//  3. encode-bypass  — re-encoding a fixed command sequence every step (MLX's
//     ~68 ms/step host cost) vs recording it ONCE into an Indirect Command Buffer
//     and replaying — the decode/diffusion lever, in pure Go;
//  4. kernel reach   — our real 150 MB mlx.metallib loads + enumerates through the
//     same no-cgo path: the foundation for keeping the kernels and dropping mlx-c.
package main

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/tmc/apple/foundation"
	"github.com/tmc/apple/metal"
	"github.com/tmc/apple/objc"
)

const src = `
#include <metal_stdlib>
using namespace metal;
kernel void doubleit(device float* data [[buffer(0)]],
                     uint i [[thread_position_in_grid]]) {
    data[i] = data[i] * 2.0f;
}
`

func main() {
	runtime.LockOSThread()

	dev := metal.MTLCreateSystemDefaultDevice()
	fmt.Printf("device: %v\n", dev.Name())

	lib, err := dev.NewLibraryWithSourceOptionsError(src, nil)
	if err != nil {
		panic(fmt.Sprintf("compile MSL: %v", err))
	}
	fn := lib.NewFunctionWithName("doubleit")
	pso, err := dev.NewComputePipelineStateWithFunctionError(fn)
	if err != nil {
		panic(fmt.Sprintf("pipeline: %v", err))
	}
	queue := dev.NewCommandQueue()

	const n = 8
	in := make([]float32, n)
	for i := range in {
		in[i] = float32(i + 1)
	}
	buf := dev.NewBufferWithBytesLengthOptions(unsafe.Pointer(&in[0]), uint(n*4), metal.MTLResourceStorageModeShared)

	cb := queue.CommandBuffer()
	enc := cb.ComputeCommandEncoder()
	enc.SetComputePipelineState(pso)
	enc.SetBufferWithOffsetAtIndex(buf, 0, 0)
	enc.DispatchThreadgroupsThreadsPerThreadgroup(
		metal.MTLSize{Width: n, Height: 1, Depth: 1},
		metal.MTLSize{Width: 1, Height: 1, Depth: 1},
	)
	enc.EndEncoding()

	t := time.Now()
	cb.Commit()
	cb.WaitUntilCompleted()
	dur := time.Since(t)

	out := unsafe.Slice((*float32)(buf.Contents()), n)
	fmt.Printf("in:  %v\n", in)
	fmt.Printf("out: %v\n", out)
	fmt.Printf("commit+wait: %v\n", dur)

	// --- throughput: GPU held-but-idle vs GPU fed ---------------------------
	// A real-sized buffer so the dispatch does measurable GPU work, then the
	// same N dispatches two ways: SERIAL (commit+wait each — host blocks on the
	// GPU round-trip every time, GPU idle between) vs BATCHED (commit all,
	// wait once — GPU runs back-to-back). The gap is the throughput the
	// host-busy/GPU-waiting pattern leaves on the table.
	const big = 1 << 20 // 1M floats
	const iters = 48    // under the default ~64 in-flight CB limit (no autorelease pool here)
	bdata := make([]float32, big)
	bbuf := dev.NewBufferWithBytesLengthOptions(unsafe.Pointer(&bdata[0]), uint(big*4), metal.MTLResourceStorageModeShared)
	grid := metal.MTLSize{Width: big / 256, Height: 1, Depth: 1}
	group := metal.MTLSize{Width: 256, Height: 1, Depth: 1}
	dispatch := func() metal.MTLCommandBuffer {
		c := queue.CommandBuffer()
		e := c.ComputeCommandEncoder()
		e.SetComputePipelineState(pso)
		e.SetBufferWithOffsetAtIndex(bbuf, 0, 0)
		e.DispatchThreadgroupsThreadsPerThreadgroup(grid, group)
		e.EndEncoding()
		return c
	}
	w := dispatch()
	w.Commit()
	w.WaitUntilCompleted() // warm

	ts := time.Now()
	for i := 0; i < iters; i++ {
		c := dispatch()
		c.Commit()
		c.WaitUntilCompleted()
	}
	serial := time.Since(ts)

	tb := time.Now()
	var last metal.MTLCommandBuffer
	for i := 0; i < iters; i++ {
		c := dispatch()
		c.Commit()
		last = c
	}
	last.WaitUntilCompleted()
	batched := time.Since(tb)

	fmt.Printf("\n%d dispatches of %dK floats:\n", iters, big/1024)
	fmt.Printf("  serial  (commit+wait each): %7.2f ms  =  %6.0f disp/s\n", float64(serial.Microseconds())/1000, float64(iters)/serial.Seconds())
	fmt.Printf("  batched (wait once at end): %7.2f ms  =  %6.0f disp/s  (%.1fx)\n", float64(batched.Microseconds())/1000, float64(iters)/batched.Seconds(), serial.Seconds()/batched.Seconds())

	// --- encode-bypass: ICB record-once/replay vs re-encode-each ------------
	// The decode/diffusion wall: a FIXED per-step command sequence (the graph)
	// that MLX re-encodes on the HOST every single step — that re-encode is the
	// ~68 ms/step we measured, not GPU work. An Indirect Command Buffer records
	// the sequence ONCE; each step replays it with a single executeCommandsInBuffer
	// call — zero per-op host marshalling. Same GPU work either way. Tiny buffer
	// (trivial GPU) so what we read IS the host encode cost the ICB eliminates.
	const cmds = 512 // ops in the fixed per-step "graph"
	const steps = 64 // times the sequence runs (denoise steps / decoded tokens)
	sbuf := dev.NewBufferWithBytesLengthOptions(unsafe.Pointer(&in[0]), uint(n*4), metal.MTLResourceStorageModeShared)
	sgrid := metal.MTLSize{Width: n, Height: 1, Depth: 1}
	sgroup := metal.MTLSize{Width: 1, Height: 1, Depth: 1}

	var encEach, recordOnce, icbReplay time.Duration

	// (A) re-encode each step: fresh encoder, encode `cmds` ops, commit, wait.
	objc.AutoreleasePool(func() {
		ta := time.Now()
		for s := 0; s < steps; s++ {
			c := queue.CommandBuffer()
			e := c.ComputeCommandEncoder()
			for k := 0; k < cmds; k++ {
				e.SetComputePipelineState(pso)
				e.SetBufferWithOffsetAtIndex(sbuf, 0, 0)
				e.DispatchThreadgroupsThreadsPerThreadgroup(sgrid, sgroup)
			}
			e.EndEncoding()
			c.Commit()
			c.WaitUntilCompleted()
		}
		encEach = time.Since(ta)
	})

	// (B) ICB: an ICB-capable PSO, record `cmds` ONCE, replay each step with one call.
	pdesc := metal.NewMTLComputePipelineDescriptor()
	pdesc.SetComputeFunction(fn)
	pdesc.SetSupportIndirectCommandBuffers(true)
	psoICB, err := dev.NewComputePipelineStateWithDescriptorOptionsReflectionError(pdesc, 0, nil)
	if err != nil {
		panic(fmt.Sprintf("icb pipeline: %v", err))
	}
	icbDesc := metal.NewMTLIndirectCommandBufferDescriptor()
	icbDesc.SetCommandTypes(metal.MTLIndirectCommandTypeConcurrentDispatch)
	icbDesc.SetInheritBuffers(false)      // we bind the buffer per command
	icbDesc.SetInheritPipelineState(false) // we set the PSO per command
	icbDesc.SetMaxKernelBufferBindCount(1)
	icb := dev.NewIndirectCommandBufferWithDescriptorMaxCommandCountOptions(icbDesc, cmds, metal.MTLResourceStorageModeShared)

	objc.AutoreleasePool(func() {
		tr := time.Now()
		for k := 0; k < cmds; k++ {
			c := icb.IndirectComputeCommandAtIndex(uint(k))
			c.SetComputePipelineState(psoICB)
			c.SetKernelBufferOffsetAtIndex(sbuf, 0, 0)
			c.ConcurrentDispatchThreadgroupsThreadsPerThreadgroup(sgrid, sgroup)
		}
		recordOnce = time.Since(tr)

		rng := foundation.NSRange{Location: 0, Length: cmds}
		tbb := time.Now()
		for s := 0; s < steps; s++ {
			c := queue.CommandBuffer()
			e := c.ComputeCommandEncoder()
			e.UseResourceUsage(sbuf, metal.MTLResourceUsageRead|metal.MTLResourceUsageWrite) // ICBs don't auto-track residency
			e.ExecuteCommandsInBufferWithRange(icb, rng)
			e.EndEncoding()
			c.Commit()
			c.WaitUntilCompleted()
		}
		icbReplay = time.Since(tbb)
	})

	msA := float64(encEach.Microseconds()) / 1000.0 / float64(steps)
	msB := float64(icbReplay.Microseconds()) / 1000.0 / float64(steps)
	fmt.Printf("\n%d-op fixed sequence x %d steps (host RE-ENCODE is the cost):\n", cmds, steps)
	fmt.Printf("  re-encode each step:   %7.2f ms  (%.3f ms/step)\n", float64(encEach.Microseconds())/1000.0, msA)
	fmt.Printf("  ICB record-once:       %7.2f ms  (paid once, then free)\n", float64(recordOnce.Microseconds())/1000.0)
	fmt.Printf("  ICB replay each step:  %7.2f ms  (%.3f ms/step)  (%.1fx)\n", float64(icbReplay.Microseconds())/1000.0, msB, msA/msB)

	// --- our real kernels are reachable from Go: load mlx.metallib ----------
	// The structural premise of "keep the kernels, drop mlx-c": our compiled
	// kernel library (150 MB) loads + enumerates through the same no-cgo path.
	// Reach the functions → we can drive them. MLX's C++ graph/encode machinery
	// is what we'd replace; the kernels themselves come straight across.
	const libPath = "/Users/snider/Code/core/go-mlx/dist/lib/mlx.metallib"
	objc.AutoreleasePool(func() {
		url := foundation.GetNSURLClass().FileURLWithPath(libPath)
		mlxLib, lerr := dev.NewLibraryWithURLError(url)
		if lerr != nil {
			fmt.Printf("\nmlx.metallib: load failed: %v\n", lerr)
			return
		}
		names := mlxLib.FunctionNames()
		fmt.Printf("\nmlx.metallib loaded (no cgo): %d kernels reachable\n", len(names))
		for i, nm := range names {
			if i >= 6 {
				fmt.Printf("  · … and %d more\n", len(names)-6)
				break
			}
			fmt.Printf("  · %s\n", nm)
		}

		// Drive a REAL mlx kernel — not just reach it. v_Squarefloat32float32 is
		// the contiguous unary square (out = in*in). ABI straight from MLX's host
		// dispatch (mlx/backend/metal/unary.cpp): in=buffer(0), out=buffer(1),
		// element count=buffer(2) (uint), dispatchThreads(count). If our output
		// matches the Go reference, we can drive the existing kernel set — the
		// whole premise of keeping the kernels and replacing only the encode layer.
		const sq = 1024
		sqfn := mlxLib.NewFunctionWithName("v_Squarefloat32float32")
		sqpso, perr := dev.NewComputePipelineStateWithFunctionError(sqfn)
		if perr != nil {
			fmt.Printf("v_Squarefloat32float32 pipeline: %v\n", perr)
			return
		}
		sin := make([]float32, sq)
		for i := range sin {
			sin[i] = float32(i + 1)
		}
		inBuf := dev.NewBufferWithBytesLengthOptions(unsafe.Pointer(&sin[0]), uint(sq*4), metal.MTLResourceStorageModeShared)
		outBuf := dev.NewBufferWithLengthOptions(uint(sq*4), metal.MTLResourceStorageModeShared)
		szc := uint32(sq)
		szb := unsafe.Slice((*byte)(unsafe.Pointer(&szc)), 4) // the uint count as 4 bytes
		c := queue.CommandBuffer()
		e := c.ComputeCommandEncoder()
		e.SetComputePipelineState(sqpso)
		e.SetBufferWithOffsetAtIndex(inBuf, 0, 0)
		e.SetBufferWithOffsetAtIndex(outBuf, 0, 1)
		e.SetBytesLengthAtIndex(szb, 4, 2)
		e.DispatchThreadsThreadsPerThreadgroup(
			metal.MTLSize{Width: sq, Height: 1, Depth: 1},
			metal.MTLSize{Width: 256, Height: 1, Depth: 1},
		)
		e.EndEncoding()
		c.Commit()
		c.WaitUntilCompleted()

		got := unsafe.Slice((*float32)(outBuf.Contents()), sq)
		var maxErr float32
		for i := 0; i < sq; i++ {
			want := sin[i] * sin[i]
			d := got[i] - want
			if d < 0 {
				d = -d
			}
			if d > maxErr {
				maxErr = d
			}
		}
		fmt.Printf("drove v_Squarefloat32float32 (%d elems): out[:4]=%v want=[1 4 9 16]  maxErr=%g\n", sq, got[:4], maxErr)
	})
}
