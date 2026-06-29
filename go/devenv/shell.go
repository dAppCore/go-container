package devenv

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/container"

	"dappco.re/go/container/internal/proc"
)

// ShellOptions configures the shell connection.
type ShellOptions struct {
	Console bool     // Use serial console instead of SSH
	Command []string // Command to run (empty = interactive shell)
}

// Shell connects to the dev environment.
//
// Usage:
//
//	if r := dev.Shell(ctx, devenv.ShellOptions{}); !r.OK { return r }
func (d *DevOps) Shell(ctx context.Context, opts ShellOptions) core.Result { // Value: nil
	runningRes := d.IsRunning(ctx)
	if !runningRes.OK {
		return runningRes
	}
	if !core.MustCast[bool](runningRes) {
		return core.Fail(core.E("DevOps.Shell", "dev environment not running (run 'core dev boot' first)", nil))
	}

	if opts.Console {
		return d.serialConsole(ctx)
	}

	return d.sshShell(ctx, opts.Command)
}

// sshShell connects via SSH.
func (d *DevOps) sshShell(ctx context.Context, command []string) core.Result { // Value: nil
	args := []string{
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=~/.core/known_hosts",
		"-o", "LogLevel=ERROR",
		"-A", // Agent forwarding
		"-p", core.Sprintf("%d", DefaultSSHPort),
		"root@localhost",
	}

	if len(command) > 0 {
		args = append(args, command...)
	}

	cmd := proc.NewCommandContext(ctx, "ssh", args...)
	cmd.Stdin = proc.Stdin
	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr

	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("DevOps.sshShell", "ssh", err))
	}
	return core.Ok(nil)
}

// serialConsole attaches to the QEMU serial console.
func (d *DevOps) serialConsole(ctx context.Context) core.Result { // Value: nil
	// Find the container to get its console socket
	findRes := d.findContainer(ctx, "core-dev")
	if !findRes.OK {
		return findRes
	}
	c := core.MustCast[*container.Container](findRes)
	if c == nil {
		return core.Fail(core.E("DevOps.serialConsole", "console not available: container not found", nil))
	}

	// Use socat to connect to the console socket
	socketPath := core.Sprintf("/tmp/core-%s-console.sock", c.ID)
	cmd := proc.NewCommandContext(ctx, "socat", "-,raw,echo=0", "unix-connect:"+socketPath)
	cmd.Stdin = proc.Stdin
	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("DevOps.serialConsole", "socat", err))
	}
	return core.Ok(nil)
}
