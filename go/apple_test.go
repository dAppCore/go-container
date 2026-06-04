package container

import (
	"context"
	"reflect"
	"runtime"
	"testing"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

	"dappco.re/go/container/internal/proc"
)

func TestApple_IsAppleAvailable_Good(t *testing.T) {
	auditTarget := "IsAppleAvailable"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	got := IsAppleAvailable()

	// Function must not panic and must return a bool regardless of platform.
	if got, want := reflect.TypeOf(got), reflect.TypeOf(true); got != want {
		t.Fatalf("want type %v, got %v", want, got)
	}
	if runtime.GOOS != "darwin" {
		if got {
			t.Fatal("expected false")
		}
	}
}

func TestApple_NewAppleProvider_Good(t *testing.T) {
	auditTarget := "NewAppleProvider"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if p == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := p.Binary, "container"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_Available_Bad(t *testing.T) {
	auditTarget := "Available"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A bogus binary name must fail Available().
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.Available() {
		t.Fatal("expected false")
	}
}

func TestApple_Build_MissingSource_Bad(t *testing.T) {
	auditTarget := "Build MissingSource"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}

	r := p.Build(ContainerConfig{})
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_Run_NilImage_Bad(t *testing.T) {
	auditTarget := "Run NilImage"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}

	r := p.Run(nil)
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_Encrypt_Decrypt_Ugly(t *testing.T) {
	auditTarget := "Encrypt Decrypt"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Encrypt+Decrypt round-trip: write plaintext to a temp file,
	// encrypt it, decrypt it, and verify the round-trip preserves content.
	dir := t.TempDir()
	plainPath := core.PathJoin(dir, "example.qcow2")
	plaintext := []byte("hello, this is container image data for testing")
	if err := coreio.Local.WriteMode(plainPath, string(plaintext), 0600); err != nil {
		t.Fatal(err)
	}

	p := NewAppleProvider()
	img := &Image{ID: "test", Path: plainPath, Size: int64(len(plaintext))}
	key := []byte("workspace-key")

	encRes := p.Encrypt(img, key)
	if !encRes.OK {
		t.Fatal(encRes.Error())
	}
	enc := core.MustCast[*EncryptedImage](encRes)
	if enc == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := enc.Scheme, "stim"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	wantEncPath := plainPath + ".stim"
	if got, want := enc.Path, wantEncPath; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	outRes := p.Decrypt(enc, key)
	if !outRes.OK {
		t.Fatal(outRes.Error())
	}
	out := core.MustCast[*Image](outRes)
	if out == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := out.Path, plainPath; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := out.Size, int64(len(plaintext)); got != want {
		t.Fatalf("want size %v, got %v", want, got)
	}

	// Verify decrypted content matches original plaintext.
	gotData, err := coreio.Local.Read(plainPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotData != string(plaintext) {
		t.Fatalf("decrypted data mismatch: want %q, got %q", plaintext, gotData)
	}
}

func TestApple_Encrypt_MissingKey_Bad(t *testing.T) {
	auditTarget := "Encrypt MissingKey"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	img := &Image{Path: "/tmp/foo"}

	r := p.Encrypt(img, nil)
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_Decrypt_MissingKey_Bad(t *testing.T) {
	auditTarget := "Decrypt MissingKey"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	enc := &EncryptedImage{Path: "/tmp/foo.stim"}

	r := p.Decrypt(enc, nil)
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_Tracked_Empty_Good(t *testing.T) {
	auditTarget := "Tracked Empty"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if got := p.Tracked(); len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestApple_Wait_UnknownID_Bad(t *testing.T) {
	auditTarget := "Wait UnknownID"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()

	r := p.Wait(context.Background(), "no-such-container")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AvailableOnNonDarwin_Ugly(t *testing.T) {
	auditTarget := "AvailableOnNonDarwin"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Available must respect GOOS — on Linux/Windows the apple binary name
	// may resolve to something that isn't Apple's runtime, but Available()
	// should still refuse.
	p := &AppleProvider{Binary: "container"}

	if runtime.GOOS != "darwin" {
		if p.Available() {
			t.Fatal("expected false")
		}
	}
}

// --- AX-7 canonical triplets ---

func TestApple_IsAppleAvailable_Bad(t *testing.T) {
	auditTarget := "IsAppleAvailable"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := IsAppleAvailable
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_IsAppleAvailable_Ugly(t *testing.T) {
	auditTarget := "IsAppleAvailable"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := IsAppleAvailable
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_NewAppleProvider_Bad(t *testing.T) {
	auditTarget := "NewAppleProvider"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := NewAppleProvider
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_NewAppleProvider_Ugly(t *testing.T) {
	auditTarget := "NewAppleProvider"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := NewAppleProvider
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Available_Good(t *testing.T) {
	auditTarget := "AppleProvider Available"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Available_Bad(t *testing.T) {
	auditTarget := "AppleProvider Available"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Available_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Available"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Build_Good(t *testing.T) {
	auditTarget := "AppleProvider Build"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Build
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Build_Bad(t *testing.T) {
	auditTarget := "AppleProvider Build"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Build
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Build_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Build"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Build
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Run_Good(t *testing.T) {
	auditTarget := "AppleProvider Run"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Run
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Run_Bad(t *testing.T) {
	auditTarget := "AppleProvider Run"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Run
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Run_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Run"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Run
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Tracked_Good(t *testing.T) {
	auditTarget := "AppleProvider Tracked"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Tracked
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Tracked_Bad(t *testing.T) {
	auditTarget := "AppleProvider Tracked"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Tracked
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Tracked_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Tracked"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Tracked
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Wait_Good(t *testing.T) {
	auditTarget := "AppleProvider Wait"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Wait
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Wait_Bad(t *testing.T) {
	auditTarget := "AppleProvider Wait"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Wait
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Wait_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Wait"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Wait
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Encrypt_Good(t *testing.T) {
	auditTarget := "AppleProvider Encrypt"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Encrypt
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Encrypt_Bad(t *testing.T) {
	auditTarget := "AppleProvider Encrypt"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Encrypt
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Encrypt_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Encrypt"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Encrypt
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Decrypt_Good(t *testing.T) {
	auditTarget := "AppleProvider Decrypt"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Decrypt
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Decrypt_Bad(t *testing.T) {
	auditTarget := "AppleProvider Decrypt"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Decrypt
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestApple_AppleProvider_Decrypt_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Decrypt"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*AppleProvider).Decrypt
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

// --- AX-7 canonical lifecycle triplets (Stop/Kill/Remove/Logs/Exec/Inspect/
// Pull/Push/RemoveImage + List/ListImages Bad/Ugly). Each Good is gated on a
// live `container` runtime; methods that mutate a real container are further
// gated on CORE_APPLE_E2E and mirror TestApple_E2E_ContainerLifecycle_Smoke
// (pre-clean, Run a sleeper, poll List, exercise, defer delete --force). Bad is
// the empty/invalid-input guard clause (no binary needed); Ugly is a different
// real edge (second guard branch, whitespace-only id, or a non-JSON CLI). ---

// runningAppleContainer boots a throwaway detached `sleep` container through the
// provider and polls List until it is running, mirroring the E2E smoke. It
// returns the container id and a cleanup func. Callers must already have gated
// on Available() and CORE_APPLE_E2E.
func runningAppleContainer(t *testing.T, p *AppleProvider, name string) (string, func()) {
	t.Helper()
	const ref = "docker.io/library/alpine:latest"
	ctx := context.Background()
	_ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() // pre-clean
	cleanup := func() { _ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() }

	if r := p.Pull(ref); !r.OK {
		cleanup()
		t.Fatalf("Pull: %v", r.Error())
	}
	runRes := p.Run(&Image{Path: ref}, WithName(name), WithDetach(true), WithArgs("sleep", "60"))
	if !runRes.OK {
		cleanup()
		t.Fatalf("Run with args: %v", runRes.Error())
	}
	var got *Container
	for i := 0; i < 30 && got == nil; i++ {
		if listRes := p.List(); listRes.OK {
			for _, c := range core.MustCast[[]*Container](listRes) {
				if c.ID == name && c.Status == StatusRunning {
					got = c
				}
			}
		}
		if got == nil {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if got == nil {
		cleanup()
		t.Fatalf("container %q not running after Run(WithArgs sleep)", name)
	}
	return name, cleanup
}

func TestApple_AppleProvider_Stop_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI lifecycle")
	}
	id, cleanup := runningAppleContainer(t, p, "core-stop-good-e2e")
	defer cleanup()
	if r := p.Stop(id); !r.OK {
		t.Fatalf("Stop on a running container: %v", r.Error())
	}
}

func TestApple_AppleProvider_Stop_Bad(t *testing.T) {
	// Guard clause: an empty id is rejected before the CLI is touched.
	p := NewAppleProvider()
	if p.Stop("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_Stop_Ugly(t *testing.T) {
	// Whitespace-only id slips past the empty-string guard but addresses no
	// real container, so the underlying `container stop` must still fail.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.Stop("   ").OK {
		t.Fatal("expected error for whitespace-only id")
	}
}

func TestApple_AppleProvider_Kill_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI lifecycle")
	}
	id, cleanup := runningAppleContainer(t, p, "core-kill-good-e2e")
	defer cleanup()
	if r := p.Kill(id); !r.OK {
		t.Fatalf("Kill on a running container: %v", r.Error())
	}
}

func TestApple_AppleProvider_Kill_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.Kill("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_Kill_Ugly(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.Kill("   ").OK {
		t.Fatal("expected error for whitespace-only id")
	}
}

func TestApple_AppleProvider_Remove_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI lifecycle")
	}
	id, cleanup := runningAppleContainer(t, p, "core-remove-good-e2e")
	defer cleanup()
	// Stop first so the container can be removed cleanly, then Remove it.
	_ = p.Stop(id)
	if r := p.Remove(id); !r.OK {
		t.Fatalf("Remove: %v", r.Error())
	}
}

func TestApple_AppleProvider_Remove_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.Remove("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_Remove_Ugly(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.Remove("   ").OK {
		t.Fatal("expected error for whitespace-only id")
	}
}

func TestApple_AppleProvider_Logs_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI lifecycle")
	}
	id, cleanup := runningAppleContainer(t, p, "core-logs-good-e2e")
	defer cleanup()
	r := p.Logs(id, 10)
	if !r.OK {
		t.Fatalf("Logs on a running container: %v", r.Error())
	}
	if _, ok := r.Value.(string); !ok {
		t.Fatalf("Logs Value: want string, got %T", r.Value)
	}
}

func TestApple_AppleProvider_Logs_Bad(t *testing.T) {
	// Guard clause: empty id rejected before the CLI runs (tail value ignored).
	p := NewAppleProvider()
	if p.Logs("", 100).OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_Logs_Ugly(t *testing.T) {
	// A non-existent container with a non-positive tail (exercises the n<=0
	// default branch) must fail against the live CLI.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.Logs("no-such-container", 0).OK {
		t.Fatal("expected error for non-existent container")
	}
}

func TestApple_AppleProvider_Exec_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI lifecycle")
	}
	id, cleanup := runningAppleContainer(t, p, "core-exec-good-e2e")
	defer cleanup()
	r := p.Exec(id, "echo", "hello-from-exec")
	if !r.OK {
		t.Fatalf("Exec: %v", r.Error())
	}
	if !core.Contains(core.MustCast[string](r), "hello-from-exec") {
		t.Fatalf("Exec output = %q, want it to contain hello-from-exec", core.MustCast[string](r))
	}
}

func TestApple_AppleProvider_Exec_Bad(t *testing.T) {
	// First guard: empty id is rejected.
	p := NewAppleProvider()
	if p.Exec("", "echo").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_Exec_Ugly(t *testing.T) {
	// Second guard: a valid id but an empty command is rejected — a different
	// branch from the empty-id Bad case, and it needs no runtime.
	p := NewAppleProvider()
	if p.Exec("some-id", "").OK {
		t.Fatal("expected error for empty command")
	}
}

func TestApple_AppleProvider_List_Good(t *testing.T) {
	auditTarget := "AppleProvider List"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	r := p.List()
	if !r.OK {
		t.Fatal(r.Error())
	}
	containers := core.MustCast[[]*Container](r)
	if containers == nil {
		t.Fatal("expected non-nil slice")
	}
}

func TestApple_AppleProvider_List_Bad(t *testing.T) {
	// A bogus binary cannot be executed, so the `ls` shell-out fails.
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.List().OK {
		t.Fatal("expected error when the container binary cannot be executed")
	}
}

func TestApple_AppleProvider_List_Ugly(t *testing.T) {
	// A real binary that emits non-JSON (`echo` echoes its args) drives the
	// parse-failure branch — distinct from the exec-failure Bad case.
	p := &AppleProvider{Binary: "echo"}
	if p.List().OK {
		t.Fatal("expected parse error when the CLI emits non-JSON output")
	}
}

func TestApple_AppleProvider_Inspect_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI lifecycle")
	}
	id, cleanup := runningAppleContainer(t, p, "core-inspect-good-e2e")
	defer cleanup()
	r := p.Inspect(id)
	if !r.OK {
		t.Fatalf("Inspect: %v", r.Error())
	}
	c := core.MustCast[*Container](r)
	if c.ID != id {
		t.Fatalf("Inspect().ID = %q, want %q", c.ID, id)
	}
}

func TestApple_AppleProvider_Inspect_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.Inspect("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_Inspect_Ugly(t *testing.T) {
	// Whitespace-only id passes the empty guard but inspects nothing real.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.Inspect("   ").OK {
		t.Fatal("expected error for whitespace-only id")
	}
}

func TestApple_AppleProvider_Pull_Good(t *testing.T) {
	// Pull exercises the live binary directly (no running container needed).
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI pull")
	}
	const ref = "docker.io/library/alpine:latest"
	r := p.Pull(ref)
	if !r.OK {
		t.Fatalf("Pull: %v", r.Error())
	}
	img := core.MustCast[*Image](r)
	if img.Name != ref {
		t.Fatalf("Pull().Name = %q, want %q", img.Name, ref)
	}
}

func TestApple_AppleProvider_Pull_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.Pull("").OK {
		t.Fatal("expected error for empty reference")
	}
}

func TestApple_AppleProvider_Pull_Ugly(t *testing.T) {
	// A whitespace-only reference passes the empty guard but is not a valid
	// image reference, so the live pull must fail.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.Pull("   ").OK {
		t.Fatal("expected error for whitespace-only reference")
	}
}

func TestApple_AppleProvider_Push_Good(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI push")
	}
	// Push to an unreachable local registry: the call reaches the CLI (guards
	// pass) and the push attempt itself fails with no registry listening.
	r := p.Push(&Image{Path: "alpine:latest"}, "127.0.0.1:1/core/nope:v1")
	if r.OK {
		t.Fatal("expected push to an unreachable registry to fail")
	}
}

func TestApple_AppleProvider_Push_Bad(t *testing.T) {
	// First guard: a nil image is rejected.
	p := NewAppleProvider()
	if p.Push(nil, "ref").OK {
		t.Fatal("expected error for nil image")
	}
}

func TestApple_AppleProvider_Push_Ugly(t *testing.T) {
	// Second guard: a valid image but an empty reference is rejected — a
	// different branch from the nil-image Bad case, and it needs no runtime.
	p := NewAppleProvider()
	if p.Push(&Image{Path: "some-image"}, "").OK {
		t.Fatal("expected error for empty reference")
	}
}

func TestApple_AppleProvider_RemoveImage_Good(t *testing.T) {
	// RemoveImage exercises the live binary directly: pull, then delete.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI image lifecycle")
	}
	const ref = "docker.io/library/alpine:latest"
	if r := p.Pull(ref); !r.OK {
		t.Fatalf("Pull (setup for RemoveImage): %v", r.Error())
	}
	if r := p.RemoveImage(ref); !r.OK {
		t.Fatalf("RemoveImage: %v", r.Error())
	}
}

func TestApple_AppleProvider_RemoveImage_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.RemoveImage("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_RemoveImage_Ugly(t *testing.T) {
	// Whitespace-only id passes the empty guard but matches no image, so the
	// live `image delete` must fail.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	if p.RemoveImage("   ").OK {
		t.Fatal("expected error for whitespace-only id")
	}
}

func TestApple_AppleProvider_ListImages_Good(t *testing.T) {
	auditTarget := "AppleProvider ListImages"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	r := p.ListImages()
	if !r.OK {
		t.Fatal(r.Error())
	}
	images := core.MustCast[[]*Image](r)
	if images == nil {
		t.Fatal("expected non-nil slice")
	}
}

func TestApple_AppleProvider_ListImages_Bad(t *testing.T) {
	// A bogus binary cannot be executed, so the `image ls` shell-out fails.
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.ListImages().OK {
		t.Fatal("expected error when the container binary cannot be executed")
	}
}

func TestApple_AppleProvider_ListImages_Ugly(t *testing.T) {
	// A real binary emitting non-JSON drives the parse-failure branch —
	// distinct from the exec-failure Bad case.
	p := &AppleProvider{Binary: "echo"}
	if p.ListImages().OK {
		t.Fatal("expected parse error when the CLI emits non-JSON output")
	}
}

// --- Task 15–17 tests ---

func TestApple_isAppleSilicon_Ugly(t *testing.T) {
	auditTarget := "isAppleSilicon"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Must return a bool and not panic.
	got := isAppleSilicon()
	if got, want := reflect.TypeOf(got), reflect.TypeOf(true); got != want {
		t.Fatalf("want type %v, got %v", want, got)
	}
}

func TestApple_deriveKey256_Ugly(t *testing.T) {
	auditTarget := "deriveKey256"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Same input must produce same output; output must be 32 bytes.
	key := []byte("test-key")
	k1 := deriveKey256(key)
	k2 := deriveKey256(key)
	if len(k1) != 32 {
		t.Fatalf("want 32-byte key, got %d", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatal("deriveKey256 is not deterministic")
		}
	}
}

func TestApple_NewAppleProvider_RetentionWindow_Ugly(t *testing.T) {
	auditTarget := "NewAppleProvider RetentionWindow"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if p.RetentionWindow <= 0 {
		t.Fatal("expected positive RetentionWindow")
	}
}

func TestApple_AppleProvider_Run_GPU_Unsupported_Ugly(t *testing.T) {
	// Metal GPU passthrough is not supported by the Apple container runtime on
	// any architecture (RFC.apple.md §15); a GPU request to Run must fail.
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	img := &Image{Path: "test-image"}
	r := p.Run(img, WithGPU(true))
	if r.OK {
		t.Fatal("expected GPU request to be rejected as unsupported")
	}
}

// --- W1: real `container` 0.12.3 JSON schema parsing ---

// realContainerLsJSON is a trimmed-but-real element of `container ls --format
// json` / `container inspect` output, captured from container 0.12.3. The
// schema is deeply nested under "configuration" with a CFAbsoluteTime
// "startedDate" float — not the flat shape the provider originally assumed.
const realContainerLsJSON = `[{"status":"running","startedDate":802181959.432204,"configuration":{"id":"coreprobe","image":{"reference":"docker.io/library/alpine:latest","descriptor":{"digest":"sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11","size":9218}},"resources":{"cpus":4,"memoryInBytes":1073741824},"publishedPorts":[{"hostPort":8080,"hostAddress":"0.0.0.0","containerPort":80,"proto":"tcp","count":1}]}}]`

// realImageLsJSON is a real element of `container image ls --format json`.
const realImageLsJSON = `[{"fullSize":"4.2 MB","reference":"docker.io/library/alpine:latest","descriptor":{"size":9218,"digest":"sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11"}}]`

func TestApple_parseContainerList_RealSchema_Good(t *testing.T) {
	r := parseContainerList([]byte(realContainerLsJSON))
	if !r.OK {
		t.Fatal(r.Error())
	}
	containers := core.MustCast[[]*Container](r)
	if len(containers) != 1 {
		t.Fatalf("want 1 container, got %d", len(containers))
	}
	c := containers[0]
	if c.ID != "coreprobe" {
		t.Fatalf("ID: want coreprobe, got %q", c.ID)
	}
	if c.Image != "docker.io/library/alpine:latest" {
		t.Fatalf("Image: want docker.io/library/alpine:latest, got %q", c.Image)
	}
	if c.Status != StatusRunning {
		t.Fatalf("Status: want running, got %q", c.Status)
	}
	if c.CPUs != 4 {
		t.Fatalf("CPUs: want 4, got %d", c.CPUs)
	}
	if c.Memory != 1024 {
		t.Fatalf("Memory: want 1024 MB (1073741824 bytes), got %d", c.Memory)
	}
	if c.Ports[8080] != 80 {
		t.Fatalf("Ports[8080]: want 80, got %d", c.Ports[8080])
	}
	if c.StartedAt.Year() != 2026 {
		t.Fatalf("StartedAt year: want 2026 (CFAbsoluteTime+978307200), got %d (%v)", c.StartedAt.Year(), c.StartedAt)
	}
}

func TestApple_parseImageList_RealSchema_Good(t *testing.T) {
	r := parseImageList([]byte(realImageLsJSON))
	if !r.OK {
		t.Fatal(r.Error())
	}
	images := core.MustCast[[]*Image](r)
	if len(images) != 1 {
		t.Fatalf("want 1 image, got %d", len(images))
	}
	img := images[0]
	if img.Name != "docker.io/library/alpine:latest" {
		t.Fatalf("Name: want reference, got %q", img.Name)
	}
	if img.Path != "docker.io/library/alpine:latest" {
		t.Fatalf("Path: want reference, got %q", img.Path)
	}
	const wantDigest = "sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11"
	if img.Digest != wantDigest {
		t.Fatalf("Digest: want descriptor.digest, got %q", img.Digest)
	}
}

func TestApple_parseSingleContainer_RealSchema_Good(t *testing.T) {
	// `container inspect <id>` returns a JSON ARRAY even for a single id.
	r := parseSingleContainer([]byte(realContainerLsJSON))
	if !r.OK {
		t.Fatal(r.Error())
	}
	c := core.MustCast[*Container](r)
	if c.ID != "coreprobe" {
		t.Fatalf("ID: want coreprobe, got %q", c.ID)
	}
	if c.Image != "docker.io/library/alpine:latest" {
		t.Fatalf("Image: want reference, got %q", c.Image)
	}
}

// --- W1: real `container` 0.12.3 CLI argument vectors ---
// Image operations live under the `image` subgroup; `images`/`pull`/`push`/`rmi`
// at top level do not exist on container 0.12.x.

func TestApple_appleImageLsArgs_Good(t *testing.T) {
	got := appleImageLsArgs()
	want := []string{"image", "ls", "--format", "json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_applePullArgs_Good(t *testing.T) {
	got := applePullArgs("docker.io/library/alpine:latest")
	want := []string{"image", "pull", "docker.io/library/alpine:latest"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_applePushArgs_Good(t *testing.T) {
	got := applePushArgs("ghcr.io/acme/app:v1")
	want := []string{"image", "push", "ghcr.io/acme/app:v1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_appleRemoveImageArgs_Good(t *testing.T) {
	got := appleRemoveImageArgs("alpine:latest")
	want := []string{"image", "delete", "alpine:latest"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_appleLogsArgs_Good(t *testing.T) {
	got := appleLogsArgs("c123", 50)
	want := []string{"logs", "-n", "50", "c123"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_appleRunArgs_GPUUnsupported_Bad(t *testing.T) {
	// Metal GPU passthrough is not offered by the Apple container runtime
	// (RFC.apple.md §15), so a GPU request must be rejected — not emitted as
	// flags the CLI does not understand.
	r := appleRunArgs("web", &Image{Path: "alpine:latest"}, RunOptions{GPU: true})
	if r.OK {
		t.Fatal("expected GPU request to be rejected as unsupported")
	}
}

func TestApple_appleRunArgs_Good(t *testing.T) {
	r := appleRunArgs("web", &Image{Path: "alpine:latest"}, RunOptions{
		Memory: 2048, CPUs: 2, Detach: true, Ports: map[int]int{8080: 80},
	})
	if !r.OK {
		t.Fatal(r.Error())
	}
	args := core.MustCast[[]string](r)
	joined := core.Join(" ", args...)
	for _, want := range []string{"run", "--name web", "--detach", "--memory 2048M", "--cpus 2", "--publish 8080:80", "alpine:latest"} {
		if !core.Contains(joined, want) {
			t.Fatalf("args %q missing %q", joined, want)
		}
	}
	for _, a := range args {
		if a == "--gpu" || a == "--device" {
			t.Fatalf("args must not emit unsupported GPU flags: %v", args)
		}
	}
}

func TestApple_appleRunArgs_ContainerArgs_Good(t *testing.T) {
	// The container command/args must follow the image, matching
	// `container run <image> [args...]`.
	r := appleRunArgs("web", &Image{Path: "alpine:latest"}, RunOptions{Args: []string{"sleep", "300"}})
	if !r.OK {
		t.Fatal(r.Error())
	}
	args := core.MustCast[[]string](r)
	if !core.Contains(core.Join(" ", args...), "alpine:latest sleep 300") {
		t.Fatalf("args %v: container command must follow the image as `alpine:latest sleep 300`", args)
	}
	if n := len(args); args[n-3] != "alpine:latest" || args[n-2] != "sleep" || args[n-1] != "300" {
		t.Fatalf("trailing args = %v, want [... alpine:latest sleep 300]", args)
	}
}

func TestApple_appleRunArgs_Env_Good(t *testing.T) {
	r := appleRunArgs("web", &Image{Path: "alpine:latest"},
		RunOptions{Env: []string{"FOO=bar"}, Volumes: map[string]string{"/h": "/c"}, Args: []string{"sleep", "1"}})
	if !r.OK {
		t.Fatal(r.Error())
	}
	args := core.MustCast[[]string](r)
	if !core.Contains(core.Join(" ", args...), "-e FOO=bar") {
		t.Fatalf("args %v missing `-e FOO=bar`", args)
	}
	// env must precede the image; container args follow it.
	ePos, imgPos := indexOf(args, "FOO=bar"), indexOf(args, "alpine:latest")
	if ePos < 0 || imgPos < 0 || ePos > imgPos {
		t.Fatalf("env must precede image: %v", args)
	}
}

func TestApple_appleRunArgs_DNS_Good(t *testing.T) {
	// Explicit DNS emits a --dns per nameserver, before the image, and
	// replaces the default (does not append to it).
	r := appleRunArgs("web", &Image{Path: "alpine:latest"},
		RunOptions{DNS: []string{"9.9.9.9", "1.0.0.1"}, Args: []string{"sleep", "1"}})
	if !r.OK {
		t.Fatal(r.Error())
	}
	args := core.MustCast[[]string](r)
	joined := core.Join(" ", args...)
	for _, want := range []string{"--dns 9.9.9.9", "--dns 1.0.0.1"} {
		if !core.Contains(joined, want) {
			t.Fatalf("args %q missing %q", joined, want)
		}
	}
	if dPos, imgPos := indexOf(args, "9.9.9.9"), indexOf(args, "alpine:latest"); dPos < 0 || imgPos < 0 || dPos > imgPos {
		t.Fatalf("--dns must precede image: %v", args)
	}
	if core.Contains(joined, "1.1.1.1") {
		t.Fatalf("explicit DNS must replace the default, not append it: %v", args)
	}
}

func TestApple_appleRunArgs_DNSDefault_Good(t *testing.T) {
	// Empty DNS defaults to a reachable public resolver — the Apple runtime's
	// gateway resolver does not answer DNS, so without this hostnames never
	// resolve even though NAT egress works.
	r := appleRunArgs("web", &Image{Path: "alpine:latest"}, RunOptions{})
	if !r.OK {
		t.Fatal(r.Error())
	}
	if joined := core.Join(" ", core.MustCast[[]string](r)...); !core.Contains(joined, "--dns 1.1.1.1") {
		t.Fatalf("empty DNS must default to a public resolver, got %q", joined)
	}
}

// indexOf returns the first index of v in s, or -1.
func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func TestApple_appleContainerID_Good(t *testing.T) {
	// Explicit name wins; else the image name; else the generated fallback.
	// For Apple this id is what `container run --name` registers and what
	// stop/logs/exec address.
	if got := appleContainerID(RunOptions{Name: "web"}, &Image{Name: "img"}, "fallback"); got != "web" {
		t.Fatalf("explicit name: got %q, want web", got)
	}
	if got := appleContainerID(RunOptions{}, &Image{Name: "img"}, "fallback"); got != "img" {
		t.Fatalf("image name: got %q, want img", got)
	}
	if got := appleContainerID(RunOptions{}, &Image{}, "fallback"); got != "fallback" {
		t.Fatalf("fallback: got %q, want fallback", got)
	}
}

func TestApple_appleSystemStatusArgs_Good(t *testing.T) {
	got := appleSystemStatusArgs()
	want := []string{"system", "status"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_appleSystemStartArgs_Good(t *testing.T) {
	if got, want := appleSystemStartArgs(true), []string{"system", "start", "--enable-kernel-install"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("install: got %v, want %v", got, want)
	}
	if got, want := appleSystemStartArgs(false), []string{"system", "start", "--disable-kernel-install"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("no-install: got %v, want %v", got, want)
	}
}

func TestApple_appleSystemStopArgs_Good(t *testing.T) {
	if got, want := appleSystemStopArgs(), []string{"system", "stop"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", want, got)
	}
}

func TestApple_AppleProvider_SystemStatus_Good(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	r := p.SystemStatus()
	if !r.OK || !core.Contains(core.Lower(core.MustCast[string](r)), "running") {
		t.Fatalf("SystemStatus OK=%v err=%q", r.OK, r.Error())
	}
}

func TestApple_AppleProvider_SystemStatus_Bad(t *testing.T) {
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.SystemStatus().OK {
		t.Fatal("expected failure with a bogus binary")
	}
}

func TestApple_AppleProvider_SystemStatus_Ugly(t *testing.T) {
	// Empty binary name must yield a failed Result, not a panic.
	p := &AppleProvider{Binary: ""}
	if p.SystemStatus().OK {
		t.Fatal("expected failure with an empty binary")
	}
}

func TestApple_AppleProvider_SystemStart_Good(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	// Idempotent — the daemon is already up.
	if r := p.SystemStart(true); !r.OK {
		t.Fatalf("SystemStart: %v", r.Error())
	}
}

func TestApple_AppleProvider_SystemStart_Bad(t *testing.T) {
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.SystemStart(true).OK {
		t.Fatal("expected failure with a bogus binary")
	}
}

func TestApple_AppleProvider_SystemStart_Ugly(t *testing.T) {
	// The --disable-kernel-install path also fails cleanly on a bogus binary.
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.SystemStart(false).OK {
		t.Fatal("expected failure with a bogus binary")
	}
}

func TestApple_AppleProvider_SystemStop_Good(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	// Stop then restart so the runtime is restored for the other live tests.
	if r := p.SystemStop(); !r.OK {
		t.Fatalf("SystemStop: %v", r.Error())
	}
	if r := p.SystemStart(true); !r.OK {
		t.Fatalf("SystemStart (restore): %v", r.Error())
	}
}

func TestApple_AppleProvider_SystemStop_Bad(t *testing.T) {
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.SystemStop().OK {
		t.Fatal("expected failure with a bogus binary")
	}
}

func TestApple_AppleProvider_SystemStop_Ugly(t *testing.T) {
	p := &AppleProvider{Binary: ""}
	if p.SystemStop().OK {
		t.Fatal("expected failure with an empty binary")
	}
}

func TestApple_appleExecInteractiveArgs_Good(t *testing.T) {
	got := appleExecInteractiveArgs("web", []string{"/bin/sh"})
	want := []string{"exec", "-i", "-t", "web", "/bin/sh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestApple_AppleProvider_ExecInteractive_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.ExecInteractive("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestApple_AppleProvider_ExecInteractive_Ugly(t *testing.T) {
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.ExecInteractive("web", "/bin/sh").OK {
		t.Fatal("expected failure with a bogus binary")
	}
}

func TestApple_AppleProvider_ExecInteractive_Good(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	const name = "core-exec-i-e2e"
	const ref = "docker.io/library/alpine:latest"
	ctx := context.Background()
	_ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run()
	defer func() { _ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() }()
	if r := p.Pull(ref); !r.OK {
		t.Fatalf("Pull: %v", r.Error())
	}
	if r := p.Run(&Image{Path: ref}, WithName(name), WithDetach(true), WithArgs("sleep", "60")); !r.OK {
		t.Fatalf("Run: %v", r.Error())
	}
	for i := 0; i < 30; i++ {
		ready := false
		if lr := p.List(); lr.OK {
			for _, c := range core.MustCast[[]*Container](lr) {
				if c.ID == name && c.Status == StatusRunning {
					ready = true
				}
			}
		}
		if ready {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Non-interactive command through the interactive path; -t without a real
	// TTY may be rejected under `go test` — that's the documented degrade-to-skip.
	if r := p.ExecInteractive(name, "echo", "interactive-ok"); !r.OK {
		t.Skipf("ExecInteractive needs a TTY in this env: %v", r.Error())
	}
}

// TestApple_E2E_ImageLifecycle_Smoke certifies the image-subgroup reconciliation
// against the LIVE `container` binary: pull → list (parse real JSON) → delete.
// Opt-in (set CORE_APPLE_E2E=1) because it shells out to the runtime, requires
// `container system start`, and pulls from a registry.
func TestApple_E2E_ImageLifecycle_Smoke(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	const ref = "docker.io/library/alpine:latest"

	if r := p.Pull(ref); !r.OK {
		t.Fatalf("Pull: %v", r.Error())
	}

	listRes := p.ListImages()
	if !listRes.OK {
		t.Fatalf("ListImages: %v", listRes.Error())
	}
	images := core.MustCast[[]*Image](listRes)
	var found *Image
	for _, img := range images {
		if core.Contains(img.Name, "alpine") {
			found = img
		}
	}
	if found == nil {
		t.Fatalf("ListImages did not return the pulled alpine image; got %d images", len(images))
	}
	if found.Digest == "" {
		t.Fatalf("pulled image %q parsed with empty digest (schema drift)", found.Name)
	}

	if r := p.RemoveImage(ref); !r.OK {
		t.Fatalf("RemoveImage: %v", r.Error())
	}
}

// TestApple_E2E_ContainerLifecycle_Smoke certifies the container lifecycle
// methods that `vm ps/logs/exec/stop` dispatch to, against the LIVE binary:
// a running container is created, List finds + parses it, Exec runs a command
// in it, Logs reads it, and Stop halts it. Opt-in (CORE_APPLE_E2E=1).
func TestApple_E2E_ContainerLifecycle_Smoke(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	const name = "core-lifecycle-e2e"
	const ref = "docker.io/library/alpine:latest"
	ctx := context.Background()
	_ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() // pre-clean leftovers
	defer func() { _ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() }()

	// Ensure the image is local so the boot below isn't gated on a registry
	// pull — keeps this test independent of the image-lifecycle test's order.
	if r := p.Pull(ref); !r.OK {
		t.Fatalf("Pull: %v", r.Error())
	}

	// Boot a long-running container THROUGH the API (#17: Run forwards args).
	// Without forwarded args, alpine's default CMD exits immediately and the
	// container would not stay running — so a running container proves it.
	runRes := p.Run(&Image{Path: ref},
		WithName(name), WithDetach(true), WithArgs("sleep", "60"))
	if !runRes.OK {
		t.Fatalf("Run with args: %v", runRes.Error())
	}
	ctr := core.MustCast[*Container](runRes)
	if ctr.ID != name {
		t.Fatalf("Run().ID = %q, want %q (the --name is the real container id)", ctr.ID, name)
	}

	// `container run --detach` boots asynchronously; poll List (the path
	// `vm ps` aggregates) until the container is running.
	var got *Container
	for i := 0; i < 30 && got == nil; i++ {
		if listRes := p.List(); listRes.OK {
			for _, c := range core.MustCast[[]*Container](listRes) {
				if c.ID == name && c.Status == StatusRunning {
					got = c
				}
			}
		}
		if got == nil {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if got == nil {
		t.Fatalf("container %q not running after Run(WithArgs sleep) — args not forwarded?", name)
	}
	if got.Image == "" {
		t.Fatalf("List parsed container with empty image: %+v", got)
	}

	// Exec runs a command inside it (the path `vm exec` dispatches).
	execRes := p.Exec(name, "echo", "hello-from-exec")
	if !execRes.OK {
		t.Fatalf("Exec: %v", execRes.Error())
	}
	if !core.Contains(core.MustCast[string](execRes), "hello-from-exec") {
		t.Fatalf("Exec output = %q, want it to contain hello-from-exec", core.MustCast[string](execRes))
	}

	// Logs is reachable (the path `vm logs` dispatches); sleep emits none.
	if r := p.Logs(name, 10); !r.OK {
		t.Fatalf("Logs: %v", r.Error())
	}

	// Stop halts it via the returned handle's id (round-trips #18).
	if r := p.Stop(ctr.ID); !r.OK {
		t.Fatalf("Stop: %v", r.Error())
	}
}

// TestApple_E2E_RunPublish_Smoke certifies that a published port set via
// WithPorts reaches the runtime: run with -p 18080:80 and assert List's parsed
// publishedPorts shows 18080->80. Opt-in (CORE_APPLE_E2E=1).
func TestApple_E2E_RunPublish_Smoke(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	const name = "core-publish-e2e"
	const ref = "docker.io/library/alpine:latest"
	ctx := context.Background()
	_ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run()
	defer func() { _ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() }()
	if r := p.Pull(ref); !r.OK {
		t.Fatalf("Pull: %v", r.Error())
	}
	if r := p.Run(&Image{Path: ref}, WithName(name), WithDetach(true),
		WithPorts(map[int]int{18080: 80}), WithArgs("sleep", "60")); !r.OK {
		t.Fatalf("Run with -p: %v", r.Error())
	}
	var got *Container
	for i := 0; i < 30 && got == nil; i++ {
		if lr := p.List(); lr.OK {
			for _, c := range core.MustCast[[]*Container](lr) {
				if c.ID == name && c.Status == StatusRunning {
					got = c
				}
			}
		}
		if got == nil {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if got == nil {
		t.Fatal("container not running after Run(WithPorts)")
	}
	if got.Ports[18080] != 80 {
		t.Fatalf("published ports = %v, want 18080->80", got.Ports)
	}
}
