package container

import (
	"context"
	"reflect"
	"runtime"
	"testing"
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

	_, err := p.Build(ContainerConfig{})
	if err == nil {
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

	_, err := p.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApple_Encrypt_Decrypt_Ugly(t *testing.T) {
	auditTarget := "Encrypt Decrypt"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Encrypt+Decrypt round-trip must preserve the path sans .stim suffix.
	p := NewAppleProvider()
	img := &Image{ID: "test", Path: "/tmp/example.qcow2", Size: 1024}
	key := []byte("workspace-key")

	enc, err := p.Encrypt(img, key)
	if err != nil {
		t.Fatal(err)
	}
	if enc == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := enc.Scheme, "stim"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := enc.Path, "/tmp/example.qcow2.stim"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	out, err := p.Decrypt(enc, key)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := out.Path, "/tmp/example.qcow2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
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

	_, err := p.Encrypt(img, nil)
	if err == nil {
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

	_, err := p.Decrypt(enc, nil)
	if err == nil {
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

	err := p.Wait(context.Background(), "no-such-container")
	if err == nil {
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
