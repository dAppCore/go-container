package container

import (
	"bytes"

	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/core/io"
	"reflect"
	"testing"
)

func TestTIM_NewTIMBundle_Good(t *testing.T) {
	bundle := NewTIMBundle("worker-01", "/var/tim/worker-01")
	if got, want := bundle.ID, "worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := bundle.Root, "/var/tim/worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := bundle.Layers, []string{TIMLayerBase, TIMLayerApp, TIMLayerData}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestTIM_SaveTIM_LoadTIM_Good(t *testing.T) {
	// SaveTIM followed by LoadTIM must round-trip the configuration.
	tmp := t.TempDir()
	root := coreutil.JoinPath(tmp, "worker-01")

	bundle := NewTIMBundle("worker-01", root)
	bundle.Config = TIMConfig{
		EntryPoint: []string{"/app/server"},
		Env:        []string{"CORE_ENV=production"},
		ReadOnly:   true,
	}

	err := SaveTIM(io.Local, bundle)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadTIM(io.Local, root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := loaded.Config.EntryPoint, []string{"/app/server"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(loaded.Config.ReadOnly) {
		t.Fatal("expected true")
	}
}

func TestTIM_SaveTIM_MissingBundle_Bad(t *testing.T) {
	err := SaveTIM(io.Local, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTIM_LoadTIM_MissingConfig_Bad(t *testing.T) {
	tmp := t.TempDir()

	_, err := LoadTIM(io.Local, tmp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTIM_EncryptTIM_DecryptSTIM_Good(t *testing.T) {
	// Round-trip the bundle through STIM encryption.
	bundle := NewTIMBundle("worker-01", "/var/tim/worker-01")
	bundle.Config.EntryPoint = []string{"/app"}
	key := []byte("workspace-key-32-bytes-xxxxxxxxxx")

	stim, err := EncryptTIM(bundle, key)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stim.Scheme, "stim"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := len(stim.Layers), len(bundle.Layers); got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}

	out, err := DecryptSTIM(stim, key)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.ID, bundle.ID; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := out.Layers, bundle.Layers; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestTIM_EncryptTIM_MissingKey_Bad(t *testing.T) {
	bundle := NewTIMBundle("a", "/tmp/a")

	_, err := EncryptTIM(bundle, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTIM_EncryptLayer_DecryptLayer_Good(t *testing.T) {
	// The layer-level AES-GCM round-trip must recover the plaintext.
	key := []byte("workspace-key-32-bytes-xxxxxxxxxx")
	plain := []byte("hello TIM layer")

	ct, err := EncryptLayer(key, "worker-01", TIMLayerApp, plain)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ct, plain; reflect.DeepEqual(got, want) {
		t.Fatalf("did not expect %v", got)
	}

	pt, err := DecryptLayer(key, "worker-01", TIMLayerApp, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !(bytes.Equal(plain, pt)) {
		t.Fatal("expected true")
	}
}

func TestTIM_DecryptLayer_ShortCiphertext_Bad(t *testing.T) {
	key := []byte("workspace-key-32-bytes-xxxxxxxxxx")

	_, err := DecryptLayer(key, "worker-01", TIMLayerApp, []byte("x"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTIM_DecryptLayer_WrongKey_Ugly(t *testing.T) {
	// A ciphertext produced with workspace key A must not decrypt with key B.
	keyA := []byte("key-a-32-bytes-xxxxxxxxxxxxxxxxx")
	keyB := []byte("key-b-32-bytes-xxxxxxxxxxxxxxxxx")
	ct, err := EncryptLayer(keyA, "worker-01", TIMLayerApp, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptLayer(keyB, "worker-01", TIMLayerApp, ct)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTIM_EncryptTIMOnMedium_DecryptSTIMOnMedium_Good(t *testing.T) {
	// Full on-disk round-trip: lay down a plaintext layer, seal it, and
	// verify DecryptSTIMOnMedium restores the payload.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	if err != nil {
		t.Fatal(err)
	}

	root := "bundle-01"
	appDir := coreutil.JoinPath(root, "rootfs", TIMLayerApp)
	if err := sandbox.EnsureDir(appDir); err != nil {
		t.Fatal(err)
	}
	if err := sandbox.Write(coreutil.JoinPath(appDir, "server.bin"), "hello server"); err != nil {
		t.Fatal(err)
	}

	bundle := NewTIMBundle("worker-01", root)
	key := []byte("workspace-key-32-bytes-xxxxxxxxx")

	stim, err := EncryptTIMOnMedium(sandbox, bundle, key)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stim.Scheme, "stim"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	sealedPath := coreutil.JoinPath(root, "rootfs", TIMLayerApp+".stim")
	if !(sandbox.IsFile(sealedPath)) {
		t.Fatal("expected true")
	}

	// Remove plaintext — decryption must recreate it.
	if err := sandbox.DeleteAll(appDir); err != nil {
		t.Fatal(err)
	}

	out, err := DecryptSTIMOnMedium(sandbox, stim, key)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.ID, "worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(sandbox.IsDir(appDir)) {
		t.Fatal("expected true")
	}
}

func TestTIM_EncryptTIMOnMedium_MissingMedium_Bad(t *testing.T) {
	bundle := NewTIMBundle("worker-01", "/tmp/x")

	_, err := EncryptTIMOnMedium(nil, bundle, []byte("k"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTIM_DecryptSTIMOnMedium_WrongKey_Ugly(t *testing.T) {
	// Sealing with key A and unsealing with key B must fail — the payload
	// must not leak under key mismatch.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	if err != nil {
		t.Fatal(err)
	}

	root := "bundle-01"
	appDir := coreutil.JoinPath(root, "rootfs", TIMLayerApp)
	if err := sandbox.EnsureDir(appDir); err != nil {
		t.Fatal(err)
	}
	if err := sandbox.Write(coreutil.JoinPath(appDir, "server.bin"), "hello"); err != nil {
		t.Fatal(err)
	}

	bundle := NewTIMBundle("worker-01", root)
	stim, err := EncryptTIMOnMedium(sandbox, bundle, []byte("key-a-32-bytes-xxxxxxxxxxxxxxxxx"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSTIMOnMedium(sandbox, stim, []byte("key-b-32-bytes-xxxxxxxxxxxxxxxxx"))
	if err == nil {
		t.Fatal("expected error")
	}
}
