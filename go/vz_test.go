//go:build darwin

package container

import (
	"testing"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

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
