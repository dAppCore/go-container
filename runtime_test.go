package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntime_Detect_Good(t *testing.T) {
	rt := Detect()

	// Detect must always return a valid runtime record — even the None zero value.
	assert.NotEmpty(t, rt.Type)
}

func TestRuntime_DetectAll_Good(t *testing.T) {
	runtimes := DetectAll()

	// Must not panic; slice may be empty on a host with no runtimes installed.
	assert.NotNil(t, runtimes)
	for _, rt := range runtimes {
		assert.NotEmpty(t, rt.Type)
	}
}

func TestRuntime_ContainerRuntime_Capabilities_Good(t *testing.T) {
	// Synthesise a runtime with every capability set and verify the predicates.
	rt := ContainerRuntime{
		Type: RuntimeApple,
		caps: capGPU | capNetworkIsolation | capVolumeMounts | capEncryption | capHardwareIsolation | capSubSecondStart,
	}

	assert.True(t, rt.HasGPU())
	assert.True(t, rt.HasNetworkIsolation())
	assert.True(t, rt.HasVolumeMounts())
	assert.True(t, rt.HasEncryption())
	assert.True(t, rt.IsHardwareIsolated())
	assert.True(t, rt.HasSubSecondStart())
	assert.NotZero(t, rt.Caps())
}

func TestRuntime_ContainerRuntime_NoCapabilities_Bad(t *testing.T) {
	rt := ContainerRuntime{Type: RuntimeNone}

	assert.False(t, rt.HasGPU())
	assert.False(t, rt.HasNetworkIsolation())
	assert.False(t, rt.HasVolumeMounts())
	assert.False(t, rt.HasEncryption())
	assert.False(t, rt.IsHardwareIsolated())
	assert.False(t, rt.HasSubSecondStart())
	assert.Zero(t, rt.Caps())
}

func TestRuntime_RequireGPU_Ugly(t *testing.T) {
	// RequireGPU must error when the runtime has no GPU capability,
	// and succeed when it does.
	noGPU := ContainerRuntime{Type: RuntimeDocker}
	assert.Error(t, RequireGPU(noGPU))

	gpu := ContainerRuntime{Type: RuntimeApple, caps: capGPU}
	assert.NoError(t, RequireGPU(gpu))
}

func TestRuntime_ProviderFor_UnsupportedType_Bad(t *testing.T) {
	_, err := ProviderFor(RuntimeDocker)

	assert.Error(t, err, "docker has no wired Provider yet")
}

func TestRuntime_ProviderFor_Unknown_Bad(t *testing.T) {
	_, err := ProviderFor(RuntimeType("not-a-runtime"))

	assert.Error(t, err)
}

func TestRuntime_HasRuntime_None_Good(t *testing.T) {
	// Asking for RuntimeNone never matches — even a pristine host would not
	// return None from DetectAll.
	assert.False(t, HasRuntime(RuntimeNone))
}
