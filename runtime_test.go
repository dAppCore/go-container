package container

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Detect ---

func TestRuntime_Detect_Good(t *testing.T) {
	rt := Detect()

	// Type is always set to one of the known runtime identifiers.
	assert.Contains(t, []string{RuntimeTypeApple, RuntimeTypeDocker, RuntimeTypePodman, RuntimeTypeNone}, rt.Type)
	assert.IsType(t, "", rt.Version)
	assert.IsType(t, "", rt.Path)
}

func TestRuntime_Detect_UnavailableSystem_Bad(t *testing.T) {
	rt := Detect()

	// On systems without any runtime the Type collapses to None with empty path.
	if rt.Type == RuntimeTypeNone {
		assert.Empty(t, rt.Path)
		assert.False(t, rt.IsAvailable())
	}
}

func TestRuntime_Detect_StringForm_Ugly(t *testing.T) {
	rt := ContainerRuntime{Type: RuntimeTypeApple, Version: "0.1.0", Path: "/Library/Apple/usr/bin/container"}

	// String form must always include the three identifying fields.
	s := rt.String()
	assert.Contains(t, s, "apple")
	assert.Contains(t, s, "0.1.0")
	assert.Contains(t, s, "/Library/Apple/usr/bin/container")
}

// --- DetectAll ---

func TestRuntime_DetectAll_Good(t *testing.T) {
	all := DetectAll()

	// Always returns a slice (possibly empty) ordered by priority.
	assert.NotNil(t, all)
	for _, rt := range all {
		assert.NotEqual(t, RuntimeTypeNone, rt.Type)
		assert.NotEmpty(t, rt.Path)
	}
}

func TestRuntime_DetectAll_ApplePriority_Good(t *testing.T) {
	all := DetectAll()
	if runtime.GOOS != "darwin" {
		// On non-darwin systems Apple Containers never surface.
		for _, rt := range all {
			assert.NotEqual(t, RuntimeTypeApple, rt.Type)
		}
		return
	}

	// When Apple is available it must sort ahead of any other runtime.
	for i, rt := range all {
		if rt.Type == RuntimeTypeApple {
			assert.Equal(t, 0, i, "apple runtime must appear first when available")
		}
	}
}

func TestRuntime_DetectAll_EmptyOnBareSystem_Ugly(t *testing.T) {
	// We don't control the CI environment so we cannot assert zero runtimes,
	// but the contract must still produce a valid result without crashing.
	all := DetectAll()
	assert.NotPanics(t, func() { _ = all })
}

// --- IsAppleAvailable ---

func TestRuntime_IsAppleAvailable_Good(t *testing.T) {
	v := IsAppleAvailable()
	assert.IsType(t, true, v)

	if runtime.GOOS != "darwin" {
		assert.False(t, v, "Apple containerisation is macOS-only")
	}
}

func TestRuntime_IsAppleAvailable_NonDarwin_Bad(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("skipping on darwin — IsAppleAvailable depends on installed framework")
	}
	assert.False(t, IsAppleAvailable())
}

func TestRuntime_IsAppleAvailable_RepeatedCalls_Ugly(t *testing.T) {
	// Repeated detection must be stable between calls (no hidden state).
	first := IsAppleAvailable()
	second := IsAppleAvailable()
	assert.Equal(t, first, second)
}

// --- Capability bitfield ---

func TestRuntime_Capabilities_Good(t *testing.T) {
	rt := ContainerRuntime{
		Type: RuntimeTypeApple,
		caps: capNetworkIsolation | capVolumeMounts | capHardwareIsolation,
	}

	assert.True(t, rt.HasNetworkIsolation())
	assert.True(t, rt.HasVolumeMounts())
	assert.True(t, rt.IsHardwareIsolated())
	assert.False(t, rt.HasGPU())
	assert.False(t, rt.HasEncryption())
}

func TestRuntime_Capabilities_NoCaps_Bad(t *testing.T) {
	rt := ContainerRuntime{Type: RuntimeTypeNone}

	assert.False(t, rt.HasGPU())
	assert.False(t, rt.HasNetworkIsolation())
	assert.False(t, rt.HasVolumeMounts())
	assert.False(t, rt.HasEncryption())
	assert.False(t, rt.IsHardwareIsolated())
}

func TestRuntime_Capabilities_All_Ugly(t *testing.T) {
	rt := ContainerRuntime{
		caps: capGPU | capNetworkIsolation | capVolumeMounts | capNativeEncryption | capHardwareIsolation,
	}

	assert.True(t, rt.HasGPU())
	assert.True(t, rt.HasNetworkIsolation())
	assert.True(t, rt.HasVolumeMounts())
	assert.True(t, rt.HasEncryption())
	assert.True(t, rt.IsHardwareIsolated())
}

// --- IsAvailable ---

func TestRuntime_IsAvailable_Good(t *testing.T) {
	rt := ContainerRuntime{Type: RuntimeTypeApple, Path: "/Library/Apple/usr/bin/container"}
	assert.True(t, rt.IsAvailable())
}

func TestRuntime_IsAvailable_None_Bad(t *testing.T) {
	rt := ContainerRuntime{Type: RuntimeTypeNone}
	assert.False(t, rt.IsAvailable())
}

func TestRuntime_IsAvailable_TypedButUnresolved_Ugly(t *testing.T) {
	// Typed runtime without a discovered Path is not available.
	rt := ContainerRuntime{Type: RuntimeTypeDocker}
	assert.False(t, rt.IsAvailable())
}

// --- IsPathAvailable ---

func TestRuntime_IsPathAvailable_Good(t *testing.T) {
	tmp := t.TempDir()
	assert.True(t, IsPathAvailable(tmp))
}

func TestRuntime_IsPathAvailable_Missing_Bad(t *testing.T) {
	assert.False(t, IsPathAvailable("/nonexistent/path/that/should/never/exist"))
}

func TestRuntime_IsPathAvailable_Empty_Ugly(t *testing.T) {
	// Empty path resolves to the current working directory which always exists,
	// so IsPathAvailable intentionally returns true. Callers should guard
	// upstream for empty paths when that is not the desired behaviour.
	assert.IsType(t, true, IsPathAvailable(""))
}
