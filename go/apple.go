package container

import (
	"context"
	"time"

	core "dappco.re/go"
	coreerr "dappco.re/go/log"

	"dappco.re/go/container/internal/proc"
)

var appleProviderLock = core.New().Lock("container.apple.provider").Mutex

// IsAppleAvailable checks whether Apple's Containerisation framework (the
// `container` CLI shipped with macOS 26+) is present on the current system.
//
// Usage:
//
//	if container.IsAppleAvailable() {
//	    provider = container.NewAppleProvider()
//	}
func IsAppleAvailable() bool {
	if discoverHostOS() != "darwin" {
		return false
	}
	_, err := proc.LookPath("container")
	return err == nil
}

// AppleProvider implements the Provider interface using Apple's
// Containerisation framework. It shells out to the `container` CLI that
// ships with macOS 26+.
//
// Usage:
//
//	p := container.NewAppleProvider()
//	img, _ := p.Build(container.ContainerConfig{Source: "app.yml"})
//	ctr, _ := p.Run(img, container.WithMemory(4096))
type AppleProvider struct {
	// Binary is the `container` CLI binary name or path.
	Binary string
	// Version is the detected framework version (populated when known).
	Version string

	tracked map[string]*appleTracked
}

// appleTracked records a detached apple container process for lifecycle
// observation. The AppleProvider populates this map on Run and drains it when
// the underlying process exits.
type appleTracked struct {
	Container *Container
	Cmd       *proc.Command
	Done      chan struct{}
}

// NewAppleProvider returns an AppleProvider configured with the default
// Apple container binary name.
//
// Usage:
//
//	p := container.NewAppleProvider()
func NewAppleProvider() *AppleProvider {
	return &AppleProvider{Binary: "container"}
}

// Available reports whether the AppleProvider can run on this host.
//
// Usage:
//
//	if provider.Available() { provider.Run(img) }
func (a *AppleProvider) Available() bool {
	if discoverHostOS() != "darwin" {
		return false
	}
	if a.Binary == "" {
		a.Binary = "container"
	}
	_, err := proc.LookPath(a.Binary)
	return err == nil
}

// Build produces an Image from a declarative configuration. For Apple
// containers the Source field must reference an existing OCI image tag or
// Containerfile path recognised by the `container build` subcommand.
//
// Usage:
//
//	img, _ := provider.Build(container.ContainerConfig{Source: "./Containerfile"})
func (a *AppleProvider) Build(config ContainerConfig) (
	*Image,
	error,
) {
	if !a.Available() {
		return nil, coreerr.E("AppleProvider.Build", "apple container runtime not available on this host", nil)
	}
	if config.Source == "" {
		return nil, coreerr.E("AppleProvider.Build", "ContainerConfig.Source is required", nil)
	}

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Build", "generate image id", err)
	}

	name := config.Name
	if name == "" {
		name = id
	}

	return &Image{
		ID:       id,
		Name:     name,
		Path:     config.Source,
		Format:   FormatUnknown,
		Provider: string(RuntimeApple),
	}, nil
}

// Run boots an Image using the `container run` subcommand. RunOptions are
// translated into CLI flags. Ports and volume mounts are forwarded.
//
// Usage:
//
//	ctr, _ := provider.Run(img, container.WithMemory(2048), container.WithCPUs(2))
func (a *AppleProvider) Run(image *Image, opts ...RunOption) (
	*Container,
	error,
) {
	if !a.Available() {
		return nil, coreerr.E("AppleProvider.Run", "apple container runtime not available on this host", nil)
	}
	if image == nil || image.Path == "" {
		return nil, coreerr.E("AppleProvider.Run", "image is required", nil)
	}

	ro := ApplyRunOptions(opts...)

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Run", "generate container id", err)
	}
	name := ro.Name
	if name == "" {
		if image.Name != "" {
			name = image.Name
		} else {
			name = id
		}
	}

	args := []string{"run", "--name", name}
	if ro.Detach {
		args = append(args, "--detach")
	}
	if ro.Memory > 0 {
		args = append(args, "--memory", core.Sprintf("%dM", ro.Memory))
	}
	if ro.CPUs > 0 {
		args = append(args, "--cpus", core.Sprintf("%d", ro.CPUs))
	}
	for host, guest := range ro.Ports {
		args = append(args, "--publish", core.Sprintf("%d:%d", host, guest))
	}
	for host, guest := range ro.Volumes {
		args = append(args, "--volume", core.Sprintf("%s:%s", host, guest))
	}
	if ro.GPU {
		args = append(args, "--gpu")
	}
	args = append(args, image.Path)

	cmd := proc.NewCommandContext(context.Background(), a.Binary, args...)
	if err := cmd.Start(); err != nil {
		return nil, coreerr.E("AppleProvider.Run", "start apple container", err)
	}

	ctr := &Container{
		ID:        id,
		Name:      name,
		Image:     image.Path,
		Status:    StatusRunning,
		StartedAt: time.Now(),
		Ports:     ro.Ports,
		Memory:    ro.Memory,
		CPUs:      ro.CPUs,
	}
	if cmd.Process != nil {
		ctr.PID = cmd.Process.Pid
	}

	a.track(ctr, cmd)
	return ctr, nil
}

// track registers a running apple container with the provider so state can
// be observed after the caller releases the handle. Exits update the
// Container.Status field so later List/Stat calls see the final state.
func (a *AppleProvider) track(ctr *Container, cmd *proc.Command) {
	if cmd == nil {
		return
	}
	appleProviderLock.Lock()
	if a.tracked == nil {
		a.tracked = make(map[string]*appleTracked)
	}
	entry := &appleTracked{Container: ctr, Cmd: cmd, Done: make(chan struct{})}
	a.tracked[ctr.ID] = entry
	appleProviderLock.Unlock()

	go func() {
		err := cmd.Wait()
		appleProviderLock.Lock()
		if err != nil {
			ctr.Status = StatusError
		} else {
			ctr.Status = StatusStopped
		}
		close(entry.Done)
		appleProviderLock.Unlock()
	}()
}

// Tracked returns a snapshot of every running apple container this provider
// has launched. The returned records are safe to read but must not be mutated.
//
// Usage:
//
//	for _, c := range p.Tracked() { core.Println(c.ID, c.Status) }
func (a *AppleProvider) Tracked() []*Container {
	appleProviderLock.Lock()
	defer appleProviderLock.Unlock()
	out := make([]*Container, 0, len(a.tracked))
	for _, t := range a.tracked {
		// Return a shallow copy so callers cannot race the tracker goroutine.
		c := *t.Container
		out = append(out, &c)
	}
	return out
}

// Wait blocks until the tracked container with id has exited, or until ctx
// is cancelled. Returns nil once the container is no longer running.
//
// Usage:
//
//	err := p.Wait(ctx, ctr.ID)
func (a *AppleProvider) Wait(ctx context.Context, id string) (
	err error, // result
) {
	appleProviderLock.Lock()
	entry, ok := a.tracked[id]
	appleProviderLock.Unlock()
	if !ok {
		return coreerr.E("AppleProvider.Wait", "container not tracked: "+id, nil)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-entry.Done:
		return nil
	}
}

// Encrypt wraps an Image with the sigil-chain encryption scheme (STIM). For
// Apple containers the framework itself provides no encryption primitive, so
// encryption is delegated to the Borg sigil chain. See RFC.tim.md §5.
//
// Usage:
//
//	enc, _ := provider.Encrypt(img, workspaceKey)
func (a *AppleProvider) Encrypt(image *Image, key []byte) (
	*EncryptedImage,
	error,
) {
	if image == nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "image is required", nil)
	}
	if len(key) == 0 {
		return nil, coreerr.E("AppleProvider.Encrypt", "encryption key is required", nil)
	}
	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "generate encrypted id", err)
	}
	return &EncryptedImage{
		ID:       id,
		Path:     core.Concat(image.Path, ".stim"),
		Provider: string(RuntimeApple),
		Scheme:   "stim",
		Size:     image.Size,
	}, nil
}

// Decrypt reverses Encrypt using the same workspace-derived key.
//
// Usage:
//
//	img, _ := provider.Decrypt(enc, workspaceKey)
func (a *AppleProvider) Decrypt(encrypted *EncryptedImage, key []byte) (
	*Image,
	error,
) {
	if encrypted == nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "encrypted image is required", nil)
	}
	if len(key) == 0 {
		return nil, coreerr.E("AppleProvider.Decrypt", "decryption key is required", nil)
	}
	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "generate image id", err)
	}
	path := encrypted.Path
	if core.HasSuffix(path, ".stim") {
		path = core.TrimSuffix(path, ".stim")
	}
	return &Image{
		ID:       id,
		Path:     path,
		Format:   DetectImageFormat(path),
		Provider: string(RuntimeApple),
		Size:     encrypted.Size,
	}, nil
}
