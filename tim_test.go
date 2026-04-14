package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewTIMProvider ---

func TestTIM_NewTIMProvider_Good(t *testing.T) {
	p := NewTIMProvider()

	assert.NotNil(t, p)
	var _ Provider = p
}

func TestTIM_TIMProvider_Run_Bad(t *testing.T) {
	p := NewTIMProvider()

	_, err := p.Run(&Image{Path: "/tmp/x"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestTIM_TIMProvider_Build_MissingSource_Ugly(t *testing.T) {
	p := NewTIMProvider()

	_, err := p.Build(ContainerConfig{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing tim source")
}

// --- EncryptTIM / DecryptSTIM round-trip ---

func TestTIM_EncryptTIM_RoundTrip_Good(t *testing.T) {
	tmpDir := t.TempDir()
	bundle := setupTestTIMBundle(t, tmpDir)

	workspaceKey := make([]byte, 32)
	for i := range workspaceKey {
		workspaceKey[i] = byte(i)
	}

	stim, err := EncryptTIM(bundle, workspaceKey)
	require.NoError(t, err)
	require.NotNil(t, stim)
	assert.Equal(t, timFormatVersion, stim.Version)
	assert.NotEmpty(t, stim.Layers)

	// Encrypted rootfs files exist on disk.
	for name := range stim.Layers {
		encPath := filepath.Join(stim.Path, timLayerDir, name+timLayerFileExt)
		_, err := os.Stat(encPath)
		require.NoError(t, err)
	}

	decrypted, err := DecryptSTIM(stim, workspaceKey)
	require.NoError(t, err)
	require.NotNil(t, decrypted)
	assert.Equal(t, stim.ID, decrypted.ID)
	assert.Equal(t, bundle.Config.EntryPoint, decrypted.Config.EntryPoint)

	// Recovered the original plaintext file.
	recovered := filepath.Join(decrypted.RootFS, "app", "hello.txt")
	data, err := os.ReadFile(recovered)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestTIM_EncryptTIM_NilBundle_Bad(t *testing.T) {
	_, err := EncryptTIM(nil, []byte{1})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing tim bundle")
}

func TestTIM_EncryptTIM_EmptyKey_Bad(t *testing.T) {
	_, err := EncryptTIM(&TIMBundle{Path: "/tmp"}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace key")
}

func TestTIM_EncryptTIM_RejectsSTIMInput_Ugly(t *testing.T) {
	// The encrypt function must not accept an already-encrypted path.
	_, err := EncryptTIM(&TIMBundle{Path: "/tmp/app.stim"}, []byte{1, 2, 3})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stim bundle")
}

// --- DecryptSTIM ---

func TestTIM_DecryptSTIM_NilBundle_Bad(t *testing.T) {
	_, err := DecryptSTIM(nil, []byte{1})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing stim bundle")
}

func TestTIM_DecryptSTIM_EmptyKey_Bad(t *testing.T) {
	_, err := DecryptSTIM(&STIMBundle{Path: "/tmp"}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace key")
}

func TestTIM_DecryptSTIM_MissingManifest_Ugly(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := DecryptSTIM(&STIMBundle{Path: tmpDir}, []byte{1, 2, 3})

	require.Error(t, err)
}

// --- TIM provider encryption round-trip via Image contract ---

func TestTIM_Provider_EncryptDecrypt_Good(t *testing.T) {
	tmpDir := t.TempDir()
	bundle := setupTestTIMBundle(t, tmpDir)

	provider := NewTIMProvider()
	image := &Image{ID: bundle.ID, Name: "demo", Path: bundle.Path, Runtime: "tim"}

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i * 3)
	}

	encrypted, err := provider.Encrypt(image, key)
	require.NoError(t, err)
	assert.Equal(t, "tim", encrypted.Runtime)
	assert.Equal(t, "stim", encrypted.Metadata["format"])

	decrypted, err := provider.Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, "tim", decrypted.Metadata["format"])
}

func TestTIM_Provider_Encrypt_MissingImage_Bad(t *testing.T) {
	provider := NewTIMProvider()
	_, err := provider.Encrypt(nil, []byte{1, 2, 3})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing image")
}

func TestTIM_Provider_Decrypt_MissingImage_Ugly(t *testing.T) {
	provider := NewTIMProvider()
	_, err := provider.Decrypt(nil, []byte{1, 2, 3})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing encrypted image")
}

// --- Key derivation ---

func TestTIM_DeriveTIMKey_Deterministic_Good(t *testing.T) {
	a := deriveTIMKey([]byte("workspace"), "container", "abc")
	b := deriveTIMKey([]byte("workspace"), "container", "abc")

	assert.Equal(t, a, b)
	assert.Len(t, a, 32)
}

func TestTIM_DeriveTIMKey_Distinct_Bad(t *testing.T) {
	a := deriveTIMKey([]byte("workspace"), "container", "abc")
	b := deriveTIMKey([]byte("workspace"), "container", "xyz")

	assert.NotEqual(t, a, b)
}

func TestTIM_DeriveTIMKey_HierarchyIsolation_Ugly(t *testing.T) {
	// Workspace → container → layer. Changing any segment must produce a new key.
	wks := deriveTIMKey([]byte("workspace"), "container", "abc")
	ctr := deriveTIMKey(wks, "base")
	app := deriveTIMKey(wks, "app")
	data := deriveTIMKey(wks, "data")

	assert.NotEqual(t, ctr, app)
	assert.NotEqual(t, app, data)
	assert.NotEqual(t, ctr, data)
}

// setupTestTIMBundle writes a minimal three-layer TIM rootfs with a known
// file under the app/ layer so round-trip decryption can be verified.
// Also writes config.json so that loadTIMBundle can rehydrate the bundle
// from disk when the provider Encrypt path is exercised.
func setupTestTIMBundle(t *testing.T, tmpDir string) *TIMBundle {
	t.Helper()
	root := filepath.Join(tmpDir, "tim-bundle")
	rootfs := filepath.Join(root, timRootFSDir)

	for _, layer := range timLayers {
		require.NoError(t, os.MkdirAll(filepath.Join(rootfs, layer), 0o755))
	}

	payload := filepath.Join(rootfs, "app", "hello.txt")
	require.NoError(t, os.WriteFile(payload, []byte("hello"), 0o600))

	cfg := TIMConfig{
		EntryPoint: []string{"/app/hello"},
		Env:        []string{"CORE_ENV=test"},
		WorkDir:    "/app",
	}
	cfgBytes := []byte(`{"entrypoint":["/app/hello"],"env":["CORE_ENV=test"],"workdir":"/app","mounts":null,"capabilities":null,"readonly":false}`)
	require.NoError(t, os.WriteFile(filepath.Join(root, timConfigFile), cfgBytes, 0o600))

	return &TIMBundle{
		ID:     "test-bundle",
		Path:   root,
		RootFS: rootfs,
		Config: cfg,
	}
}
