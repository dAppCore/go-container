package container

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"runtime"
	"strconv"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

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
	return NewAppleProvider().Available()
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
	// Binary-present is not runtime-serving: Build/Run/List need the system
	// services up (`container system start`) — without this probe Detect
	// selects a runtime whose every call exits 1.
	return a.systemRunning()
}

// systemRunning reports whether the Apple container system services (apiserver
// + plugins) are started. Build and Run require them; without them the CLI
// fails with a "plugins unavailable" error. Bring them up with
// `container system start`.
func (a *AppleProvider) systemRunning() bool {
	r := a.SystemStatus()
	return r.OK && core.Contains(core.Lower(core.MustCast[string](r)), "running")
}

// Build produces an Image from a declarative configuration. For Apple
// containers the Source field must reference an existing OCI image tag or
// Containerfile path recognised by the `container build` subcommand.
//
// Usage:
//
//	img := core.MustCast[*Image](provider.Build(container.ContainerConfig{Source: "./Containerfile"}))
func (a *AppleProvider) Build(config ContainerConfig) core.Result { // Value: *Image
	if !a.Available() {
		return core.Fail(core.E("AppleProvider.Build", "apple container runtime not available on this host", nil))
	}
	if !a.systemRunning() {
		return core.Fail(core.E("AppleProvider.Build", "apple container system is not running; start it with: container system start", nil))
	}
	if config.Source == "" {
		return core.Fail(core.E("AppleProvider.Build", "ContainerConfig.Source is required", nil))
	}

	name := config.Name
	if name == "" {
		idRes := GenerateID()
		if !idRes.OK {
			return core.Fail(core.E("AppleProvider.Build", "generate image id", idRes.Value.(error)))
		}
		name = core.MustCast[string](idRes)
	}

	args := []string{"build"}
	if name != "" {
		args = append(args, "--tag", name)
	}

	contextDir := "."
	var isContainerfile bool
	if coreio.Local.IsFile(config.Source) {
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
		return core.Fail(core.E("AppleProvider.Build", "container build", err))
	}

	digest := parseDigestFromOutput(out)

	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("AppleProvider.Build", "generate image id", idRes.Value.(error)))
	}

	return core.Ok(&Image{
		ID:       core.MustCast[string](idRes),
		Name:     name,
		Path:     core.Concat(name, ":", digest),
		Format:   FormatOCI,
		Provider: string(RuntimeApple),
		Digest:   digest,
	})
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

// --- CLI argument builders (container 0.12.x) ---
// Image operations are nested under the `image` subgroup; the top-level
// `images`/`pull`/`push`/`rmi` verbs of older assumptions do not exist.

// appleImageLsArgs builds the `container image ls --format json` argument vector.
func appleImageLsArgs() []string {
	return []string{"image", "ls", "--format", "json"}
}

// applePullArgs builds the `container image pull <ref>` argument vector.
func applePullArgs(ref string) []string {
	return []string{"image", "pull", ref}
}

// applePushArgs builds the `container image push <ref>` argument vector. The
// reference must already be tagged locally.
func applePushArgs(ref string) []string {
	return []string{"image", "push", ref}
}

// appleRemoveImageArgs builds the `container image delete <ref>` argument vector.
func appleRemoveImageArgs(ref string) []string {
	return []string{"image", "delete", ref}
}

// appleLogsArgs builds the `container logs -n <n> <id>` argument vector. The
// real CLI uses -n for line count, not --tail.
func appleLogsArgs(id string, n int) []string {
	return []string{"logs", "-n", strconv.Itoa(n), id}
}

// defaultAppleDNS is the resolver appleRunArgs sets when RunOptions.DNS is
// empty. The Apple runtime points new containers at the gateway as their
// resolver but it doesn't answer DNS, and the host's LAN resolver is often
// unreachable from the container subnet — so a reachable public resolver is
// the default. Override per-run via WithDNS.
var defaultAppleDNS = []string{"1.1.1.1", "8.8.8.8"}

// appleRunArgs builds the `container run` argument vector for the resolved
// container name. Metal GPU passthrough is not offered by the Apple container
// runtime (RFC.apple.md §15), so a GPU request is rejected rather than emitting
// flags the CLI does not understand.
//
// Usage:
//
//	r := appleRunArgs(name, image, ro); if !r.OK { return r }
func appleRunArgs(name string, image *Image, ro RunOptions) core.Result { // Value: []string
	if ro.GPU {
		return core.Fail(core.E("appleRunArgs", "Metal GPU passthrough is not supported by the Apple container runtime", nil))
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
	for _, e := range ro.Env {
		args = append(args, "-e", e)
	}
	// DNS — the Apple runtime points new containers at the gateway as their
	// resolver, but that gateway does not answer DNS: NAT egress works, name
	// resolution doesn't. Default to a reachable public resolver unless the
	// caller set one explicitly (WithDNS).
	dns := ro.DNS
	if len(dns) == 0 {
		dns = defaultAppleDNS
	}
	for _, ns := range dns {
		args = append(args, "--dns", ns)
	}
	args = append(args, image.Path)
	args = append(args, ro.Args...)
	return core.Ok(args)
}

// appleContainerID resolves the id/name for a container run: the explicit
// RunOptions.Name, else the image name, else the generated fallback. For Apple
// the `--name` value IS the container's id, so this is what stop/logs/exec
// address and what Run records as Container.ID.
func appleContainerID(ro RunOptions, image *Image, fallback string) string {
	if ro.Name != "" {
		return ro.Name
	}
	if image != nil && image.Name != "" {
		return image.Name
	}
	return fallback
}

// appleSystemStatusArgs builds the `container system status` argument vector.
func appleSystemStatusArgs() []string {
	return []string{"system", "status"}
}

// appleSystemStartArgs builds the `container system start` argument vector. The
// kernel-install flag is forced because the CLI otherwise prompts interactively.
func appleSystemStartArgs(installKernel bool) []string {
	flag := "--enable-kernel-install"
	if !installKernel {
		flag = "--disable-kernel-install"
	}
	return []string{"system", "start", flag}
}

// appleSystemStopArgs builds the `container system stop` argument vector.
func appleSystemStopArgs() []string {
	return []string{"system", "stop"}
}

// SystemStart brings up the apiserver + background services. installKernel
// chooses --enable-kernel-install vs --disable-kernel-install (the CLI would
// otherwise prompt, which is impossible non-interactively).
//
// Usage:
//
//	if r := p.SystemStart(true); !r.OK { return r }
func (a *AppleProvider) SystemStart(installKernel bool) core.Result { // Value: nil
	if err := proc.NewCommand(a.Binary, appleSystemStartArgs(installKernel)...).Run(); err != nil {
		return core.Fail(core.E("AppleProvider.SystemStart", "start container system", err))
	}
	return core.Ok(nil)
}

// SystemStop stops all `container` services.
//
// Usage:
//
//	if r := p.SystemStop(); !r.OK { return r }
func (a *AppleProvider) SystemStop() core.Result { // Value: nil
	if err := proc.NewCommand(a.Binary, appleSystemStopArgs()...).Run(); err != nil {
		return core.Fail(core.E("AppleProvider.SystemStop", "stop container system", err))
	}
	return core.Ok(nil)
}

// SystemStatus returns the raw `container system status` output.
//
// Usage:
//
//	status := core.MustCast[string](p.SystemStatus())
func (a *AppleProvider) SystemStatus() core.Result { // Value: string
	out, err := proc.NewCommand(a.Binary, appleSystemStatusArgs()...).Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.SystemStatus", "container system status", err))
	}
	return core.Ok(string(out))
}

// Run boots an Image using the `container run` subcommand. RunOptions are
// translated into CLI flags. Ports and volume mounts are forwarded.
//
// Usage:
//
//	ctr := core.MustCast[*Container](provider.Run(img, container.WithMemory(2048), container.WithCPUs(2)))
func (a *AppleProvider) Run(image *Image, opts ...RunOption) core.Result { // Value: *Container
	if !a.Available() {
		return core.Fail(core.E("AppleProvider.Run", "apple container runtime not available on this host", nil))
	}
	if !a.systemRunning() {
		return core.Fail(core.E("AppleProvider.Run", "apple container system is not running; start it with: container system start", nil))
	}
	if image == nil || image.Path == "" {
		return core.Fail(core.E("AppleProvider.Run", "image is required", nil))
	}

	ro := ApplyRunOptions(opts...)

	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("AppleProvider.Run", "generate container id", idRes.Value.(error)))
	}
	name := appleContainerID(ro, image, core.MustCast[string](idRes))

	argsRes := appleRunArgs(name, image, ro)
	if !argsRes.OK {
		return argsRes
	}
	args := core.MustCast[[]string](argsRes)

	cmd := proc.NewCommandContext(context.Background(), a.Binary, args...)
	if err := cmd.Start(); err != nil {
		return core.Fail(core.E("AppleProvider.Run", "start apple container", err))
	}

	ctr := &Container{
		ID:        name,
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
	return core.Ok(ctr)
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
//	if r := p.Wait(ctx, ctr.ID); !r.OK { return r }
func (a *AppleProvider) Wait(ctx context.Context, id string) core.Result { // Value: nil
	appleProviderLock.Lock()
	entry, ok := a.tracked[id]
	appleProviderLock.Unlock()
	if !ok {
		return core.Fail(core.E("AppleProvider.Wait", "container not tracked: "+id, nil))
	}
	select {
	case <-ctx.Done():
		return core.Fail(core.E("AppleProvider.Wait", "context cancelled", ctx.Err()))
	case <-entry.Done:
		return core.Ok(nil)
	}
}

// Encrypt wraps an Image with the sigil-chain encryption scheme (STIM). For
// Apple containers the framework itself provides no encryption primitive, so
// encryption is delegated to the Borg sigil chain. See RFC.tim.md §5.
//
// Usage:
//
//	enc := core.MustCast[*EncryptedImage](provider.Encrypt(img, workspaceKey))
func (a *AppleProvider) Encrypt(image *Image, key []byte) core.Result { // Value: *EncryptedImage
	if image == nil {
		return core.Fail(core.E("AppleProvider.Encrypt", "image is required", nil))
	}
	if len(key) == 0 {
		return core.Fail(core.E("AppleProvider.Encrypt", "encryption key is required", nil))
	}

	content, err := coreio.Local.Read(image.Path)
	if err != nil {
		return core.Fail(core.E("AppleProvider.Encrypt", "read image", err))
	}
	plaintext := []byte(content)

	block, err := aes.NewCipher(deriveKey256(key))
	if err != nil {
		return core.Fail(core.E("AppleProvider.Encrypt", "create cipher", err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return core.Fail(core.E("AppleProvider.Encrypt", "create gcm", err))
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return core.Fail(core.E("AppleProvider.Encrypt", "generate nonce", err))
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	stimPath := core.Concat(image.Path, ".stim")
	if err := coreio.Local.WriteMode(stimPath, string(ciphertext), 0600); err != nil {
		return core.Fail(core.E("AppleProvider.Encrypt", "write encrypted image", err))
	}

	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("AppleProvider.Encrypt", "generate encrypted id", idRes.Value.(error)))
	}

	return core.Ok(&EncryptedImage{
		ID:       core.MustCast[string](idRes),
		Path:     stimPath,
		Provider: string(RuntimeApple),
		Scheme:   "stim",
		Size:     int64(len(ciphertext)),
	})
}

// Decrypt reverses Encrypt using the same workspace-derived key.
//
// Usage:
//
//	img := core.MustCast[*Image](provider.Decrypt(enc, workspaceKey))
func (a *AppleProvider) Decrypt(encrypted *EncryptedImage, key []byte) core.Result { // Value: *Image
	if encrypted == nil {
		return core.Fail(core.E("AppleProvider.Decrypt", "encrypted image is required", nil))
	}
	if len(key) == 0 {
		return core.Fail(core.E("AppleProvider.Decrypt", "decryption key is required", nil))
	}

	content, err := coreio.Local.Read(encrypted.Path)
	if err != nil {
		return core.Fail(core.E("AppleProvider.Decrypt", "read encrypted image", err))
	}
	ciphertext := []byte(content)

	block, err := aes.NewCipher(deriveKey256(key))
	if err != nil {
		return core.Fail(core.E("AppleProvider.Decrypt", "create cipher", err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return core.Fail(core.E("AppleProvider.Decrypt", "create gcm", err))
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return core.Fail(core.E("AppleProvider.Decrypt", "ciphertext too short", nil))
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return core.Fail(core.E("AppleProvider.Decrypt", "decrypt", err))
	}

	path := encrypted.Path
	if core.HasSuffix(path, ".stim") {
		path = core.TrimSuffix(path, ".stim")
	}
	if err := coreio.Local.WriteMode(path, string(plaintext), 0600); err != nil {
		return core.Fail(core.E("AppleProvider.Decrypt", "write decrypted image", err))
	}

	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("AppleProvider.Decrypt", "generate image id", idRes.Value.(error)))
	}

	return core.Ok(&Image{
		ID:       core.MustCast[string](idRes),
		Path:     path,
		Format:   DetectImageFormat(path),
		Provider: string(RuntimeApple),
		Size:     int64(len(plaintext)),
	})
}

// Stop stops a running container by ID through the Apple container CLI.
//
// Usage:
//
//	if r := p.Stop(ctr.ID); !r.OK { return r }
func (a *AppleProvider) Stop(id string) core.Result { // Value: nil
	if id == "" {
		return core.Fail(core.E("AppleProvider.Stop", "container id is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "stop", id)
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.Stop", "stop container", err))
	}
	appleProviderLock.Lock()
	if entry, ok := a.tracked[id]; ok {
		entry.Container.Status = StatusStopped
	}
	appleProviderLock.Unlock()
	return core.Ok(nil)
}

// Kill sends SIGKILL to a running container by ID through the Apple container CLI.
//
// Usage:
//
//	if r := p.Kill(ctr.ID); !r.OK { return r }
func (a *AppleProvider) Kill(id string) core.Result { // Value: nil
	if id == "" {
		return core.Fail(core.E("AppleProvider.Kill", "container id is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "kill", id)
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.Kill", "kill container", err))
	}
	appleProviderLock.Lock()
	if entry, ok := a.tracked[id]; ok {
		entry.Container.Status = StatusKilled
	}
	appleProviderLock.Unlock()
	return core.Ok(nil)
}

// Remove removes a container by ID through the Apple container CLI and
// deletes its entry from the tracked map.
//
// Usage:
//
//	if r := p.Remove(ctr.ID); !r.OK { return r }
func (a *AppleProvider) Remove(id string) core.Result { // Value: nil
	if id == "" {
		return core.Fail(core.E("AppleProvider.Remove", "container id is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "rm", id)
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.Remove", "remove container", err))
	}
	appleProviderLock.Lock()
	delete(a.tracked, id)
	appleProviderLock.Unlock()
	return core.Ok(nil)
}

// Logs returns the combined stdout/stderr log output for a container.
// tail specifies the number of lines to return (defaults to 200 if <= 0).
//
// Usage:
//
//	out := core.MustCast[string](p.Logs(ctr.ID, 100))
func (a *AppleProvider) Logs(id string, tail int) core.Result { // Value: string
	if id == "" {
		return core.Fail(core.E("AppleProvider.Logs", "container id is required", nil))
	}
	n := tail
	if n <= 0 {
		n = 200
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, appleLogsArgs(id, n)...)
	out, err := cmd.Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.Logs", "get logs", err))
	}
	return core.Ok(string(out))
}

// Exec runs a command inside a container by ID through the Apple container CLI.
//
// Usage:
//
//	out := core.MustCast[string](p.Exec(ctr.ID, "/bin/sh", "-c", "echo hello"))
func (a *AppleProvider) Exec(id, command string, args ...string) core.Result { // Value: string
	if id == "" {
		return core.Fail(core.E("AppleProvider.Exec", "container id is required", nil))
	}
	if command == "" {
		return core.Fail(core.E("AppleProvider.Exec", "command is required", nil))
	}
	cliArgs := append([]string{"exec", id, command}, args...)
	cmd := proc.NewCommandContext(context.Background(), a.Binary, cliArgs...)
	out, err := cmd.Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.Exec", "exec command", err))
	}
	return core.Ok(string(out))
}

// appleExecInteractiveArgs builds `container exec -i -t <id> <cmd…>`.
func appleExecInteractiveArgs(id string, cmd []string) []string {
	return append([]string{"exec", "-i", "-t", id}, cmd...)
}

// ExecInteractive runs an interactive command in a container with a TTY, wiring
// the child's stdin/stdout/stderr to the terminal and blocking until it exits.
// Unlike Exec (which captures output), this is for shells and other interactive
// programs.
//
// Usage:
//
//	if r := p.ExecInteractive(id, "/bin/sh"); !r.OK { return r }
func (a *AppleProvider) ExecInteractive(id string, cmd ...string) core.Result { // Value: nil
	if id == "" {
		return core.Fail(core.E("AppleProvider.ExecInteractive", "container id is required", nil))
	}
	c := proc.NewCommandContext(context.Background(), a.Binary, appleExecInteractiveArgs(id, cmd)...)
	c.Stdin = proc.Stdin
	c.Stdout = proc.Stdout
	c.Stderr = proc.Stderr
	if err := c.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.ExecInteractive", "interactive exec", err))
	}
	return core.Ok(nil)
}

// List returns all containers known to the Apple container CLI.
//
// Usage:
//
//	containers := core.MustCast[[]*Container](p.List())
//	for _, c := range containers {
//	    core.Println(c.ID, c.Status)
//	}
func (a *AppleProvider) List() core.Result { // Value: []*Container
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "ls", "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.List", "list containers", err))
	}
	return parseContainerList(out)
}

// Inspect returns detailed information about a single container from the
// Apple container CLI.
//
// Usage:
//
//	ctr := core.MustCast[*Container](p.Inspect(ctr.ID))
func (a *AppleProvider) Inspect(id string) core.Result { // Value: *Container
	if id == "" {
		return core.Fail(core.E("AppleProvider.Inspect", "container id is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, "inspect", id)
	out, err := cmd.Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.Inspect", "inspect container", err))
	}
	return parseSingleContainer(out)
}

// Pull fetches an image from a registry using the Apple container CLI.
//
// Usage:
//
//	img := core.MustCast[*Image](p.Pull("ghcr.io/user/app:latest"))
func (a *AppleProvider) Pull(ref string) core.Result { // Value: *Image
	if ref == "" {
		return core.Fail(core.E("AppleProvider.Pull", "image reference is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, applePullArgs(ref)...)
	out, err := cmd.Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.Pull", "pull image", err))
	}

	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("AppleProvider.Pull", "generate image id", idRes.Value.(error)))
	}

	digest := parseDigestFromOutput(out)
	return core.Ok(&Image{
		ID:       core.MustCast[string](idRes),
		Name:     ref,
		Path:     ref,
		Format:   FormatOCI,
		Provider: string(RuntimeApple),
		Digest:   digest,
	})
}

// Push uploads an image to a registry using the Apple container CLI.
//
// Usage:
//
//	if r := p.Push(img, "ghcr.io/user/app:v1"); !r.OK { return r }
func (a *AppleProvider) Push(image *Image, ref string) core.Result { // Value: nil
	if image == nil || image.Path == "" {
		return core.Fail(core.E("AppleProvider.Push", "image is required", nil))
	}
	if ref == "" {
		return core.Fail(core.E("AppleProvider.Push", "image reference is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, applePushArgs(ref)...)
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.Push", "push image", err))
	}
	return core.Ok(nil)
}

// RemoveImage removes a container image by ID using the Apple container CLI.
//
// Usage:
//
//	if r := p.RemoveImage(img.ID); !r.OK { return r }
func (a *AppleProvider) RemoveImage(id string) core.Result { // Value: nil
	if id == "" {
		return core.Fail(core.E("AppleProvider.RemoveImage", "image id is required", nil))
	}
	cmd := proc.NewCommandContext(context.Background(), a.Binary, appleRemoveImageArgs(id)...)
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.RemoveImage", "remove image", err))
	}
	return core.Ok(nil)
}

// ListImages returns all images known to the Apple container CLI.
//
// Usage:
//
//	images := core.MustCast[[]*Image](p.ListImages())
//	for _, img := range images {
//	    core.Println(img.ID, img.Name)
//	}
func (a *AppleProvider) ListImages() core.Result { // Value: []*Image
	cmd := proc.NewCommandContext(context.Background(), a.Binary, appleImageLsArgs()...)
	out, err := cmd.Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.ListImages", "list images", err))
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

// appleContainerJSON is the JSON schema returned by `container ls --format
// json` and `container inspect` (container 0.12.x). The runtime nests almost
// everything under "configuration"; "status" is top-level and "startedDate"
// is a CFAbsoluteTime float (seconds since 2001-01-01, not RFC3339).
type appleContainerJSON struct {
	Status        string  `json:"status"`
	StartedDate   float64 `json:"startedDate"`
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
		Resources struct {
			CPUs          int   `json:"cpus"`
			MemoryInBytes int64 `json:"memoryInBytes"`
		} `json:"resources"`
		PublishedPorts []struct {
			HostPort      int `json:"hostPort"`
			ContainerPort int `json:"containerPort"`
		} `json:"publishedPorts"`
	} `json:"configuration"`
}

// appleImageJSON is the JSON schema returned by `container image ls --format
// json` (container 0.12.x): a registry "reference" plus a content
// "descriptor" carrying the digest.
type appleImageJSON struct {
	Reference  string `json:"reference"`
	FullSize   string `json:"fullSize"`
	Descriptor struct {
		Digest string `json:"digest"`
	} `json:"descriptor"`
}

// cfAbsoluteTimeUnixOffset is the seconds between the CFAbsoluteTime epoch
// (2001-01-01 00:00:00 UTC) and the Unix epoch (1970-01-01 00:00:00 UTC).
const cfAbsoluteTimeUnixOffset = 978307200

// parseContainerList parses the JSON array output of `container ls --format json`.
func parseContainerList(data []byte) core.Result { // Value: []*Container
	var raw []appleContainerJSON
	if res := core.JSONUnmarshal(data, &raw); !res.OK {
		if e, ok := res.Value.(error); ok {
			return core.Fail(core.E("parseContainerList", "parse container list json", e))
		}
		return core.Fail(core.E("parseContainerList", "parse container list json", nil))
	}
	out := make([]*Container, 0, len(raw))
	for _, r := range raw {
		c := containerFromJSON(r)
		out = append(out, c)
	}
	return core.Ok(out)
}

// parseSingleContainer parses the JSON output of `container inspect <id>`,
// which is a JSON ARRAY even for a single id.
func parseSingleContainer(data []byte) core.Result { // Value: *Container
	var raw []appleContainerJSON
	if res := core.JSONUnmarshal(data, &raw); !res.OK {
		if e, ok := res.Value.(error); ok {
			return core.Fail(core.E("parseSingleContainer", "parse container json", e))
		}
		return core.Fail(core.E("parseSingleContainer", "parse container json", nil))
	}
	if len(raw) == 0 {
		return core.Fail(core.E("parseSingleContainer", "no container in inspect output", nil))
	}
	return core.Ok(containerFromJSON(raw[0]))
}

// containerFromJSON maps the container 0.12.x JSON schema to a Container.
// Memory is converted from bytes to MB; StartedDate is converted from the
// CFAbsoluteTime epoch to a Unix time.
func containerFromJSON(raw appleContainerJSON) *Container {
	cfg := raw.Configuration
	c := &Container{
		ID:     cfg.ID,
		Name:   cfg.ID, // the --name becomes the container id; there is no separate name field
		Image:  cfg.Image.Reference,
		Status: Status(raw.Status),
		CPUs:   cfg.Resources.CPUs,
		Ports:  make(map[int]int),
	}
	if cfg.Resources.MemoryInBytes > 0 {
		c.Memory = int(cfg.Resources.MemoryInBytes / (1024 * 1024))
	}
	if raw.StartedDate > 0 {
		sec := int64(raw.StartedDate)
		nsec := int64((raw.StartedDate - float64(sec)) * 1e9)
		c.StartedAt = time.Unix(sec+cfAbsoluteTimeUnixOffset, nsec)
	}
	for _, p := range cfg.PublishedPorts {
		if p.HostPort != 0 {
			c.Ports[p.HostPort] = p.ContainerPort
		}
	}
	return c
}

// parseImageList parses the JSON array output of `container images --format json`.
func parseImageList(data []byte) core.Result { // Value: []*Image
	var raw []appleImageJSON
	if res := core.JSONUnmarshal(data, &raw); !res.OK {
		if e, ok := res.Value.(error); ok {
			return core.Fail(core.E("parseImageList", "parse image list json", e))
		}
		return core.Fail(core.E("parseImageList", "parse image list json", nil))
	}
	out := make([]*Image, 0, len(raw))
	for _, r := range raw {
		out = append(out, &Image{
			Name:     r.Reference,
			Path:     r.Reference,
			Format:   FormatOCI,
			Provider: string(RuntimeApple),
			Digest:   r.Descriptor.Digest,
		})
	}
	return core.Ok(out)
}
