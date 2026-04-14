package container

import (
	"runtime"

	core "dappco.re/go/core"

	"dappco.re/go/core/container/internal/proc"
)

// RuntimeType identifies a container runtime backend detected on the host.
type RuntimeType string

const (
	// RuntimeApple is Apple's Containerisation framework (macOS 26+).
	RuntimeApple RuntimeType = "apple"
	// RuntimeDocker is Docker / dockerd on any platform.
	RuntimeDocker RuntimeType = "docker"
	// RuntimePodman is Podman on Linux or macOS.
	RuntimePodman RuntimeType = "podman"
	// RuntimeLinuxKit is the bundled LinuxKit + QEMU/Hyperkit runtime.
	RuntimeLinuxKit RuntimeType = "linuxkit"
	// RuntimeNone signals no supported runtime was detected.
	RuntimeNone RuntimeType = "none"
)

// Runtime capability bits. Providers report capabilities via the caps field
// on ContainerRuntime so consumers can adapt behaviour without branching on
// RuntimeType directly.
const (
	capGPU uint32 = 1 << iota
	capNetworkIsolation
	capVolumeMounts
	capEncryption
	capHardwareIsolation
	capSubSecondStart
)

// ContainerRuntime describes a detected container runtime and its capabilities.
//
// Usage:
//
//	rt := container.Detect()
//	if rt.HasGPU() {
//	    opts = append(opts, container.WithGPU(true))
//	}
type ContainerRuntime struct {
	// Type is the canonical runtime identifier.
	Type RuntimeType
	// Version is the reported runtime version string.
	Version string
	// Path is the path to the runtime binary or the detection marker.
	Path string
	// caps is the capability bitfield set by detection.
	caps uint32
}

// HasGPU reports whether the runtime can expose a GPU to the container.
//
// Usage:
//
//	if rt.HasGPU() { opts = append(opts, container.WithGPU(true)) }
func (r ContainerRuntime) HasGPU() bool { return r.caps&capGPU != 0 }

// HasNetworkIsolation reports whether network namespaces or VM networking
// are available.
//
// Usage:
//
//	if rt.HasNetworkIsolation() { cfg.Isolate = true }
func (r ContainerRuntime) HasNetworkIsolation() bool { return r.caps&capNetworkIsolation != 0 }

// HasVolumeMounts reports whether the runtime supports host volume mounting.
//
// Usage:
//
//	if rt.HasVolumeMounts() { opts = append(opts, container.WithVolumes(v)) }
func (r ContainerRuntime) HasVolumeMounts() bool { return r.caps&capVolumeMounts != 0 }

// HasEncryption reports whether the runtime provides native encrypted storage.
//
// Usage:
//
//	if rt.HasEncryption() { cfg.Encrypted = true }
func (r ContainerRuntime) HasEncryption() bool { return r.caps&capEncryption != 0 }

// IsHardwareIsolated reports whether the runtime uses hardware virtualisation
// for isolation (rather than namespaces).
//
// Usage:
//
//	if rt.IsHardwareIsolated() { auditLog("hardware-isolated workload") }
func (r ContainerRuntime) IsHardwareIsolated() bool { return r.caps&capHardwareIsolation != 0 }

// HasSubSecondStart reports whether the runtime boots in under one second.
//
// Usage:
//
//	if rt.HasSubSecondStart() { fast = true }
func (r ContainerRuntime) HasSubSecondStart() bool { return r.caps&capSubSecondStart != 0 }

// Caps returns the raw capability bitfield for callers that need it.
//
// Usage:
//
//	bits := rt.Caps()
func (r ContainerRuntime) Caps() uint32 { return r.caps }

// Detect probes the system for available container runtimes and returns the
// highest-priority runtime found. Priority order:
//
//	Apple Containers → Docker → Podman → LinuxKit → None.
//
// Usage:
//
//	rt := container.Detect()
//	fmt.Println(rt.Type)  // "apple", "docker", "podman", "linuxkit" or "none"
func Detect() ContainerRuntime {
	for _, rt := range DetectAll() {
		return rt
	}
	return ContainerRuntime{Type: RuntimeNone}
}

// DetectAll returns every container runtime found on the host ordered by
// priority (highest first). Empty slice means no runtime is available.
//
// Usage:
//
//	for _, rt := range container.DetectAll() {
//	    fmt.Printf("%s %s at %s\n", rt.Type, rt.Version, rt.Path)
//	}
func DetectAll() []ContainerRuntime {
	var out []ContainerRuntime

	if rt, ok := detectApple(); ok {
		out = append(out, rt)
	}
	if rt, ok := detectDocker(); ok {
		out = append(out, rt)
	}
	if rt, ok := detectPodman(); ok {
		out = append(out, rt)
	}
	if rt, ok := detectLinuxKit(); ok {
		out = append(out, rt)
	}

	return out
}

// detectApple probes for the Apple Containerisation framework CLI.
func detectApple() (ContainerRuntime, bool) {
	if runtime.GOOS != "darwin" {
		return ContainerRuntime{}, false
	}
	path, err := proc.LookPath("container")
	if err != nil {
		return ContainerRuntime{}, false
	}
	rt := ContainerRuntime{
		Type:    RuntimeApple,
		Path:    path,
		Version: captureVersion(path, "--version"),
	}
	rt.caps = capNetworkIsolation | capVolumeMounts | capHardwareIsolation | capSubSecondStart
	return rt, true
}

// detectDocker probes for the Docker CLI and daemon.
func detectDocker() (ContainerRuntime, bool) {
	path, err := proc.LookPath("docker")
	if err != nil {
		return ContainerRuntime{}, false
	}
	rt := ContainerRuntime{
		Type:    RuntimeDocker,
		Path:    path,
		Version: captureVersion(path, "--version"),
	}
	rt.caps = capNetworkIsolation | capVolumeMounts
	if runtime.GOOS == "linux" {
		rt.caps |= capGPU
	}
	return rt, true
}

// detectPodman probes for the Podman CLI.
func detectPodman() (ContainerRuntime, bool) {
	path, err := proc.LookPath("podman")
	if err != nil {
		return ContainerRuntime{}, false
	}
	rt := ContainerRuntime{
		Type:    RuntimePodman,
		Path:    path,
		Version: captureVersion(path, "--version"),
	}
	rt.caps = capNetworkIsolation | capVolumeMounts
	if runtime.GOOS == "linux" {
		rt.caps |= capGPU
	}
	return rt, true
}

// detectLinuxKit reports LinuxKit support when a compatible hypervisor is present.
func detectLinuxKit() (ContainerRuntime, bool) {
	hv, err := DetectHypervisor()
	if err != nil {
		return ContainerRuntime{}, false
	}
	rt := ContainerRuntime{
		Type:    RuntimeLinuxKit,
		Path:    hv.Name(),
		Version: "", // Hypervisor-specific; intentionally left empty.
	}
	rt.caps = capNetworkIsolation | capVolumeMounts | capEncryption | capHardwareIsolation
	return rt, true
}

// captureVersion invokes a runtime with the given version flag and returns the
// first line of stdout. Errors are suppressed — detection must not panic.
func captureVersion(path string, flag string) string {
	cmd := proc.NewCommand(path, flag)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	parts := core.SplitN(string(out), "\n", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
