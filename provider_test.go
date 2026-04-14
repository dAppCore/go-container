package container

import (
	"runtime"
	"testing"

	"dappco.re/go/core/io"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewProvider ---

func TestProvider_NewProvider_LinuxKitDefault_Good(t *testing.T) {
	p, err := NewProvider("", io.Local)

	require.NoError(t, err)
	_, ok := p.(*LinuxKitProvider)
	assert.True(t, ok, "empty runtime name must default to LinuxKit")
}

func TestProvider_NewProvider_ExplicitLinuxKit_Good(t *testing.T) {
	p, err := NewProvider("linuxkit", io.Local)

	require.NoError(t, err)
	_, ok := p.(*LinuxKitProvider)
	assert.True(t, ok)
}

func TestProvider_NewProvider_TIM_Good(t *testing.T) {
	p, err := NewProvider("tim", io.Local)

	require.NoError(t, err)
	_, ok := p.(*TIMProvider)
	assert.True(t, ok)
}

func TestProvider_NewProvider_UnknownRuntime_Bad(t *testing.T) {
	_, err := NewProvider("unicorn", io.Local)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported runtime")
}

func TestProvider_NewProvider_DockerUnimplemented_Bad(t *testing.T) {
	_, err := NewProvider(RuntimeTypeDocker, io.Local)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker")
}

func TestProvider_NewProvider_AppleWhenUnavailable_Ugly(t *testing.T) {
	if IsAppleAvailable() {
		t.Skip("apple runtime is available — tested in Good path")
	}
	_, err := NewProvider("apple", io.Local)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apple")
}

// --- AppleProvider ---

func TestProvider_NewAppleProvider_Good(t *testing.T) {
	p := NewAppleProvider()

	assert.NotNil(t, p)
	if runtime.GOOS == "darwin" && IsAppleAvailable() {
		assert.NotEmpty(t, p.runtime)
	}
}

func TestProvider_AppleProvider_Run_GPURefused_Bad(t *testing.T) {
	// Even with a fake runtime path, GPU passthrough is architecturally
	// unsupported and must error out before invoking the CLI.
	provider := &AppleProvider{runtime: "/path/to/container-binary"}
	image := &Image{ID: "aaa", Path: "/tmp/img", Runtime: "apple"}

	_, err := provider.Run(image, WithGPU(true))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "GPU")
}

func TestProvider_AppleProvider_Run_Unavailable_Bad(t *testing.T) {
	provider := &AppleProvider{}
	image := &Image{ID: "aaa", Path: "/tmp/img", Runtime: "apple"}

	_, err := provider.Run(image)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestProvider_AppleProvider_Build_MissingConfig_Ugly(t *testing.T) {
	provider := &AppleProvider{runtime: "/path/to/container-binary"}

	_, err := provider.Build(ContainerConfig{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing config")
}

// --- WithGPU / RunOption defaults ---

func TestProvider_RunOption_Defaults_Good(t *testing.T) {
	cfg := resolveRunConfig()

	assert.Empty(t, cfg.name)
	assert.False(t, cfg.detach)
	assert.Equal(t, 0, cfg.memory)
	assert.Equal(t, 0, cfg.cpus)
	assert.NotNil(t, cfg.ports)
	assert.NotNil(t, cfg.volumes)
	assert.False(t, cfg.gpu)
}

func TestProvider_RunOption_Overrides_Bad(t *testing.T) {
	cfg := resolveRunConfig(
		WithName(""),
		WithMemoryMB(-1),
	)

	// Negative and empty values are preserved verbatim — callers decide sanity.
	assert.Equal(t, "", cfg.name)
	assert.Equal(t, -1, cfg.memory)
}

func TestProvider_RunOption_Combined_Ugly(t *testing.T) {
	cfg := resolveRunConfig(
		WithName("alpha"),
		WithDetach(true),
		WithMemory("2G"),
		WithCPUs(4),
		WithPorts(map[int]int{8080: 80}),
		WithVolumes(map[string]string{"/src": "/app"}),
		WithSSHPort(2222),
		WithSSHKey("/tmp/key"),
		WithGPU(false),
	)

	assert.Equal(t, "alpha", cfg.name)
	assert.True(t, cfg.detach)
	assert.Equal(t, 2048, cfg.memory)
	assert.Equal(t, 4, cfg.cpus)
	assert.Equal(t, 80, cfg.ports[8080])
	assert.Equal(t, "/app", cfg.volumes["/src"])
	assert.Equal(t, 2222, cfg.sshPort)
	assert.Equal(t, "/tmp/key", cfg.sshKey)
	assert.False(t, cfg.gpu)
}

// --- Memory parsing ---

func TestProvider_ParseMemorySize_Good(t *testing.T) {
	assert.Equal(t, 1024, parseMemorySize("1G"))
	assert.Equal(t, 2048, parseMemorySize("2g"))
	assert.Equal(t, 512, parseMemorySize("512M"))
	assert.Equal(t, 1024, parseMemorySize("1024"))
}

func TestProvider_ParseMemorySize_Invalid_Bad(t *testing.T) {
	assert.Equal(t, 0, parseMemorySize(""))
	assert.Equal(t, 0, parseMemorySize("abc"))
	assert.Equal(t, 0, parseMemorySize("XG"))
}

func TestProvider_ParseMemorySize_Rounding_Ugly(t *testing.T) {
	// Fractional values round up.
	assert.Equal(t, 1536, parseMemorySize("1.5G"))
	// Kilobytes round up to the nearest MiB.
	assert.Equal(t, 1, parseMemorySize("1023K"))
}

// --- Image JSON round-trip ---

func TestProvider_MarshalImageJSON_Good(t *testing.T) {
	image := &Image{ID: "abc", Name: "demo", Path: "/tmp/x", Runtime: "linuxkit"}

	data, err := MarshalImageJSON(image)

	require.NoError(t, err)
	assert.Contains(t, string(data), "\"ID\"")
	assert.Contains(t, string(data), "\"abc\"")
}

func TestProvider_MarshalImageJSON_Nil_Bad(t *testing.T) {
	_, err := MarshalImageJSON(nil)
	require.Error(t, err)
}

func TestProvider_ParseImageJSON_RoundTrip_Ugly(t *testing.T) {
	image := &Image{ID: "abc", Name: "demo", Path: "/tmp/x", Runtime: "linuxkit"}

	data, err := MarshalImageJSON(image)
	require.NoError(t, err)

	parsed, err := ParseImageJSON(data)
	require.NoError(t, err)
	assert.Equal(t, image.ID, parsed.ID)
	assert.Equal(t, image.Name, parsed.Name)
	assert.Equal(t, image.Path, parsed.Path)
	assert.Equal(t, image.Runtime, parsed.Runtime)
}

func TestProvider_ParseImageJSON_Empty_Bad(t *testing.T) {
	_, err := ParseImageJSON(nil)
	require.Error(t, err)
}

// --- LinuxKit provider ---

func TestProvider_NewLinuxKitProvider_Good(t *testing.T) {
	p, err := NewLinuxKitProvider(io.Local)
	// On a bare system without hyperkit or qemu this may fail; tolerate either
	// outcome and verify the contract.
	if err != nil {
		assert.Nil(t, p)
		return
	}
	assert.NotNil(t, p)
}

func TestProvider_LinuxKitProvider_Run_NilProvider_Bad(t *testing.T) {
	var p *LinuxKitProvider
	_, err := p.Run(&Image{Path: "/tmp/x"})

	require.Error(t, err)
}

func TestProvider_LinuxKitProvider_Run_GPURefused_Ugly(t *testing.T) {
	p := NewLinuxKitProviderWithHypervisor(io.Local, NewState("/tmp/x.json"), NewQemuHypervisor())
	_, err := p.Run(&Image{Path: "/tmp/x.iso"}, WithGPU(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GPU")
}
