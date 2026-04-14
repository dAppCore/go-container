package container

import (
	"fmt"
	"runtime"
	"strings"

	"dappco.re/go/core/container/internal/proc"
	"dappco.re/go/core/io"
)

// RuntimeType identifies a discovered runtime.
const (
	RuntimeTypeApple  = "apple"
	RuntimeTypeDocker = "docker"
	RuntimeTypePodman = "podman"
	RuntimeTypeNone   = "none"
)

// Capability flags for ContainerRuntime.
const (
	capGPU uint32 = 1 << iota
	capNetworkIsolation
	capVolumeMounts
	capNativeEncryption
	capHardwareIsolation
)

// ContainerRuntime describes a detected container runtime.
type ContainerRuntime struct {
	Type    string
	Version string
	Path    string
	caps    uint32
}

func (r ContainerRuntime) HasGPU() bool {
	return r.caps&capGPU != 0
}

func (r ContainerRuntime) HasNetworkIsolation() bool {
	return r.caps&capNetworkIsolation != 0
}

func (r ContainerRuntime) HasVolumeMounts() bool {
	return r.caps&capVolumeMounts != 0
}

func (r ContainerRuntime) HasEncryption() bool {
	return r.caps&capNativeEncryption != 0
}

func (r ContainerRuntime) IsHardwareIsolated() bool {
	return r.caps&capHardwareIsolation != 0
}

func (r ContainerRuntime) String() string {
	return fmt.Sprintf("%s %s at %s", r.Type, r.Version, r.Path)
}

func (r ContainerRuntime) IsAvailable() bool {
	return r.Type != RuntimeTypeNone && r.Path != ""
}

// Detect probes available runtimes and returns the highest-priority runtime.
func Detect() ContainerRuntime {
	runtimes := DetectAll()
	if len(runtimes) == 0 {
		return ContainerRuntime{Type: RuntimeTypeNone}
	}
	return runtimes[0]
}

// DetectAll discovers available container runtimes in priority order.
func DetectAll() []ContainerRuntime {
	runtimes := make([]ContainerRuntime, 0, 4)

	if rt := detectAppleRuntime(); rt.Type != RuntimeTypeNone {
		runtimes = append(runtimes, rt)
	}

	if rt := detectRuntimeCandidate(RuntimeTypeDocker, "docker", capNetworkIsolation|capVolumeMounts); rt.Type != RuntimeTypeNone {
		runtimes = append(runtimes, rt)
	}

	if rt := detectRuntimeCandidate(RuntimeTypePodman, "podman", capNetworkIsolation|capVolumeMounts); rt.Type != RuntimeTypeNone {
		runtimes = append(runtimes, rt)
	}

	return runtimes
}

// IsAppleAvailable checks whether Apple's Containerization runtime is available.
func IsAppleAvailable() bool {
	return detectAppleRuntime().Type == RuntimeTypeApple
}

func detectAppleRuntime() ContainerRuntime {
	if runtime.GOOS != "darwin" {
		return ContainerRuntime{Type: RuntimeTypeNone}
	}

	path, ok := detectRuntimePath("container", []string{
		"/Library/Apple/usr/bin/container",
		"/usr/local/bin/container",
		"/usr/bin/container",
	})
	if !ok {
		return ContainerRuntime{Type: RuntimeTypeNone}
	}

	version := detectRuntimeVersion(path, "--version")
	if version == "" {
		version = "unknown"
	}

	return ContainerRuntime{
		Type:    RuntimeTypeApple,
		Version: version,
		Path:    path,
		caps:    capNetworkIsolation | capVolumeMounts | capHardwareIsolation,
	}
}

func detectRuntimeCandidate(name, binary string, caps uint32) ContainerRuntime {
	path, err := proc.LookPath(binary)
	if err != nil {
		return ContainerRuntime{Type: RuntimeTypeNone}
	}

	version := detectRuntimeVersion(path, "--version")
	if version == "" {
		version = "unknown"
	}

	return ContainerRuntime{
		Type:    name,
		Version: version,
		Path:    path,
		caps:    caps,
	}
}

func detectRuntimeVersion(path, arg string) string {
	cmd := proc.NewCommand(path, arg)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, token := range strings.Fields(string(out)) {
		candidate := strings.Trim(token, ",")
		if isVersionCandidate(candidate) {
			return candidate
		}
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return ""
	}
	return text
}

func isVersionCandidate(value string) bool {
	hasDigit := false
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '.' || r == '-' || r == '_' || r == '+' || r == 'v' || r == 'V':
		default:
			return false
		}
	}
	if !hasDigit {
		return false
	}
	return true
}

func detectRuntimePath(binary string, fallback []string) (string, bool) {
	for _, path := range fallback {
		if IsPathAvailable(path) {
			return path, true
		}
	}
	path, err := proc.LookPath(binary)
	if err != nil {
		return "", false
	}
	return path, true
}

// IsPathAvailable checks whether a path exists.
func IsPathAvailable(path string) bool {
	return io.Local.Exists(path)
}
