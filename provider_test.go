package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvider_ApplyRunOptions_Good(t *testing.T) {
	opts := ApplyRunOptions(
		WithName("api"),
		WithMemory(2048),
		WithCPUs(4),
		WithDetach(true),
		WithPorts(map[int]int{8080: 80}),
		WithVolumes(map[string]string{"/data": "/app/data"}),
	)

	assert.Equal(t, "api", opts.Name)
	assert.Equal(t, 2048, opts.Memory)
	assert.Equal(t, 4, opts.CPUs)
	assert.True(t, opts.Detach)
	assert.Equal(t, 80, opts.Ports[8080])
	assert.Equal(t, "/app/data", opts.Volumes["/data"])
}

func TestProvider_ApplyRunOptions_NilOption_Bad(t *testing.T) {
	// Nil options must be skipped without panicking.
	opts := ApplyRunOptions(nil, WithName("ok"), nil)

	assert.Equal(t, "ok", opts.Name)
}

func TestProvider_ApplyRunOptions_OverwriteAndMerge_Ugly(t *testing.T) {
	// Applying two WithPorts calls merges maps; applying two WithMemory calls overwrites.
	opts := ApplyRunOptions(
		WithMemory(1024),
		WithMemory(4096),
		WithPorts(map[int]int{8080: 80}),
		WithPorts(map[int]int{9090: 90}),
	)

	assert.Equal(t, 4096, opts.Memory)
	assert.Equal(t, 80, opts.Ports[8080])
	assert.Equal(t, 90, opts.Ports[9090])
}

func TestProvider_WithGPU_Good(t *testing.T) {
	opts := ApplyRunOptions(WithGPU(true))

	assert.True(t, opts.GPU)
}

func TestProvider_WithGPU_Disabled_Bad(t *testing.T) {
	opts := ApplyRunOptions(WithGPU(false))

	assert.False(t, opts.GPU)
}

func TestProvider_WithGPU_OverriddenByLater_Ugly(t *testing.T) {
	opts := ApplyRunOptions(WithGPU(true), WithGPU(false))

	assert.False(t, opts.GPU)
}
