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
	"context"
	"slices" // Note: AX-6 — deterministic volume ordering; no core sort primitive exists.
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

	"github.com/tmc/apple/foundation"
	vz "github.com/tmc/apple/virtualization"
	vzvm "github.com/tmc/apple/x/vzkit/vm"
	vzvsock "github.com/tmc/apple/x/vzkit/vsock"

	"dappco.re/go/container/internal/vzproto"
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
	// vzDiskArtefact is the §4 optional root-volume filename inside an image directory.
	vzDiskArtefact = "disk.img"
	// vzReadOnlySuffix marks a volume target read-only: WithVolumes(map[string]string{
	// "/host/data.img": "/data:ro"}). Mirrors the docker -v host:guest:ro convention
	// so the same option surface drives every provider.
	vzReadOnlySuffix = ":ro"
	// vzStartTimeout bounds how long Run waits for the VZ start completion handler.
	vzStartTimeout = 60 * time.Second
	// vzStopTimeout bounds how long Stop/Kill wait for the VZ stop completion handler.
	vzStopTimeout = 30 * time.Second
	// vzWatchInterval is the state-poll cadence of the per-VM watcher goroutine.
	vzWatchInterval = 500 * time.Millisecond
	// vzLogTailLines is how many serial-console lines boot failures surface (§7).
	vzLogTailLines = 5
	// vzExecTimeout bounds one Exec round-trip on the vsock control channel.
	vzExecTimeout = 60 * time.Second
	// vzAgentStopTimeout bounds the agent-stop request inside the graceful
	// Stop chain — a guest without a listening agent must not eat the whole
	// StopDeadline before the fallback runs.
	vzAgentStopTimeout = 5 * time.Second
	// vzLogsDefaultTail is the Logs line count when tail <= 0 — AppleProvider
	// parity.
	vzLogsDefaultTail = 200
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

// detectVZ probes for the in-process Virtualization.framework provider —
// the darwin half of the detection pair (vz_other.go answers for every
// other platform). Priority sits between Apple Containers (richer:
// sub-second OCI containers, needs the CLI + services) and Docker
// (RFC.vz.md §6 Phase E: apple → vz → docker → podman).
func detectVZ() (ContainerRuntime, bool) {
	if !IsVZAvailable() {
		return ContainerRuntime{}, false
	}
	rt := ContainerRuntime{
		Type: RuntimeVZ,
		// The framework is the detection marker — there is no runtime binary
		// and no cheap version probe for an in-process provider.
		Path: "Virtualization.framework",
	}
	// Hardware isolation per RFC.apple.md §2; NAT networking and Phase C
	// block-device volumes. GPU passthrough is rejected by
	// vzBuildConfiguration and a VZ kernel boot is seconds-scale, so capGPU
	// and capSubSecondStart stay unset — HasGPU()->WithGPU->Run honesty.
	rt.caps = capNetworkIsolation | capVolumeMounts | capHardwareIsolation
	return rt, true
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
	// StatePath overrides the persistent container registry file. Empty uses
	// DefaultStatePath() (~/.core/containers.json) — one inventory across
	// providers (§3).
	StatePath string

	tracked map[string]*vzTracked
	state   *State
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
	// Vsock is the lazily-created control-channel manager (§5) wrapping the
	// VM's VZVirtioSocketDevice. Nil until the first Exec/Stop needs it.
	Vsock *vzvsock.Manager
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
	// Disk is the optional root-volume path (disk.img); "" when the image
	// directory carries none.
	Disk string
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
	if diskPath := core.JoinPath(dir, vzDiskArtefact); coreio.Local.IsFile(diskPath) {
		art.Disk = diskPath
	}
	return core.Ok(art)
}

// vzVolumeSpec is one planned virtio block attachment: a host-side raw image
// file exposed to the guest as a block device. Target is the guest-side
// mount point declared by the caller — advisory at this layer (the guest's
// init/agent decides mounts); ordering is what the host controls, so specs
// are deterministic: root disk first, then volumes sorted by Target — the
// guest sees /dev/vda, /dev/vdb… in that order.
type vzVolumeSpec struct {
	// Source is the host-side raw disk image path.
	Source string
	// Target is the guest-side mount point (advisory, ordering key).
	Target string
	// ReadOnly attaches the device read-only (":ro" target suffix).
	ReadOnly bool
}

// vzVolumeSpecs plans the §4 block-device set for one run: the image
// directory's optional disk.img root volume first (read-write), then every
// RunOptions.Volumes entry (host image path → guest target) sorted by
// target. A ":ro" suffix on the target marks the attachment read-only.
// Pure planning — no framework calls, fully unit-testable.
//
// Usage:
//
//	r := vzVolumeSpecs(art, ApplyRunOptions(WithVolumes(map[string]string{"/host/data.img": "/data:ro"})))
//	specs := core.MustCast[[]vzVolumeSpec](r)
func vzVolumeSpecs(art vzGuestArtefacts, ro RunOptions) core.Result { // Value: []vzVolumeSpec
	specs := make([]vzVolumeSpec, 0, len(ro.Volumes)+1)
	if art.Disk != "" {
		if !coreio.Local.IsFile(art.Disk) {
			return core.Fail(core.E("vzVolumeSpecs", "root disk artefact missing: "+art.Disk, nil))
		}
		specs = append(specs, vzVolumeSpec{Source: art.Disk, Target: "/"})
	}

	volumes := make([]vzVolumeSpec, 0, len(ro.Volumes))
	seen := make(map[string]string, len(ro.Volumes))
	for source, target := range ro.Volumes {
		readOnly := false
		if core.HasSuffix(target, vzReadOnlySuffix) {
			readOnly = true
			target = core.TrimSuffix(target, vzReadOnlySuffix)
		}
		if source == "" || target == "" {
			return core.Fail(core.E("vzVolumeSpecs", "volume source and target are required", nil))
		}
		if target == "/" {
			return core.Fail(core.E("vzVolumeSpecs", "volume target / collides with the root disk", nil))
		}
		if previous, dup := seen[target]; dup {
			return core.Fail(core.E("vzVolumeSpecs", "duplicate volume target "+target+" ("+previous+" and "+source+")", nil))
		}
		seen[target] = source
		if !coreio.Local.IsFile(source) {
			return core.Fail(core.E("vzVolumeSpecs", "volume source is not a file: "+source, nil))
		}
		volumes = append(volumes, vzVolumeSpec{Source: source, Target: target, ReadOnly: readOnly})
	}
	slices.SortFunc(volumes, func(a, b vzVolumeSpec) int {
		return core.Compare(a.Target, b.Target)
	})
	return core.Ok(append(specs, volumes...))
}

// vzAttachStorage constructs a virtio block device per planned spec and sets
// them on the configuration in plan order (VZDiskImageStorageDeviceAttachment
// → VZVirtioBlockDeviceConfiguration). Construction needs no entitlement;
// the framework opens each image file here, so a bad path fails loudly with
// the file named (§7).
func vzAttachStorage(config vz.VZVirtualMachineConfiguration, specs []vzVolumeSpec) core.Result { // Value: nil
	if len(specs) == 0 {
		return core.Ok(nil)
	}
	devices := make([]vz.VZStorageDeviceConfiguration, 0, len(specs))
	for _, spec := range specs {
		url := foundation.NewURLFileURLWithPath(spec.Source)
		attachment, err := vz.NewDiskImageStorageDeviceAttachmentWithURLReadOnlyError(url, spec.ReadOnly)
		if err != nil {
			return core.Fail(core.E("vzAttachStorage", "create VZDiskImageStorageDeviceAttachment for "+spec.Source, err))
		}
		blockDevice := vz.NewVirtioBlockDeviceConfigurationWithAttachment(attachment)
		if blockDevice.ID == 0 {
			return core.Fail(core.E("vzAttachStorage", "create VZVirtioBlockDeviceConfiguration for "+spec.Source, nil))
		}
		devices = append(devices, blockDevice.VZStorageDeviceConfiguration)
	}
	config.SetStorageDevices(devices)
	return core.Ok(nil)
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
// serial console to logPath, NAT network, entropy, vsock control channel,
// and virtio block storage (root disk.img + RunOptions.Volumes, in
// vzVolumeSpecs plan order). Construction needs no entitlement (verified
// empirically); validation does — see vzValidateConfiguration, which Run
// performs before creating the VM.
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

	// Control channel — vsock (VZVirtioSocketDeviceConfiguration, §5). The
	// running VM materialises the matching VZVirtioSocketDevice for the
	// agent round-trips; exactly one socket device per configuration.
	socketDevice := vz.NewVZVirtioSocketDeviceConfiguration()
	config.SetSocketDevices([]vz.VZSocketDeviceConfiguration{socketDevice.VZSocketDeviceConfiguration})

	// Storage — root disk.img + declared volumes as virtio block devices in
	// deterministic plan order (§4; Phase C).
	specsRes := vzVolumeSpecs(art, ro)
	if !specsRes.OK {
		return specsRes
	}
	if r := vzAttachStorage(config, core.MustCast[[]vzVolumeSpec](specsRes)); !r.OK {
		return r
	}

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
	if r := p.persistAdd(ctr); !r.OK {
		// LinuxKit-manager precedent: an unpersistable container fails the
		// Run and the just-booted VM is force-stopped, not leaked.
		_ = p.Kill(id)
		return core.Fail(core.E("VZProvider.Run", "persist container registry", r.Value.(error)))
	}
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
	p.persistUpdate(entry)

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

// status reads a tracked container's status under the provider lock.
func (p *VZProvider) status(entry *vzTracked) Status {
	vzProviderLock.Lock()
	defer vzProviderLock.Unlock()
	return entry.Container.Status
}

// registry returns the provider's persistent container registry (§3: one
// inventory across providers), loading it on first use from StatePath or
// the module default ~/.core/containers.json.
func (p *VZProvider) registry() core.Result { // Value: *State
	vzProviderLock.Lock()
	cached := p.state
	vzProviderLock.Unlock()
	if cached != nil {
		return core.Ok(cached)
	}

	path := p.StatePath
	if path == "" {
		pathRes := DefaultStatePath()
		if !pathRes.OK {
			return pathRes
		}
		path = core.MustCast[string](pathRes)
	}
	stateRes := LoadState(path)
	if !stateRes.OK {
		return stateRes
	}
	loaded := core.MustCast[*State](stateRes)

	vzProviderLock.Lock()
	if p.state == nil {
		p.state = loaded
	} else {
		loaded = p.state // lost a benign race — reuse the winner
	}
	vzProviderLock.Unlock()
	return core.Ok(loaded)
}

// persistAdd records a new container in the registry. Load-bearing on Run:
// a registry that cannot persist fails the boot (LinuxKit-manager
// precedent) rather than leaking an uninventoried VM.
func (p *VZProvider) persistAdd(ctr *Container) core.Result { // Value: nil
	stateRes := p.registry()
	if !stateRes.OK {
		return stateRes
	}
	record := *ctr // the registry owns a copy, never the live tracked struct
	return core.MustCast[*State](stateRes).Add(&record)
}

// persistUpdate best-effort-syncs a status transition into the registry.
// Verb results reflect the VM operation itself — a stopped VM with an
// unwritable registry file is still stopped, so persistence failures here
// never fail the verb.
func (p *VZProvider) persistUpdate(entry *vzTracked) {
	stateRes := p.registry()
	if !stateRes.OK {
		return
	}
	vzProviderLock.Lock()
	record := *entry.Container
	vzProviderLock.Unlock()
	_ = core.MustCast[*State](stateRes).Update(&record)
}

// vsockManager returns the entry's control-channel manager, creating it on
// first use. The VZVirtioSocketDevice is read off the running VM on its own
// dispatch queue (same queue discipline as lifecycle), and every framework
// call the manager later makes is routed back through that queue.
func (p *VZProvider) vsockManager(entry *vzTracked) core.Result { // Value: *vzvsock.Manager
	vzProviderLock.Lock()
	existing := entry.Vsock
	vzProviderLock.Unlock()
	if existing != nil {
		return core.Ok(existing)
	}

	var mgr *vzvsock.Manager
	var err error
	entry.Queue.Sync(func() { mgr, err = vzvsock.NewManager(entry.Machine) })
	if err != nil {
		return core.Fail(core.E("VZProvider.vsockManager", "wrap VZVirtioSocketDevice", err))
	}
	mgr.DispatchFunc = entry.Queue.Sync

	vzProviderLock.Lock()
	if entry.Vsock == nil {
		entry.Vsock = mgr
	} else {
		mgr = entry.Vsock // lost a benign race — reuse the winner
	}
	vzProviderLock.Unlock()
	return core.Ok(mgr)
}

// vzVsockConn is the subset of net.Conn the control round-trip needs —
// keeps vzAgentCall mockable and the import surface minimal.
type vzVsockConn interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	SetDeadline(t time.Time) error
	Close() error
}

// vzConnectControl dials the guest control port under its own timeout.
// VZVirtioSocketDevice's connect completion handler is documented to never
// fire when the guest has no listener on the port (e.g. the vsock driver is
// not up yet), so the dial itself must be bounded — an abandoned dial's
// late connection is reaped and closed.
func vzConnectControl(mgr *vzvsock.Manager, timeout time.Duration) core.Result { // Value: vzVsockConn
	type dialResult struct {
		conn vzVsockConn
		err  error
	}
	dialled := make(chan dialResult, 1)
	go func() {
		conn, err := mgr.Connect(vzproto.ControlPort)
		dialled <- dialResult{conn: conn, err: err}
	}()
	select {
	case r := <-dialled:
		if r.err != nil {
			return core.Fail(core.E("vzConnectControl", "connect vsock control port", r.err))
		}
		return core.Ok(r.conn)
	case <-time.After(timeout):
		go func() { // reap a late connect so its fd never leaks
			if r := <-dialled; r.conn != nil {
				_ = r.conn.Close()
			}
		}()
		return core.Fail(core.E("vzConnectControl", "vsock control port connect timed out (no guest agent listening?)", nil))
	}
}

// vzAgentCall performs one control round-trip (§5): connect to the guest
// agent's vsock port, exchange a single length-prefixed JSON frame pair
// under deadline, close.
func vzAgentCall(mgr *vzvsock.Manager, req vzproto.Request, timeout time.Duration) core.Result { // Value: vzproto.Response
	if timeout <= 0 {
		timeout = vzExecTimeout
	}
	connRes := vzConnectControl(mgr, timeout)
	if !connRes.OK {
		return connRes
	}
	conn := core.MustCast[vzVsockConn](connRes)
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return core.Fail(core.E("vzAgentCall", "set control deadline", err))
	}
	resp, err := vzproto.RoundTrip(conn, req)
	if err != nil {
		return core.Fail(core.E("vzAgentCall", "control round-trip", err))
	}
	return core.Ok(resp)
}

// Exec runs a command inside the guest via the vsock control channel and
// returns its stdout. A command that runs and exits non-zero fails with the
// exit code and stderr named; a guest without a reachable agent fails with
// the connect error (§5, §7).
//
// Usage:
//
//	out := core.MustCast[string](p.Exec(ctr.ID, "uname", "-a"))
func (p *VZProvider) Exec(id, command string, args ...string) core.Result { // Value: string
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Exec", "container id is required", nil))
	}
	if command == "" {
		return core.Fail(core.E("VZProvider.Exec", "command is required", nil))
	}
	entry := p.entry(id)
	if entry == nil {
		return core.Fail(core.E("VZProvider.Exec", "container not tracked: "+id, nil))
	}
	if status := p.status(entry); status != StatusRunning {
		return core.Fail(core.E("VZProvider.Exec", "container not running: "+id+" ("+string(status)+")", nil))
	}

	mgrRes := p.vsockManager(entry)
	if !mgrRes.OK {
		return mgrRes
	}
	callRes := vzAgentCall(core.MustCast[*vzvsock.Manager](mgrRes), vzproto.Request{
		Verb:    vzproto.VerbExec,
		Command: command,
		Args:    args,
	}, vzExecTimeout)
	if !callRes.OK {
		return callRes
	}
	resp := core.MustCast[vzproto.Response](callRes)
	if !resp.OK {
		return core.Fail(core.E("VZProvider.Exec", "agent refused exec: "+resp.Error, nil))
	}
	if resp.Exit != 0 {
		return core.Fail(core.E("VZProvider.Exec", core.Sprintf("command exited %d; stderr: %s", resp.Exit, resp.Stderr), nil))
	}
	return core.Ok(resp.Stdout)
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

// vzRequestAgentStop sends the guest agent the stop verb (§5: guest
// poweroff). True means the agent acknowledged and a guest-side shutdown is
// in flight; false means no agent answered — callers fall back.
func (p *VZProvider) vzRequestAgentStop(entry *vzTracked) bool {
	mgrRes := p.vsockManager(entry)
	if !mgrRes.OK {
		return false
	}
	callRes := vzAgentCall(core.MustCast[*vzvsock.Manager](mgrRes), vzproto.Request{Verb: vzproto.VerbStop}, vzAgentStopTimeout)
	if !callRes.OK {
		return false
	}
	return core.MustCast[vzproto.Response](callRes).OK
}

// vzRequestGuestStop asks the framework to deliver a guest stop request
// (virtio power signal) — the agent-less graceful fallback. True means the
// request was delivered.
func vzRequestGuestStop(entry *vzTracked) bool {
	var canRequest bool
	entry.Queue.Sync(func() { canRequest = entry.Machine.CanRequestStop() })
	if !canRequest {
		return false
	}
	var requested bool
	var requestErr error
	entry.Queue.Sync(func() { requested, requestErr = entry.Machine.RequestStopWithError() })
	return requested && requestErr == nil
}

// Stop stops a VM gracefully per the §5 chain: ask the guest agent to power
// off over vsock; when no agent answers, deliver the framework's guest stop
// request; wait up to StopDeadline for the guest to exit; escalate to a VZ
// force stop when the deadline lapses.
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
	if status := p.status(entry); status == StatusStopped || status == StatusKilled {
		// Idempotent: the guest already exited (the watcher or an earlier
		// verb saw it) — a second stop succeeds without touching the queue.
		return core.Ok(nil)
	}

	deadline := p.StopDeadline
	if deadline <= 0 {
		deadline = 10 * time.Second
	}

	graceful := p.vzRequestAgentStop(entry)
	if !graceful {
		graceful = vzRequestGuestStop(entry)
	}
	if graceful {
		select {
		case <-entry.Done:
			p.setStatus(entry, StatusStopped)
			p.persistUpdate(entry)
			return core.Ok(nil)
		case <-time.After(deadline):
			// Guest ignored the request — escalate below.
		}
	}

	if r := vzForceStop(entry); !r.OK {
		return core.Fail(core.E("VZProvider.Stop", "force stop after graceful deadline", r.Value.(error)))
	}
	p.setStatus(entry, StatusStopped)
	p.persistUpdate(entry)
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
	if status := p.status(entry); status == StatusStopped || status == StatusKilled {
		// Idempotent: the guest already exited — same contract as Stop.
		return core.Ok(nil)
	}
	if r := vzForceStop(entry); !r.OK {
		return core.Fail(core.E("VZProvider.Kill", "force stop", r.Value.(error)))
	}
	p.setStatus(entry, StatusKilled)
	p.persistUpdate(entry)
	return core.Ok(nil)
}

// Wait blocks until the tracked VM with id has exited, or until ctx is
// cancelled — AppleProvider parity. Returns nil once the VM is no longer
// running.
//
// Usage:
//
//	if r := p.Wait(ctx, ctr.ID); !r.OK { return r }
func (p *VZProvider) Wait(ctx context.Context, id string) core.Result { // Value: nil
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Wait", "container id is required", nil))
	}
	entry := p.entry(id)
	if entry == nil {
		return core.Fail(core.E("VZProvider.Wait", "container not tracked: "+id, nil))
	}
	select {
	case <-ctx.Done():
		return core.Fail(core.E("VZProvider.Wait", "context cancelled", ctx.Err()))
	case <-entry.Done:
		return core.Ok(nil)
	}
}

// Logs returns the last tail lines of a VM's serial console capture
// (~/.core/logs/{id}.log — §3 log convention). tail <= 0 defaults to 200
// lines, AppleProvider parity. The log survives the tracked entry, so an
// exited and even removed VM stays diagnosable.
//
// Usage:
//
//	out := core.MustCast[string](p.Logs(ctr.ID, 100))
func (p *VZProvider) Logs(id string, tail int) core.Result { // Value: string
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Logs", "container id is required", nil))
	}
	lines := tail
	if lines <= 0 {
		lines = vzLogsDefaultTail
	}
	logRes := LogPath(id)
	if !logRes.OK {
		return logRes
	}
	logPath := core.MustCast[string](logRes)
	if !coreio.Local.IsFile(logPath) {
		return core.Fail(core.E("VZProvider.Logs", "no serial log for container: "+id, nil))
	}
	return core.Ok(vzSerialLogTail(logPath, lines))
}

// Remove drops an exited VM from the tracked set and the persistent
// registry. A running VM is refused — stop it first. The serial log file is
// kept for post-mortems (Logs still answers after Remove).
//
// Usage:
//
//	if r := p.Remove(ctr.ID); !r.OK { return r }
func (p *VZProvider) Remove(id string) core.Result { // Value: nil
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Remove", "container id is required", nil))
	}
	entry := p.entry(id)
	if entry != nil && p.status(entry) == StatusRunning {
		return core.Fail(core.E("VZProvider.Remove", "container is running, stop it first: "+id, nil))
	}

	inRegistry := false
	stateRes := p.registry()
	if stateRes.OK {
		_, inRegistry = core.MustCast[*State](stateRes).Get(id)
	}
	if entry == nil && !inRegistry {
		return core.Fail(core.E("VZProvider.Remove", "container not tracked: "+id, nil))
	}

	vzProviderLock.Lock()
	delete(p.tracked, id)
	vzProviderLock.Unlock()

	if inRegistry {
		if r := core.MustCast[*State](stateRes).Remove(id); !r.OK {
			return core.Fail(core.E("VZProvider.Remove", "remove from container registry", r.Value.(error)))
		}
	}
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
