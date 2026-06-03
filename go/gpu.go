package container

import (
	core "dappco.re/go"
)

// WithGPU requests Metal (Apple) or NVIDIA (Linux) GPU passthrough for the
// container. Runtimes that do not support GPU passthrough signal this via
// ContainerRuntime.HasGPU — check before adding the option when portability
// matters.
//
// Usage:
//
//	rt := container.Detect()
//	if rt.HasGPU() {
//	    opts = append(opts, container.WithGPU(true))
//	}
//	ctr, err := provider.Run(img, opts...)
func WithGPU(enabled bool) RunOption {
	return func(o *RunOptions) {
		o.GPU = enabled
	}
}

// RequireGPU returns a failed Result if the runtime does not support GPU
// passthrough. Use this when GPU access is mandatory for the workload (e.g.
// LEM inference inside the container).
//
// Usage:
//
//	if r := container.RequireGPU(container.Detect()); !r.OK {
//	    return r
//	}
func RequireGPU(rt ContainerRuntime) core.Result { // Value: nil
	if rt.HasGPU() {
		return core.Ok(nil)
	}
	return core.Fail(core.E("RequireGPU", "container runtime does not support GPU passthrough", nil))
}
