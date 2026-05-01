package sources

import (
	"context"
	goio "io"
	"net/http"

	core "dappco.re/go"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"

	"dappco.re/go/container/internal/coreutil"
)

// CDNSource downloads images from a CDN or S3 bucket.
type CDNSource struct {
	config SourceConfig
}

// Compile-time interface check.
var _ ImageSource = (*CDNSource)(nil)

// NewCDNSource creates a new CDN source.
//
// Usage:
//
//	src := NewCDNSource(cfg)
func NewCDNSource(cfg SourceConfig) *CDNSource {
	return &CDNSource{config: cfg}
}

// Name returns "cdn".
func (s *CDNSource) Name() string {
	return "cdn"
}

// Available checks if CDN URL is configured.
func (s *CDNSource) Available() bool {
	return s.config.CDNURL != ""
}

// LatestVersion fetches version from manifest or returns "latest".
func (s *CDNSource) LatestVersion(ctx context.Context) (
	string,
	error,
) {
	// Try to fetch manifest.json for version info
	url := core.Sprintf("%s/manifest.json", s.config.CDNURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "latest", nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "latest", nil
	}
	defer func() { _ = resp.Body.Close() }()

	// For now, just return latest - could parse manifest for version
	return "latest", nil
}

// Download downloads the image from CDN.
func (s *CDNSource) Download(ctx context.Context, m io.Medium, dest string, progress func(downloaded, total int64)) (
	err error, // result
) {
	url := core.Sprintf("%s/%s", s.config.CDNURL, s.config.ImageName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return coreerr.E("cdn.Download", "create request", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return coreerr.E("cdn.Download", "execute request", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return coreerr.E("cdn.Download", core.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	// Ensure dest directory exists
	if err := m.EnsureDir(dest); err != nil {
		return coreerr.E("cdn.Download", "ensure destination directory", err)
	}

	// Create destination file
	destPath := coreutil.JoinPath(dest, s.config.ImageName)
	f, err := m.Create(destPath)
	if err != nil {
		return coreerr.E("cdn.Download", "create destination file", err)
	}
	defer func() { _ = f.Close() }()

	// Copy with progress
	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return coreerr.E("cdn.Download", "write to file", werr)
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if err == goio.EOF {
			break
		}
		if err != nil {
			return coreerr.E("cdn.Download", "read response body", err)
		}
	}

	return nil
}
