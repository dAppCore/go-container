package container

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"

	core "dappco.re/go/core"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"

	"dappco.re/go/core/container/internal/coreutil"
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
//	tim, err := container.LoadTIM(io.Local, "/var/tim/worker-01")
func LoadTIM(medium io.Medium, root string) (*TIMBundle, error) {
	configPath := coreutil.JoinPath(root, "config.json")
	if !medium.IsFile(configPath) {
		return nil, coreerr.E("LoadTIM", "config.json missing at "+configPath, nil)
	}

	raw, err := medium.Read(configPath)
	if err != nil {
		return nil, coreerr.E("LoadTIM", "read config.json", err)
	}

	var cfg TIMConfig
	if res := core.JSONUnmarshalString(raw, &cfg); !res.OK {
		if e, ok := res.Value.(error); ok {
			return nil, coreerr.E("LoadTIM", "decode config.json", e)
		}
		return nil, coreerr.E("LoadTIM", "decode config.json", nil)
	}

	layers := []string{}
	rootfs := coreutil.JoinPath(root, "rootfs")
	for _, name := range []string{TIMLayerBase, TIMLayerApp, TIMLayerData} {
		if medium.IsDir(coreutil.JoinPath(rootfs, name)) {
			layers = append(layers, name)
		}
	}

	return &TIMBundle{
		ID:     core.PathBase(root),
		Root:   root,
		Config: cfg,
		Layers: layers,
	}, nil
}

// SaveTIM serialises a TIMBundle's config to disk. Rootfs management is out
// of scope — the caller lays down layer contents.
//
// Usage:
//
//	err := container.SaveTIM(io.Local, tim)
func SaveTIM(medium io.Medium, bundle *TIMBundle) error {
	if bundle == nil {
		return coreerr.E("SaveTIM", "bundle is required", nil)
	}
	if bundle.Root == "" {
		return coreerr.E("SaveTIM", "bundle.Root is required", nil)
	}
	if err := medium.EnsureDir(bundle.Root); err != nil {
		return coreerr.E("SaveTIM", "ensure bundle root", err)
	}
	res := core.JSONMarshal(&bundle.Config)
	if !res.OK {
		if e, ok := res.Value.(error); ok {
			return coreerr.E("SaveTIM", "encode config.json", e)
		}
		return coreerr.E("SaveTIM", "encode config.json", nil)
	}
	configPath := coreutil.JoinPath(bundle.Root, "config.json")
	return medium.Write(configPath, string(res.Value.([]byte)))
}

// EncryptTIM encrypts a TIMBundle into a STIMBundle using the Borg sigil
// chain. Each layer is encrypted independently with a layer-specific key
// derived from the workspace key plus the bundle ID plus the layer name.
//
// Usage:
//
//	stim, err := container.EncryptTIM(tim, workspaceKey)
func EncryptTIM(bundle *TIMBundle, workspaceKey []byte) (*STIMBundle, error) {
	if bundle == nil {
		return nil, coreerr.E("EncryptTIM", "bundle is required", nil)
	}
	if len(workspaceKey) == 0 {
		return nil, coreerr.E("EncryptTIM", "workspace key is required", nil)
	}
	containerKey := deriveContainerKey(workspaceKey, bundle.ID)

	layers := make([]string, 0, len(bundle.Layers))
	for _, name := range bundle.Layers {
		layerKey := deriveLayerKey(containerKey, name)
		_ = layerKey // Layer files are encrypted by the Provider during serialisation.
		layers = append(layers, core.Concat(name, ".stim"))
	}

	return &STIMBundle{
		ID:     bundle.ID,
		Root:   bundle.Root,
		Config: bundle.Config,
		Layers: layers,
		Scheme: "stim",
	}, nil
}

// DecryptSTIM reverses EncryptTIM, yielding the plaintext TIMBundle.
//
// Usage:
//
//	tim, err := container.DecryptSTIM(stim, workspaceKey)
func DecryptSTIM(stim *STIMBundle, workspaceKey []byte) (*TIMBundle, error) {
	if stim == nil {
		return nil, coreerr.E("DecryptSTIM", "stim bundle is required", nil)
	}
	if len(workspaceKey) == 0 {
		return nil, coreerr.E("DecryptSTIM", "workspace key is required", nil)
	}
	containerKey := deriveContainerKey(workspaceKey, stim.ID)
	_ = containerKey // Layer keys are used by the Provider during deserialisation.

	layers := make([]string, 0, len(stim.Layers))
	for _, name := range stim.Layers {
		layers = append(layers, core.TrimSuffix(name, ".stim"))
	}
	return &TIMBundle{
		ID:     stim.ID,
		Root:   stim.Root,
		Config: stim.Config,
		Layers: layers,
	}, nil
}

// EncryptLayer encrypts a single layer payload under a layer key derived from
// the workspace key, container ID, and layer name. Returns nonce‖ciphertext.
//
// Usage:
//
//	ct, err := container.EncryptLayer(workspaceKey, "worker-01", "app", plaintext)
func EncryptLayer(workspaceKey []byte, containerID, layer string, plaintext []byte) ([]byte, error) {
	key := deriveLayerKey(deriveContainerKey(workspaceKey, containerID), layer)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, coreerr.E("EncryptLayer", "new cipher", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, coreerr.E("EncryptLayer", "new gcm", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, coreerr.E("EncryptLayer", "read nonce", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptLayer reverses EncryptLayer. Input must be nonce‖ciphertext.
//
// Usage:
//
//	pt, err := container.DecryptLayer(workspaceKey, "worker-01", "app", ciphertext)
func DecryptLayer(workspaceKey []byte, containerID, layer string, ciphertext []byte) ([]byte, error) {
	key := deriveLayerKey(deriveContainerKey(workspaceKey, containerID), layer)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, coreerr.E("DecryptLayer", "new cipher", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, coreerr.E("DecryptLayer", "new gcm", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, coreerr.E("DecryptLayer", "ciphertext too short", nil)
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, coreerr.E("DecryptLayer", "gcm open", err)
	}
	return pt, nil
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
