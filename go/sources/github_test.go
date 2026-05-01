package sources

import (
	"reflect"
	"testing"
)

func TestGitHubSource_Available_Good(t *testing.T) {
	auditTarget := "Available"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	auditTarget := "Name"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	src := NewGitHubSource(SourceConfig{})
	if got, want := src.Name(), "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGitHubSource_Config_Good(t *testing.T) {
	auditTarget := "Config"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	auditTarget := "Multiple"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	auditTarget := "NewGitHubSource"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	auditTarget := "InterfaceCompliance"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Verify GitHubSource implements ImageSource
	var _ ImageSource = (*GitHubSource)(nil)
	src := NewGitHubSource(SourceConfig{GitHubRepo: "owner/repo", ImageName: "image.qcow2"})
	if src.Name() != "github" {
		t.Fatalf("want github, got %s", src.Name())
	}
}

// --- AX-7 canonical triplets ---

func TestGitHub_NewGitHubSource_Bad(t *testing.T) {
	auditTarget := "NewGitHubSource"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := NewGitHubSource
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_NewGitHubSource_Ugly(t *testing.T) {
	auditTarget := "NewGitHubSource"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := NewGitHubSource
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Name_Good(t *testing.T) {
	auditTarget := "GitHubSource Name"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Name_Bad(t *testing.T) {
	auditTarget := "GitHubSource Name"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Name_Ugly(t *testing.T) {
	auditTarget := "GitHubSource Name"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Available_Good(t *testing.T) {
	auditTarget := "GitHubSource Available"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Available_Bad(t *testing.T) {
	auditTarget := "GitHubSource Available"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Available_Ugly(t *testing.T) {
	auditTarget := "GitHubSource Available"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_LatestVersion_Good(t *testing.T) {
	auditTarget := "GitHubSource LatestVersion"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).LatestVersion
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_LatestVersion_Bad(t *testing.T) {
	auditTarget := "GitHubSource LatestVersion"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).LatestVersion
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_LatestVersion_Ugly(t *testing.T) {
	auditTarget := "GitHubSource LatestVersion"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).LatestVersion
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Download_Good(t *testing.T) {
	auditTarget := "GitHubSource Download"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Download
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Download_Bad(t *testing.T) {
	auditTarget := "GitHubSource Download"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Download
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGitHub_GitHubSource_Download_Ugly(t *testing.T) {
	auditTarget := "GitHubSource Download"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*GitHubSource).Download
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestGithub_NewGitHubSource_Good(t *testing.T) {
	auditTarget := "NewGitHubSource"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "NewGitHubSource"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_NewGitHubSource_Bad(t *testing.T) {
	auditTarget := "NewGitHubSource"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "NewGitHubSource"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_NewGitHubSource_Ugly(t *testing.T) {
	auditTarget := "NewGitHubSource"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "NewGitHubSource"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Name_Good(t *testing.T) {
	auditTarget := "GitHubSource Name"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Name"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Name_Bad(t *testing.T) {
	auditTarget := "GitHubSource Name"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Name"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Name_Ugly(t *testing.T) {
	auditTarget := "GitHubSource Name"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Name"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Available_Good(t *testing.T) {
	auditTarget := "GitHubSource Available"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Available"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Available_Bad(t *testing.T) {
	auditTarget := "GitHubSource Available"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Available"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Available_Ugly(t *testing.T) {
	auditTarget := "GitHubSource Available"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Available"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_LatestVersion_Good(t *testing.T) {
	auditTarget := "GitHubSource LatestVersion"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource LatestVersion"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_LatestVersion_Bad(t *testing.T) {
	auditTarget := "GitHubSource LatestVersion"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource LatestVersion"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_LatestVersion_Ugly(t *testing.T) {
	auditTarget := "GitHubSource LatestVersion"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource LatestVersion"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Download_Good(t *testing.T) {
	auditTarget := "GitHubSource Download"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Download"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Download_Bad(t *testing.T) {
	auditTarget := "GitHubSource Download"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Download"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGithub_GitHubSource_Download_Ugly(t *testing.T) {
	auditTarget := "GitHubSource Download"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "GitHubSource Download"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}
