package container

// Provider abstracts the container backend (LinuxKit, TIM, Apple Containers).
// Each provider implements a consistent lifecycle: build an image from a
// declarative config, run it, and optionally encrypt/decrypt the image.
//
// Usage:
//
//	p := container.NewAppleProvider()
//	img, err := p.Build(container.ContainerConfig{EntryPoint: []string{"/app"}})
//	ctr, err := p.Run(img, container.WithMemory(4096))
type Provider interface {
	// Build produces an Image from a declarative container configuration.
	//
	// Example: img, _ := p.Build(container.ContainerConfig{Name: "api"})
	Build(config ContainerConfig) (*Image, error)

	// Run boots an Image and returns the running Container record.
	//
	// Example: ctr, _ := p.Run(img, container.WithMemory(2048))
	Run(image *Image, opts ...RunOption) (*Container, error)

	// Encrypt wraps an Image with an encryption key producing an EncryptedImage.
	//
	// Example: enc, _ := p.Encrypt(img, workspaceKey)
	Encrypt(image *Image, key []byte) (*EncryptedImage, error)

	// Decrypt unwraps an EncryptedImage back into a plaintext Image.
	//
	// Example: img, _ := p.Decrypt(enc, workspaceKey)
	Decrypt(encrypted *EncryptedImage, key []byte) (*Image, error)
}

// ContainerConfig is the declarative build input for a Provider.
// Different providers map this to their native format (LinuxKit YAML,
// TIM config.json, Apple Containers spec).
//
// Usage:
//
//	cfg := container.ContainerConfig{
//	    Name:       "api",
//	    EntryPoint: []string{"/app/server"},
//	    Env:        []string{"CORE_ENV=production"},
//	}
type ContainerConfig struct {
	// Name is an optional identifier for the image.
	Name string
	// EntryPoint is the process executed on start (argv[0..]).
	EntryPoint []string
	// Env is the container environment in KEY=VALUE form.
	Env []string
	// WorkDir is the initial working directory.
	WorkDir string
	// Mounts describes host→container filesystem mappings.
	Mounts []Mount
	// Capabilities lists Linux capabilities granted to the container.
	Capabilities []string
	// ReadOnly mounts the root filesystem read-only.
	ReadOnly bool
	// Memory requests memory allocation in MB. Zero uses provider defaults.
	Memory int
	// CPUs requests CPU allocation. Zero uses provider defaults.
	CPUs int
	// Ports maps host ports to container ports.
	Ports map[int]int
	// Format is the requested output format (iso, qcow2, raw, vmdk, ami).
	// Empty uses the provider default.
	Format string
	// Source is the source image reference (LinuxKit YAML path, TIM bundle path, etc).
	Source string
}

// Mount describes a single filesystem mount into a container.
//
// Usage:
//
//	mount := container.Mount{Source: "/data", Target: "/app/data", ReadOnly: true}
type Mount struct {
	// Source is the host-side path.
	Source string
	// Target is the container-side mount point.
	Target string
	// ReadOnly mounts the path read-only.
	ReadOnly bool
}

// Image is the built container artefact returned by Provider.Build.
// The concrete on-disk layout depends on the Provider type.
//
// Usage:
//
//	img, _ := p.Build(config)
//	fmt.Println(img.Path, img.Format)
type Image struct {
	// ID is a unique image identifier (8 character hex).
	ID string
	// Name is an optional human-readable identifier.
	Name string
	// Path is the on-disk location of the built artefact.
	Path string
	// Format identifies the image format (iso, qcow2, raw, vmdk, tim).
	Format ImageFormat
	// Provider names the backend that produced this image.
	Provider string
	// Digest is an optional content digest (e.g. sha256:...).
	Digest string
	// Size is the image size in bytes.
	Size int64
}

// EncryptedImage is an Image with its payload encrypted by a Provider.
// The concrete encryption scheme depends on the Provider:
//   - LinuxKit uses dm-crypt wrapping the writable volume.
//   - TIM uses Enchantrix sigil-chain (STIM).
//   - Apple Containers uses host FileVault plus optional STIM.
//
// Usage:
//
//	enc, _ := p.Encrypt(img, key)
//	img, _  = p.Decrypt(enc, key)
type EncryptedImage struct {
	// ID is a unique encrypted image identifier.
	ID string
	// Path is the on-disk location of the encrypted artefact.
	Path string
	// Provider names the backend that produced this encryption.
	Provider string
	// Scheme names the encryption scheme (dm-crypt, stim, filevault).
	Scheme string
	// KeyHint is an optional non-secret key identifier.
	KeyHint string
	// Size is the encrypted image size in bytes.
	Size int64
}

// RunOption configures a Provider.Run call. Options compose functionally so
// providers can ignore options they do not support without error.
//
// Usage:
//
//	ctr, _ := p.Run(img, container.WithMemory(4096), container.WithGPU(true))
type RunOption func(*RunOptions)

// WithName assigns a human-readable name to the running container.
//
// Usage:
//
//	p.Run(img, container.WithName("api"))
func WithName(name string) RunOption {
	return func(o *RunOptions) {
		o.Name = name
	}
}

// WithMemory sets the memory allocation in MB for the container.
//
// Usage:
//
//	p.Run(img, container.WithMemory(4096))
func WithMemory(mb int) RunOption {
	return func(o *RunOptions) {
		o.Memory = mb
	}
}

// WithCPUs sets the CPU allocation for the container.
//
// Usage:
//
//	p.Run(img, container.WithCPUs(4))
func WithCPUs(cpus int) RunOption {
	return func(o *RunOptions) {
		o.CPUs = cpus
	}
}

// WithDetach starts the container in the background.
//
// Usage:
//
//	p.Run(img, container.WithDetach(true))
func WithDetach(detach bool) RunOption {
	return func(o *RunOptions) {
		o.Detach = detach
	}
}

// WithPorts adds host→container port forwards to the RunOptions.
//
// Usage:
//
//	p.Run(img, container.WithPorts(map[int]int{8080: 80}))
func WithPorts(ports map[int]int) RunOption {
	return func(o *RunOptions) {
		if o.Ports == nil {
			o.Ports = make(map[int]int, len(ports))
		}
		for h, c := range ports {
			o.Ports[h] = c
		}
	}
}

// WithVolumes adds host→container volume mounts to the RunOptions.
//
// Usage:
//
//	p.Run(img, container.WithVolumes(map[string]string{"/data": "/app/data"}))
func WithVolumes(vols map[string]string) RunOption {
	return func(o *RunOptions) {
		if o.Volumes == nil {
			o.Volumes = make(map[string]string, len(vols))
		}
		for h, c := range vols {
			o.Volumes[h] = c
		}
	}
}

// ApplyRunOptions folds a slice of RunOption functions into a RunOptions.
//
// Usage:
//
//	opts := container.ApplyRunOptions(container.WithMemory(2048), container.WithCPUs(2))
func ApplyRunOptions(opts ...RunOption) RunOptions {
	var o RunOptions
	for _, apply := range opts {
		if apply != nil {
			apply(&o)
		}
	}
	return o
}
