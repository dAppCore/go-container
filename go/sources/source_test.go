package sources

import (
	"reflect"
	"testing"
)

func TestSourceConfig_Empty_Good(t *testing.T) {
	auditTarget := "Empty"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	cfg := SourceConfig{}
	if got := cfg.GitHubRepo; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got := cfg.RegistryImage; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got := cfg.CDNURL; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got := cfg.ImageName; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestSourceConfig_Complete_Good(t *testing.T) {
	auditTarget := "Complete"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	cfg := SourceConfig{
		GitHubRepo:    "owner/repo",
		RegistryImage: "ghcr.io/owner/image:v1",
		CDNURL:        "https://cdn.example.com/images",
		ImageName:     "my-image-darwin-arm64.qcow2",
	}
	if got, want := cfg.GitHubRepo, "owner/repo"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.RegistryImage, "ghcr.io/owner/image:v1"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.CDNURL, "https://cdn.example.com/images"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.ImageName, "my-image-darwin-arm64.qcow2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestImageSource_Interface_Good(t *testing.T) {
	auditTarget := "Interface"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Ensure both sources implement the interface
	var _ ImageSource = (*GitHubSource)(nil)
	var _ ImageSource = (*CDNSource)(nil)
	github := NewGitHubSource(SourceConfig{GitHubRepo: "owner/repo", ImageName: "image.qcow2"})
	cdn := NewCDNSource(SourceConfig{CDNURL: "https://cdn.example.com", ImageName: "image.qcow2"})
	if github.Name() == "" || cdn.Name() == "" {
		t.Fatal("expected non-empty source names")
	}
}
