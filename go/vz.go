//go:build darwin

package container

// In-process Virtualization.framework provider (RFC.vz.md Phase A — boot).
//
// The github.com/tmc/apple bindings drive Objective-C dynamically through
// purego: importing them is safe on a darwin host without the framework
// (class lookups return zero, never panic), but the objc runtime layer only
// exists on darwin — hence the build tag. Linux/Windows builds of this
// package keep the AppleProvider/LinuxKit surface and simply have no
// VZProvider symbol.

import (
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

	"github.com/tmc/apple/foundation"
	vz "github.com/tmc/apple/virtualization"
	vzvm "github.com/tmc/apple/x/vzkit/vm"
)

var vzProviderLock = core.New().Lock("container.vz.provider").Mutex

const (
	// vzDefaultMemoryMB is the memory allocation when RunOptions.Memory is zero.
	vzDefaultMemoryMB = 1024
	// vzDefaultCPUs is the CPU allocation when RunOptions.CPUs is zero.
	vzDefaultCPUs = 1
	// vzDefaultCmdline boots the guest with its console on the virtio serial
	// port when the image directory carries no cmdline file (RFC.vz.md §4).
	vzDefaultCmdline = "console=hvc0"
	// vzKernelArtefact is the §4 kernel filename inside an image directory.
	vzKernelArtefact = "kernel"
	// vzInitrdArtefact is the §4 initial-ramdisk filename inside an image directory.
	vzInitrdArtefact = "initrd.img"
	// vzCmdlineArtefact is the §4 kernel command-line filename inside an image directory.
	vzCmdlineArtefact = "cmdline"
	// vzStartTimeout bounds how long Run waits for the VZ start completion handler.
	vzStartTimeout = 60 * time.Second
	// vzStopTimeout bounds how long Stop/Kill wait for the VZ stop completion handler.
	vzStopTimeout = 30 * time.Second
	// vzWatchInterval is the state-poll cadence of the per-VM watcher goroutine.
	vzWatchInterval = 500 * time.Millisecond
	// vzLogTailLines is how many serial-console lines boot failures surface (§7).
	vzLogTailLines = 5
)

// IsVZAvailable reports whether Virtualization.framework is usable in this
// process: darwin + Apple silicon, the framework's classes resolved at
// runtime, and the framework reports virtualization support. The
// com.apple.security.virtualization entitlement cannot be probed without
// starting a VM, so an unentitled caller sees Available()==true and receives
// the framework's verbatim entitlement error from Run (§7 failure honesty).
//
// Usage:
//
//	if container.IsVZAvailable() {
//	    provider = container.NewVZProvider()
//	}
func IsVZAvailable() bool {
	if discoverHostOS() != "darwin" || !isAppleSilicon() {
		return false
	}
	cls := vz.GetVZVirtualMachineClass()
	if cls.Class() == 0 {
		// Framework failed to load — class lookup misses, never panics.
		return false
	}
	return cls.IsSupported()
}

// VZProvider runs hardware-isolated Linux VMs in-process via
// Virtualization.framework (tmc/apple purego bindings). macOS 13+ Apple
// silicon hosts; no external binary, App-Sandbox-compatible with the
// com.apple.security.virtualization entitlement. Guest images are §4
// artefact directories (LinuxKit kernel+initrd output) — VZProvider has no
// Build verb.
//
// Usage:
//
//	provider := container.NewVZProvider()
//	if !provider.Available() { /* fall back per Detect() */ }
//	r := provider.Run(img, container.WithMemory(2048))
//	ctr := core.MustCast[*Container](r)
type VZProvider struct {
	// RetentionWindow is the duration tracked entries persist after VM exit.
	RetentionWindow time.Duration
	// StopDeadline is how long Stop waits for a graceful guest stop before
	// escalating to a VZ force stop.
	StopDeadline time.Duration

	tracked map[string]*vzTracked
}

// vzTracked records an in-process VM for lifecycle observation. The objc
// handles (machine, queue, config) are held here so the VM's Objective-C
// object graph stays referenced for as long as the VM is tracked.
type vzTracked struct {
	Container *Container
	Machine   vz.VZVirtualMachine
	Queue     *vzvm.Queue
	Config    vz.VZVirtualMachineConfiguration
	Done      chan struct{}
}

// NewVZProvider returns a VZProvider with default retention and stop
// deadlines. The constructor always succeeds — availability is a separate
// question answered by Available() (§7: a missing framework never panics).
//
// Usage:
//
//	p := container.NewVZProvider()
func NewVZProvider() *VZProvider {
	return &VZProvider{RetentionWindow: 5 * time.Minute, StopDeadline: 10 * time.Second}
}

// Available reports whether this provider can boot VMs on this host.
//
// Usage:
//
//	if provider.Available() { provider.Run(img) }
func (p *VZProvider) Available() bool {
	return IsVZAvailable()
}

// vzUnavailable is the §7 sentinel failure every verb returns when the
// framework is unusable on this host.
func vzUnavailable() core.Result {
	return core.Fail(core.E("container.vz", "virtualization framework unavailable", nil))
}

// vzGuestArtefacts is the resolved §4 guest contract for one image directory.
type vzGuestArtefacts struct {
	// Kernel is the uncompressed arm64 Image path (required).
	Kernel string
	// Initrd is the initial-ramdisk path (required).
	Initrd string
	// Cmdline is the kernel command line (file contents, or the default).
	Cmdline string
}

// vzResolveGuestArtefacts resolves kernel/initrd.img/cmdline inside an image
// directory per the §4 guest contract. kernel and initrd.img are required;
// a missing cmdline file falls back to vzDefaultCmdline.
//
// Usage:
//
//	r := vzResolveGuestArtefacts("/path/to/image-dir")
//	art := core.MustCast[vzGuestArtefacts](r)
func vzResolveGuestArtefacts(dir string) core.Result { // Value: vzGuestArtefacts
	if dir == "" {
		return core.Fail(core.E("vzResolveGuestArtefacts", "image directory is required", nil))
	}
	if !coreio.Local.IsDir(dir) {
		return core.Fail(core.E("vzResolveGuestArtefacts", "image path is not a directory: "+dir, nil))
	}
	art := vzGuestArtefacts{
		Kernel:  core.JoinPath(dir, vzKernelArtefact),
		Initrd:  core.JoinPath(dir, vzInitrdArtefact),
		Cmdline: vzDefaultCmdline,
	}
	if !coreio.Local.IsFile(art.Kernel) {
		return core.Fail(core.E("vzResolveGuestArtefacts", "kernel artefact missing: "+art.Kernel, nil))
	}
	if !coreio.Local.IsFile(art.Initrd) {
		return core.Fail(core.E("vzResolveGuestArtefacts", "initrd artefact missing: "+art.Initrd, nil))
	}
	cmdlinePath := core.JoinPath(dir, vzCmdlineArtefact)
	if coreio.Local.IsFile(cmdlinePath) {
		content, err := coreio.Local.Read(cmdlinePath)
		if err != nil {
			return core.Fail(core.E("vzResolveGuestArtefacts", "read cmdline artefact", err))
		}
		if trimmed := core.Trim(content); trimmed != "" {
			art.Cmdline = trimmed
		}
	}
	return core.Ok(art)
}

// vzClampMemoryBytes converts a MB request to bytes inside the framework's
// allowed envelope. Zero requests take the module default before clamping.
func vzClampMemoryBytes(memoryMB int) uint64 {
	if memoryMB <= 0 {
		memoryMB = vzDefaultMemoryMB
	}
	bytes := uint64(memoryMB) * 1024 * 1024
	cls := vz.GetVZVirtualMachineConfigurationClass()
	if minBytes := cls.MinimumAllowedMemorySize(); bytes < minBytes {
		bytes = minBytes
	}
	if maxBytes := cls.MaximumAllowedMemorySize(); maxBytes > 0 && bytes > maxBytes {
		bytes = maxBytes
	}
	return bytes
}

// vzClampCPUCount fits a CPU request inside the framework's allowed envelope.
// Zero requests take the module default before clamping.
func vzClampCPUCount(cpus int) uint {
	if cpus <= 0 {
		cpus = vzDefaultCPUs
	}
	count := uint(cpus)
	cls := vz.GetVZVirtualMachineConfigurationClass()
	if minCount := cls.MinimumAllowedCPUCount(); count < minCount {
		count = minCount
	}
	if maxCount := cls.MaximumAllowedCPUCount(); maxCount > 0 && count > maxCount {
		count = maxCount
	}
	return count
}

// vzBuildConfiguration constructs the §4 device wiring for one VM:
// VZLinuxBootLoader (kernel+initrd+cmdline), memory/cpu from RunOptions,
// serial console to logPath, NAT network, entropy. Construction needs no
// entitlement (verified empirically); validation does — see
// vzValidateConfiguration, which Run performs before creating the VM.
//
// Usage:
//
//	r := vzBuildConfiguration(art, "/tmp/vm.log", ApplyRunOptions(WithMemory(2048)))
//	cfg := core.MustCast[vz.VZVirtualMachineConfiguration](r)
func vzBuildConfiguration(art vzGuestArtefacts, logPath string, ro RunOptions) core.Result { // Value: vz.VZVirtualMachineConfiguration
	if ro.GPU {
		return core.Fail(core.E("vzBuildConfiguration", "GPU passthrough is not supported by the VZ provider", nil))
	}
	if logPath == "" {
		return core.Fail(core.E("vzBuildConfiguration", "serial log path is required", nil))
	}

	// Boot loader — kernel + initrd + cmdline (VZLinuxBootLoader).
	kernelURL := foundation.NewURLFileURLWithPath(art.Kernel)
	bootLoader := vz.NewLinuxBootLoaderWithKernelURL(kernelURL)
	if bootLoader.ID == 0 {
		return core.Fail(core.E("vzBuildConfiguration", "create VZLinuxBootLoader", nil))
	}
	bootLoader.SetInitialRamdiskURL(foundation.NewURLFileURLWithPath(art.Initrd))
	bootLoader.SetCommandLine(art.Cmdline)

	config := vz.NewVZVirtualMachineConfiguration()
	if config.ID == 0 {
		return core.Fail(core.E("vzBuildConfiguration", "create VZVirtualMachineConfiguration", nil))
	}
	config.SetBootLoader(&bootLoader.VZBootLoader)
	config.SetMemorySize(vzClampMemoryBytes(ro.Memory))
	config.SetCPUCount(vzClampCPUCount(ro.CPUs))

	// Serial console → log file (VZFileSerialPortAttachment, truncate mode).
	logURL := foundation.NewURLFileURLWithPath(logPath)
	serialAttachment, err := vz.NewFileSerialPortAttachmentWithURLAppendError(logURL, false)
	if err != nil {
		return core.Fail(core.E("vzBuildConfiguration", "create VZFileSerialPortAttachment for "+logPath, err))
	}
	serialPort := vz.NewVZVirtioConsoleDeviceSerialPortConfiguration()
	serialPort.SetAttachment(&serialAttachment.VZSerialPortAttachment)
	config.SetSerialPorts([]vz.VZSerialPortConfiguration{serialPort.VZSerialPortConfiguration})

	// NAT network (VZNATNetworkDeviceAttachment — no extra entitlement).
	natAttachment := vz.NewVZNATNetworkDeviceAttachment()
	networkDevice := vz.NewVZVirtioNetworkDeviceConfiguration()
	networkDevice.SetAttachment(&natAttachment.VZNetworkDeviceAttachment)
	config.SetNetworkDevices([]vz.VZNetworkDeviceConfiguration{networkDevice.VZNetworkDeviceConfiguration})

	// Entropy (VZVirtioEntropyDeviceConfiguration).
	entropyDevice := vz.NewVZVirtioEntropyDeviceConfiguration()
	config.SetEntropyDevices([]vz.VZEntropyDeviceConfiguration{entropyDevice.VZEntropyDeviceConfiguration})

	return core.Ok(config)
}

// vzValidateConfiguration asks the framework to validate a configuration
// before any start attempt — the framework names what's wrong. Empirical
// gotcha: ValidateWithError itself requires the
// com.apple.security.virtualization entitlement; an unentitled process gets
// the entitlement error here, verbatim, rather than at boot (§7).
//
// Usage:
//
//	if r := vzValidateConfiguration(config); !r.OK { return r }
func vzValidateConfiguration(config vz.VZVirtualMachineConfiguration) core.Result { // Value: nil
	valid, err := config.ValidateWithError()
	if err != nil || !valid {
		return core.Fail(core.E("vzValidateConfiguration", "VZVirtualMachineConfiguration validation", err))
	}
	return core.Ok(nil)
}

// vzSerialLogTail returns the last few serial-console lines for boot
// diagnostics (§7: the kernel's last words are the diagnosis). Best-effort —
// an unreadable or empty log yields "".
func vzSerialLogTail(logPath string, lines int) string {
	if logPath == "" || !coreio.Local.IsFile(logPath) {
		return ""
	}
	content, err := coreio.Local.Read(logPath)
	if err != nil || content == "" {
		return ""
	}
	all := core.Split(core.Trim(content), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return core.Join("\n", all...)
}

// Run boots a §4 image directory to a running VM: boot loader + devices are
// wired by vzBuildConfiguration, the VM starts on its own dispatch queue,
// and the serial console streams to ~/.core/logs/{id}.log. Phase A has no
// guest agent — Run proves boot, Stop/Kill manage lifecycle (RFC.vz.md §6).
//
// Usage:
//
//	ctr := core.MustCast[*Container](provider.Run(img, container.WithMemory(2048), container.WithCPUs(2)))
func (p *VZProvider) Run(image *Image, opts ...RunOption) core.Result { // Value: *Container
	if !p.Available() {
		return vzUnavailable()
	}
	if image == nil || image.Path == "" {
		return core.Fail(core.E("VZProvider.Run", "image is required", nil))
	}
	ro := ApplyRunOptions(opts...)

	artefactsRes := vzResolveGuestArtefacts(image.Path)
	if !artefactsRes.OK {
		return artefactsRes
	}
	artefacts := core.MustCast[vzGuestArtefacts](artefactsRes)

	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("VZProvider.Run", "generate container id", idRes.Value.(error)))
	}
	id := core.MustCast[string](idRes)
	name := ro.Name
	if name == "" {
		name = id
	}

	if r := EnsureLogsDir(); !r.OK {
		return r
	}
	logRes := LogPath(id)
	if !logRes.OK {
		return logRes
	}
	logPath := core.MustCast[string](logRes)

	configRes := vzBuildConfiguration(artefacts, logPath, ro)
	if !configRes.OK {
		return configRes
	}
	config := core.MustCast[vz.VZVirtualMachineConfiguration](configRes)
	if r := vzValidateConfiguration(config); !r.OK {
		return r
	}

	// The framework requires every VM call on the queue the VM was created
	// with; vzvm.Queue is that serial dispatch queue.
	queue := vzvm.NewQueue(core.Concat("container.vz.", id))
	machine := vzvm.Create(config, queue)
	if machine.ID == 0 {
		return core.Fail(core.E("VZProvider.Run", "create VZVirtualMachine", nil))
	}

	started := make(chan error, 1)
	vzvm.Start(queue, machine, func(err error) { started <- err })
	select {
	case err := <-started:
		if err != nil {
			msg := "start VZVirtualMachine"
			if tail := vzSerialLogTail(logPath, vzLogTailLines); tail != "" {
				msg = core.Concat(msg, "; serial tail:\n", tail)
			}
			return core.Fail(core.E("VZProvider.Run", msg, err))
		}
	case <-time.After(vzStartTimeout):
		return core.Fail(core.E("VZProvider.Run", "VZVirtualMachine start timed out", nil))
	}

	if state := vzvm.State(queue, machine); state != vz.VZVirtualMachineStateRunning {
		return core.Fail(core.E("VZProvider.Run", "VZVirtualMachine not running after start: "+state.String(), nil))
	}

	ctr := &Container{
		ID:        id,
		Name:      name,
		Image:     image.Path,
		Status:    StatusRunning,
		StartedAt: time.Now(),
		Ports:     ro.Ports,
		Memory:    int(vzClampMemoryBytes(ro.Memory) / (1024 * 1024)),
		CPUs:      int(vzClampCPUCount(ro.CPUs)),
		// PID stays zero: the VM lives inside this process, not behind one.
	}
	p.track(ctr, machine, queue, config)
	return core.Ok(ctr)
}

// track registers a running VM and watches its state from a goroutine so
// later Tracked() calls see exits even when the caller drops the handle.
func (p *VZProvider) track(ctr *Container, machine vz.VZVirtualMachine, queue *vzvm.Queue, config vz.VZVirtualMachineConfiguration) {
	vzProviderLock.Lock()
	if p.tracked == nil {
		p.tracked = make(map[string]*vzTracked)
	}
	entry := &vzTracked{Container: ctr, Machine: machine, Queue: queue, Config: config, Done: make(chan struct{})}
	p.tracked[ctr.ID] = entry
	vzProviderLock.Unlock()

	go p.watch(entry)
}

// watch polls VM state until it reaches Stopped or Error, finalises the
// tracked status, then drops the entry after RetentionWindow — the same
// observe-then-expire shape as AppleProvider.track.
func (p *VZProvider) watch(entry *vzTracked) {
	state := vzvm.State(entry.Queue, entry.Machine)
	for state != vz.VZVirtualMachineStateStopped && state != vz.VZVirtualMachineStateError {
		time.Sleep(vzWatchInterval)
		state = vzvm.State(entry.Queue, entry.Machine)
	}

	vzProviderLock.Lock()
	if entry.Container.Status == StatusRunning {
		if state == vz.VZVirtualMachineStateError {
			entry.Container.Status = StatusError
		} else {
			entry.Container.Status = StatusStopped
		}
	}
	close(entry.Done)
	vzProviderLock.Unlock()

	window := p.RetentionWindow
	if window <= 0 {
		window = 5 * time.Minute
	}
	time.AfterFunc(window, func() {
		vzProviderLock.Lock()
		defer vzProviderLock.Unlock()
		delete(p.tracked, entry.Container.ID)
	})
}

// entry returns the tracked record for id, or nil when unknown.
func (p *VZProvider) entry(id string) *vzTracked {
	vzProviderLock.Lock()
	defer vzProviderLock.Unlock()
	return p.tracked[id]
}

// setStatus records a final container status under the provider lock.
func (p *VZProvider) setStatus(entry *vzTracked, status Status) {
	vzProviderLock.Lock()
	entry.Container.Status = status
	vzProviderLock.Unlock()
}

// vzForceStop force-stops a VM and waits for the completion handler. A VM
// that already reached Stopped between request and force counts as success.
func vzForceStop(entry *vzTracked) core.Result { // Value: nil
	stopped := make(chan error, 1)
	vzvm.Stop(entry.Queue, entry.Machine, func(err error) { stopped <- err })
	select {
	case err := <-stopped:
		if err != nil {
			select {
			case <-entry.Done:
				// The guest beat the force stop to it — already stopped.
				return core.Ok(nil)
			default:
			}
			return core.Fail(core.E("vzForceStop", "stop VZVirtualMachine", err))
		}
		return core.Ok(nil)
	case <-time.After(vzStopTimeout):
		return core.Fail(core.E("vzForceStop", "VZVirtualMachine stop timed out", nil))
	}
}

// Stop stops a VM gracefully: request a guest stop (when the guest supports
// it), wait up to StopDeadline, then escalate to a VZ force stop (§3).
//
// Usage:
//
//	if r := p.Stop(ctr.ID); !r.OK { return r }
func (p *VZProvider) Stop(id string) core.Result { // Value: nil
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Stop", "container id is required", nil))
	}
	entry := p.entry(id)
	if entry == nil {
		return core.Fail(core.E("VZProvider.Stop", "container not tracked: "+id, nil))
	}

	deadline := p.StopDeadline
	if deadline <= 0 {
		deadline = 10 * time.Second
	}

	var canRequest bool
	entry.Queue.Sync(func() { canRequest = entry.Machine.CanRequestStop() })
	if canRequest {
		var requested bool
		var requestErr error
		entry.Queue.Sync(func() { requested, requestErr = entry.Machine.RequestStopWithError() })
		if requested && requestErr == nil {
			select {
			case <-entry.Done:
				p.setStatus(entry, StatusStopped)
				return core.Ok(nil)
			case <-time.After(deadline):
				// Guest ignored the request — escalate below.
			}
		}
	}

	if r := vzForceStop(entry); !r.OK {
		return core.Fail(core.E("VZProvider.Stop", "force stop after graceful deadline", r.Value.(error)))
	}
	p.setStatus(entry, StatusStopped)
	return core.Ok(nil)
}

// Kill force-stops a VM immediately — no guest-stop request, no deadline.
//
// Usage:
//
//	if r := p.Kill(ctr.ID); !r.OK { return r }
func (p *VZProvider) Kill(id string) core.Result { // Value: nil
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Kill", "container id is required", nil))
	}
	entry := p.entry(id)
	if entry == nil {
		return core.Fail(core.E("VZProvider.Kill", "container not tracked: "+id, nil))
	}
	if r := vzForceStop(entry); !r.OK {
		return core.Fail(core.E("VZProvider.Kill", "force stop", r.Value.(error)))
	}
	p.setStatus(entry, StatusKilled)
	return core.Ok(nil)
}

// Tracked returns a snapshot of every VM this provider has launched. The
// returned records are copies — safe to read, mutations don't race the
// watcher.
//
// Usage:
//
//	for _, c := range p.Tracked() { core.Println(c.ID, c.Status) }
func (p *VZProvider) Tracked() []*Container {
	vzProviderLock.Lock()
	defer vzProviderLock.Unlock()
	out := make([]*Container, 0, len(p.tracked))
	for _, t := range p.tracked {
		c := *t.Container
		out = append(out, &c)
	}
	return out
}
