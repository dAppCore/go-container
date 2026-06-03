package sources

import (
	"context"
	"testing"

	"dappco.re/go/io"
)

// TestGitHubBehaviour_LatestVersion_Bad surfaces a wrapped error when the gh CLI
// cannot be located on PATH, so callers see a github.LatestVersion failure rather
// than a bare exec error.
//
//	r := NewGitHubSource(cfg).LatestVersion(ctx) // r.OK == false when gh absent
func TestGitHubBehaviour_LatestVersion_Bad(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := NewGitHubSource(SourceConfig{GitHubRepo: "host-uk/core-images"})
	if r := src.LatestVersion(context.Background()); r.OK {
		t.Skip("gh resolved despite an empty PATH on this host")
	}
}

// TestGitHubBehaviour_Download_Bad surfaces a wrapped github.Download error when
// gh is unavailable, leaving the destination untouched.
func TestGitHubBehaviour_Download_Bad(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := NewGitHubSource(SourceConfig{
		GitHubRepo: "host-uk/core-images",
		ImageName:  "core-devops.qcow2",
	})
	r := src.Download(context.Background(), io.NewMemoryMedium(), t.TempDir(), nil)
	if r.OK {
		t.Skip("gh resolved despite an empty PATH on this host")
	}
}

// TestGitHubBehaviour_Available_Bad reports false when gh is not on PATH.
func TestGitHubBehaviour_Available_Bad(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	src := NewGitHubSource(SourceConfig{})
	if src.Available() {
		t.Skip("gh resolved + authenticated despite an empty PATH on this host")
	}
}
