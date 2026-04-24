package sources

import (
	"context"

	core "dappco.re/go/core"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"

	"dappco.re/go/container/internal/proc"
)

// GitHubSource downloads images from GitHub Releases.
type GitHubSource struct {
	config SourceConfig
}

// Compile-time interface check.
var _ ImageSource = (*GitHubSource)(nil)

// NewGitHubSource creates a new GitHub source.
//
// Usage:
//
//	src := NewGitHubSource(cfg)
func NewGitHubSource(cfg SourceConfig) *GitHubSource {
	return &GitHubSource{config: cfg}
}

// Name returns "github".
func (s *GitHubSource) Name() string {
	return "github"
}

// Available checks if gh CLI is installed and authenticated.
func (s *GitHubSource) Available() bool {
	_, err := proc.LookPath("gh")
	if err != nil {
		return false
	}
	// Check if authenticated
	cmd := proc.NewCommand("gh", "auth", "status")
	return cmd.Run() == nil
}

// LatestVersion returns the latest release tag.
func (s *GitHubSource) LatestVersion(ctx context.Context) (string, error) {
	cmd := proc.NewCommandContext(ctx, "gh", "release", "view",
		"-R", s.config.GitHubRepo,
		"--json", "tagName",
		"-q", ".tagName",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", coreerr.E("github.LatestVersion", "failed", err)
	}
	return core.Trim(string(out)), nil
}

// Download downloads the image from the latest release.
func (s *GitHubSource) Download(ctx context.Context, m io.Medium, dest string, progress func(downloaded, total int64)) error {
	// Get release assets to find our image
	cmd := proc.NewCommandContext(ctx, "gh", "release", "download",
		"-R", s.config.GitHubRepo,
		"-p", s.config.ImageName,
		"-D", dest,
		"--clobber",
	)
	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr

	if err := cmd.Run(); err != nil {
		return coreerr.E("github.Download", "failed", err)
	}
	return nil
}
