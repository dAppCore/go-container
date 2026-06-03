// Package devenv provides a portable development environment using LinuxKit images.
package devenv

import (
	"context"
	"runtime"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/io"

	"dappco.re/go/container/internal/coreutil"
)

const (
	// DefaultSSHPort is the default port for SSH connections to the dev environment.
	DefaultSSHPort = 2222
)

// DevOps manages the portable development environment.
type DevOps struct {
	medium    io.Medium
	config    *Config
	images    *ImageManager
	container *container.LinuxKitManager
}

// New creates a new DevOps instance using the provided medium.
//
// Usage:
//
//	dev := core.MustCast[*DevOps](New(io.Local))
func New(m io.Medium) core.Result { // Value: *DevOps
	cfgRes := LoadConfig(m)
	if !cfgRes.OK {
		return core.Fail(core.E("devops.New", "failed to load config", cfgRes.Value.(error)))
	}
	cfg := core.MustCast[*Config](cfgRes)

	imagesRes := NewImageManager(m, cfg)
	if !imagesRes.OK {
		return core.Fail(core.E("devops.New", "failed to create image manager", imagesRes.Value.(error)))
	}
	images := core.MustCast[*ImageManager](imagesRes)

	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return core.Fail(core.E("devops.New", "failed to create container manager", mgrRes.Value.(error)))
	}
	mgr := core.MustCast[*container.LinuxKitManager](mgrRes)

	return core.Ok(&DevOps{
		medium:    m,
		config:    cfg,
		images:    images,
		container: mgr,
	})
}

// ImageName returns the platform-specific image name.
//
// Usage:
//
//	name := ImageName()
func ImageName() string {
	return core.Sprintf("core-devops-%s-%s.qcow2", runtime.GOOS, runtime.GOARCH)
}

// ImagesDir returns the path to the images directory.
//
// Usage:
//
//	dir := core.MustCast[string](ImagesDir())
func ImagesDir() core.Result { // Value: string
	if dir := core.Env("CORE_IMAGES_DIR"); dir != "" {
		return core.Ok(dir)
	}
	home := coreutil.HomeDir()
	if home == "" {
		return core.Fail(core.E("ImagesDir", "home directory not available", nil))
	}
	return core.Ok(coreutil.JoinPath(home, ".core", "images"))
}

// ImagePath returns the full path to the platform-specific image.
//
// Usage:
//
//	path := core.MustCast[string](ImagePath())
func ImagePath() core.Result { // Value: string
	dirRes := ImagesDir()
	if !dirRes.OK {
		return dirRes
	}
	return core.Ok(coreutil.JoinPath(core.MustCast[string](dirRes), ImageName()))
}

// IsInstalled checks if the dev image is installed.
func (d *DevOps) IsInstalled() bool {
	pathRes := ImagePath()
	if !pathRes.OK {
		return false
	}
	return d.medium.IsFile(core.MustCast[string](pathRes))
}

// Install downloads and installs the dev image.
//
// Usage:
//
//	if r := dev.Install(ctx, nil); !r.OK { return r }
func (d *DevOps) Install(ctx context.Context, progress func(downloaded, total int64)) core.Result { // Value: nil
	return d.images.Install(ctx, progress)
}

// CheckUpdate checks if an update is available.
//
// Usage:
//
//	info := core.MustCast[*UpdateInfo](dev.CheckUpdate(ctx))
func (d *DevOps) CheckUpdate(ctx context.Context) core.Result { // Value: *UpdateInfo
	return d.images.CheckUpdate(ctx)
}

// BootOptions configures how to boot the dev environment.
type BootOptions struct {
	Memory int    // MB, default 4096
	CPUs   int    // default 2
	Name   string // container name
	Fresh  bool   // destroy existing and start fresh
}

// DefaultBootOptions returns sensible defaults.
//
// Usage:
//
//	opts := DefaultBootOptions()
func DefaultBootOptions() BootOptions {
	return BootOptions{
		Memory: 4096,
		CPUs:   2,
		Name:   "core-dev",
	}
}

// Boot starts the dev environment.
//
// Usage:
//
//	if r := dev.Boot(ctx, devenv.DefaultBootOptions()); !r.OK { return r }
func (d *DevOps) Boot(ctx context.Context, opts BootOptions) core.Result { // Value: nil
	if !d.images.IsInstalled() {
		return core.Fail(core.E("DevOps.Boot", "dev image not installed (run 'core dev install' first)", nil))
	}

	// Check if already running
	if !opts.Fresh {
		runningRes := d.IsRunning(ctx)
		if runningRes.OK && core.MustCast[bool](runningRes) {
			return core.Fail(core.E("DevOps.Boot", "dev environment already running (use 'core dev stop' first or --fresh)", nil))
		}
	}

	// Stop existing if fresh
	if opts.Fresh {
		if r := d.Stop(ctx); !r.OK {
			// Fresh boot should continue when there is no existing container to stop.
		}
	}

	pathRes := ImagePath()
	if !pathRes.OK {
		return pathRes
	}
	imagePath := core.MustCast[string](pathRes)

	// Build run options for LinuxKitManager
	runOpts := container.RunOptions{
		Name:    opts.Name,
		Memory:  opts.Memory,
		CPUs:    opts.CPUs,
		SSHPort: DefaultSSHPort,
		Detach:  true,
	}

	if r := d.container.Run(ctx, imagePath, runOpts); !r.OK {
		return r
	}

	// Wait for SSH to be ready and scan host key
	// We try for up to 60 seconds as the VM takes a moment to boot
	var lastErr error
	for range 30 {
		select {
		case <-ctx.Done():
			return core.Fail(core.E("DevOps.Boot", "context cancelled", ctx.Err()))
		case <-time.After(2 * time.Second):
			if r := ensureHostKey(ctx, runOpts.SSHPort); r.OK {
				return core.Ok(nil)
			} else {
				lastErr = r.Value.(error)
			}
		}
	}

	return core.Fail(core.E("DevOps.Boot", "failed to verify host key after boot", lastErr))
}

// Stop stops the dev environment.
//
// Usage:
//
//	if r := dev.Stop(ctx); !r.OK { return r }
func (d *DevOps) Stop(ctx context.Context) core.Result { // Value: nil
	findRes := d.findContainer(ctx, "core-dev")
	if !findRes.OK {
		return findRes
	}
	c := core.MustCast[*container.Container](findRes)
	if c == nil {
		return core.Fail(core.E("DevOps.Stop", "dev environment not found", nil))
	}
	return d.container.Stop(ctx, c.ID)
}

// IsRunning checks if the dev environment is running.
//
// Usage:
//
//	running := core.MustCast[bool](dev.IsRunning(ctx))
func (d *DevOps) IsRunning(ctx context.Context) core.Result { // Value: bool
	findRes := d.findContainer(ctx, "core-dev")
	if !findRes.OK {
		return findRes
	}
	c := core.MustCast[*container.Container](findRes)
	return core.Ok(c != nil && c.Status == container.StatusRunning)
}

// findContainer finds a container by name. The Result carries a nil
// *container.Container Value when no container matches the name.
func (d *DevOps) findContainer(ctx context.Context, name string) core.Result { // Value: *container.Container
	listRes := d.container.List(ctx)
	if !listRes.OK {
		return listRes
	}
	containers := core.MustCast[[]*container.Container](listRes)
	for _, c := range containers {
		if c.Name == name {
			return core.Ok(c)
		}
	}
	return core.Ok((*container.Container)(nil))
}

// DevStatus returns information about the dev environment.
type DevStatus struct {
	Installed    bool
	Running      bool
	ImageVersion string
	ContainerID  string
	Memory       int
	CPUs         int
	SSHPort      int
	Uptime       time.Duration
}

// Status returns the current dev environment status.
//
// Usage:
//
//	status := core.MustCast[*DevStatus](dev.Status(ctx))
func (d *DevOps) Status(ctx context.Context) core.Result { // Value: *DevStatus
	status := &DevStatus{
		Installed: d.images.IsInstalled(),
		SSHPort:   DefaultSSHPort,
	}

	if info, ok := d.images.manifest.Images[ImageName()]; ok {
		status.ImageVersion = info.Version
	}

	if findRes := d.findContainer(ctx, "core-dev"); findRes.OK {
		c := core.MustCast[*container.Container](findRes)
		if c != nil {
			status.Running = c.Status == container.StatusRunning
			status.ContainerID = c.ID
			status.Memory = c.Memory
			status.CPUs = c.CPUs
			if status.Running {
				status.Uptime = time.Since(c.StartedAt)
			}
		}
	}

	return core.Ok(status)
}
