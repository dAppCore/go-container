// Package sources provides image download sources for container.
package sources

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/io"
)

// ImageSource defines the interface for downloading dev images.
type ImageSource interface {
	// Name returns the source identifier.
	Name() string
	// Available checks if this source can be used.
	Available() bool
	// LatestVersion returns the latest available version.
	//
	// Example: r := src.LatestVersion(ctx); version := core.MustCast[string](r)
	LatestVersion(ctx context.Context) core.Result // Value: string
	// Download downloads the image to the destination path.
	// Reports progress via the callback if provided.
	//
	// Example: if r := src.Download(ctx, io.Local, dest, nil); !r.OK { return r }
	Download(ctx context.Context, m io.Medium, dest string, progress func(downloaded, total int64)) core.Result // Value: nil
}

// SourceConfig holds configuration for a source.
type SourceConfig struct {
	// GitHub configuration
	GitHubRepo string
	// Registry configuration
	RegistryImage string
	// CDN configuration
	CDNURL string
	// Image name (e.g., core-devops-darwin-arm64.qcow2)
	ImageName string
}
