package devenv

import (
	"context"
	"io/fs"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container/sources"
	"dappco.re/go/io"

	"dappco.re/go/container/internal/coreutil"
)

// ImageManager handles image downloads and updates.
type ImageManager struct {
	medium   io.Medium
	config   *Config
	manifest *Manifest
	sources  []sources.ImageSource
}

// Manifest tracks installed images.
type Manifest struct {
	medium io.Medium
	Images map[string]ImageInfo `json:"images"`
	path   string
}

// ImageInfo holds metadata about an installed image.
type ImageInfo struct {
	Version    string    `json:"version"`
	SHA256     string    `json:"sha256,omitempty"`
	Downloaded time.Time `json:"downloaded"`
	Source     string    `json:"source"`
}

// UpdateInfo is the result of an ImageManager.CheckUpdate call. It carries
// the currently-installed version, the latest available version, and whether
// an update is available.
type UpdateInfo struct {
	// Current is the installed image version.
	Current string
	// Latest is the latest available image version.
	Latest string
	// HasUpdate reports whether Latest differs from Current.
	HasUpdate bool
}

// NewImageManager creates a new image manager.
//
// Usage:
//
//	manager := core.MustCast[*ImageManager](NewImageManager(io.Local, cfg))
func NewImageManager(m io.Medium, cfg *Config) core.Result { // Value: *ImageManager
	dirRes := ImagesDir()
	if !dirRes.OK {
		return dirRes
	}
	imagesDir := core.MustCast[string](dirRes)

	// Ensure images directory exists
	if err := m.EnsureDir(imagesDir); err != nil {
		return core.Fail(core.E("NewImageManager", "ensure images directory", err))
	}

	// Load or create manifest
	manifestPath := coreutil.JoinPath(imagesDir, "manifest.json")
	manifestRes := loadManifest(m, manifestPath)
	if !manifestRes.OK {
		return manifestRes
	}
	manifest := core.MustCast[*Manifest](manifestRes)

	// Build source list based on config
	imageName := ImageName()
	sourceCfg := sources.SourceConfig{
		GitHubRepo:    cfg.Images.GitHub.Repo,
		RegistryImage: cfg.Images.Registry.Image,
		CDNURL:        cfg.Images.CDN.URL,
		ImageName:     imageName,
	}

	var srcs []sources.ImageSource
	switch cfg.Images.Source {
	case "github":
		srcs = []sources.ImageSource{sources.NewGitHubSource(sourceCfg)}
	case "cdn":
		srcs = []sources.ImageSource{sources.NewCDNSource(sourceCfg)}
	default: // "auto"
		srcs = []sources.ImageSource{
			sources.NewGitHubSource(sourceCfg),
			sources.NewCDNSource(sourceCfg),
		}
	}

	return core.Ok(&ImageManager{
		medium:   m,
		config:   cfg,
		manifest: manifest,
		sources:  srcs,
	})
}

// IsInstalled checks if the dev image is installed.
func (m *ImageManager) IsInstalled() bool {
	pathRes := ImagePath()
	if !pathRes.OK {
		return false
	}
	return m.medium.IsFile(core.MustCast[string](pathRes))
}

// Install downloads and installs the dev image.
//
// Usage:
//
//	if r := mgr.Install(ctx, nil); !r.OK { return r }
func (m *ImageManager) Install(ctx context.Context, progress func(downloaded, total int64)) core.Result { // Value: nil
	dirRes := ImagesDir()
	if !dirRes.OK {
		return dirRes
	}
	imagesDir := core.MustCast[string](dirRes)

	// Find first available source
	var src sources.ImageSource
	for _, s := range m.sources {
		if s.Available() {
			src = s
			break
		}
	}
	if src == nil {
		return core.Fail(core.E("ImageManager.Install", "no image source available", nil))
	}

	// Get version
	versionRes := src.LatestVersion(ctx)
	if !versionRes.OK {
		return core.Fail(core.E("ImageManager.Install", "failed to get latest version", versionRes.Value.(error)))
	}
	version := core.MustCast[string](versionRes)

	core.Print(nil, "Downloading %s from %s...", ImageName(), src.Name())

	// Download
	if r := src.Download(ctx, m.medium, imagesDir, progress); !r.OK {
		return r
	}

	// Update manifest
	m.manifest.Images[ImageName()] = ImageInfo{
		Version:    version,
		Downloaded: time.Now(),
		Source:     src.Name(),
	}

	return m.manifest.Save()
}

// CheckUpdate checks if an update is available.
//
// Usage:
//
//	info := core.MustCast[*UpdateInfo](mgr.CheckUpdate(ctx))
func (m *ImageManager) CheckUpdate(ctx context.Context) core.Result { // Value: *UpdateInfo
	info, ok := m.manifest.Images[ImageName()]
	if !ok {
		return core.Fail(core.E("ImageManager.CheckUpdate", "image not installed", nil))
	}
	current := info.Version

	// Find first available source
	var src sources.ImageSource
	for _, s := range m.sources {
		if s.Available() {
			src = s
			break
		}
	}
	if src == nil {
		return core.Fail(core.E("ImageManager.CheckUpdate", "no image source available", nil))
	}

	versionRes := src.LatestVersion(ctx)
	if !versionRes.OK {
		return core.Fail(core.E("ImageManager.CheckUpdate", "failed to get latest version", versionRes.Value.(error)))
	}
	latest := core.MustCast[string](versionRes)

	return core.Ok(&UpdateInfo{
		Current:   current,
		Latest:    latest,
		HasUpdate: current != latest,
	})
}

func loadManifest(m io.Medium, path string) core.Result { // Value: *Manifest
	manifest := &Manifest{
		medium: m,
		Images: make(map[string]ImageInfo),
		path:   path,
	}

	content, err := m.Read(path)
	if err != nil {
		if core.Is(err, fs.ErrNotExist) {
			return core.Ok(manifest)
		}
		return core.Fail(core.E("loadManifest", "read manifest", err))
	}

	result := core.JSONUnmarshalString(content, manifest)
	if !result.OK {
		return result
	}
	manifest.medium = m
	manifest.path = path

	return core.Ok(manifest)
}

// Save writes the manifest to disk.
//
// Usage:
//
//	if r := manifest.Save(); !r.OK { return r }
func (m *Manifest) Save() core.Result { // Value: nil
	result := core.JSONMarshal(m)
	if !result.OK {
		return result
	}
	if err := m.medium.Write(m.path, string(core.MustCast[[]byte](result))); err != nil {
		return core.Fail(core.E("Manifest.Save", "write manifest", err))
	}
	return core.Ok(nil)
}
