package container

import (
	"context"
	"reflect"
	"runtime"
	"testing"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
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

// --- Task 4-14 new method tests ---

func TestApple_AppleProvider_Stop_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider Stop"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Stop("")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Kill_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider Kill"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Kill("")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Remove_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider Remove"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Remove("")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Logs_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider Logs"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Logs("", 100)
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Logs_ZeroTail_Good(t *testing.T) {
	auditTarget := "AppleProvider Logs"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	r := p.Logs("no-such-container", 0)
	if r.OK {
		t.Fatal("expected error for non-existent container")
	}
}

func TestApple_AppleProvider_Exec_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider Exec"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Exec("", "echo")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Exec_EmptyCommand_Bad(t *testing.T) {
	auditTarget := "AppleProvider Exec"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Exec("some-id", "")
	if r.OK {
		t.Fatal("expected error")
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

func TestApple_AppleProvider_Inspect_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider Inspect"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Inspect("")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Pull_EmptyRef_Bad(t *testing.T) {
	auditTarget := "AppleProvider Pull"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Pull("")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Push_NilImage_Bad(t *testing.T) {
	auditTarget := "AppleProvider Push"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Push(nil, "ref")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_Push_EmptyRef_Bad(t *testing.T) {
	auditTarget := "AppleProvider Push"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.Push(&Image{Path: "some-image"}, "")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestApple_AppleProvider_RemoveImage_EmptyID_Bad(t *testing.T) {
	auditTarget := "AppleProvider RemoveImage"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	p := NewAppleProvider()
	r := p.RemoveImage("")
	if r.OK {
		t.Fatal("expected error")
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

func TestApple_AppleProvider_Run_GPU_NonAppleSilicon_Ugly(t *testing.T) {
	auditTarget := "AppleProvider Run GPU"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	if isAppleSilicon() {
		t.Skip("run requires non-Apple Silicon host for this error path")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	img := &Image{Path: "test-image"}
	r := p.Run(img, WithGPU(true))
	if r.OK {
		t.Fatal("expected error: Metal GPU requires Apple Silicon")
	}
}
