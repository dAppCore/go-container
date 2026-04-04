package devenv

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/container/sources"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageManager_IsInstalled_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	// Not installed yet
	assert.False(t, mgr.IsInstalled())

	// Create fake image
	imagePath := coreutil.JoinPath(tmpDir, ImageName())
	err = io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	// Now installed
	assert.True(t, mgr.IsInstalled())
}

func TestImages_NewImageManager_Good(t *testing.T) {
	t.Run("creates manager with cdn source", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tmpDir)

		cfg := DefaultConfig()
		cfg.Images.Source = "cdn"

		mgr, err := NewImageManager(io.Local, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
		assert.Len(t, mgr.sources, 1)
		assert.Equal(t, "cdn", mgr.sources[0].Name())
	})

	t.Run("creates manager with github source", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tmpDir)

		cfg := DefaultConfig()
		cfg.Images.Source = "github"

		mgr, err := NewImageManager(io.Local, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
		assert.Len(t, mgr.sources, 1)
		assert.Equal(t, "github", mgr.sources[0].Name())
	})
}

func TestManifest_Save_Good(t *testing.T) {
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "manifest.json")

	m := &Manifest{
		medium: io.Local,
		Images: make(map[string]ImageInfo),
		path:   path,
	}

	m.Images["test.img"] = ImageInfo{
		Version: "1.0.0",
		Source:  "test",
	}

	err := m.Save()
	assert.NoError(t, err)

	// Verify file exists and has content
	assert.True(t, io.Local.IsFile(path))

	// Reload
	m2, err := loadManifest(io.Local, path)
	assert.NoError(t, err)
	assert.Equal(t, "1.0.0", m2.Images["test.img"].Version)
}

func TestImages_LoadManifest_Bad(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := coreutil.JoinPath(tmpDir, "manifest.json")
		err := io.Local.Write(path, "invalid json")
		require.NoError(t, err)

		_, err = loadManifest(io.Local, path)
		assert.Error(t, err)
	})
}

func TestImages_CheckUpdate_Bad(t *testing.T) {
	t.Run("image not installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tmpDir)

		cfg := DefaultConfig()
		mgr, err := NewImageManager(io.Local, cfg)
		require.NoError(t, err)

		_, _, _, err = mgr.CheckUpdate(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image not installed")
	})
}

func TestNewImageManager_AutoSource_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	cfg := DefaultConfig()
	cfg.Images.Source = "auto"

	mgr, err := NewImageManager(io.Local, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.Len(t, mgr.sources, 2) // github and cdn
}

func TestNewImageManager_UnknownSourceFallsToAuto_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	cfg := DefaultConfig()
	cfg.Images.Source = "unknown"

	mgr, err := NewImageManager(io.Local, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.Len(t, mgr.sources, 2) // falls to default (auto) which is github + cdn
}

func TestLoadManifest_Empty_Good(t *testing.T) {
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "nonexistent.json")

	m, err := loadManifest(io.Local, path)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.NotNil(t, m.Images)
	assert.Empty(t, m.Images)
	assert.Equal(t, path, m.path)
}

func TestLoadManifest_ExistingData_Good(t *testing.T) {
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "manifest.json")

	data := `{"images":{"test.img":{"version":"2.0.0","source":"cdn"}}}`
	err := io.Local.Write(path, data)
	require.NoError(t, err)

	m, err := loadManifest(io.Local, path)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, "2.0.0", m.Images["test.img"].Version)
	assert.Equal(t, "cdn", m.Images["test.img"].Source)
}

func TestImageInfo_Struct_Good(t *testing.T) {
	info := ImageInfo{
		Version:    "1.0.0",
		SHA256:     "abc123",
		Downloaded: time.Now(),
		Source:     "github",
	}
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "abc123", info.SHA256)
	assert.False(t, info.Downloaded.IsZero())
	assert.Equal(t, "github", info.Source)
}

func TestManifest_Save_CreatesDirs_Good(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := coreutil.JoinPath(tmpDir, "nested", "dir", "manifest.json")

	m := &Manifest{
		medium: io.Local,
		Images: make(map[string]ImageInfo),
		path:   nestedPath,
	}
	m.Images["test.img"] = ImageInfo{Version: "1.0.0"}

	// Save creates parent directories automatically via io.Local.Write
	err := m.Save()
	assert.NoError(t, err)

	// Verify file was created
	assert.True(t, io.Local.IsFile(nestedPath))
}

func TestManifest_Save_Overwrite_Good(t *testing.T) {
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "manifest.json")

	// First save
	m1 := &Manifest{
		medium: io.Local,
		Images: make(map[string]ImageInfo),
		path:   path,
	}
	m1.Images["test.img"] = ImageInfo{Version: "1.0.0"}
	err := m1.Save()
	require.NoError(t, err)

	// Second save with different data
	m2 := &Manifest{
		medium: io.Local,
		Images: make(map[string]ImageInfo),
		path:   path,
	}
	m2.Images["other.img"] = ImageInfo{Version: "2.0.0"}
	err = m2.Save()
	require.NoError(t, err)

	// Verify second data
	loaded, err := loadManifest(io.Local, path)
	assert.NoError(t, err)
	assert.Equal(t, "2.0.0", loaded.Images["other.img"].Version)
	_, exists := loaded.Images["test.img"]
	assert.False(t, exists)
}

func TestImageManager_Install_NoSourceAvailable_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	// Create manager with empty sources
	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  nil, // no sources
	}

	err := mgr.Install(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image source available")
}

func TestNewImageManager_CreatesDir_Good(t *testing.T) {
	tmpDir := t.TempDir()
	imagesDir := coreutil.JoinPath(tmpDir, "images")
	t.Setenv("CORE_IMAGES_DIR", imagesDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mgr)

	// Verify directory was created
	info, err := io.Local.Stat(imagesDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// mockImageSource is a test helper for simulating image sources
type mockImageSource struct {
	name          string
	available     bool
	latestVersion string
	latestErr     error
	downloadErr   error
}

func (m *mockImageSource) Name() string    { return m.name }
func (m *mockImageSource) Available() bool { return m.available }
func (m *mockImageSource) LatestVersion(ctx context.Context) (string, error) {
	return m.latestVersion, m.latestErr
}
func (m *mockImageSource) Download(ctx context.Context, medium io.Medium, dest string, progress func(downloaded, total int64)) error {
	if m.downloadErr != nil {
		return m.downloadErr
	}
	// Create a fake image file
	imagePath := coreutil.JoinPath(dest, ImageName())
	return medium.Write(imagePath, "mock image content")
}

func TestImageManager_Install_WithMockSource_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:          "mock",
		available:     true,
		latestVersion: "v1.0.0",
	}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{mock},
	}

	err := mgr.Install(context.Background(), nil)
	assert.NoError(t, err)
	assert.True(t, mgr.IsInstalled())

	// Verify manifest was updated
	info, ok := mgr.manifest.Images[ImageName()]
	assert.True(t, ok)
	assert.Equal(t, "v1.0.0", info.Version)
	assert.Equal(t, "mock", info.Source)
}

func TestImageManager_Install_DownloadError_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:          "mock",
		available:     true,
		latestVersion: "v1.0.0",
		downloadErr:   assert.AnError,
	}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{mock},
	}

	err := mgr.Install(context.Background(), nil)
	assert.Error(t, err)
}

func TestImageManager_Install_VersionError_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:      "mock",
		available: true,
		latestErr: assert.AnError,
	}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{mock},
	}

	err := mgr.Install(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get latest version")
}

func TestImageManager_Install_SkipsUnavailableSource_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	unavailableMock := &mockImageSource{
		name:      "unavailable",
		available: false,
	}
	availableMock := &mockImageSource{
		name:          "available",
		available:     true,
		latestVersion: "v2.0.0",
	}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{unavailableMock, availableMock},
	}

	err := mgr.Install(context.Background(), nil)
	assert.NoError(t, err)

	// Should have used the available source
	info := mgr.manifest.Images[ImageName()]
	assert.Equal(t, "available", info.Source)
}

func TestImageManager_CheckUpdate_WithMockSource_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:          "mock",
		available:     true,
		latestVersion: "v2.0.0",
	}

	mgr := &ImageManager{
		medium: io.Local,
		config: DefaultConfig(),
		manifest: &Manifest{
			medium: io.Local,
			Images: map[string]ImageInfo{
				ImageName(): {Version: "v1.0.0", Source: "mock"},
			},
			path: coreutil.JoinPath(tmpDir, "manifest.json"),
		},
		sources: []sources.ImageSource{mock},
	}

	current, latest, hasUpdate, err := mgr.CheckUpdate(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "v1.0.0", current)
	assert.Equal(t, "v2.0.0", latest)
	assert.True(t, hasUpdate)
}

func TestImageManager_CheckUpdate_NoUpdate_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:          "mock",
		available:     true,
		latestVersion: "v1.0.0",
	}

	mgr := &ImageManager{
		medium: io.Local,
		config: DefaultConfig(),
		manifest: &Manifest{
			medium: io.Local,
			Images: map[string]ImageInfo{
				ImageName(): {Version: "v1.0.0", Source: "mock"},
			},
			path: coreutil.JoinPath(tmpDir, "manifest.json"),
		},
		sources: []sources.ImageSource{mock},
	}

	current, latest, hasUpdate, err := mgr.CheckUpdate(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "v1.0.0", current)
	assert.Equal(t, "v1.0.0", latest)
	assert.False(t, hasUpdate)
}

func TestImageManager_CheckUpdate_NoSource_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	unavailableMock := &mockImageSource{
		name:      "mock",
		available: false,
	}

	mgr := &ImageManager{
		medium: io.Local,
		config: DefaultConfig(),
		manifest: &Manifest{
			medium: io.Local,
			Images: map[string]ImageInfo{
				ImageName(): {Version: "v1.0.0", Source: "mock"},
			},
			path: coreutil.JoinPath(tmpDir, "manifest.json"),
		},
		sources: []sources.ImageSource{unavailableMock},
	}

	_, _, _, err := mgr.CheckUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image source available")
}

func TestImageManager_CheckUpdate_VersionError_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:      "mock",
		available: true,
		latestErr: assert.AnError,
	}

	mgr := &ImageManager{
		medium: io.Local,
		config: DefaultConfig(),
		manifest: &Manifest{
			medium: io.Local,
			Images: map[string]ImageInfo{
				ImageName(): {Version: "v1.0.0", Source: "mock"},
			},
			path: coreutil.JoinPath(tmpDir, "manifest.json"),
		},
		sources: []sources.ImageSource{mock},
	}

	current, _, _, err := mgr.CheckUpdate(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "v1.0.0", current) // Current should still be returned
}

func TestImageManager_Install_EmptySources_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{}, // Empty slice, not nil
	}

	err := mgr.Install(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image source available")
}

func TestImageManager_Install_AllUnavailable_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock1 := &mockImageSource{name: "mock1", available: false}
	mock2 := &mockImageSource{name: "mock2", available: false}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{mock1, mock2},
	}

	err := mgr.Install(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no image source available")
}

func TestImageManager_CheckUpdate_FirstSourceUnavailable_Good(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	unavailable := &mockImageSource{name: "unavailable", available: false}
	available := &mockImageSource{name: "available", available: true, latestVersion: "v2.0.0"}

	mgr := &ImageManager{
		medium: io.Local,
		config: DefaultConfig(),
		manifest: &Manifest{
			medium: io.Local,
			Images: map[string]ImageInfo{
				ImageName(): {Version: "v1.0.0", Source: "available"},
			},
			path: coreutil.JoinPath(tmpDir, "manifest.json"),
		},
		sources: []sources.ImageSource{unavailable, available},
	}

	current, latest, hasUpdate, err := mgr.CheckUpdate(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "v1.0.0", current)
	assert.Equal(t, "v2.0.0", latest)
	assert.True(t, hasUpdate)
}

func TestManifest_Struct_Good(t *testing.T) {
	m := &Manifest{
		Images: map[string]ImageInfo{
			"test.img": {Version: "1.0.0"},
		},
		path: "/path/to/manifest.json",
	}
	assert.Equal(t, "/path/to/manifest.json", m.path)
	assert.Len(t, m.Images, 1)
	assert.Equal(t, "1.0.0", m.Images["test.img"].Version)
}
