package container

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/container/internal/proc"
	"dappco.re/go/core/io"
)

// Provider defines a generic image lifecycle interface.
type Provider interface {
	Build(config ContainerConfig) (*Image, error)
	Run(image *Image, opts ...RunOption) (*Container, error)
	Encrypt(image *Image, key []byte) (*EncryptedImage, error)
	Decrypt(encrypted *EncryptedImage, key []byte) (*Image, error)
}

// ContainerConfig is a generic input config used by providers.
type ContainerConfig struct {
	Name     string
	Path     string
	Source   string
	Content  string
	Runtime  string
	Metadata map[string]string
}

// Image is a provider-produced image artifact.
type Image struct {
	ID       string
	Name     string
	Path     string
	Runtime  string
	Metadata map[string]string
}

// EncryptedImage is a provider-produced encrypted image artifact.
type EncryptedImage struct {
	ID       string
	Name     string
	Path     string
	Runtime  string
	Metadata map[string]string
}

type runConfig struct {
	name    string
	detach  bool
	memory  int
	cpus    int
	ports   map[int]int
	volumes map[string]string
	sshPort int
	sshKey  string
	gpu     bool
}

// RunOption configures container run behaviour.
type RunOption func(*runConfig)

// WithName sets the container name.
func WithName(name string) RunOption {
	return func(cfg *runConfig) {
		cfg.name = name
	}
}

// WithDetach enables/disable detached mode.
func WithDetach(detach bool) RunOption {
	return func(cfg *runConfig) {
		cfg.detach = detach
	}
}

// WithMemory sets container memory in human-readable form.
//
// Supported values: "1024", "1024M", "2G", "4g", "512m".
func WithMemory(size string) RunOption {
	return func(cfg *runConfig) {
		cfg.memory = parseMemorySize(size)
	}
}

// WithMemoryMB sets container memory directly in MiB.
func WithMemoryMB(mb int) RunOption {
	return func(cfg *runConfig) {
		cfg.memory = mb
	}
}

// WithCPUs sets container CPU count.
func WithCPUs(cpus int) RunOption {
	return func(cfg *runConfig) {
		cfg.cpus = cpus
	}
}

// WithPorts sets host:guest port forwarding.
func WithPorts(ports map[int]int) RunOption {
	return func(cfg *runConfig) {
		cfg.ports = ports
	}
}

// WithVolumes sets host:container volume mapping.
func WithVolumes(volumes map[string]string) RunOption {
	return func(cfg *runConfig) {
		cfg.volumes = volumes
	}
}

// WithSSHPort sets the forwarded SSH port.
func WithSSHPort(port int) RunOption {
	return func(cfg *runConfig) {
		cfg.sshPort = port
	}
}

// WithSSHKey sets SSH key path for exec/management.
func WithSSHKey(path string) RunOption {
	return func(cfg *runConfig) {
		cfg.sshKey = path
	}
}

// WithGPU requests GPU passthrough.
func WithGPU(enabled bool) RunOption {
	return func(cfg *runConfig) {
		cfg.gpu = enabled
	}
}

func defaultRunConfig() *runConfig {
	return &runConfig{
		ports:   map[int]int{},
		volumes: map[string]string{},
	}
}

func parseMemorySize(value string) int {
	v := strings.TrimSpace(value)
	if v == "" {
		return 0
	}

	upper := strings.ToUpper(v)
	last := upper[len(upper)-1]
	switch last {
	case 'K', 'M', 'G', 'T':
		number := strings.TrimSpace(upper[:len(upper)-1])
		magnitude, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return 0
		}
		switch last {
		case 'K':
			return int(math.Ceil(magnitude / 1024))
		case 'M':
			return int(math.Ceil(magnitude))
		case 'G':
			return int(math.Ceil(magnitude * 1024))
		case 'T':
			return int(math.Ceil(magnitude * 1024 * 1024))
		}
	}

	parsed, err := strconv.Atoi(upper)
	if err != nil {
		return 0
	}
	return parsed
}

func resolveRunConfig(opts ...RunOption) *runConfig {
	cfg := defaultRunConfig()
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

func normalizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return map[string]string{}
	}
	copyMetadata := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copyMetadata[key] = value
	}
	return copyMetadata
}

func isImageConfigPath(path string) bool {
	ext := core.Lower(core.PathExt(path))
	return ext == ".yml" || ext == ".yaml"
}

func findBuiltImage(basePath string) string {
	extensions := []string{".ami", ".qcow2", ".raw", ".vmdk", ".iso", "-bios.iso"}

	for _, ext := range extensions {
		candidate := core.Concat(basePath, ext)
		if io.Local.IsFile(candidate) {
			return candidate
		}
	}

	base := filepath.Base(basePath)
	dir := filepath.Dir(basePath)
	entries, err := io.Local.List(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		name := entry.Name()
		if !core.HasPrefix(name, base) {
			continue
		}
		for _, ext := range extensions {
			if core.HasSuffix(name, ext) {
				return coreutil.JoinPath(dir, name)
			}
		}
	}

	return ""
}

func lookupLinuxKit() (string, error) {
	if path, err := proc.LookPath("linuxkit"); err == nil {
		return path, nil
	}

	paths := []string{"/usr/local/bin/linuxkit", "/opt/homebrew/bin/linuxkit"}
	for _, p := range paths {
		if io.Local.Exists(p) {
			return p, nil
		}
	}

	return "", coreerr.E("lookupLinuxKit", "linuxkit executable not found", nil)
}

// NewProvider returns a provider by runtime name.
//
// Supported runtime values: "linuxkit" (default), "apple", "tim".
func NewProvider(runtimeName string, m io.Medium) (Provider, error) {
	target := strings.TrimSpace(strings.ToLower(runtimeName))
	if target == "" {
		target = "linuxkit"
	}
	if target == "auto" {
		rt := Detect()
		target = rt.Type
	}

	switch target {
	case "", "linuxkit":
		return NewLinuxKitProvider(m)
	case "apple":
		provider := NewAppleProvider()
		if provider == nil || provider.runtime == "" {
			return nil, coreerr.E("NewProvider", "apple runtime not available", nil)
		}
		return provider, nil
	case "tim":
		return NewTIMProvider(), nil
	case RuntimeTypeDocker:
		return nil, coreerr.E("NewProvider", "docker provider not implemented", nil)
	case RuntimeTypePodman:
		return nil, coreerr.E("NewProvider", "podman provider not implemented", nil)
	default:
		return nil, coreerr.E("NewProvider", "unsupported runtime: "+target, nil)
	}
}

// LinuxKitProvider implements Provider for LinuxKit images.
type LinuxKitProvider struct {
	m     io.Medium
	hv    Hypervisor
	state *State
}

// Compile-time interface check.
var _ Provider = (*LinuxKitProvider)(nil)

// NewLinuxKitProvider creates a LinuxKit provider.
func NewLinuxKitProvider(m io.Medium) (*LinuxKitProvider, error) {
	statePath, err := DefaultStatePath()
	if err != nil {
		return nil, coreerr.E("NewLinuxKitProvider", "failed to determine state path", err)
	}

	state, err := LoadState(statePath)
	if err != nil {
		return nil, coreerr.E("NewLinuxKitProvider", "failed to load state", err)
	}

	hypervisor, err := DetectHypervisor()
	if err != nil {
		return nil, err
	}

	return &LinuxKitProvider{
		m:     m,
		hv:    hypervisor,
		state: state,
	}, nil
}

// NewLinuxKitProviderWithHypervisor injects custom dependencies.
func NewLinuxKitProviderWithHypervisor(m io.Medium, state *State, hv Hypervisor) *LinuxKitProvider {
	return &LinuxKitProvider{
		m:     m,
		hv:    hv,
		state: state,
	}
}

// Build returns a LinuxKit image from config.
func (p *LinuxKitProvider) Build(config ContainerConfig) (*Image, error) {
	if p == nil {
		return nil, coreerr.E("LinuxKitProvider.Build", "provider is nil", nil)
	}

	cfgPath := strings.TrimSpace(config.Path)
	if cfgPath == "" {
		cfgPath = strings.TrimSpace(config.Source)
	}
	if cfgPath == "" {
		if config.Content == "" {
			return nil, coreerr.E("LinuxKitProvider.Build", "missing config content or path", nil)
		}

		tmpDir, err := coreutil.MkdirTemp("core-lk-config-")
		if err != nil {
			return nil, coreerr.E("LinuxKitProvider.Build", "create config temp dir", err)
		}

		name := strings.TrimSpace(config.Name)
		if name == "" {
			name = "container"
		}
		cfgPath = coreutil.JoinPath(tmpDir, core.Concat(name, ".yml"))
		if err := io.Local.Write(cfgPath, config.Content); err != nil {
			return nil, coreerr.E("LinuxKitProvider.Build", "write config", err)
		}
	}

	if !isImageConfigPath(cfgPath) && DetectImageFormat(cfgPath) != FormatUnknown {
		imageID, err := GenerateID()
		if err != nil {
			return nil, coreerr.E("LinuxKitProvider.Build", "generate image id", err)
		}
		return &Image{
			ID:       imageID,
			Name:     strings.TrimSpace(config.Name),
			Path:     cfgPath,
			Runtime:  "linuxkit",
			Metadata: normalizeMetadata(config.Metadata),
		}, nil
	}

	lkPath, err := lookupLinuxKit()
	if err != nil {
		return nil, err
	}

	tmpBuildDir, err := coreutil.MkdirTemp("core-lk-build-")
	if err != nil {
		return nil, coreerr.E("LinuxKitProvider.Build", "create build temp dir", err)
	}

	imageID, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("LinuxKitProvider.Build", "generate image id", err)
	}

	baseName := strings.TrimSpace(config.Name)
	if baseName == "" {
		baseName = core.PathBase(cfgPath)
		baseName = strings.TrimSuffix(baseName, core.PathExt(baseName))
		if baseName == "" {
			baseName = "container"
		}
	}

	outputPath := coreutil.JoinPath(tmpBuildDir, core.Concat(baseName, "-", imageID[:4]))

	cmd := proc.NewCommand(lkPath, "build", "--format", "qcow2", "--name", outputPath, cfgPath)
	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr
	if err := cmd.Run(); err != nil {
		return nil, coreerr.E("LinuxKitProvider.Build", "linuxkit build", err)
	}

	imagePath := findBuiltImage(outputPath)
	if imagePath == "" {
		return nil, coreerr.E("LinuxKitProvider.Build", "no image produced", nil)
	}

	return &Image{
		ID:       imageID,
		Name:     strings.TrimSpace(config.Name),
		Path:     imagePath,
		Runtime:  "linuxkit",
		Metadata: normalizeMetadata(config.Metadata),
	}, nil
}

// Run executes a LinuxKit image.
func (p *LinuxKitProvider) Run(image *Image, opts ...RunOption) (*Container, error) {
	if p == nil || p.state == nil {
		return nil, coreerr.E("LinuxKitProvider.Run", "provider not initialised", nil)
	}
	if image == nil {
		return nil, coreerr.E("LinuxKitProvider.Run", "missing image", nil)
	}
	if image.Path == "" {
		return nil, coreerr.E("LinuxKitProvider.Run", "image path missing", nil)
	}

	cfg := resolveRunConfig(opts...)
	if cfg.gpu {
		return nil, coreerr.E("LinuxKitProvider.Run", "GPU passthrough is not implemented for LinuxKit provider", nil)
	}

	manager := &LinuxKitManager{
		state:      p.state,
		hypervisor: p.hv,
		medium:     p.m,
	}

	container, err := manager.Run(context.Background(), image.Path, RunOptions{
		Name:    cfg.name,
		Detach:  cfg.detach,
		Memory:  cfg.memory,
		CPUs:    cfg.cpus,
		Ports:   cfg.ports,
		Volumes: cfg.volumes,
		SSHPort: cfg.sshPort,
		SSHKey:  cfg.sshKey,
	})
	if err != nil {
		return nil, coreerr.E("LinuxKitProvider.Run", "run linuxkit image", err)
	}

	return container, nil
}

// Encrypt is not currently supported for LinuxKit images.
func (p *LinuxKitProvider) Encrypt(_ *Image, _ []byte) (*EncryptedImage, error) {
	return nil, coreerr.E("LinuxKitProvider.Encrypt", "linuxkit encryption is not supported in this package", nil)
}

// Decrypt is not currently supported for LinuxKit images.
func (p *LinuxKitProvider) Decrypt(_ *EncryptedImage, _ []byte) (*Image, error) {
	return nil, coreerr.E("LinuxKitProvider.Decrypt", "linuxkit decryption is not supported in this package", nil)
}

// AppleProvider implements Provider for Apple Containers.
type AppleProvider struct {
	runtime string
	version string
}

// Compile-time interface check.
var _ Provider = (*AppleProvider)(nil)

// NewAppleProvider creates a provider backed by Apple Containers.
func NewAppleProvider() *AppleProvider {
	rt := DetectAll()
	for _, candidate := range rt {
		if candidate.Type == "apple" {
			return &AppleProvider{runtime: candidate.Path, version: candidate.Version}
		}
	}
	return &AppleProvider{}
}

// Build validates an Apple container config and returns an image wrapper.
func (a *AppleProvider) Build(config ContainerConfig) (*Image, error) {
	if a == nil || a.runtime == "" {
		return nil, coreerr.E("AppleProvider.Build", "apple provider unavailable", nil)
	}

	cfgPath := strings.TrimSpace(config.Path)
	if cfgPath == "" {
		cfgPath = strings.TrimSpace(config.Source)
	}
	if cfgPath == "" {
		if config.Content == "" {
			return nil, coreerr.E("AppleProvider.Build", "missing config content or path", nil)
		}
		tmpDir, err := coreutil.MkdirTemp("core-apple-config-")
		if err != nil {
			return nil, coreerr.E("AppleProvider.Build", "create config temp dir", err)
		}
		name := strings.TrimSpace(config.Name)
		if name == "" {
			name = "container"
		}
		cfgPath = coreutil.JoinPath(tmpDir, core.Concat(name, ".yml"))
		if err := io.Local.Write(cfgPath, config.Content); err != nil {
			return nil, coreerr.E("AppleProvider.Build", "write config", err)
		}
	}

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Build", "generate image id", err)
	}

	return &Image{
		ID:       id,
		Name:     strings.TrimSpace(config.Name),
		Path:     cfgPath,
		Runtime:  "apple",
		Metadata: normalizeMetadata(config.Metadata),
	}, nil
}

// Run starts an Apple container image.
func (a *AppleProvider) Run(image *Image, opts ...RunOption) (*Container, error) {
	if a == nil || a.runtime == "" {
		return nil, coreerr.E("AppleProvider.Run", "apple provider unavailable", nil)
	}
	if image == nil {
		return nil, coreerr.E("AppleProvider.Run", "missing image", nil)
	}
	if image.Path == "" {
		return nil, coreerr.E("AppleProvider.Run", "image path missing", nil)
	}

	cfg := resolveRunConfig(opts...)
	_ = cfg

	if cfg.gpu {
		return nil, coreerr.E("AppleProvider.Run", "Apple provider does not expose GPU capability", nil)
	}

	// Apple Containers are expected to be executed via containerd bindings.
	return nil, coreerr.E("AppleProvider.Run", "apple runtime execution is not wired in this build", nil)
}

// Encrypt is not yet supported for Apple container images.
func (a *AppleProvider) Encrypt(_ *Image, _ []byte) (*EncryptedImage, error) {
	return nil, coreerr.E("AppleProvider.Encrypt", "apple encryption is delegated to STIM workflow", nil)
}

// Decrypt is not yet supported for Apple container images.
func (a *AppleProvider) Decrypt(_ *EncryptedImage, _ []byte) (*Image, error) {
	return nil, coreerr.E("AppleProvider.Decrypt", "apple decryption is delegated to STIM workflow", nil)
}

// MarshalImageJSON marshals image metadata for external persistence.
func MarshalImageJSON(image *Image) ([]byte, error) {
	if image == nil {
		return nil, coreerr.E("MarshalImageJSON", "missing image", nil)
	}
	return json.Marshal(image)
}

// ParseImageJSON parses an image from JSON bytes.
func ParseImageJSON(data []byte) (*Image, error) {
	if len(data) == 0 {
		return nil, coreerr.E("ParseImageJSON", "empty image payload", nil)
	}
	var image Image
	if err := json.Unmarshal(data, &image); err != nil {
		return nil, coreerr.E("ParseImageJSON", "decode image", err)
	}
	return &image, nil
}
