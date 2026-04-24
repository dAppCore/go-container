package sources

import (
	"reflect"
	"testing"
)

func TestGitHubSource_Available_Good(t *testing.T) {
	src := NewGitHubSource(SourceConfig{
		GitHubRepo: "host-uk/core-images",
		ImageName:  "core-devops-darwin-arm64.qcow2",
	})

	if src.Name() != "github" {
		t.Errorf("expected name 'github', got %q", src.Name())
	}

	// Available depends on gh CLI being installed
	_ = src.Available()
}

func TestGitHubSource_Name_Good(t *testing.T) {
	src := NewGitHubSource(SourceConfig{})
	if got, want := src.Name(), "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGitHubSource_Config_Good(t *testing.T) {
	cfg := SourceConfig{
		GitHubRepo: "owner/repo",
		ImageName:  "test-image.qcow2",
	}
	src := NewGitHubSource(cfg)

	// Verify the config is stored
	if got, want := src.config.GitHubRepo, "owner/repo"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src.config.ImageName, "test-image.qcow2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGitHubSource_Multiple_Good(t *testing.T) {
	// Test creating multiple sources with different configs
	src1 := NewGitHubSource(SourceConfig{GitHubRepo: "org1/repo1", ImageName: "img1.qcow2"})
	src2 := NewGitHubSource(SourceConfig{GitHubRepo: "org2/repo2", ImageName: "img2.qcow2"})
	if got, want := src1.config.GitHubRepo, "org1/repo1"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src2.config.GitHubRepo, "org2/repo2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src1.Name(), "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src2.Name(), "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGitHub_NewGitHubSource_Good(t *testing.T) {
	cfg := SourceConfig{
		GitHubRepo:    "host-uk/core-images",
		RegistryImage: "ghcr.io/host-uk/core-devops",
		CDNURL:        "https://cdn.example.com",
		ImageName:     "core-devops-darwin-arm64.qcow2",
	}

	src := NewGitHubSource(cfg)
	if src == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := src.Name(), "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := src.config.GitHubRepo, cfg.GitHubRepo; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGitHubSource_InterfaceCompliance_Good(t *testing.T) {
	// Verify GitHubSource implements ImageSource
	var _ ImageSource = (*GitHubSource)(nil)
}
