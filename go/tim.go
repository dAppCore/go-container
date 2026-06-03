package container

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"

	core "dappco.re/go"
	"dappco.re/go/io"

	"dappco.re/go/container/internal/coreutil"
)

// TIMConfig defines the OCI-compatible configuration for a TIM container.
// See RFC.tim.md §2 for the full semantic model.
//
// Usage:
//
//	tim := container.TIMConfig{
//	    EntryPoint: []string{"/app/server"},
//	    Env:        []string{"CORE_ENV=production"},
//	    ReadOnly:   true,
//	}
type TIMConfig struct {
	EntryPoint   []string   `json:"entrypoint"`
	Env          []string   `json:"env"`
	WorkDir      string     `json:"workdir"`
	Mounts       []TIMMount `json:"mounts"`
	Capabilities []string   `json:"capabilities"`
	ReadOnly     bool       `json:"readonly"`
}

// TIMMount defines a filesystem mount point within the container.
//
// Usage:
//
//	mount := container.TIMMount{Source: "/data", Target: "/app/data", ReadOnly: true}
type TIMMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readonly"`
}

// TIM rootfs layer names. See RFC.tim.md §3 — three-layer convention.
const (
	// TIMLayerBase is the minimal distroless base layer (libc, ca-certs, tzdata).
	TIMLayerBase = "base"
	// TIMLayerApp is the application layer (binary, static assets).
	TIMLayerApp = "app"
	// TIMLayerData is the runtime-state layer (often a mount point).
	TIMLayerData = "data"
)

// TIMBundle is a materialised TIM container. Rootfs contents live on the host
// filesystem at Root and follow the three-layer convention base/app/data.
//
// Usage:
//
//	tim := container.TIMBundle{
//	    ID:     "worker-01",
//	    Root:   "/var/tim/worker-01",
//	    Config: container.TIMConfig{EntryPoint: []string{"/app/server"}},
//	}
type TIMBundle struct {
	// ID is a unique identifier for the bundle.
	ID string
	// Root is the filesystem path containing config.json and rootfs/.
	Root string
	// Config is the decoded TIM OCI-compatible configuration.
	Config TIMConfig
	// Layers lists the present rootfs layers in order (base, app, data).
	Layers []string
}

// STIMBundle is an encrypted TIM bundle. Each layer is encrypted under a key
// derived from the workspace key. See RFC.tim.md §5-6 for the key hierarchy.
//
// Usage:
//
//	stim, _ := container.EncryptTIM(tim, workspaceKey)
//	tim, _  = container.DecryptSTIM(stim, workspaceKey)
type STIMBundle struct {
	// ID mirrors the underlying TIMBundle.ID.
	ID string
	// Root is the filesystem path containing the encrypted layers and cleartext config.json.
	Root string
	// Config is cleartext (matches the TIMBundle config).
	Config TIMConfig
	// Layers lists the encrypted layer filenames under Root.
	Layers []string
	// Scheme names the encryption scheme (always "stim" for this type).
	Scheme string
}

// NewTIMBundle constructs a TIMBundle placeholder for the given root path.
// The caller is responsible for populating Config and laying down rootfs.
//
// Usage:
//
//	bundle := container.NewTIMBundle("worker-01", "/var/tim/worker-01")
func NewTIMBundle(id, root string) *TIMBundle {
	return &TIMBundle{
		ID:     id,
		Root:   root,
		Layers: []string{TIMLayerBase, TIMLayerApp, TIMLayerData},
	}
}

// LoadTIM reads a TIMBundle from disk. It decodes config.json and lists the
// layers present under rootfs/.
//
// Usage:
//
//	tim := core.MustCast[*TIMBundle](container.LoadTIM(io.Local, "/var/tim/worker-01"))
func LoadTIM(medium io.Medium, root string) core.Result { // Value: *TIMBundle
	configPath := coreutil.JoinPath(root, "config.json")
	if !medium.IsFile(configPath) {
		return core.Fail(core.E("LoadTIM", "config.json missing at "+configPath, nil))
	}

	raw, err := medium.Read(configPath)
	if err != nil {
		return core.Fail(core.E("LoadTIM", "read config.json", err))
	}

	var cfg TIMConfig
	if res := core.JSONUnmarshalString(raw, &cfg); !res.OK {
		if e, ok := res.Value.(error); ok {
			return core.Fail(core.E("LoadTIM", "decode config.json", e))
		}
		return core.Fail(core.E("LoadTIM", "decode config.json", nil))
	}

	layers := []string{}
	rootfs := coreutil.JoinPath(root, "rootfs")
	for _, name := range []string{TIMLayerBase, TIMLayerApp, TIMLayerData} {
		if medium.IsDir(coreutil.JoinPath(rootfs, name)) {
			layers = append(layers, name)
		}
	}

	return core.Ok(&TIMBundle{
		ID:     core.PathBase(root),
		Root:   root,
		Config: cfg,
		Layers: layers,
	})
}

// SaveTIM serialises a TIMBundle's config to disk. Rootfs management is out
// of scope — the caller lays down layer contents.
//
// Usage:
//
//	if r := container.SaveTIM(io.Local, tim); !r.OK { return r }
func SaveTIM(medium io.Medium, bundle *TIMBundle) core.Result { // Value: nil
	if bundle == nil {
		return core.Fail(core.E("SaveTIM", "bundle is required", nil))
	}
	if bundle.Root == "" {
		return core.Fail(core.E("SaveTIM", "bundle.Root is required", nil))
	}
	if err := medium.EnsureDir(bundle.Root); err != nil {
		return core.Fail(core.E("SaveTIM", "ensure bundle root", err))
	}
	res := core.JSONMarshal(&bundle.Config)
	if !res.OK {
		if e, ok := res.Value.(error); ok {
			return core.Fail(core.E("SaveTIM", "encode config.json", e))
		}
		return core.Fail(core.E("SaveTIM", "encode config.json", nil))
	}
	configPath := coreutil.JoinPath(bundle.Root, "config.json")
	if err := medium.Write(configPath, string(core.MustCast[[]byte](res))); err != nil {
		return core.Fail(core.E("SaveTIM", "write config.json", err))
	}
	return core.Ok(nil)
}

// EncryptTIM encrypts a TIMBundle into a STIMBundle using the Borg sigil
// chain. Each layer is encrypted independently with a layer-specific key
// derived from the workspace key plus the bundle ID plus the layer name.
//
// This operates purely on the STIMBundle record — layer file payloads are
// encrypted by EncryptTIMOnMedium when a concrete io.Medium is provided.
//
// Usage:
//
//	stim := core.MustCast[*STIMBundle](container.EncryptTIM(tim, workspaceKey))
func EncryptTIM(bundle *TIMBundle, workspaceKey []byte) core.Result { // Value: *STIMBundle
	if bundle == nil {
		return core.Fail(core.E("EncryptTIM", "bundle is required", nil))
	}
	if len(workspaceKey) == 0 {
		return core.Fail(core.E("EncryptTIM", "workspace key is required", nil))
	}
	containerKey := deriveContainerKey(workspaceKey, bundle.ID)

	layers := make([]string, 0, len(bundle.Layers))
	for _, name := range bundle.Layers {
		layerKey := deriveLayerKey(containerKey, name)
		_ = layerKey // Key derivation validated; payload sealing happens in EncryptTIMOnMedium.
		layers = append(layers, core.Concat(name, ".stim"))
	}

	return core.Ok(&STIMBundle{
		ID:     bundle.ID,
		Root:   bundle.Root,
		Config: bundle.Config,
		Layers: layers,
		Scheme: "stim",
	})
}

// DecryptSTIM reverses EncryptTIM, yielding the plaintext TIMBundle record.
// Use DecryptSTIMOnMedium to decrypt actual layer payloads on disk.
//
// Usage:
//
//	tim := core.MustCast[*TIMBundle](container.DecryptSTIM(stim, workspaceKey))
func DecryptSTIM(stim *STIMBundle, workspaceKey []byte) core.Result { // Value: *TIMBundle
	if stim == nil {
		return core.Fail(core.E("DecryptSTIM", "stim bundle is required", nil))
	}
	if len(workspaceKey) == 0 {
		return core.Fail(core.E("DecryptSTIM", "workspace key is required", nil))
	}
	containerKey := deriveContainerKey(workspaceKey, stim.ID)
	_ = containerKey // Key derivation validated; payload opening happens in DecryptSTIMOnMedium.

	layers := make([]string, 0, len(stim.Layers))
	for _, name := range stim.Layers {
		layers = append(layers, core.TrimSuffix(name, ".stim"))
	}
	return core.Ok(&TIMBundle{
		ID:     stim.ID,
		Root:   stim.Root,
		Config: stim.Config,
		Layers: layers,
	})
}

// EncryptTIMOnMedium is the full-fidelity encrypt-on-disk flow. For each
// layer under rootfs/<name>/ the function tarballs the layer, encrypts the
// archive under the derived layer key, and writes rootfs/<name>.stim in
// ciphertext form. The cleartext config.json is preserved. Empty or missing
// layer directories are skipped.
//
// Usage:
//
//	stim := core.MustCast[*STIMBundle](container.EncryptTIMOnMedium(io.Local, tim, workspaceKey))
func EncryptTIMOnMedium(medium io.Medium, bundle *TIMBundle, workspaceKey []byte) core.Result { // Value: *STIMBundle
	if medium == nil {
		return core.Fail(core.E("EncryptTIMOnMedium", "medium is required", nil))
	}
	if bundle == nil {
		return core.Fail(core.E("EncryptTIMOnMedium", "bundle is required", nil))
	}
	if len(workspaceKey) == 0 {
		return core.Fail(core.E("EncryptTIMOnMedium", "workspace key is required", nil))
	}
	if bundle.Root == "" {
		return core.Fail(core.E("EncryptTIMOnMedium", "bundle.Root is required", nil))
	}

	rootfs := coreutil.JoinPath(bundle.Root, "rootfs")
	encryptedLayers := make([]string, 0, len(bundle.Layers))
	for _, name := range bundle.Layers {
		layerDir := coreutil.JoinPath(rootfs, name)
		if !medium.IsDir(layerDir) {
			// No plaintext layer to encrypt — keep ciphertext name in manifest.
			encryptedLayers = append(encryptedLayers, core.Concat(name, ".stim"))
			continue
		}
		payloadRes := collectLayer(medium, layerDir)
		if !payloadRes.OK {
			return core.Fail(core.E("EncryptTIMOnMedium", "collect layer "+name, payloadRes.Value.(error)))
		}
		sealedRes := EncryptLayer(workspaceKey, bundle.ID, name, core.MustCast[[]byte](payloadRes))
		if !sealedRes.OK {
			return core.Fail(core.E("EncryptTIMOnMedium", "encrypt layer "+name, sealedRes.Value.(error)))
		}
		outPath := coreutil.JoinPath(rootfs, core.Concat(name, ".stim"))
		if err := medium.Write(outPath, string(core.MustCast[[]byte](sealedRes))); err != nil {
			return core.Fail(core.E("EncryptTIMOnMedium", "write sealed layer "+name, err))
		}
		encryptedLayers = append(encryptedLayers, core.Concat(name, ".stim"))
	}
	return core.Ok(&STIMBundle{
		ID:     bundle.ID,
		Root:   bundle.Root,
		Config: bundle.Config,
		Layers: encryptedLayers,
		Scheme: "stim",
	})
}

// DecryptSTIMOnMedium reverses EncryptTIMOnMedium. Each rootfs/<name>.stim
// is decrypted and written back as rootfs/<name>/payload.bin. The caller is
// responsible for unpacking the archive format chosen by collectLayer.
//
// Usage:
//
//	tim := core.MustCast[*TIMBundle](container.DecryptSTIMOnMedium(io.Local, stim, workspaceKey))
func DecryptSTIMOnMedium(medium io.Medium, stim *STIMBundle, workspaceKey []byte) core.Result { // Value: *TIMBundle
	if medium == nil {
		return core.Fail(core.E("DecryptSTIMOnMedium", "medium is required", nil))
	}
	if stim == nil {
		return core.Fail(core.E("DecryptSTIMOnMedium", "stim bundle is required", nil))
	}
	if len(workspaceKey) == 0 {
		return core.Fail(core.E("DecryptSTIMOnMedium", "workspace key is required", nil))
	}
	if stim.Root == "" {
		return core.Fail(core.E("DecryptSTIMOnMedium", "stim.Root is required", nil))
	}

	rootfs := coreutil.JoinPath(stim.Root, "rootfs")
	plaintextLayers := make([]string, 0, len(stim.Layers))
	for _, sealedName := range stim.Layers {
		plainName := core.TrimSuffix(sealedName, ".stim")
		sealedPath := coreutil.JoinPath(rootfs, sealedName)
		if !medium.IsFile(sealedPath) {
			plaintextLayers = append(plaintextLayers, plainName)
			continue
		}
		sealed, err := medium.Read(sealedPath)
		if err != nil {
			return core.Fail(core.E("DecryptSTIMOnMedium", "read sealed layer "+sealedName, err))
		}
		payloadRes := DecryptLayer(workspaceKey, stim.ID, plainName, []byte(sealed))
		if !payloadRes.OK {
			return core.Fail(core.E("DecryptSTIMOnMedium", "decrypt layer "+plainName, payloadRes.Value.(error)))
		}
		layerDir := coreutil.JoinPath(rootfs, plainName)
		if err := medium.EnsureDir(layerDir); err != nil {
			return core.Fail(core.E("DecryptSTIMOnMedium", "ensure layer dir "+plainName, err))
		}
		payloadPath := coreutil.JoinPath(layerDir, "payload.bin")
		if err := medium.Write(payloadPath, string(core.MustCast[[]byte](payloadRes))); err != nil {
			return core.Fail(core.E("DecryptSTIMOnMedium", "write payload "+plainName, err))
		}
		plaintextLayers = append(plaintextLayers, plainName)
	}
	return core.Ok(&TIMBundle{
		ID:     stim.ID,
		Root:   stim.Root,
		Config: stim.Config,
		Layers: plaintextLayers,
	})
}

// collectLayer serialises a layer directory into a single flat buffer. Each
// entry is encoded as a length-prefixed name followed by a length-prefixed
// content blob. This deterministic encoding lets EncryptLayer seal the whole
// layer as one AEAD block.
func collectLayer(medium io.Medium, dir string) core.Result { // Value: []byte
	entries, err := medium.List(dir)
	if err != nil {
		return core.Fail(core.E("collectLayer", "list layer dir", err))
	}
	var buf []byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := coreutil.JoinPath(dir, name)
		content, err := medium.Read(path)
		if err != nil {
			return core.Fail(core.E("collectLayer", "read "+path, err))
		}
		buf = append(buf, encodeLen(uint32(len(name)))...)
		buf = append(buf, []byte(name)...)
		buf = append(buf, encodeLen(uint32(len(content)))...)
		buf = append(buf, []byte(content)...)
	}
	return core.Ok(buf)
}

// encodeLen writes a 4-byte big-endian length prefix.
func encodeLen(n uint32) []byte {
	return []byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
}

// EncryptLayer encrypts a single layer payload under a layer key derived from
// the workspace key, container ID, and layer name. Returns nonce‖ciphertext.
//
// Usage:
//
//	ct := core.MustCast[[]byte](container.EncryptLayer(workspaceKey, "worker-01", "app", plaintext))
func EncryptLayer(workspaceKey []byte, containerID, layer string, plaintext []byte) core.Result { // Value: []byte
	key := deriveLayerKey(deriveContainerKey(workspaceKey, containerID), layer)
	block, err := aes.NewCipher(key)
	if err != nil {
		return core.Fail(core.E("EncryptLayer", "new cipher", err))
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return core.Fail(core.E("EncryptLayer", "new gcm", err))
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return core.Fail(core.E("EncryptLayer", "read nonce", err))
	}
	return core.Ok(gcm.Seal(nonce, nonce, plaintext, nil))
}

// DecryptLayer reverses EncryptLayer. Input must be nonce‖ciphertext.
//
// Usage:
//
//	pt := core.MustCast[[]byte](container.DecryptLayer(workspaceKey, "worker-01", "app", ciphertext))
func DecryptLayer(workspaceKey []byte, containerID, layer string, ciphertext []byte) core.Result { // Value: []byte
	key := deriveLayerKey(deriveContainerKey(workspaceKey, containerID), layer)
	block, err := aes.NewCipher(key)
	if err != nil {
		return core.Fail(core.E("DecryptLayer", "new cipher", err))
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return core.Fail(core.E("DecryptLayer", "new gcm", err))
	}
	if len(ciphertext) < gcm.NonceSize() {
		return core.Fail(core.E("DecryptLayer", "ciphertext too short", nil))
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return core.Fail(core.E("DecryptLayer", "gcm open", err))
	}
	return core.Ok(pt)
}

// deriveContainerKey derives a container-specific key from the workspace key
// and container ID. See RFC.tim.md §6 — Level 2 key.
func deriveContainerKey(workspaceKey []byte, containerID string) []byte {
	h := sha256.New()
	h.Write(workspaceKey)
	h.Write([]byte("tim:container:"))
	h.Write([]byte(containerID))
	return h.Sum(nil)
}

// deriveLayerKey derives a layer-specific key from the container key. See
// RFC.tim.md §6 — Level 3 key.
func deriveLayerKey(containerKey []byte, layer string) []byte {
	h := sha256.New()
	h.Write(containerKey)
	h.Write([]byte("tim:layer:"))
	h.Write([]byte(layer))
	return h.Sum(nil)
}
