package container

import (
	"bytes"
	"testing"

	"dappco.re/go/core/io"

	"dappco.re/go/core/container/internal/coreutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTIM_NewTIMBundle_Good(t *testing.T) {
	bundle := NewTIMBundle("worker-01", "/var/tim/worker-01")

	assert.Equal(t, "worker-01", bundle.ID)
	assert.Equal(t, "/var/tim/worker-01", bundle.Root)
	assert.Equal(t, []string{TIMLayerBase, TIMLayerApp, TIMLayerData}, bundle.Layers)
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
	require.NoError(t, err)

	loaded, err := LoadTIM(io.Local, root)
	require.NoError(t, err)
	assert.Equal(t, []string{"/app/server"}, loaded.Config.EntryPoint)
	assert.True(t, loaded.Config.ReadOnly)
}

func TestTIM_SaveTIM_MissingBundle_Bad(t *testing.T) {
	err := SaveTIM(io.Local, nil)

	assert.Error(t, err)
}

func TestTIM_LoadTIM_MissingConfig_Bad(t *testing.T) {
	tmp := t.TempDir()

	_, err := LoadTIM(io.Local, tmp)

	assert.Error(t, err)
}

func TestTIM_EncryptTIM_DecryptSTIM_Good(t *testing.T) {
	// Round-trip the bundle through STIM encryption.
	bundle := NewTIMBundle("worker-01", "/var/tim/worker-01")
	bundle.Config.EntryPoint = []string{"/app"}
	key := []byte("workspace-key-32-bytes-xxxxxxxxxx")

	stim, err := EncryptTIM(bundle, key)
	require.NoError(t, err)
	assert.Equal(t, "stim", stim.Scheme)
	assert.Len(t, stim.Layers, len(bundle.Layers))

	out, err := DecryptSTIM(stim, key)
	require.NoError(t, err)
	assert.Equal(t, bundle.ID, out.ID)
	assert.Equal(t, bundle.Layers, out.Layers)
}

func TestTIM_EncryptTIM_MissingKey_Bad(t *testing.T) {
	bundle := NewTIMBundle("a", "/tmp/a")

	_, err := EncryptTIM(bundle, nil)

	assert.Error(t, err)
}

func TestTIM_EncryptLayer_DecryptLayer_Good(t *testing.T) {
	// The layer-level AES-GCM round-trip must recover the plaintext.
	key := []byte("workspace-key-32-bytes-xxxxxxxxxx")
	plain := []byte("hello TIM layer")

	ct, err := EncryptLayer(key, "worker-01", TIMLayerApp, plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, ct)

	pt, err := DecryptLayer(key, "worker-01", TIMLayerApp, ct)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(plain, pt))
}

func TestTIM_DecryptLayer_ShortCiphertext_Bad(t *testing.T) {
	key := []byte("workspace-key-32-bytes-xxxxxxxxxx")

	_, err := DecryptLayer(key, "worker-01", TIMLayerApp, []byte("x"))

	assert.Error(t, err)
}

func TestTIM_DecryptLayer_WrongKey_Ugly(t *testing.T) {
	// A ciphertext produced with workspace key A must not decrypt with key B.
	keyA := []byte("key-a-32-bytes-xxxxxxxxxxxxxxxxx")
	keyB := []byte("key-b-32-bytes-xxxxxxxxxxxxxxxxx")
	ct, err := EncryptLayer(keyA, "worker-01", TIMLayerApp, []byte("secret"))
	require.NoError(t, err)

	_, err = DecryptLayer(keyB, "worker-01", TIMLayerApp, ct)

	assert.Error(t, err)
}

func TestTIM_EncryptTIMOnMedium_DecryptSTIMOnMedium_Good(t *testing.T) {
	// Full on-disk round-trip: lay down a plaintext layer, seal it, and
	// verify DecryptSTIMOnMedium restores the payload.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	require.NoError(t, err)

	root := "bundle-01"
	appDir := coreutil.JoinPath(root, "rootfs", TIMLayerApp)
	require.NoError(t, sandbox.EnsureDir(appDir))
	require.NoError(t, sandbox.Write(coreutil.JoinPath(appDir, "server.bin"), "hello server"))

	bundle := NewTIMBundle("worker-01", root)
	key := []byte("workspace-key-32-bytes-xxxxxxxxx")

	stim, err := EncryptTIMOnMedium(sandbox, bundle, key)
	require.NoError(t, err)
	assert.Equal(t, "stim", stim.Scheme)
	sealedPath := coreutil.JoinPath(root, "rootfs", TIMLayerApp+".stim")
	assert.True(t, sandbox.IsFile(sealedPath), "sealed layer artefact must exist on disk")

	// Remove plaintext — decryption must recreate it.
	require.NoError(t, sandbox.DeleteAll(appDir))

	out, err := DecryptSTIMOnMedium(sandbox, stim, key)
	require.NoError(t, err)
	assert.Equal(t, "worker-01", out.ID)
	assert.True(t, sandbox.IsDir(appDir))
}

func TestTIM_EncryptTIMOnMedium_MissingMedium_Bad(t *testing.T) {
	bundle := NewTIMBundle("worker-01", "/tmp/x")

	_, err := EncryptTIMOnMedium(nil, bundle, []byte("k"))

	assert.Error(t, err)
}

func TestTIM_DecryptSTIMOnMedium_WrongKey_Ugly(t *testing.T) {
	// Sealing with key A and unsealing with key B must fail — the payload
	// must not leak under key mismatch.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	require.NoError(t, err)

	root := "bundle-01"
	appDir := coreutil.JoinPath(root, "rootfs", TIMLayerApp)
	require.NoError(t, sandbox.EnsureDir(appDir))
	require.NoError(t, sandbox.Write(coreutil.JoinPath(appDir, "server.bin"), "hello"))

	bundle := NewTIMBundle("worker-01", root)
	stim, err := EncryptTIMOnMedium(sandbox, bundle, []byte("key-a-32-bytes-xxxxxxxxxxxxxxxxx"))
	require.NoError(t, err)

	_, err = DecryptSTIMOnMedium(sandbox, stim, []byte("key-b-32-bytes-xxxxxxxxxxxxxxxxx"))
	assert.Error(t, err)
}
