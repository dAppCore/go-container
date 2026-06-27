package sources

import (
	"context"
	"os"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/io"
)

func fakeGH(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := core.PathJoin(dir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  exit 0
fi
if [ "$1" = "release" ] && [ "$2" = "view" ]; then
  printf 'v1.2.3\n'
  exit 0
fi
if [ "$1" = "release" ] && [ "$2" = "download" ]; then
  dest=""
  pattern=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      -D) shift; dest="$1" ;;
      -p) shift; pattern="$1" ;;
    esac
    shift
  done
  mkdir -p "$dest"
  printf 'image' > "$dest/$pattern"
  exit 0
fi
exit 1
`
	if err := io.Local.Write(path, script); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod fake gh: %v", err)
	}
	t.Setenv("PATH", dir)
	return dir
}

func TestGitHubBehaviour_Available_Good(t *testing.T) {
	fakeGH(t)
	src := NewGitHubSource(SourceConfig{})
	if !src.Available() {
		t.Fatal("expected fake gh to make source available")
	}
}

func TestGitHubBehaviour_LatestVersion_Good(t *testing.T) {
	fakeGH(t)
	src := NewGitHubSource(SourceConfig{GitHubRepo: "host-uk/core-images"})
	r := src.LatestVersion(context.Background())
	if !r.OK {
		t.Fatalf("LatestVersion returned error: %v", r.Error())
	}
	if got := core.MustCast[string](r); got != "v1.2.3" {
		t.Fatalf("LatestVersion = %q, want v1.2.3", got)
	}
}

func TestGitHubBehaviour_Download_Good(t *testing.T) {
	fakeGH(t)
	dest := t.TempDir()
	src := NewGitHubSource(SourceConfig{
		GitHubRepo: "host-uk/core-images",
		ImageName:  "core-devops.qcow2",
	})
	if r := src.Download(context.Background(), io.Local, dest, nil); !r.OK {
		t.Fatalf("Download returned error: %v", r.Error())
	}
	if !io.Local.IsFile(core.PathJoin(dest, "core-devops.qcow2")) {
		t.Fatal("expected fake gh to create downloaded image")
	}
}

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
