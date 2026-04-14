package container

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
)

// TIMConfig is a subset of the OCI-compatible runtime config used by TIM.
type TIMConfig struct {
	EntryPoint   []string    `json:"entrypoint"`
	Env          []string    `json:"env"`
	WorkDir      string      `json:"workdir"`
	Mounts       []TIMMount  `json:"mounts"`
	Capabilities []string    `json:"capabilities"`
	ReadOnly     bool        `json:"readonly"`
}

// TIMMount defines a mount within a TIM rootfs layer.
type TIMMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readonly"`
}

// TIMBundle represents an unencrypted TIM bundle.
type TIMBundle struct {
	ID     string   `json:"id"`
	Path   string   `json:"path"`
	RootFS string   `json:"rootfs"`
	Config TIMConfig `json:"config"`
}

// STIMBundle is an encrypted TIM bundle.
type STIMBundle struct {
	ID      string                    `json:"id"`
	Path    string                    `json:"path"`
	Config  TIMConfig                `json:"config"`
	Layers  map[string]stimLayerMeta `json:"layers"`
	Version string                  `json:"version"`
}

type stimLayerMeta struct {
	File   string `json:"file"`
	Nonce  string `json:"nonce"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

type stimManifest struct {
	ID      string                    `json:"id"`
	Config  TIMConfig                `json:"config"`
	Layers  map[string]stimLayerMeta `json:"layers"`
	Version string                  `json:"version"`
}

var timLayers = []string{"base", "app", "data"}

const (
	timManifestFile  = "manifest.json"
	timConfigFile    = "config.json"
	timRootFSDir     = "rootfs"
	timLayerDir      = "layers"
	timLayerFileExt  = ".enc"
	timFormatVersion = "tim-stim-1"
)

// EncryptTIM encrypts a TIM bundle into a STIM bundle.
//
//   stim, err := container.EncryptTIM(bundle, workspaceKey)
func EncryptTIM(tim *TIMBundle, workspaceKey []byte) (*STIMBundle, error) {
	if tim == nil {
		return nil, coreerr.E("EncryptTIM", "missing tim bundle", nil)
	}
	if len(workspaceKey) == 0 {
		return nil, coreerr.E("EncryptTIM", "missing workspace key", nil)
	}

	timPath := strings.TrimSpace(tim.Path)
	if timPath == "" {
		return nil, coreerr.E("EncryptTIM", "missing tim path", nil)
	}

	rootPath := timPath
	if ext := filepath.Ext(timPath); strings.EqualFold(ext, ".stim") {
		return nil, coreerr.E("EncryptTIM", "input path must not be a stim bundle", nil)
	}

	rootFS := strings.TrimSpace(tim.RootFS)
	if rootFS == "" {
		rootFS = filepath.Join(rootPath, timRootFSDir)
	}

	id := strings.TrimSpace(tim.ID)
	if id == "" {
		id = generateTIMID(rootPath, tim.Config)
	}

	stimPath := core.Concat(rootPath, ".stim")
	if err := os.MkdirAll(stimPath, 0o755); err != nil {
		return nil, coreerr.E("EncryptTIM", "create stim directory", err)
	}

	manifest := &stimManifest{
		ID:      id,
		Config:  tim.Config,
		Layers:  map[string]stimLayerMeta{},
		Version: timFormatVersion,
	}

	containerKey := deriveTIMKey(workspaceKey, "container", id)

	for _, layer := range timLayers {
		layerPath := filepath.Join(rootFS, layer)
		layerExists := false
		if info, err := os.Stat(layerPath); err == nil && info.IsDir() {
			layerExists = true
		}
		if !layerExists {
			continue
		}

		plaintext, err := packLayer(layerPath)
		if err != nil {
			return nil, coreerr.E("EncryptTIM", "pack layer: "+layer, err)
		}

		layerKey := deriveTIMKey(containerKey, layer)
		encrypted, nonce, err := encryptPayload(layerKey, plaintext)
		if err != nil {
			return nil, coreerr.E("EncryptTIM", "encrypt layer: "+layer, err)
		}

		layerFile := filepath.Join(stimPath, timLayerDir)
		if err := os.MkdirAll(layerFile, 0o755); err != nil {
			return nil, coreerr.E("EncryptTIM", "create layers directory", err)
		}
		destination := filepath.Join(layerFile, layer+timLayerFileExt)
		if err := os.WriteFile(destination, encrypted, 0o600); err != nil {
			return nil, coreerr.E("EncryptTIM", "write encrypted layer: "+layer, err)
		}

		manifest.Layers[layer] = stimLayerMeta{
			File:   filepath.Base(destination),
			Nonce: base64.StdEncoding.EncodeToString(nonce),
			Size:  len(encrypted),
			SHA256: checksumSHA256(encrypted),
		}
	}

	manifestPath := filepath.Join(stimPath, timManifestFile)
	rawManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, coreerr.E("EncryptTIM", "marshal manifest", err)
	}
	if err := os.WriteFile(manifestPath, rawManifest, 0o600); err != nil {
		return nil, coreerr.E("EncryptTIM", "write manifest", err)
	}

	configPath := filepath.Join(stimPath, timConfigFile)
	rawConfig, err := json.MarshalIndent(tim.Config, "", "  ")
	if err != nil {
		return nil, coreerr.E("EncryptTIM", "marshal config", err)
	}
	if err := os.WriteFile(configPath, rawConfig, 0o600); err != nil {
		return nil, coreerr.E("EncryptTIM", "write config", err)
	}

	return &STIMBundle{
		ID:      id,
		Path:    stimPath,
		Config:  tim.Config,
		Layers:  manifest.Layers,
		Version: timFormatVersion,
	}, nil
}

// DecryptSTIM decrypts a STIM bundle into a TIM bundle.
//
//   tim, err := container.DecryptSTIM(stim, workspaceKey)
func DecryptSTIM(stim *STIMBundle, workspaceKey []byte) (*TIMBundle, error) {
	if stim == nil {
		return nil, coreerr.E("DecryptSTIM", "missing stim bundle", nil)
	}
	if len(workspaceKey) == 0 {
		return nil, coreerr.E("DecryptSTIM", "missing workspace key", nil)
	}
	if strings.TrimSpace(stim.Path) == "" {
		return nil, coreerr.E("DecryptSTIM", "missing stim path", nil)
	}

	rawManifest, err := os.ReadFile(filepath.Join(stim.Path, timManifestFile))
	if err != nil {
		return nil, coreerr.E("DecryptSTIM", "read manifest", err)
	}

	var manifest stimManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, coreerr.E("DecryptSTIM", "parse manifest", err)
	}

	tmpDir, err := os.MkdirTemp("", "core-tim-decrypt-")
	if err != nil {
		return nil, coreerr.E("DecryptSTIM", "create temporary directory", err)
	}

	rootFS := filepath.Join(tmpDir, timRootFSDir)
	for _, layer := range timLayers {
		if err := os.MkdirAll(filepath.Join(rootFS, layer), 0o755); err != nil {
			return nil, coreerr.E("DecryptSTIM", "create rootfs layer", err)
		}
	}

	containerKey := deriveTIMKey(workspaceKey, "container", manifest.ID)
	for _, layer := range timLayers {
		meta, found := manifest.Layers[layer]
		if !found {
			continue
		}

		encoded := filepath.Join(stim.Path, timLayerDir, meta.File)
		ciphertext, err := os.ReadFile(encoded)
		if err != nil {
			return nil, coreerr.E("DecryptSTIM", "read encrypted layer: "+layer, err)
		}

		layerKey := deriveTIMKey(containerKey, layer)
		plaintext, err := decryptPayload(layerKey, ciphertext)
		if err != nil {
			return nil, coreerr.E("DecryptSTIM", "decrypt layer: "+layer, err)
		}

		layerRoot := filepath.Join(rootFS, layer)
		if err := unpackLayer(layerRoot, plaintext); err != nil {
			return nil, coreerr.E("DecryptSTIM", "unpack layer: "+layer, err)
		}
	}

	configPath := filepath.Join(tmpDir, timConfigFile)
	rawConfig := stim.Config
	if len(manifest.Config.EntryPoint) > 0 || len(manifest.Config.Mounts) > 0 || manifest.Config.WorkDir != "" {
		rawConfig = manifest.Config
	} else {
		// Keep cleartext config if manifest lost it.
		if fallback, err := os.ReadFile(filepath.Join(stim.Path, timConfigFile)); err == nil {
			var fromFile TIMConfig
			if json.Unmarshal(fallback, &fromFile) == nil {
				rawConfig = fromFile
			}
		}
	}
	configBytes, err := json.MarshalIndent(rawConfig, "", "  ")
	if err != nil {
		return nil, coreerr.E("DecryptSTIM", "marshal config", err)
	}
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		return nil, coreerr.E("DecryptSTIM", "write config", err)
	}

	return &TIMBundle{
		ID:     manifest.ID,
		Path:   tmpDir,
		RootFS: rootFS,
		Config: rawConfig,
	}, nil
}

// NewTIMProvider returns a TIM provider.
func NewTIMProvider() *TIMProvider {
	return &TIMProvider{}
}

// TIMProvider implements the experimental TIM path.
type TIMProvider struct{}

var _ Provider = (*TIMProvider)(nil)

// Build validates and returns an image wrapper for a TIM bundle.
func (t *TIMProvider) Build(config ContainerConfig) (*Image, error) {
	if t == nil {
		return nil, coreerr.E("TIMProvider.Build", "provider is nil", nil)
	}

	source := strings.TrimSpace(config.Path)
	if source == "" {
		source = strings.TrimSpace(config.Source)
	}
	if source == "" {
		return nil, coreerr.E("TIMProvider.Build", "missing tim source", nil)
	}

	if !filepath.IsAbs(source) {
		abs, err := filepath.Abs(source)
		if err != nil {
			return nil, coreerr.E("TIMProvider.Build", "resolve tim source", err)
		}
		source = abs
	}

	id, err := GenerateID()
	if err != nil {
		return nil, coreerr.E("TIMProvider.Build", "generate image id", err)
	}

	return &Image{
		ID:       id,
		Name:     strings.TrimSpace(config.Name),
		Path:     source,
		Runtime:  "tim",
		Metadata: map[string]string{"runtime": "tim"},
	}, nil
}

// Run starts a TIM bundle. This is intentionally unsupported in this build.
func (t *TIMProvider) Run(_ *Image, _ ...RunOption) (*Container, error) {
	return nil, coreerr.E("TIMProvider.Run", "TIM execution is not implemented in this build", nil)
}

// Encrypt wraps EncryptTIM for the Image contract.
func (t *TIMProvider) Encrypt(image *Image, key []byte) (*EncryptedImage, error) {
	if image == nil {
		return nil, coreerr.E("TIMProvider.Encrypt", "missing image", nil)
	}

	bundle, err := loadTIMBundle(image.Path)
	if err != nil {
		return nil, coreerr.E("TIMProvider.Encrypt", "load tim bundle", err)
	}

	stim, err := EncryptTIM(bundle, key)
	if err != nil {
		return nil, coreerr.E("TIMProvider.Encrypt", "encrypt", err)
	}

	return &EncryptedImage{
		ID:       image.ID,
		Name:     image.Name,
		Path:     stim.Path,
		Runtime:  "tim",
		Metadata: map[string]string{"format": "stim"},
	}, nil
}

// Decrypt wraps DecryptSTIM for the Image contract.
func (t *TIMProvider) Decrypt(encrypted *EncryptedImage, key []byte) (*Image, error) {
	if encrypted == nil {
		return nil, coreerr.E("TIMProvider.Decrypt", "missing encrypted image", nil)
	}

	tim, err := DecryptSTIM(&STIMBundle{
		ID:   encrypted.ID,
		Path: encrypted.Path,
	}, key)
	if err != nil {
		return nil, coreerr.E("TIMProvider.Decrypt", "decrypt", err)
	}

	return &Image{
		ID:       encrypted.ID,
		Name:     encrypted.Name,
		Path:     tim.Path,
		Runtime:  "tim",
		Metadata: map[string]string{"format": "tim"},
	}, nil
}

func loadTIMBundle(path string) (*TIMBundle, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil, coreerr.E("loadTIMBundle", "missing path", nil)
	}

	fi, err := os.Stat(p)
	if err != nil {
		return nil, coreerr.E("loadTIMBundle", "missing tim path", err)
	}

	root := p
	if !fi.IsDir() {
		return nil, coreerr.E("loadTIMBundle", "tim path is not a directory", nil)
	}

	configPath := filepath.Join(root, timConfigFile)
	rawConfig, err := os.ReadFile(configPath)
	if err != nil {
		return nil, coreerr.E("loadTIMBundle", "read config.json", err)
	}

	var config TIMConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, coreerr.E("loadTIMBundle", "parse config.json", err)
	}

	return &TIMBundle{
		ID:     generateTIMID(root, config),
		Path:   root,
		RootFS: filepath.Join(root, timRootFSDir),
		Config: config,
	}, nil
}

func generateTIMID(path string, config TIMConfig) string {
	sum := sha256.Sum256([]byte(core.Sprintf("%s:%v", path, config)))
	return hex.EncodeToString(sum[:4])
}

func deriveTIMKey(secret []byte, labels ...string) []byte {
	hash := sha256.New()
	_, _ = hash.Write(secret)
	for _, label := range labels {
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(label))
	}
	sum := hash.Sum(nil)
	return sum
}

func packLayer(layerPath string) ([]byte, error) {
	buf := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzipWriter)

	err := filepath.WalkDir(layerPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, d.Name())
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(layerPath, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := tarWriter.Write(data); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		_ = tarWriter.Close()
		_ = gzipWriter.Close()
		return nil, err
	}

	if err := tarWriter.Close(); err != nil {
		_ = gzipWriter.Close()
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unpackLayer(target string, packed []byte) error {
	reader := bytes.NewReader(packed)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		dest := filepath.Join(target, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			dir := filepath.Dir(dest)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tarReader)
			closeErr := out.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink, tar.TypeLink, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			// Preserve as regular empty files when unpacking in this implementation.
			dir := filepath.Dir(dest)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			_, _ = os.Create(dest)
		default:
			return fmt.Errorf("unsupported tar entry type: %v", header.Typeflag)
		}
	}
	return nil
}

func encryptPayload(key, plain []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plain, nil)
	packed := append(nonce, ciphertext...)
	return packed, nonce, nil
}

func decryptPayload(key, packed []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(packed) < nonceSize {
		return nil, coreerr.E("decryptPayload", "ciphertext too small", nil)
	}

	nonce := packed[:nonceSize]
	ciphertext := packed[nonceSize:]
	return aead.Open(nil, nonce, ciphertext, nil)
}

func checksumSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
