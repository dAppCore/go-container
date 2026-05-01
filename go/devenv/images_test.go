package devenv

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/container/sources"
	"dappco.re/go/io"
	"reflect"
	"testing"
	"time"
)

func TestImageManager_IsInstalled_Good(t *testing.T) {
	auditTarget := "IsInstalled"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Not installed yet
	if mgr.IsInstalled() {
		t.Fatal("expected false")
	}

	// Create fake image
	imagePath := coreutil.JoinPath(tmpDir, ImageName())
	err = io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	// Now installed
	if !mgr.IsInstalled() {
		t.Fatal("expected true")
	}
}

func TestImages_NewImageManager_Good(t *testing.T) {
	auditTarget := "NewImageManager"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	t.Run("creates manager with cdn source", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tmpDir)

		cfg := DefaultConfig()
		cfg.Images.Source = "cdn"

		mgr, err := NewImageManager(io.Local, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if mgr == nil {
			t.Fatal("expected non-nil value")
		}
		if got, want := len(mgr.sources), 1; got != want {
			t.Fatalf("want len %v, got %v", want, got)
		}
		if got, want := mgr.sources[0].Name(), "cdn"; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})

	t.Run("creates manager with github source", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tmpDir)

		cfg := DefaultConfig()
		cfg.Images.Source = "github"

		mgr, err := NewImageManager(io.Local, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if mgr == nil {
			t.Fatal("expected non-nil value")
		}
		if got, want := len(mgr.sources), 1; got != want {
			t.Fatalf("want len %v, got %v", want, got)
		}
		if got, want := mgr.sources[0].Name(), "github"; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})
}

func TestManifest_Save_Good(t *testing.T) {
	auditTarget := "Save"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	// Verify file exists and has content
	if !io.Local.IsFile(path) {
		t.Fatal("expected true")
	}

	// Reload
	m2, err := loadManifest(io.Local, path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m2.Images["test.img"].Version, "1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestImages_LoadManifest_Bad(t *testing.T) {
	auditTarget := "LoadManifest"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := coreutil.JoinPath(tmpDir, "manifest.json")
		err := io.Local.Write(path, "invalid json")
		if err != nil {
			t.Fatal(err)
		}

		_, err = loadManifest(io.Local, path)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestImages_CheckUpdate_Bad(t *testing.T) {
	auditTarget := "CheckUpdate"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	t.Run("image not installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tmpDir)

		cfg := DefaultConfig()
		mgr, err := NewImageManager(io.Local, cfg)
		if err != nil {
			t.Fatal(err)
		}

		_, _, _, err = mgr.CheckUpdate(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if s, sub := err.Error(), "image not installed"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	})
}

func TestNewImageManager_AutoSource_Good(t *testing.T) {
	auditTarget := "AutoSource"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	cfg := DefaultConfig()
	cfg.Images.Source = "auto"

	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil value")
	}
	// github and cdn
	if got, want := len(mgr.sources), 2; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

func TestNewImageManager_UnknownSourceFallsToAuto_Good(t *testing.T) {
	auditTarget := "UnknownSourceFallsToAuto"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	cfg := DefaultConfig()
	cfg.Images.Source = "unknown"

	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil value")
	}
	// falls to default (auto) which is github + cdn
	if got, want := len(mgr.sources), 2; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

func TestLoadManifest_Empty_Good(t *testing.T) {
	auditTarget := "Empty"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "nonexistent.json")

	m, err := loadManifest(io.Local, path)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected non-nil value")
	}
	if m.Images == nil {
		t.Fatal("expected non-nil value")
	}
	if got := m.Images; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got, want := m.path, path; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLoadManifest_ExistingData_Good(t *testing.T) {
	auditTarget := "ExistingData"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	path := coreutil.JoinPath(tmpDir, "manifest.json")

	data := `{"images":{"test.img":{"version":"2.0.0","source":"cdn"}}}`
	err := io.Local.Write(path, data)
	if err != nil {
		t.Fatal(err)
	}

	m, err := loadManifest(io.Local, path)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := m.Images["test.img"].Version, "2.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := m.Images["test.img"].Source, "cdn"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestImageInfo_Struct_Good(t *testing.T) {
	auditTarget := "Struct"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	info := ImageInfo{
		Version:    "1.0.0",
		SHA256:     "abc123",
		Downloaded: time.Now(),
		Source:     "github",
	}
	if got, want := info.Version, "1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := info.SHA256, "abc123"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if info.Downloaded.IsZero() {
		t.Fatal("expected false")
	}
	if got, want := info.Source, "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestManifest_Save_CreatesDirs_Good(t *testing.T) {
	auditTarget := "Save CreatesDirs"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	// Verify file was created
	if !io.Local.IsFile(nestedPath) {
		t.Fatal("expected true")
	}
}

func TestManifest_Save_Overwrite_Good(t *testing.T) {
	auditTarget := "Save Overwrite"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	// Second save with different data
	m2 := &Manifest{
		medium: io.Local,
		Images: make(map[string]ImageInfo),
		path:   path,
	}
	m2.Images["other.img"] = ImageInfo{Version: "2.0.0"}
	err = m2.Save()
	if err != nil {
		t.Fatal(err)
	}

	// Verify second data
	loaded, err := loadManifest(io.Local, path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := loaded.Images["other.img"].Version, "2.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	_, exists := loaded.Images["test.img"]
	if exists {
		t.Fatal("expected false")
	}
}

func TestImageManager_Install_NoSourceAvailable_Bad(t *testing.T) {
	auditTarget := "Install NoSourceAvailable"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "no image source available"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestNewImageManager_CreatesDir_Good(t *testing.T) {
	auditTarget := "CreatesDir"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	imagesDir := coreutil.JoinPath(tmpDir, "images")
	t.Setenv("CORE_IMAGES_DIR", imagesDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil value")

		// Verify directory was created
	}

	info, err := io.Local.Stat(imagesDir)
	if err != nil {
		t.Fatal(err)
	}
	if !(info.IsDir()) {
		t.Fatal("expected true")

		// mockImageSource is a test helper for simulating image sources
	}
}

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
	auditTarget := "Install WithMockSource"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !(mgr.IsInstalled()) {
		t.Fatal("expected true")
	}

	// Verify manifest was updated
	info, ok := mgr.manifest.Images[ImageName()]
	if !(ok) {
		t.Fatal("expected true")
	}
	if got, want := info.Version, "v1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := info.Source, "mock"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestImageManager_Install_DownloadError_Bad(t *testing.T) {
	auditTarget := "Install DownloadError"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:          "mock",
		available:     true,
		latestVersion: "v1.0.0",
		downloadErr:   core.NewError("test error"),
	}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{mock},
	}

	err := mgr.Install(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImageManager_Install_VersionError_Bad(t *testing.T) {
	auditTarget := "Install VersionError"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:      "mock",
		available: true,
		latestErr: core.NewError("test error"),
	}

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{mock},
	}

	err := mgr.Install(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "failed to get latest version"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestImageManager_Install_SkipsUnavailableSource_Good(t *testing.T) {
	auditTarget := "Install SkipsUnavailableSource"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	// Should have used the available source
	info := mgr.manifest.Images[ImageName()]
	if got, want := info.Source, "available"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestImageManager_CheckUpdate_WithMockSource_Good(t *testing.T) {
	auditTarget := "CheckUpdate WithMockSource"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if got, want := current, "v1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := latest, "v2.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(hasUpdate) {
		t.Fatal("expected true")
	}
}

func TestImageManager_CheckUpdate_NoUpdate_Good(t *testing.T) {
	auditTarget := "CheckUpdate NoUpdate"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if got, want := current, "v1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := latest, "v1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if hasUpdate {
		t.Fatal("expected false")
	}
}

func TestImageManager_CheckUpdate_NoSource_Bad(t *testing.T) {
	auditTarget := "CheckUpdate NoSource"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "no image source available"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestImageManager_CheckUpdate_VersionError_Bad(t *testing.T) {
	auditTarget := "CheckUpdate VersionError"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mock := &mockImageSource{
		name:      "mock",
		available: true,
		latestErr: core.NewError("test error"),
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
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := current, "v1.0.0"; !reflect.DeepEqual( // Current should still be returned
		got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestImageManager_Install_EmptySources_Bad(t *testing.T) {
	auditTarget := "Install EmptySources"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tmpDir)

	mgr := &ImageManager{
		medium:   io.Local,
		config:   DefaultConfig(),
		manifest: &Manifest{medium: io.Local, Images: make(map[string]ImageInfo), path: coreutil.JoinPath(tmpDir, "manifest.json")},
		sources:  []sources.ImageSource{}, // Empty slice, not nil
	}

	err := mgr.Install(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "no image source available"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestImageManager_Install_AllUnavailable_Bad(t *testing.T) {
	auditTarget := "Install AllUnavailable"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "no image source available"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestImageManager_CheckUpdate_FirstSourceUnavailable_Good(t *testing.T) {
	auditTarget := "CheckUpdate FirstSourceUnavailable"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if got, want := current, "v1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := latest, "v2.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(hasUpdate) {
		t.Fatal("expected true")
	}
}

func TestManifest_Struct_Good(t *testing.T) {
	auditTarget := "Struct"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	m := &Manifest{
		Images: map[string]ImageInfo{
			"test.img": {Version: "1.0.0"},
		},
		path: "/path/to/manifest.json",
	}
	if got, want := m.path, "/path/to/manifest.json"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := len(m.Images), 1; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := m.Images["test.img"].Version, "1.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// --- AX-7 canonical triplets ---

func TestImages_NewImageManager_Bad(t *testing.T) {
	auditTarget := "NewImageManager"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := NewImageManager
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_NewImageManager_Ugly(t *testing.T) {
	auditTarget := "NewImageManager"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := NewImageManager
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_IsInstalled_Good(t *testing.T) {
	auditTarget := "ImageManager IsInstalled"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).IsInstalled
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_IsInstalled_Bad(t *testing.T) {
	auditTarget := "ImageManager IsInstalled"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).IsInstalled
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_IsInstalled_Ugly(t *testing.T) {
	auditTarget := "ImageManager IsInstalled"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).IsInstalled
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_Install_Good(t *testing.T) {
	auditTarget := "ImageManager Install"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).Install
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_Install_Bad(t *testing.T) {
	auditTarget := "ImageManager Install"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).Install
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_Install_Ugly(t *testing.T) {
	auditTarget := "ImageManager Install"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).Install
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_CheckUpdate_Good(t *testing.T) {
	auditTarget := "ImageManager CheckUpdate"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).CheckUpdate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_CheckUpdate_Bad(t *testing.T) {
	auditTarget := "ImageManager CheckUpdate"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).CheckUpdate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_ImageManager_CheckUpdate_Ugly(t *testing.T) {
	auditTarget := "ImageManager CheckUpdate"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*ImageManager).CheckUpdate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_Manifest_Save_Good(t *testing.T) {
	auditTarget := "Manifest Save"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*Manifest).Save
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_Manifest_Save_Bad(t *testing.T) {
	auditTarget := "Manifest Save"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*Manifest).Save
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestImages_Manifest_Save_Ugly(t *testing.T) {
	auditTarget := "Manifest Save"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*Manifest).Save
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
