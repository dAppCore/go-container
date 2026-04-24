package devenv

import (
	"context"

	core "dappco.re/go/core"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"

	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/container/internal/proc"
)

// ServeOptions configures the dev server.
type ServeOptions struct {
	Port int    // Port to serve on (default 8000)
	Path string // Subdirectory to serve (default: current dir)
}

// Serve mounts the project and starts a dev server.
func (d *DevOps) Serve(ctx context.Context, projectDir string, opts ServeOptions) error {
	running, err := d.IsRunning(ctx)
	if err != nil {
		return err
	}
	if !running {
		return coreerr.E("DevOps.Serve", "dev environment not running (run 'core dev boot' first)", nil)
	}

	if opts.Port == 0 {
		opts.Port = 8000
	}

	servePath := projectDir
	if opts.Path != "" {
		servePath = coreutil.JoinPath(projectDir, opts.Path)
	}

	// Mount project directory via SSHFS
	if err := d.mountProject(ctx, servePath); err != nil {
		return coreerr.E("DevOps.Serve", "failed to mount project", err)
	}

	// Detect and run serve command
	serveCmd := DetectServeCommand(d.medium, servePath)
	core.Print(nil, "Starting server: %s", serveCmd)
	core.Print(nil, "Listening on http://localhost:%d", opts.Port)

	// Run serve command via SSH
	return d.sshShell(ctx, []string{"cd", "/app", "&&", serveCmd})
}

// mountProject mounts a directory into the VM via SSHFS.
func (d *DevOps) mountProject(ctx context.Context, path string) error {
	absPath := coreutil.AbsPath(path)

	// Use reverse SSHFS mount
	// The VM connects back to host to mount the directory
	cmd := proc.NewCommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=~/.core/known_hosts",
		"-o", "LogLevel=ERROR",
		"-R", "10000:localhost:22", // Reverse tunnel for SSHFS
		"-p", core.Sprintf("%d", DefaultSSHPort),
		"root@localhost",
		core.Sprintf("mkdir -p /app && sshfs -p 10000 %s@localhost:%s /app -o allow_other", core.Env("USER"), absPath),
	)
	return cmd.Run()
}

// DetectServeCommand auto-detects the serve command for a project.
//
// Usage:
//
//	cmd := DetectServeCommand(io.Local, ".")
func DetectServeCommand(m io.Medium, projectDir string) string {
	// Laravel/Octane
	if hasFile(m, projectDir, "artisan") {
		return "php artisan octane:start --host=0.0.0.0 --port=8000"
	}

	// Node.js with dev script
	if hasFile(m, projectDir, "package.json") {
		if hasPackageScript(m, projectDir, "dev") {
			return "npm run dev -- --host 0.0.0.0"
		}
		if hasPackageScript(m, projectDir, "start") {
			return "npm start"
		}
	}

	// PHP with composer
	if hasFile(m, projectDir, "composer.json") {
		return "frankenphp php-server -l :8000"
	}

	// Go
	if hasFile(m, projectDir, "go.mod") {
		if hasFile(m, projectDir, "main.go") {
			return "go run ."
		}
	}

	// Python Django
	if hasFile(m, projectDir, "manage.py") {
		return "python manage.py runserver 0.0.0.0:8000"
	}

	// Fallback: simple HTTP server
	return "python3 -m http.server 8000"
}
