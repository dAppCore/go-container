package container

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"os"
	"runtime"
	"strconv"
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
	// RetentionWindow is the duration tracked entries persist after container exit.
	RetentionWindow time.Duration

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
	return &AppleProvider{Binary: "container", RetentionWindow: 5 * time.Minute}
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
	path, err := proc.LookPath(a.Binary)
	if err != nil {
		return false
	}
	if a.Version == "" {
		out, err := proc.NewCommand(path, "--version").Output()
		if err == nil && len(out) > 0 {
			a.Version = string(out)
		}
	}
	return true
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

	name := config.Name
	if name == "" {
		id, err := GenerateID()
		if err != nil {
			return nil, coreerr.E("AppleProvider.Build", "generate image id", err)
		}
		name = id
	}

	args := []string{"build"}
	if name != "" {
		args = append(args, "--tag", name)
	}

	contextDir := "."
	var isContainerfile bool
	if info, err := os.Stat(config.Source); err == nil && !info.IsDir() {
		isContainerfile = true
		args = append(args, "--file", config.Source)
		if workDir := core.PathDir(config.Source); workDir != "" {
			contextDir = workDir
		}
	}
	if !isContainerfile {
		contextDir = core.PathDir(config.Source)
		if contextDir == "" {
			contextDir = "."
		}
	}
	args = append(args, contextDir)

	cmd := proc.NewCommandContext(context.Background(), a.Binary, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Build", "container build", err)
	}

	digest := parseDigestFromOutput(out)

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Build", "generate image id", err)
	}

	return &Image{
		ID:       id,
		Name:     name,
		Path:     core.Concat(name, ":", digest),
		Format:   FormatOCI,
		Provider: string(RuntimeApple),
		Digest:   digest,
	}, nil
}

// parseDigestFromOutput extracts a content digest from container build stdout.
func parseDigestFromOutput(out []byte) string {
	line := firstLine(out)
	words := core.Split(line, " ")
	for i := len(words) - 1; i >= 0; i-- {
		w := words[i]
		if core.HasPrefix(w, "sha256:") {
			return w
		}
	}
	return string(out)
}

// firstLine returns the first line of output.
func firstLine(out []byte) string {
	s := string(out)
	parts := core.SplitN(s, "\n", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return s
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
		if !isAppleSilicon() {
			return nil, coreerr.E("AppleProvider.Run", "Metal GPU passthrough requires Apple Silicon", nil)
		}
		args = append(args, "--gpu", "--device", "metal")
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

		window := a.RetentionWindow
		if window <= 0 {
			window = 5 * time.Minute
		}
		time.AfterFunc(window, func() {
			appleProviderLock.Lock()
			defer appleProviderLock.Unlock()
			delete(a.tracked, ctr.ID)
		})
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

	plaintext, err := os.ReadFile(image.Path)
	if err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "read image", err)
	}

	block, err := aes.NewCipher(deriveKey256(key))
	if err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "create gcm", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "generate nonce", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	stimPath := core.Concat(image.Path, ".stim")
	if err := os.WriteFile(stimPath, ciphertext, 0600); err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "write encrypted image", err)
	}

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Encrypt", "generate encrypted id", err)
	}

	return &EncryptedImage{
		ID:       id,
		Path:     stimPath,
		Provider: string(RuntimeApple),
		Scheme:   "stim",
		Size:     int64(len(ciphertext)),
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

	ciphertext, err := os.ReadFile(encrypted.Path)
	if err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "read encrypted image", err)
	}

	block, err := aes.NewCipher(deriveKey256(key))
	if err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "create gcm", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, coreerr.E("AppleProvider.Decrypt", "ciphertext too short", nil)
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "decrypt", err)
	}

	path := encrypted.Path
	if core.HasSuffix(path, ".stim") {
		path = core.TrimSuffix(path, ".stim")
	}
	if err := os.WriteFile(path, plaintext, 0600); err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "write decrypted image", err)
	}

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Decrypt", "generate image id", err)
	}

	return &Image{
		ID:       id,
		Path:     path,
		Format:   DetectImageFormat(path),
		Provider: string(RuntimeApple),
		Size:     int64(len(plaintext)),
	}, nil
}

// Stop stops a running container by ID through the Apple container CLI.
//
// Usage:
//
//	err := p.Stop(ctr.ID)
func (a *AppleProvider) Stop(id string) error {
	if id == "" {
		return coreerr.E("AppleProvider.Stop", "container id is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "stop", id)
	if err := cmd.Run(); err != nil {
		return coreerr.E("AppleProvider.Stop", "stop container", err)
	}
	appleProviderLock.Lock()
	if entry, ok := a.tracked[id]; ok {
		entry.Container.Status = StatusStopped
	}
	appleProviderLock.Unlock()
	return nil
}

// Kill sends SIGKILL to a running container by ID through the Apple container CLI.
//
// Usage:
//
//	err := p.Kill(ctr.ID)
func (a *AppleProvider) Kill(id string) error {
	if id == "" {
		return coreerr.E("AppleProvider.Kill", "container id is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "kill", id)
	if err := cmd.Run(); err != nil {
		return coreerr.E("AppleProvider.Kill", "kill container", err)
	}
	appleProviderLock.Lock()
	if entry, ok := a.tracked[id]; ok {
		entry.Container.Status = StatusKilled
	}
	appleProviderLock.Unlock()
	return nil
}

// Remove removes a container by ID through the Apple container CLI and
// deletes its entry from the tracked map.
//
// Usage:
//
//	err := p.Remove(ctr.ID)
func (a *AppleProvider) Remove(id string) error {
	if id == "" {
		return coreerr.E("AppleProvider.Remove", "container id is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "rm", id)
	if err := cmd.Run(); err != nil {
		return coreerr.E("AppleProvider.Remove", "remove container", err)
	}
	appleProviderLock.Lock()
	delete(a.tracked, id)
	appleProviderLock.Unlock()
	return nil
}

// Logs returns the combined stdout/stderr log output for a container.
// tail specifies the number of lines to return (defaults to 200 if <= 0).
//
// Usage:
//
//	out, _ := p.Logs(ctr.ID, 100)
func (a *AppleProvider) Logs(id string, tail int) (
	string,
	error,
) {
	if id == "" {
		return "", coreerr.E("AppleProvider.Logs", "container id is required", nil)
	}
	n := tail
	if n <= 0 {
		n = 200
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "logs", "--tail", strconv.Itoa(n), id)
	out, err := cmd.Output()
	if err != nil {
		return "", coreerr.E("AppleProvider.Logs", "get logs", err)
	}
	return string(out), nil
}

// Exec runs a command inside a container by ID through the Apple container CLI.
//
// Usage:
//
//	out, _ := p.Exec(ctr.ID, "/bin/sh", "-c", "echo hello")
func (a *AppleProvider) Exec(id, command string, args ...string) (
	string,
	error,
) {
	if id == "" {
		return "", coreerr.E("AppleProvider.Exec", "container id is required", nil)
	}
	if command == "" {
		return "", coreerr.E("AppleProvider.Exec", "command is required", nil)
	}
	cliArgs := append([]string{"exec", id, command}, args...)
	cmd := proc.NewCommandContext(context.Background(), a.Binary, cliArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", coreerr.E("AppleProvider.Exec", "exec command", err)
	}
	return string(out), nil
}

// List returns all containers known to the Apple container CLI.
//
// Usage:
//
//	containers, _ := p.List()
//	for _, c := range containers {
//	    fmt.Println(c.ID, c.Status)
//	}
func (a *AppleProvider) List() (
	[]*Container,
	error,
) {
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "ls", "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, coreerr.E("AppleProvider.List", "list containers", err)
	}
	return parseContainerList(out)
}

// Inspect returns detailed information about a single container from the
// Apple container CLI.
//
// Usage:
//
//	ctr, _ := p.Inspect(ctr.ID)
func (a *AppleProvider) Inspect(id string) (
	*Container,
	error,
) {
	if id == "" {
		return nil, coreerr.E("AppleProvider.Inspect", "container id is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "inspect", id)
	out, err := cmd.Output()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Inspect", "inspect container", err)
	}
	return parseSingleContainer(out)
}

// Pull fetches an image from a registry using the Apple container CLI.
//
// Usage:
//
//	img, _ := p.Pull("ghcr.io/user/app:latest")
func (a *AppleProvider) Pull(ref string) (
	*Image,
	error,
) {
	if ref == "" {
		return nil, coreerr.E("AppleProvider.Pull", "image reference is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "pull", ref)
	out, err := cmd.Output()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Pull", "pull image", err)
	}

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("AppleProvider.Pull", "generate image id", err)
	}

	digest := parseDigestFromOutput(out)
	return &Image{
		ID:       id,
		Name:     ref,
		Path:     ref,
		Format:   FormatOCI,
		Provider: string(RuntimeApple),
		Digest:   digest,
	}, nil
}

// Push uploads an image to a registry using the Apple container CLI.
//
// Usage:
//
//	err := p.Push(img, "ghcr.io/user/app:v1")
func (a *AppleProvider) Push(image *Image, ref string) error {
	if image == nil || image.Path == "" {
		return coreerr.E("AppleProvider.Push", "image is required", nil)
	}
	if ref == "" {
		return coreerr.E("AppleProvider.Push", "image reference is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "push", image.Path, ref)
	if err := cmd.Run(); err != nil {
		return coreerr.E("AppleProvider.Push", "push image", err)
	}
	return nil
}

// RemoveImage removes a container image by ID using the Apple container CLI.
//
// Usage:
//
//	err := p.RemoveImage(img.ID)
func (a *AppleProvider) RemoveImage(id string) error {
	if id == "" {
		return coreerr.E("AppleProvider.RemoveImage", "image id is required", nil)
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "rmi", id)
	if err := cmd.Run(); err != nil {
		return coreerr.E("AppleProvider.RemoveImage", "remove image", err)
	}
	return nil
}

// ListImages returns all images known to the Apple container CLI.
//
// Usage:
//
//	images, _ := p.ListImages()
//	for _, img := range images {
//	    fmt.Println(img.ID, img.Name)
//	}
func (a *AppleProvider) ListImages() (
	[]*Image,
	error,
) {
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "images", "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, coreerr.E("AppleProvider.ListImages", "list images", err)
	}
	return parseImageList(out)
}

// isAppleSilicon reports whether the host is running on Apple Silicon (ARM64 macOS).
//
// Usage:
//
//	if isAppleSilicon() { enableMetal() }
func isAppleSilicon() bool {
	return runtime.GOARCH == "arm64" && runtime.GOOS == "darwin"
}

// deriveKey256 derives a 32-byte AES-256 key from arbitrary key material
// using SHA-256.
func deriveKey256(key []byte) []byte {
	hash := sha256.Sum256(key)
	return hash[:]
}

// appleContainerJSON is the JSON schema returned by `container ls --format json`.
type appleContainerJSON struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    string            `json:"status"`
	CreatedAt string            `json:"created_at"`
	Ports     map[string]string `json:"ports"`
}

// appleImageJSON is the JSON schema returned by `container images --format json`.
type appleImageJSON struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

// parseContainerList parses the JSON array output of `container ls --format json`.
func parseContainerList(data []byte) ([]*Container, error) {
	var raw []appleContainerJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, coreerr.E("parseContainerList", "parse container list json", err)
	}
	out := make([]*Container, 0, len(raw))
	for _, r := range raw {
		c := containerFromJSON(r)
		out = append(out, c)
	}
	return out, nil
}

// parseSingleContainer parses the JSON output of `container inspect id`.
func parseSingleContainer(data []byte) (*Container, error) {
	var raw appleContainerJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, coreerr.E("parseSingleContainer", "parse container json", err)
	}
	return containerFromJSON(raw), nil
}

// containerFromJSON maps the Apple CLI JSON schema to a Container struct.
func containerFromJSON(raw appleContainerJSON) *Container {
	c := &Container{
		ID:     raw.ID,
		Name:   raw.Name,
		Image:  raw.Image,
		Status: Status(raw.Status),
		Ports:  make(map[int]int),
	}
	if raw.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, raw.CreatedAt); err == nil {
			c.StartedAt = t
		}
	}
	for host, guest := range raw.Ports {
		h, err := strconv.Atoi(host)
		if err != nil {
			continue
		}
		g, err := strconv.Atoi(guest)
		if err != nil {
			continue
		}
		c.Ports[h] = g
	}
	return c
}

// parseImageList parses the JSON array output of `container images --format json`.
func parseImageList(data []byte) ([]*Image, error) {
	var raw []appleImageJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, coreerr.E("parseImageList", "parse image list json", err)
	}
	out := make([]*Image, 0, len(raw))
	for _, r := range raw {
		out = append(out, &Image{
			ID:       r.ID,
			Name:     r.Name,
			Path:     r.Name,
			Format:   FormatOCI,
			Provider: string(RuntimeApple),
			Digest:   r.Digest,
		})
	}
	return out, nil
}
