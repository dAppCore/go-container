//go:build darwin

package container

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

	"dappco.re/go/container/internal/vzproto"

	vz "github.com/tmc/apple/virtualization"
)

// vzWriteGuestDir lays out a §4 image directory in a temp dir. Artefact
// content is arbitrary bytes — configuration construction never reads the
// kernel, only the live boot path does.
func vzWriteGuestDir(t *testing.T, withCmdline bool, cmdline string) string {
	t.Helper()
	dir := t.TempDir()
	if err := coreio.Local.Write(core.JoinPath(dir, "kernel"), "not-a-real-kernel"); err != nil {
		t.Fatalf("write kernel: %v", err)
	}
	if err := coreio.Local.Write(core.JoinPath(dir, "initrd.img"), "not-a-real-initrd"); err != nil {
		t.Fatalf("write initrd: %v", err)
	}
	if withCmdline {
		if err := coreio.Local.Write(core.JoinPath(dir, "cmdline"), cmdline); err != nil {
			t.Fatalf("write cmdline: %v", err)
		}
	}
	return dir
}

// vzLiveFixtureDir is where the live boot test looks for real LinuxKit
// artefacts (kernel + initrd.img [+ cmdline]). Relative to the package dir.
const vzLiveFixtureDir = "testdata/vz"

// vzFabricateTracked registers a synthetic tracked entry (no VM, no queue
// behind it) so verb contract paths — status guards, Wait, Logs, Remove —
// are testable without booting. Only safe for paths that never touch
// entry.Machine/entry.Queue.
func vzFabricateTracked(t *testing.T, p *VZProvider, ctr *Container) *vzTracked {
	t.Helper()
	entry := &vzTracked{Container: ctr, Done: make(chan struct{})}
	vzProviderLock.Lock()
	if p.tracked == nil {
		p.tracked = make(map[string]*vzTracked)
	}
	p.tracked[ctr.ID] = entry
	vzProviderLock.Unlock()
	t.Cleanup(func() {
		vzProviderLock.Lock()
		delete(p.tracked, ctr.ID)
		vzProviderLock.Unlock()
	})
	return entry
}

func TestVz_IsVZAvailable_Good(t *testing.T) {
	auditTarget := "IsVZAvailable"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Must not panic regardless of framework presence — §7 failure honesty.
	got := IsVZAvailable()
	t.Logf("IsVZAvailable() = %v", got)
}

func TestVz_IsVZAvailable_Bad(t *testing.T) {
	auditTarget := "IsVZAvailable"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A simulated non-darwin host is never VZ-capable.
	t.Setenv("GOOS", "linux")
	if IsVZAvailable() {
		t.Fatal("expected false")
	}
}

func TestVz_IsVZAvailable_Ugly(t *testing.T) {
	auditTarget := "IsVZAvailable"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Repeated probes are stable — class lookup caching must not flip answers.
	first := IsVZAvailable()
	second := IsVZAvailable()
	if first != second {
		t.Fatalf("availability flapped: first %v, second %v", first, second)
	}
}

func TestVz_DetectVZ_Good(t *testing.T) {
	auditTarget := "detectVZ"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Detection mirrors availability; a found runtime reports the §6 Phase E
	// capability honesty: hardware isolation yes, GPU and sub-second no.
	rt, found := detectVZ()
	if found != IsVZAvailable() {
		t.Fatalf("detection %v disagrees with availability %v", found, IsVZAvailable())
	}
	if !found {
		t.Skip("virtualization framework not available")
	}
	if rt.Type != RuntimeVZ {
		t.Fatalf("expected vz runtime, got %q", rt.Type)
	}
	if rt.Path == "" {
		t.Fatal("expected a detection marker path")
	}
	if !rt.IsHardwareIsolated() {
		t.Fatal("expected hardware isolation capability")
	}
	if !rt.HasVolumeMounts() || !rt.HasNetworkIsolation() {
		t.Fatalf("expected volume + network capabilities, caps %b", rt.Caps())
	}
	if rt.HasGPU() {
		t.Fatal("GPU passthrough is rejected by the provider; capGPU must stay unset")
	}
	if rt.HasSubSecondStart() {
		t.Fatal("a VZ kernel boot is seconds-scale; capSubSecondStart must stay unset")
	}
}

func TestVz_DetectVZ_Bad(t *testing.T) {
	auditTarget := "detectVZ"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A simulated non-darwin host never detects vz, and the whole detection
	// chain stays vz-free.
	t.Setenv("GOOS", "linux")
	if _, found := detectVZ(); found {
		t.Fatal("expected no vz on a simulated non-darwin host")
	}
	for _, rt := range DetectAll() {
		if rt.Type == RuntimeVZ {
			t.Fatal("DetectAll leaked a vz runtime on a simulated non-darwin host")
		}
	}
}

func TestVz_DetectVZ_Ugly(t *testing.T) {
	auditTarget := "detectVZ"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Priority ordering (§6 Phase E: apple → vz → docker → podman): vz
	// appears exactly once, after any apple entry, before any of the rest —
	// on a CLI-less host (no apple) this makes vz the Detect() winner.
	all := DetectAll()
	vzIndex := -1
	for i, rt := range all {
		switch rt.Type {
		case RuntimeVZ:
			if vzIndex != -1 {
				t.Fatal("vz detected twice")
			}
			vzIndex = i
		case RuntimeApple:
			if vzIndex != -1 {
				t.Fatalf("apple at %d after vz at %d", i, vzIndex)
			}
		case RuntimeDocker, RuntimePodman, RuntimeLinuxKit:
			if vzIndex == -1 {
				t.Fatalf("%s at %d before vz", rt.Type, i)
			}
		}
	}
	if vzIndex == -1 {
		t.Fatal("expected vz in DetectAll on a VZ-capable host")
	}
	if !HasRuntime(RuntimeVZ) {
		t.Fatal("HasRuntime(vz) disagrees with DetectAll")
	}
}

func TestVz_NewVZProvider_Good(t *testing.T) {
	auditTarget := "NewVZProvider"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	if p == nil {
		t.Fatal("expected non-nil value")
	}
	if p.RetentionWindow <= 0 {
		t.Fatal("expected a default RetentionWindow")
	}
	if p.StopDeadline <= 0 {
		t.Fatal("expected a default StopDeadline")
	}
}

func TestVz_NewVZProvider_Bad(t *testing.T) {
	auditTarget := "NewVZProvider"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A zero-value provider must stay safe: no panics, verbs fail cleanly.
	var p VZProvider
	if got := p.Tracked(); len(got) != 0 {
		t.Fatalf("expected empty tracked, got %d", len(got))
	}
	if r := p.Stop("nope"); r.OK {
		t.Fatal("expected error")
	}
	if r := p.Kill("nope"); r.OK {
		t.Fatal("expected error")
	}
}

func TestVz_NewVZProvider_Ugly(t *testing.T) {
	auditTarget := "NewVZProvider"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Caller-tuned windows are respected, not reset.
	p := NewVZProvider()
	p.RetentionWindow = time.Second
	p.StopDeadline = time.Millisecond
	if p.RetentionWindow != time.Second || p.StopDeadline != time.Millisecond {
		t.Fatal("expected tuned windows to stick")
	}
}

func TestVz_VZProvider_Available_Good(t *testing.T) {
	auditTarget := "VZProvider Available"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	if got, want := p.Available(), IsVZAvailable(); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestVz_VZProvider_Available_Bad(t *testing.T) {
	auditTarget := "VZProvider Available"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	t.Setenv("GOOS", "windows")
	p := NewVZProvider()
	if p.Available() {
		t.Fatal("expected false")
	}
}

func TestVz_VZProvider_Available_Ugly(t *testing.T) {
	auditTarget := "VZProvider Available"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// An explicit darwin override on a darwin host matches the real probe.
	t.Setenv("GOOS", "darwin")
	p := NewVZProvider()
	if got, want := p.Available(), IsVZAvailable(); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestVz_ResolveGuestArtefacts_Good(t *testing.T) {
	auditTarget := "vzResolveGuestArtefacts"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	dir := vzWriteGuestDir(t, true, "console=hvc0 loglevel=7\n")
	r := vzResolveGuestArtefacts(dir)
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	art := core.MustCast[vzGuestArtefacts](r)
	if art.Kernel != core.JoinPath(dir, "kernel") {
		t.Fatalf("unexpected kernel path %q", art.Kernel)
	}
	if art.Initrd != core.JoinPath(dir, "initrd.img") {
		t.Fatalf("unexpected initrd path %q", art.Initrd)
	}
	// File content is used verbatim, trimmed of surrounding whitespace.
	if got, want := art.Cmdline, "console=hvc0 loglevel=7"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	// No disk.img in the directory → no root disk resolved (§4 optional).
	if art.Disk != "" {
		t.Fatalf("expected no root disk, got %q", art.Disk)
	}

	// A disk.img present in the directory resolves as the root volume.
	if err := coreio.Local.Write(core.JoinPath(dir, "disk.img"), "not-a-real-disk"); err != nil {
		t.Fatalf("write disk: %v", err)
	}
	r2 := vzResolveGuestArtefacts(dir)
	if !r2.OK {
		t.Fatalf("expected ok, got %v", r2.Value)
	}
	if got, want := core.MustCast[vzGuestArtefacts](r2).Disk, core.JoinPath(dir, "disk.img"); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestVz_ResolveGuestArtefacts_Bad(t *testing.T) {
	auditTarget := "vzResolveGuestArtefacts"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if r := vzResolveGuestArtefacts(""); r.OK {
		t.Fatal("expected error for empty dir")
	}
	if r := vzResolveGuestArtefacts(core.JoinPath(t.TempDir(), "missing")); r.OK {
		t.Fatal("expected error for non-directory")
	}

	// Missing kernel.
	dir := t.TempDir()
	if err := coreio.Local.Write(core.JoinPath(dir, "initrd.img"), "x"); err != nil {
		t.Fatalf("write initrd: %v", err)
	}
	r := vzResolveGuestArtefacts(dir)
	if r.OK {
		t.Fatal("expected error for missing kernel")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "kernel") {
		t.Fatalf("expected kernel-naming error, got %v", r.Value)
	}

	// Missing initrd.
	dir2 := t.TempDir()
	if err := coreio.Local.Write(core.JoinPath(dir2, "kernel"), "x"); err != nil {
		t.Fatalf("write kernel: %v", err)
	}
	r2 := vzResolveGuestArtefacts(dir2)
	if r2.OK {
		t.Fatal("expected error for missing initrd")
	}
	if err, ok := r2.Value.(error); !ok || !core.Contains(err.Error(), "initrd") {
		t.Fatalf("expected initrd-naming error, got %v", r2.Value)
	}
}

func TestVz_ResolveGuestArtefacts_Ugly(t *testing.T) {
	auditTarget := "vzResolveGuestArtefacts"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// No cmdline file → §4 default.
	dir := vzWriteGuestDir(t, false, "")
	r := vzResolveGuestArtefacts(dir)
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if got, want := core.MustCast[vzGuestArtefacts](r).Cmdline, "console=hvc0"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}

	// Whitespace-only cmdline file → default too, not an empty command line.
	dir2 := vzWriteGuestDir(t, true, "  \n\t\n")
	r2 := vzResolveGuestArtefacts(dir2)
	if !r2.OK {
		t.Fatalf("expected ok, got %v", r2.Value)
	}
	if got, want := core.MustCast[vzGuestArtefacts](r2).Cmdline, "console=hvc0"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// vzWriteVolumeFile lays out a minimal RAW disk image for volume tests:
// 1MiB of zeros. Empirical framework contract: the
// VZDiskImageStorageDeviceAttachment constructor validates the image format
// ("Invalid disk image. The disk image format is not recognized.") — raw
// images must be sector-aligned, so an arbitrary text blob is rejected at
// construction, not at boot.
func vzWriteVolumeFile(t *testing.T, name string) string {
	t.Helper()
	path := core.JoinPath(t.TempDir(), name)
	if err := coreio.Local.Write(path, string(make([]byte, 1<<20))); err != nil {
		t.Fatalf("write volume file: %v", err)
	}
	return path
}

func TestVz_VolumeSpecs_Good(t *testing.T) {
	auditTarget := "vzVolumeSpecs"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Root disk first, then volumes sorted by target — deterministic
	// /dev/vdX ordering regardless of map iteration; :ro parses off.
	disk := vzWriteVolumeFile(t, "disk.img")
	volB := vzWriteVolumeFile(t, "b.img")
	volA := vzWriteVolumeFile(t, "a.img")
	r := vzVolumeSpecs(vzGuestArtefacts{Disk: disk}, ApplyRunOptions(WithVolumes(map[string]string{
		volB: "/srv/b",
		volA: "/data/a:ro",
	})))
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	specs := core.MustCast[[]vzVolumeSpec](r)
	if len(specs) != 3 {
		t.Fatalf("expected 3 specs, got %d", len(specs))
	}
	if specs[0].Source != disk || specs[0].Target != "/" || specs[0].ReadOnly {
		t.Fatalf("expected read-write root disk first, got %+v", specs[0])
	}
	if specs[1].Target != "/data/a" || !specs[1].ReadOnly || specs[1].Source != volA {
		t.Fatalf("expected sorted read-only /data/a second, got %+v", specs[1])
	}
	if specs[2].Target != "/srv/b" || specs[2].ReadOnly || specs[2].Source != volB {
		t.Fatalf("expected read-write /srv/b third, got %+v", specs[2])
	}
}

func TestVz_VolumeSpecs_Bad(t *testing.T) {
	auditTarget := "vzVolumeSpecs"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A missing volume source fails with the file named.
	missing := core.JoinPath(t.TempDir(), "missing.img")
	r := vzVolumeSpecs(vzGuestArtefacts{}, ApplyRunOptions(WithVolumes(map[string]string{missing: "/data"})))
	if r.OK {
		t.Fatal("expected error for missing source")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), missing) {
		t.Fatalf("expected source-naming error, got %v", r.Value)
	}
	// A directory source is not a raw image.
	r2 := vzVolumeSpecs(vzGuestArtefacts{}, ApplyRunOptions(WithVolumes(map[string]string{t.TempDir(): "/data"})))
	if r2.OK {
		t.Fatal("expected error for directory source")
	}
	// An empty target (also the bare ":ro" form) is refused.
	vol := vzWriteVolumeFile(t, "v.img")
	if r := vzVolumeSpecs(vzGuestArtefacts{}, ApplyRunOptions(WithVolumes(map[string]string{vol: ":ro"}))); r.OK {
		t.Fatal("expected error for empty target")
	}
	// A volume targeting / collides with the root disk's seat.
	if r := vzVolumeSpecs(vzGuestArtefacts{}, ApplyRunOptions(WithVolumes(map[string]string{vol: "/"}))); r.OK {
		t.Fatal("expected error for / target")
	}
}

func TestVz_VolumeSpecs_Ugly(t *testing.T) {
	auditTarget := "vzVolumeSpecs"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// No disk, no volumes → empty plan, storage stays unset.
	r := vzVolumeSpecs(vzGuestArtefacts{}, ApplyRunOptions())
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if got := len(core.MustCast[[]vzVolumeSpec](r)); got != 0 {
		t.Fatalf("expected empty plan, got %d", got)
	}
	// Two sources claiming one target is ambiguous — refused, not last-wins.
	volA := vzWriteVolumeFile(t, "a.img")
	volB := vzWriteVolumeFile(t, "b.img")
	r2 := vzVolumeSpecs(vzGuestArtefacts{}, ApplyRunOptions(WithVolumes(map[string]string{
		volA: "/data",
		volB: "/data:ro",
	})))
	if r2.OK {
		t.Fatal("expected error for duplicate target")
	}
	if err, ok := r2.Value.(error); !ok || !core.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate-target error, got %v", r2.Value)
	}
}

func TestVz_AttachStorage_Good(t *testing.T) {
	auditTarget := "vzAttachStorage"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Attachment construction opens the image files but needs no entitlement
	// and no boot — the §2.2 construction/validation split.
	config := vz.NewVZVirtualMachineConfiguration()
	disk := vzWriteVolumeFile(t, "disk.img")
	vol := vzWriteVolumeFile(t, "data.img")
	r := vzAttachStorage(config, []vzVolumeSpec{
		{Source: disk, Target: "/"},
		{Source: vol, Target: "/data", ReadOnly: true},
	})
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if got := len(config.StorageDevices()); got != 2 {
		t.Fatalf("expected 2 storage devices, got %d", got)
	}
}

func TestVz_AttachStorage_Bad(t *testing.T) {
	auditTarget := "vzAttachStorage"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// A vanished source fails at attachment construction with the file named
	// — the framework opens the image here, not at boot.
	config := vz.NewVZVirtualMachineConfiguration()
	missing := core.JoinPath(t.TempDir(), "gone.img")
	r := vzAttachStorage(config, []vzVolumeSpec{{Source: missing, Target: "/data"}})
	if r.OK {
		t.Fatal("expected error for missing image file")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), missing) {
		t.Fatalf("expected file-naming error, got %v", r.Value)
	}
}

func TestVz_AttachStorage_Ugly(t *testing.T) {
	auditTarget := "vzAttachStorage"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// An empty plan is a no-op: no devices set, no framework calls, OK.
	config := vz.NewVZVirtualMachineConfiguration()
	if r := vzAttachStorage(config, nil); !r.OK {
		t.Fatalf("expected ok for empty plan, got %v", r.Value)
	}
	if got := len(config.StorageDevices()); got != 0 {
		t.Fatalf("expected no storage devices, got %d", got)
	}
}

func TestVz_AttachFileSystems_Good(t *testing.T) {
	auditTarget := "vzAttachFileSystems"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// virtio-fs device construction opens no image and needs no entitlement
	// and no boot — the same construction/validation split as storage,
	// verified empirically by this test running unsigned. Shares attach in
	// slice order (no sort: insertion order is the contract).
	config := vz.NewVZVirtualMachineConfiguration()
	r := vzAttachFileSystems(config, []FSShare{
		{HostDir: t.TempDir(), Tag: "workspace"},
		{HostDir: t.TempDir(), Tag: "inputs", ReadOnly: true},
	})
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if got := len(config.DirectorySharingDevices()); got != 2 {
		t.Fatalf("expected 2 directory-sharing devices, got %d", got)
	}
}

func TestVz_AttachFileSystems_Bad(t *testing.T) {
	auditTarget := "vzAttachFileSystems"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Validation happens before any framework call, so the Bad contract holds
	// even on an entitlement-less / non-darwin host — no IsVZAvailable gate.
	config := vz.NewVZVirtualMachineConfiguration()
	// An empty host directory is refused.
	if r := vzAttachFileSystems(config, []FSShare{{HostDir: "", Tag: "workspace"}}); r.OK {
		t.Fatal("expected error for empty host directory")
	}
	// An empty tag is refused — never silently auto-derived from the path.
	if r := vzAttachFileSystems(config, []FSShare{{HostDir: t.TempDir(), Tag: ""}}); r.OK {
		t.Fatal("expected error for empty tag")
	}
	// A host path that is not a directory fails with the path named.
	missing := core.JoinPath(t.TempDir(), "nope")
	r := vzAttachFileSystems(config, []FSShare{{HostDir: missing, Tag: "workspace"}})
	if r.OK {
		t.Fatal("expected error for non-directory host path")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), missing) {
		t.Fatalf("expected path-naming error, got %v", r.Value)
	}
	// A file (not a directory) is also refused — a share is a directory.
	file := vzWriteVolumeFile(t, "data.img")
	if r := vzAttachFileSystems(config, []FSShare{{HostDir: file, Tag: "workspace"}}); r.OK {
		t.Fatal("expected error for file host path")
	}
}

func TestVz_AttachFileSystems_Ugly(t *testing.T) {
	auditTarget := "vzAttachFileSystems"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// An empty share set is a no-op: no devices set, no framework calls, OK —
	// so the existing BuildConfiguration tests see no new device list. Holds
	// without entitlement (the early return precedes any framework call).
	config := vz.NewVZVirtualMachineConfiguration()
	if r := vzAttachFileSystems(config, nil); !r.OK {
		t.Fatalf("expected ok for empty share set, got %v", r.Value)
	}
	if got := len(config.DirectorySharingDevices()); got != 0 {
		t.Fatalf("expected no directory-sharing devices, got %d", got)
	}
	// Two shares claiming one tag are ambiguous — refused, not last-wins.
	r := vzAttachFileSystems(config, []FSShare{
		{HostDir: t.TempDir(), Tag: "dup"},
		{HostDir: t.TempDir(), Tag: "dup"},
	})
	if r.OK {
		t.Fatal("expected error for duplicate tag")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate-tag error, got %v", r.Value)
	}
}

func TestVz_BuildConfiguration_Shares_Good(t *testing.T) {
	auditTarget := "vzBuildConfiguration"
	auditVariant := "Shares"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// WithSharedDir threads through ApplyRunOptions into the built config as a
	// directory-sharing device — the SP3-U1 workspace path. Construction stays
	// unentitled (no boot).
	dir := vzWriteGuestDir(t, true, "console=hvc0")
	art := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))
	r := vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "vm.log"),
		ApplyRunOptions(WithSharedDir(t.TempDir(), "workspace")))
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	config := core.MustCast[vz.VZVirtualMachineConfiguration](r)
	if got := len(config.DirectorySharingDevices()); got != 1 {
		t.Fatalf("expected 1 directory-sharing device, got %d", got)
	}
	// A bad share fails the whole build (the share error surfaces, §7).
	r2 := vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "vm2.log"),
		ApplyRunOptions(WithSharedDir(core.JoinPath(t.TempDir(), "nope"), "workspace")))
	if r2.OK {
		t.Fatal("expected build failure for a non-directory share")
	}
}

func TestVz_BuildConfiguration_Good(t *testing.T) {
	auditTarget := "vzBuildConfiguration"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Constructing a configuration needs no entitlement and no boot —
	// verified empirically by this test running unsigned. (Validation DOES
	// need the entitlement — see TestVz_ValidateConfiguration_Good.)
	dir := vzWriteGuestDir(t, true, "console=hvc0")
	art := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))
	logPath := core.JoinPath(t.TempDir(), "vm.log")

	r := vzBuildConfiguration(art, logPath, ApplyRunOptions(WithMemory(1024), WithCPUs(1)))
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	config := core.MustCast[vz.VZVirtualMachineConfiguration](r)
	if config.ID == 0 {
		t.Fatal("expected a live configuration object")
	}
	// §5 control channel: exactly one socket device rides every VM config.
	if got := len(config.SocketDevices()); got != 1 {
		t.Fatalf("expected 1 socket device, got %d", got)
	}
	// No disk.img, no volumes → no storage devices on the config.
	if got := len(config.StorageDevices()); got != 0 {
		t.Fatalf("expected no storage devices, got %d", got)
	}

	// With a root disk and a declared volume the config carries both, in
	// plan order (§4 root first). The root image must be sector-aligned RAW
	// — the attachment constructor validates the format (see
	// vzWriteVolumeFile).
	if err := coreio.Local.Write(core.JoinPath(dir, "disk.img"), string(make([]byte, 1<<20))); err != nil {
		t.Fatalf("write disk: %v", err)
	}
	vol := vzWriteVolumeFile(t, "data.img")
	art2 := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))
	r2 := vzBuildConfiguration(art2, core.JoinPath(t.TempDir(), "vm2.log"),
		ApplyRunOptions(WithMemory(1024), WithCPUs(1), WithVolumes(map[string]string{vol: "/data:ro"})))
	if !r2.OK {
		t.Fatalf("expected ok with storage, got %v", r2.Value)
	}
	if got := len(core.MustCast[vz.VZVirtualMachineConfiguration](r2).StorageDevices()); got != 2 {
		t.Fatalf("expected 2 storage devices, got %d", got)
	}
}

func TestVz_BuildConfiguration_Bad(t *testing.T) {
	auditTarget := "vzBuildConfiguration"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// GPU passthrough is rejected before any framework call.
	r := vzBuildConfiguration(vzGuestArtefacts{Kernel: "/k", Initrd: "/i", Cmdline: "c"}, "/tmp/x.log", RunOptions{GPU: true})
	if r.OK {
		t.Fatal("expected error for GPU request")
	}
	// A missing serial log path is rejected before any framework call.
	r2 := vzBuildConfiguration(vzGuestArtefacts{Kernel: "/k", Initrd: "/i", Cmdline: "c"}, "", RunOptions{})
	if r2.OK {
		t.Fatal("expected error for empty log path")
	}
}

func TestVz_BuildConfiguration_Ugly(t *testing.T) {
	auditTarget := "vzBuildConfiguration"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Zero memory/cpu fall back to defaults and clamp into the framework's
	// envelope; an absurd request clamps instead of failing validation.
	dir := vzWriteGuestDir(t, false, "")
	art := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))

	r := vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "a.log"), RunOptions{})
	if !r.OK {
		t.Fatalf("expected ok for defaults, got %v", r.Value)
	}
	r2 := vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "b.log"), RunOptions{Memory: 1 << 30, CPUs: 4096})
	if !r2.OK {
		t.Fatalf("expected ok for clamped extremes, got %v", r2.Value)
	}
}

// vzEntitlementErrorText is the framework's verbatim complaint when the
// calling process lacks com.apple.security.virtualization. Pinned so the
// validation tests distinguish "unentitled but healthy" from real breakage.
const vzEntitlementErrorText = "com.apple.security.virtualization"

func TestVz_ValidateConfiguration_Good(t *testing.T) {
	auditTarget := "vzValidateConfiguration"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Empirical contract: validation requires the virtualization
	// entitlement. An entitled process validates this complete config OK;
	// an unentitled one (plain `go test`) gets the entitlement error
	// verbatim — any OTHER failure is a regression.
	dir := vzWriteGuestDir(t, true, "console=hvc0")
	art := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))
	config := core.MustCast[vz.VZVirtualMachineConfiguration](
		vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "v.log"), ApplyRunOptions()))

	r := vzValidateConfiguration(config)
	if r.OK {
		return // entitled process — full pass
	}
	err, ok := r.Value.(error)
	if !ok {
		t.Fatalf("expected error value, got %T", r.Value)
	}
	if !core.Contains(err.Error(), vzEntitlementErrorText) {
		t.Fatalf("expected entitlement error in unentitled process, got %q", err.Error())
	}
}

func TestVz_ValidateConfiguration_Bad(t *testing.T) {
	auditTarget := "vzValidateConfiguration"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// An empty configuration (no boot loader) must never validate OK —
	// entitled processes get the missing-bootloader error, unentitled ones
	// the entitlement error. Either way: clean failure, no panic.
	config := vz.NewVZVirtualMachineConfiguration()
	if r := vzValidateConfiguration(config); r.OK {
		t.Fatal("expected error for empty configuration")
	}
}

func TestVz_ValidateConfiguration_Ugly(t *testing.T) {
	auditTarget := "vzValidateConfiguration"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Repeated validation of the same object is stable.
	dir := vzWriteGuestDir(t, false, "")
	art := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))
	config := core.MustCast[vz.VZVirtualMachineConfiguration](
		vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "w.log"), ApplyRunOptions()))
	first := vzValidateConfiguration(config)
	second := vzValidateConfiguration(config)
	if first.OK != second.OK {
		t.Fatalf("validation flapped: first %v, second %v", first.OK, second.OK)
	}
}

func TestVz_ClampMemoryBytes_Good(t *testing.T) {
	auditTarget := "vzClampMemoryBytes"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	got := vzClampMemoryBytes(2048)
	if want := uint64(2048) * 1024 * 1024; got != want {
		t.Fatalf("want %d, got %d", want, got)
	}
}

func TestVz_ClampMemoryBytes_Bad(t *testing.T) {
	auditTarget := "vzClampMemoryBytes"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Zero and negative requests resolve to the default, inside the envelope.
	if got := vzClampMemoryBytes(0); got == 0 {
		t.Fatal("expected non-zero default")
	}
	if got := vzClampMemoryBytes(-5); got == 0 {
		t.Fatal("expected non-zero default")
	}
}

func TestVz_ClampMemoryBytes_Ugly(t *testing.T) {
	auditTarget := "vzClampMemoryBytes"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// 1MB request sits below the framework minimum and clamps up to it.
	got := vzClampMemoryBytes(1)
	if got < uint64(1)*1024*1024 {
		t.Fatalf("expected clamp >= request, got %d", got)
	}
}

func TestVz_Run_Good(t *testing.T) {
	auditTarget := "VZProvider Run"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// The Good path for Run is a live boot — exercised by
	// TestVz_Run_LiveBoot_Good under its double gate. Here the pre-boot
	// pipeline (artefacts + config) is proven OK-shaped without booting.
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	dir := vzWriteGuestDir(t, true, "console=hvc0")
	art := core.MustCast[vzGuestArtefacts](vzResolveGuestArtefacts(dir))
	r := vzBuildConfiguration(art, core.JoinPath(t.TempDir(), "run.log"), ApplyRunOptions())
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
}

func TestVz_Run_Bad(t *testing.T) {
	auditTarget := "VZProvider Run"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	if r := p.Run(nil); r.OK {
		t.Fatal("expected error for nil image")
	}
	if r := p.Run(&Image{}); r.OK {
		t.Fatal("expected error for empty image path")
	}
	if !p.Available() {
		t.Skip("virtualization framework not available")
	}
	// A directory without §4 artefacts fails before any VM is created.
	r := p.Run(&Image{Path: t.TempDir()})
	if r.OK {
		t.Fatal("expected error for missing artefacts")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "kernel") {
		t.Fatalf("expected kernel-naming error, got %v", r.Value)
	}
}

func TestVz_Run_Ugly(t *testing.T) {
	auditTarget := "VZProvider Run"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// On a simulated non-darwin host every verb fails with the §7 sentinel.
	t.Setenv("GOOS", "linux")
	p := NewVZProvider()
	r := p.Run(&Image{Path: "/nonexistent"})
	if r.OK {
		t.Fatal("expected error")
	}
	err, ok := r.Value.(error)
	if !ok {
		t.Fatalf("expected error value, got %T", r.Value)
	}
	if !core.Contains(err.Error(), "virtualization framework unavailable") {
		t.Fatalf("expected §7 sentinel, got %q", err.Error())
	}
}

func TestVz_Stop_Good(t *testing.T) {
	auditTarget := "VZProvider Stop"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// The Good Stop is exercised against a live VM in
	// TestVz_Run_LiveBoot_Good; without a VM the contract is clean failure.
	p := NewVZProvider()
	if r := p.Stop("never-ran"); r.OK {
		t.Fatal("expected error")
	}
	// Double-stop: a VM the watcher already saw exit stops idempotently —
	// Ok without touching the dispatch queue (provider conformance).
	if IsVZAvailable() {
		vzFabricateTracked(t, p, &Container{ID: "vz-stop-exited", Status: StatusStopped})
		if r := p.Stop("vz-stop-exited"); !r.OK {
			t.Fatalf("expected idempotent stop, got %v", r.Value)
		}
	}
}

func TestVz_Stop_Bad(t *testing.T) {
	auditTarget := "VZProvider Stop"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	if !p.Available() {
		t.Skip("virtualization framework not available")
	}
	r := p.Stop("")
	if r.OK {
		t.Fatal("expected error for empty id")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "required") {
		t.Fatalf("expected id-required error, got %v", r.Value)
	}
}

func TestVz_Stop_Ugly(t *testing.T) {
	auditTarget := "VZProvider Stop"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	t.Setenv("GOOS", "linux")
	p := NewVZProvider()
	r := p.Stop("anything")
	if r.OK {
		t.Fatal("expected error")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "virtualization framework unavailable") {
		t.Fatalf("expected §7 sentinel, got %v", r.Value)
	}
}

func TestVz_Kill_Good(t *testing.T) {
	auditTarget := "VZProvider Kill"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// As with Stop: the live kill runs under the live gate; without a VM the
	// contract is clean failure for an untracked id.
	p := NewVZProvider()
	if r := p.Kill("never-ran"); r.OK {
		t.Fatal("expected error")
	}
	// Kill after exit is idempotent, like Stop (provider conformance).
	if IsVZAvailable() {
		vzFabricateTracked(t, p, &Container{ID: "vz-kill-exited", Status: StatusKilled})
		if r := p.Kill("vz-kill-exited"); !r.OK {
			t.Fatalf("expected idempotent kill, got %v", r.Value)
		}
	}
}

func TestVz_Kill_Bad(t *testing.T) {
	auditTarget := "VZProvider Kill"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	if !p.Available() {
		t.Skip("virtualization framework not available")
	}
	if r := p.Kill(""); r.OK {
		t.Fatal("expected error for empty id")
	}
}

func TestVz_Kill_Ugly(t *testing.T) {
	auditTarget := "VZProvider Kill"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	t.Setenv("GOOS", "linux")
	p := NewVZProvider()
	r := p.Kill("anything")
	if r.OK {
		t.Fatal("expected error")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "virtualization framework unavailable") {
		t.Fatalf("expected §7 sentinel, got %v", r.Value)
	}
}

func TestVz_Wait_Good(t *testing.T) {
	auditTarget := "VZProvider Wait"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Wait unblocks the moment the watcher observes guest exit (Done).
	p := NewVZProvider()
	entry := vzFabricateTracked(t, p, &Container{ID: "vz-wait-good", Status: StatusRunning})
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(entry.Done)
	}()
	if r := p.Wait(context.Background(), "vz-wait-good"); !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
}

func TestVz_Wait_Bad(t *testing.T) {
	auditTarget := "VZProvider Wait"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	if r := p.Wait(context.Background(), ""); r.OK {
		t.Fatal("expected error for empty id")
	}
	r := p.Wait(context.Background(), "never-ran")
	if r.OK {
		t.Fatal("expected error for untracked id")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "not tracked") {
		t.Fatalf("expected not-tracked error, got %v", r.Value)
	}
}

func TestVz_Wait_Ugly(t *testing.T) {
	auditTarget := "VZProvider Wait"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// A cancelled context aborts the wait with the cancellation named — the
	// VM (Done never closed) is untouched.
	p := NewVZProvider()
	vzFabricateTracked(t, p, &Container{ID: "vz-wait-ugly", Status: StatusRunning})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := p.Wait(ctx, "vz-wait-ugly")
	if r.OK {
		t.Fatal("expected error for cancelled context")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "context cancelled") {
		t.Fatalf("expected cancellation error, got %v", r.Value)
	}
}

func TestVz_Logs_Good(t *testing.T) {
	auditTarget := "VZProvider Logs"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Logs tails the §3 serial capture for the id — no tracked entry needed,
	// an exited VM's log stays readable.
	id := "vz-test-logs-" + core.MustCast[string](GenerateID())
	if r := EnsureLogsDir(); !r.OK {
		t.Fatalf("ensure logs dir: %v", r.Value)
	}
	logPath := core.MustCast[string](LogPath(id))
	if err := coreio.Local.Write(logPath, "boot one\nboot two\nboot three\n"); err != nil {
		t.Fatalf("write log: %v", err)
	}
	t.Cleanup(func() { _ = coreio.Local.Delete(logPath) })

	p := NewVZProvider()
	r := p.Logs(id, 2)
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if got, want := core.MustCast[string](r), "boot two\nboot three"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestVz_Logs_Bad(t *testing.T) {
	auditTarget := "VZProvider Logs"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	if r := p.Logs("", 10); r.OK {
		t.Fatal("expected error for empty id")
	}
	// An id with no serial capture fails with the id named.
	r := p.Logs("vz-never-logged-"+core.MustCast[string](GenerateID()), 10)
	if r.OK {
		t.Fatal("expected error for unknown id")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "no serial log") {
		t.Fatalf("expected no-serial-log error, got %v", r.Value)
	}
}

func TestVz_Logs_Ugly(t *testing.T) {
	auditTarget := "VZProvider Logs"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// tail <= 0 falls back to the 200-line AppleProvider-parity default —
	// shorter logs come back whole.
	id := "vz-test-logs-ugly-" + core.MustCast[string](GenerateID())
	if r := EnsureLogsDir(); !r.OK {
		t.Fatalf("ensure logs dir: %v", r.Value)
	}
	logPath := core.MustCast[string](LogPath(id))
	if err := coreio.Local.Write(logPath, "alpha\nbeta\n"); err != nil {
		t.Fatalf("write log: %v", err)
	}
	t.Cleanup(func() { _ = coreio.Local.Delete(logPath) })

	p := NewVZProvider()
	r := p.Logs(id, 0)
	if !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if got, want := core.MustCast[string](r), "alpha\nbeta"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	if r := p.Logs(id, -7); !r.OK {
		t.Fatalf("expected ok for negative tail, got %v", r.Value)
	}
}

func TestVz_Remove_Good(t *testing.T) {
	auditTarget := "VZProvider Remove"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Remove drops an exited VM from both inventories: tracked map AND the
	// §3 registry file.
	p := NewVZProvider()
	p.StatePath = core.JoinPath(t.TempDir(), "containers.json")
	ctr := &Container{ID: "vz-remove-good", Status: StatusStopped}
	vzFabricateTracked(t, p, ctr)
	if r := p.persistAdd(ctr); !r.OK {
		t.Fatalf("persist add: %v", r.Value)
	}

	if r := p.Remove(ctr.ID); !r.OK {
		t.Fatalf("expected ok, got %v", r.Value)
	}
	if len(p.Tracked()) != 0 {
		t.Fatal("expected tracked entry gone")
	}
	fresh := core.MustCast[*State](LoadState(p.StatePath))
	if _, still := fresh.Get(ctr.ID); still {
		t.Fatal("expected registry entry gone")
	}
}

func TestVz_Remove_Bad(t *testing.T) {
	auditTarget := "VZProvider Remove"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	p.StatePath = core.JoinPath(t.TempDir(), "containers.json")
	if r := p.Remove(""); r.OK {
		t.Fatal("expected error for empty id")
	}
	r := p.Remove("never-existed")
	if r.OK {
		t.Fatal("expected error for unknown id")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "not tracked") {
		t.Fatalf("expected not-tracked error, got %v", r.Value)
	}
}

func TestVz_Remove_Ugly(t *testing.T) {
	auditTarget := "VZProvider Remove"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// A running VM is refused; the same id removes cleanly once stopped.
	p := NewVZProvider()
	p.StatePath = core.JoinPath(t.TempDir(), "containers.json")
	ctr := &Container{ID: "vz-remove-ugly", Status: StatusRunning}
	entry := vzFabricateTracked(t, p, ctr)

	r := p.Remove(ctr.ID)
	if r.OK {
		t.Fatal("expected refusal for a running container")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "running") {
		t.Fatalf("expected running refusal, got %v", r.Value)
	}

	vzProviderLock.Lock()
	entry.Container.Status = StatusStopped
	vzProviderLock.Unlock()
	if r := p.Remove(ctr.ID); !r.OK {
		t.Fatalf("expected ok after stop, got %v", r.Value)
	}
}

func TestVz_Registry_Good(t *testing.T) {
	auditTarget := "VZProvider registry"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// persistAdd lands the container in the registry file; persistUpdate
	// syncs a status transition — the §3 Add/Update lifecycle.
	p := NewVZProvider()
	p.StatePath = core.JoinPath(t.TempDir(), "containers.json")
	ctr := &Container{ID: "vz-reg-good", Status: StatusRunning, Image: "/img"}
	if r := p.persistAdd(ctr); !r.OK {
		t.Fatalf("persist add: %v", r.Value)
	}
	loaded := core.MustCast[*State](LoadState(p.StatePath))
	got, ok := loaded.Get(ctr.ID)
	if !ok || got.Status != StatusRunning {
		t.Fatalf("expected running record, got %+v ok=%v", got, ok)
	}

	entry := vzFabricateTracked(t, p, ctr)
	vzProviderLock.Lock()
	entry.Container.Status = StatusStopped
	vzProviderLock.Unlock()
	p.persistUpdate(entry)
	reloaded := core.MustCast[*State](LoadState(p.StatePath))
	got2, ok2 := reloaded.Get(ctr.ID)
	if !ok2 || got2.Status != StatusStopped {
		t.Fatalf("expected stopped record, got %+v ok=%v", got2, ok2)
	}
}

func TestVz_Registry_Bad(t *testing.T) {
	auditTarget := "VZProvider registry"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// A registry whose parent path is a FILE cannot persist — persistAdd
	// fails loudly (this is the path that fails a live Run, LinuxKit
	// precedent).
	blocker := core.JoinPath(t.TempDir(), "blocker")
	if err := coreio.Local.Write(blocker, "in the way"); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	p := NewVZProvider()
	p.StatePath = core.JoinPath(blocker, "containers.json")
	if r := p.persistAdd(&Container{ID: "vz-reg-bad"}); r.OK {
		t.Fatal("expected persist failure")
	}

	// A corrupt registry file refuses to load.
	corrupt := core.JoinPath(t.TempDir(), "containers.json")
	if err := coreio.Local.Write(corrupt, "{not json"); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	p2 := NewVZProvider()
	p2.StatePath = corrupt
	if r := p2.registry(); r.OK {
		t.Fatal("expected corrupt registry to refuse loading")
	}
}

func TestVz_Registry_Ugly(t *testing.T) {
	auditTarget := "VZProvider registry"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	// Two providers sharing one StatePath see one inventory (§3): a record
	// added through the first is visible to — and removable through — the
	// second.
	path := core.JoinPath(t.TempDir(), "containers.json")
	p1 := NewVZProvider()
	p1.StatePath = path
	ctr := &Container{ID: "vz-reg-ugly", Status: StatusStopped}
	if r := p1.persistAdd(ctr); !r.OK {
		t.Fatalf("persist add: %v", r.Value)
	}

	p2 := NewVZProvider()
	p2.StatePath = path
	if r := p2.Remove(ctr.ID); !r.OK {
		t.Fatalf("expected cross-provider remove, got %v", r.Value)
	}
	final := core.MustCast[*State](LoadState(path))
	if _, still := final.Get(ctr.ID); still {
		t.Fatal("expected one shared inventory, record still present")
	}
}

func TestVz_Registry_Cache_Ugly(t *testing.T) {
	auditTarget := "VZProvider registry cache"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	p.StatePath = core.JoinPath(t.TempDir(), "containers.json")

	firstRes := p.registry()
	if !firstRes.OK {
		t.Fatalf("first registry load: %v", firstRes.Value)
	}
	secondRes := p.registry()
	if !secondRes.OK {
		t.Fatalf("second registry load: %v", secondRes.Value)
	}
	if core.MustCast[*State](firstRes) != core.MustCast[*State](secondRes) {
		t.Fatal("expected registry cache to return the same State pointer")
	}
}

func TestVz_Tracked_Good(t *testing.T) {
	auditTarget := "VZProvider Tracked"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	got := p.Tracked()
	if got == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestVz_Tracked_Bad(t *testing.T) {
	auditTarget := "VZProvider Tracked"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Mutating a snapshot must not affect provider state.
	p := NewVZProvider()
	snapshot := p.Tracked()
	snapshot = append(snapshot, &Container{ID: "intruder"})
	if len(p.Tracked()) != 0 {
		t.Fatalf("snapshot mutation leaked into provider: %v", snapshot)
	}
}

func TestVz_Tracked_Ugly(t *testing.T) {
	auditTarget := "VZProvider Tracked"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Concurrent snapshots must not race (run with -race).
	p := NewVZProvider()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			p.Tracked()
		}
		close(done)
	}()
	for i := 0; i < 50; i++ {
		p.Tracked()
	}
	<-done
}

func TestVz_Tracked_WithEntry_Ugly(t *testing.T) {
	auditTarget := "VZProvider Tracked with entry"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewVZProvider()
	entry := vzFabricateTracked(t, p, &Container{ID: "vz-tracked-entry", Status: StatusRunning})

	if got := p.entry(entry.Container.ID); got != entry {
		t.Fatal("entry lookup did not return the tracked record")
	}
	p.setStatus(entry, StatusStopped)
	if got := p.status(entry); got != StatusStopped {
		t.Fatalf("status = %s, want %s", got, StatusStopped)
	}

	snapshot := p.Tracked()
	if len(snapshot) != 1 {
		t.Fatalf("expected one tracked snapshot, got %d", len(snapshot))
	}
	snapshot[0].Status = StatusError
	if got := p.status(entry); got != StatusStopped {
		t.Fatalf("snapshot mutation changed live status to %s", got)
	}
}

func TestVz_PersistUpdate_RegistryFailure_Ugly(t *testing.T) {
	auditTarget := "VZProvider persistUpdate registry failure"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	corrupt := core.JoinPath(t.TempDir(), "containers.json")
	if err := coreio.Local.Write(corrupt, "{not json"); err != nil {
		t.Fatalf("write corrupt registry: %v", err)
	}
	p := NewVZProvider()
	p.StatePath = corrupt
	entry := &vzTracked{Container: &Container{ID: "vz-persist-update", Status: StatusStopped}}

	p.persistUpdate(entry)
	if got := p.status(entry); got != StatusStopped {
		t.Fatalf("persistUpdate changed live status to %s", got)
	}
}

func TestVz_SerialLogTail_Good(t *testing.T) {
	auditTarget := "vzSerialLogTail"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	logPath := core.JoinPath(t.TempDir(), "boot.log")
	if err := coreio.Local.Write(logPath, "one\ntwo\nthree\nfour\nfive\nsix\nseven\n"); err != nil {
		t.Fatalf("write log: %v", err)
	}
	got := vzSerialLogTail(logPath, 3)
	if got != "five\nsix\nseven" {
		t.Fatalf("unexpected tail %q", got)
	}
}

func TestVz_SerialLogTail_Bad(t *testing.T) {
	auditTarget := "vzSerialLogTail"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if got := vzSerialLogTail("", 3); got != "" {
		t.Fatalf("expected empty for empty path, got %q", got)
	}
	if got := vzSerialLogTail(core.JoinPath(t.TempDir(), "missing.log"), 3); got != "" {
		t.Fatalf("expected empty for missing file, got %q", got)
	}
}

func TestVz_SerialLogTail_Ugly(t *testing.T) {
	auditTarget := "vzSerialLogTail"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Fewer lines than requested returns everything, no padding, no panic.
	logPath := core.JoinPath(t.TempDir(), "short.log")
	if err := coreio.Local.Write(logPath, "only\n"); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if got := vzSerialLogTail(logPath, 10); got != "only" {
		t.Fatalf("unexpected tail %q", got)
	}
}

func TestVz_Exec_Good(t *testing.T) {
	auditTarget := "VZProvider Exec"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// The Good exec is a live guest round-trip — TestVz_Exec_LiveAgent_Good
	// under its triple gate. Without a VM the contract is: a tracked but
	// not-running container is refused BEFORE any vsock dial.
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	ctr := &Container{ID: "vz-exec-good", Status: StatusStopped}
	vzFabricateTracked(t, p, ctr)
	r := p.Exec(ctr.ID, "uname", "-a")
	if r.OK {
		t.Fatal("expected refusal for a stopped container")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "not running") {
		t.Fatalf("expected not-running refusal, got %v", r.Value)
	}
}

func TestVz_Exec_Bad(t *testing.T) {
	auditTarget := "VZProvider Exec"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	if r := p.Exec("", "uname"); r.OK {
		t.Fatal("expected error for empty id")
	}
	if r := p.Exec("some-id", ""); r.OK {
		t.Fatal("expected error for empty command")
	}
	r := p.Exec("never-ran", "uname")
	if r.OK {
		t.Fatal("expected error for untracked id")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "not tracked") {
		t.Fatalf("expected not-tracked error, got %v", r.Value)
	}
}

func TestVz_Exec_Ugly(t *testing.T) {
	auditTarget := "VZProvider Exec"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// On a simulated non-darwin host Exec fails with the §7 sentinel.
	t.Setenv("GOOS", "linux")
	p := NewVZProvider()
	r := p.Exec("anything", "uname")
	if r.OK {
		t.Fatal("expected error")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "virtualization framework unavailable") {
		t.Fatalf("expected §7 sentinel, got %v", r.Value)
	}
}

// TestVz_Exec_LiveAgent_Good is the §6 Phase B proof: Exec(id, "uname",
// "-a") returns guest output over the vsock control channel. Triple-gated:
// CONTAINER_VZ_LIVE=1, CONTAINER_VZ_LIVE_AGENT=1 (the fixture image must
// carry vzagent — see testdata/vz/linuxkit-vzagent.yml), and real artefacts
// at testdata/vz/.
func TestVz_Exec_LiveAgent_Good(t *testing.T) {
	auditTarget := "VZProvider Exec LiveAgent"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if core.Env("CONTAINER_VZ_LIVE") != "1" {
		t.Skip("live boot gated: set CONTAINER_VZ_LIVE=1")
	}
	if core.Env("CONTAINER_VZ_LIVE_AGENT") != "1" {
		t.Skip("live agent gated: set CONTAINER_VZ_LIVE_AGENT=1 (image must carry vzagent)")
	}
	if !coreio.Local.IsFile(core.JoinPath(vzLiveFixtureDir, "kernel")) ||
		!coreio.Local.IsFile(core.JoinPath(vzLiveFixtureDir, "initrd.img")) {
		t.Skip("live boot gated: no LinuxKit artefacts at " + vzLiveFixtureDir)
	}
	p := NewVZProvider()
	if !p.Available() {
		t.Skip("virtualization framework not available")
	}

	r := p.Run(&Image{Path: vzLiveFixtureDir}, WithMemory(1024), WithCPUs(1), WithName("vz-live-agent-test"))
	if !r.OK {
		t.Fatalf("boot failed: %v", r.Value)
	}
	ctr := core.MustCast[*Container](r)
	defer func() { _ = p.Kill(ctr.ID) }()

	// Give the guest a moment to bring up the vsock driver and the agent.
	time.Sleep(10 * time.Second)

	execRes := p.Exec(ctr.ID, "uname", "-a")
	if !execRes.OK {
		t.Fatalf("exec failed: %v", execRes.Value)
	}
	out := core.MustCast[string](execRes)
	if !core.Contains(out, "Linux") {
		t.Fatalf("expected guest uname output, got %q", out)
	}

	// §5 graceful stop: the agent acks, the guest powers off, no force stop.
	if r := p.Stop(ctr.ID); !r.OK {
		t.Fatalf("graceful stop failed: %v", r.Value)
	}
}

func TestVz_ExecResult_Good(t *testing.T) {
	auditTarget := "VZProvider ExecResult"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// The whole verb over a live VM is TestVz_ExecResult_LiveAgent_Good. The
	// contract that distinguishes ExecResult from Exec is the Response→result
	// mapping (vzExecResult), which unit-tests without a VM: a command that ran
	// and exited succeeds at the verb level, exit code and stderr preserved.
	r := vzExecResult(vzproto.Response{OK: true, Stdout: "Linux\n", Exit: 0})
	if !r.OK {
		t.Fatalf("expected ok for exit 0, got %v", r.Value)
	}
	if got := core.MustCast[ExecResult](r); got.Stdout != "Linux\n" || got.Exit != 0 {
		t.Fatalf("unexpected zero-exit result %+v", got)
	}
	// The gap vs Exec: a non-zero exit is a SUCCESSFUL exec — OK=true with the
	// real exit code and stderr surviving, where Exec would have folded both
	// into a failure.
	r2 := vzExecResult(vzproto.Response{OK: true, Stdout: "partial", Stderr: "boom: not found", Exit: 127})
	if !r2.OK {
		t.Fatalf("expected ok for non-zero exit (ran-and-exited), got %v", r2.Value)
	}
	got := core.MustCast[ExecResult](r2)
	if got.Exit != 127 {
		t.Fatalf("exit code lost: want 127, got %d", got.Exit)
	}
	if got.Stderr != "boom: not found" {
		t.Fatalf("stderr lost: want %q, got %q", "boom: not found", got.Stderr)
	}
	if got.Stdout != "partial" {
		t.Fatalf("stdout lost: want %q, got %q", "partial", got.Stdout)
	}
}

func TestVz_ExecResult_Bad(t *testing.T) {
	auditTarget := "VZProvider ExecResult"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A refused verb (OK=false) is a verb-level failure naming the agent's
	// error — not an ExecResult with a zero exit. Pure mapping, no VM needed.
	r := vzExecResult(vzproto.Response{OK: false, Error: "unknown verb"})
	if r.OK {
		t.Fatal("expected failure for a refused verb")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "unknown verb") {
		t.Fatalf("expected agent-error naming, got %v", r.Value)
	}

	// Verb-level argument guards mirror Exec.
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	if r := p.ExecResult("", "uname"); r.OK {
		t.Fatal("expected error for empty id")
	}
	if r := p.ExecResult("some-id", ""); r.OK {
		t.Fatal("expected error for empty command")
	}
	r2 := p.ExecResult("never-ran", "uname")
	if r2.OK {
		t.Fatal("expected error for untracked id")
	}
	if err, ok := r2.Value.(error); !ok || !core.Contains(err.Error(), "not tracked") {
		t.Fatalf("expected not-tracked error, got %v", r2.Value)
	}
}

func TestVz_ExecResult_Ugly(t *testing.T) {
	auditTarget := "VZProvider ExecResult"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A tracked but not-running container is refused BEFORE any vsock dial —
	// same guard as Exec.
	if IsVZAvailable() {
		p := NewVZProvider()
		ctr := &Container{ID: "vz-execresult-stopped", Status: StatusStopped}
		vzFabricateTracked(t, p, ctr)
		r := p.ExecResult(ctr.ID, "uname", "-a")
		if r.OK {
			t.Fatal("expected refusal for a stopped container")
		}
		if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "not running") {
			t.Fatalf("expected not-running refusal, got %v", r.Value)
		}
	}

	// On a simulated non-darwin host ExecResult fails with the §7 sentinel.
	t.Setenv("GOOS", "linux")
	p := NewVZProvider()
	r := p.ExecResult("anything", "uname")
	if r.OK {
		t.Fatal("expected error")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "virtualization framework unavailable") {
		t.Fatalf("expected §7 sentinel, got %v", r.Value)
	}
}

// TestVz_ExecResult_LiveAgent_Good is the structured-exec proof over a live
// guest: a command that exits non-zero returns OK=true with the real exit code
// and stderr (the gap vs Exec). Triple-gated exactly like
// TestVz_Exec_LiveAgent_Good.
func TestVz_ExecResult_LiveAgent_Good(t *testing.T) {
	auditTarget := "VZProvider ExecResult LiveAgent"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if core.Env("CONTAINER_VZ_LIVE") != "1" {
		t.Skip("live boot gated: set CONTAINER_VZ_LIVE=1")
	}
	if core.Env("CONTAINER_VZ_LIVE_AGENT") != "1" {
		t.Skip("live agent gated: set CONTAINER_VZ_LIVE_AGENT=1 (image must carry vzagent)")
	}
	if !coreio.Local.IsFile(core.JoinPath(vzLiveFixtureDir, "kernel")) ||
		!coreio.Local.IsFile(core.JoinPath(vzLiveFixtureDir, "initrd.img")) {
		t.Skip("live boot gated: no LinuxKit artefacts at " + vzLiveFixtureDir)
	}
	p := NewVZProvider()
	if !p.Available() {
		t.Skip("virtualization framework not available")
	}

	r := p.Run(&Image{Path: vzLiveFixtureDir}, WithMemory(1024), WithCPUs(1), WithName("vz-execresult-test"))
	if !r.OK {
		t.Fatalf("boot failed: %v", r.Value)
	}
	ctr := core.MustCast[*Container](r)
	defer func() { _ = p.Kill(ctr.ID) }()

	time.Sleep(10 * time.Second)

	// A command that exits non-zero: ExecResult succeeds and preserves the code.
	res := p.ExecResult(ctr.ID, "sh", "-c", "echo out; echo err 1>&2; exit 3")
	if !res.OK {
		t.Fatalf("execresult failed at the verb level: %v", res.Value)
	}
	got := core.MustCast[ExecResult](res)
	if got.Exit != 3 {
		t.Fatalf("expected exit 3, got %d", got.Exit)
	}
	if !core.Contains(got.Stderr, "err") {
		t.Fatalf("expected stderr captured, got %q", got.Stderr)
	}
	if !core.Contains(got.Stdout, "out") {
		t.Fatalf("expected stdout captured, got %q", got.Stdout)
	}

	if r := p.Stop(ctr.ID); !r.OK {
		t.Fatalf("graceful stop failed: %v", r.Value)
	}
}

// TestVz_Run_LiveBoot_Good is the §6 Phase A proof: a real LinuxKit
// kernel+initrd boots to a running VM and the serial log captures kernel
// output. Double-gated: CONTAINER_VZ_LIVE=1 AND real artefacts at
// testdata/vz/ — CI and entitlement-less hosts skip cleanly.
func TestVz_Run_LiveBoot_Good(t *testing.T) {
	auditTarget := "VZProvider Run LiveBoot"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if core.Env("CONTAINER_VZ_LIVE") != "1" {
		t.Skip("live boot gated: set CONTAINER_VZ_LIVE=1")
	}
	if !coreio.Local.IsFile(core.JoinPath(vzLiveFixtureDir, "kernel")) ||
		!coreio.Local.IsFile(core.JoinPath(vzLiveFixtureDir, "initrd.img")) {
		t.Skip("live boot gated: no LinuxKit artefacts at " + vzLiveFixtureDir)
	}
	p := NewVZProvider()
	if !p.Available() {
		t.Skip("virtualization framework not available")
	}

	r := p.Run(&Image{Path: vzLiveFixtureDir}, WithMemory(1024), WithCPUs(1), WithName("vz-live-test"))
	if !r.OK {
		t.Fatalf("boot failed: %v", r.Value)
	}
	ctr := core.MustCast[*Container](r)
	if ctr.Status != StatusRunning {
		t.Fatalf("expected running, got %s", ctr.Status)
	}
	if len(p.Tracked()) != 1 {
		t.Fatalf("expected 1 tracked VM, got %d", len(p.Tracked()))
	}

	// Give the kernel a moment to write boot lines to the serial console.
	time.Sleep(5 * time.Second)
	logPath := core.MustCast[string](LogPath(ctr.ID))
	if tail := vzSerialLogTail(logPath, 10); tail == "" {
		t.Fatalf("expected serial console output at %s", logPath)
	}

	if r := p.Stop(ctr.ID); !r.OK {
		t.Fatalf("stop failed: %v", r.Value)
	}
}
