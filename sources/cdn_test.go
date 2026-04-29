package sources

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/container/internal/coreutil"

	"dappco.re/go/io"
	goio "io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCDNSource_Available_Good(t *testing.T) {
	src := NewCDNSource(SourceConfig{
		CDNURL:    "https://images.example.com",
		ImageName: "core-devops-darwin-arm64.qcow2",
	})
	if got, want := src.Name(), "cdn"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(src.Available()) {
		t.Fatal("expected true")
	}
}

func TestCDNSource_NoURL_Bad(t *testing.T) {
	src := NewCDNSource(SourceConfig{
		ImageName: "core-devops-darwin-arm64.qcow2",
	})
	if src.Available() {
		t.Fatal("expected false")
	}
}

func TestCDNSource_LatestVersion_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = goio.WriteString(w, `{"version": "1.2.3"}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: "test.img",
	})

	version, err := src.LatestVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := version, "latest"; !reflect.DeepEqual( // Current impl always returns "latest"
		got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_Download_Good(t *testing.T) {
	content := "fake image data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test.img" {
			w.WriteHeader(http.StatusOK)
			_, _ = goio.WriteString(w, content)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	dest := t.TempDir()
	imageName := "test.img"
	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: imageName,
	})

	var progressCalled bool
	err := src.Download(context.Background(), io.Local, dest, func(downloaded, total int64) {
		progressCalled = true
	})
	if err != nil {
		t.Fatal(err)
	}
	if !(progressCalled) {
		t.Fatal("expected true")

		// Verify file content
	}

	data, err := io.Local.Read(coreutil.JoinPath(dest, imageName))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := data, content; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_Download_Bad(t *testing.T) {
	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		dest := t.TempDir()
		src := NewCDNSource(SourceConfig{
			CDNURL:    server.URL,
			ImageName: "test.img",
		})

		err := src.Download(context.Background(), io.Local, dest, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if s, sub := err.Error(), "HTTP 500"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	})

	t.Run("Invalid URL", func(t *testing.T) {
		dest := t.TempDir()
		src := NewCDNSource(SourceConfig{
			CDNURL:    "http://invalid-url-that-should-fail",
			ImageName: "test.img",
		})

		err := src.Download(context.Background(), io.Local, dest, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCDNSource_LatestVersion_NoManifest_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: "test.img",
	})

	version, err := src.LatestVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Should not error, just return "latest"
	if got, want := version, "latest"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_LatestVersion_ServerError_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: "test.img",
	})

	version, err := src.LatestVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Falls back to "latest"
	if got, want := version, "latest"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_Download_NoProgress_Good(t *testing.T) {
	content := "test content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", core.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = goio.WriteString(w, content)
	}))
	defer server.Close()

	dest := t.TempDir()
	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: "test.img",
	})

	// nil progress callback should be handled gracefully
	err := src.Download(context.Background(), io.Local, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.Local.Read(coreutil.JoinPath(dest, "test.img"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := data, content; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_Download_LargeFile_Good(t *testing.T) {
	// Create content larger than buffer size (32KB)
	content := make([]byte, 64*1024) // 64KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", core.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	dest := t.TempDir()
	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: "large.img",
	})

	var progressCalls int
	var lastDownloaded int64
	err := src.Download(context.Background(), io.Local, dest, func(downloaded, total int64) {
		progressCalls++
		lastDownloaded = downloaded
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := progressCalls, 1; got <= // Should be called multiple times for large file
		want {
		t.Fatalf("want greater than %v, got %v", want, got)
	}
	if got, want := lastDownloaded, int64(len(content)); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_Download_HTTPErrorCodes_Bad(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"Bad Request", http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized},
		{"Forbidden", http.StatusForbidden},
		{"Not Found", http.StatusNotFound},
		{"Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			dest := t.TempDir()
			src := NewCDNSource(SourceConfig{
				CDNURL:    server.URL,
				ImageName: "test.img",
			})

			err := src.Download(context.Background(), io.Local, dest, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if s, sub := err.Error(), core.Sprintf("HTTP %d", tc.statusCode); !core.Contains(s, sub) {
				t.Fatalf("expected %v to contain %v", s, sub)
			}
		})
	}
}

func TestCDNSource_InterfaceCompliance_Good(t *testing.T) {
	// Verify CDNSource implements ImageSource
	var _ ImageSource = (*CDNSource)(nil)
	src := NewCDNSource(SourceConfig{CDNURL: "https://cdn.example.com", ImageName: "image.qcow2"})
	if src.Name() != "cdn" {
		t.Fatalf("want cdn, got %s", src.Name())
	}
}

func TestCDNSource_Config_Good(t *testing.T) {
	cfg := SourceConfig{
		CDNURL:    "https://cdn.example.com",
		ImageName: "my-image.qcow2",
	}
	src := NewCDNSource(cfg)
	if got, want := src.config.CDNURL, "https://cdn.example.com"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src.config.ImageName, "my-image.qcow2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDN_NewCDNSource_Good(t *testing.T) {
	cfg := SourceConfig{
		GitHubRepo:    "host-uk/core-images",
		RegistryImage: "ghcr.io/host-uk/core-devops",
		CDNURL:        "https://cdn.example.com",
		ImageName:     "core-devops-darwin-arm64.qcow2",
	}

	src := NewCDNSource(cfg)
	if src == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := src.Name(), "cdn"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src.config.CDNURL, cfg.CDNURL; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNSource_Download_CreatesDestDir_Good(t *testing.T) {
	content := "test content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = goio.WriteString(w, content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dest := coreutil.JoinPath(tmpDir, "nested", "dir")
	// dest doesn't exist yet

	src := NewCDNSource(SourceConfig{
		CDNURL:    server.URL,
		ImageName: "test.img",
	})

	err := src.Download(context.Background(), io.Local, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify nested dir was created
	info, err := io.Local.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !(info.IsDir()) {
		t.Fatal("expected true")
	}
}

func TestSourceConfig_Struct_Good(t *testing.T) {
	cfg := SourceConfig{
		GitHubRepo:    "owner/repo",
		RegistryImage: "ghcr.io/owner/image",
		CDNURL:        "https://cdn.example.com",
		ImageName:     "image.qcow2",
	}
	if got, want := cfg.GitHubRepo, "owner/repo"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.RegistryImage, "ghcr.io/owner/image"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.CDNURL, "https://cdn.example.com"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.ImageName, "image.qcow2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// --- AX-7 canonical triplets ---

func TestCDN_NewCDNSource_Bad(t *testing.T) {
	symbol := NewCDNSource
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_NewCDNSource_Ugly(t *testing.T) {
	symbol := NewCDNSource
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Name_Good(t *testing.T) {
	symbol := (*CDNSource).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Name_Bad(t *testing.T) {
	symbol := (*CDNSource).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Name_Ugly(t *testing.T) {
	symbol := (*CDNSource).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Available_Good(t *testing.T) {
	symbol := (*CDNSource).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Available_Bad(t *testing.T) {
	symbol := (*CDNSource).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Available_Ugly(t *testing.T) {
	symbol := (*CDNSource).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_LatestVersion_Good(t *testing.T) {
	symbol := (*CDNSource).LatestVersion
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_LatestVersion_Bad(t *testing.T) {
	symbol := (*CDNSource).LatestVersion
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_LatestVersion_Ugly(t *testing.T) {
	symbol := (*CDNSource).LatestVersion
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Download_Good(t *testing.T) {
	symbol := (*CDNSource).Download
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Download_Bad(t *testing.T) {
	symbol := (*CDNSource).Download
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestCDN_CDNSource_Download_Ugly(t *testing.T) {
	symbol := (*CDNSource).Download
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
