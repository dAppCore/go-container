package devenv

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/io"

	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/container/internal/proc"
)

// ClaudeOptions configures the Claude sandbox session.
type ClaudeOptions struct {
	NoAuth bool     // Don't forward any auth
	Auth   []string // Selective auth: "gh", "anthropic", "ssh", "git"
	Model  string   // Model to use: opus, sonnet
}

// Claude starts a sandboxed Claude session in the dev environment.
//
// Usage:
//
//	if r := dev.Claude(ctx, ".", devenv.ClaudeOptions{}); !r.OK { return r }
func (d *DevOps) Claude(ctx context.Context, projectDir string, opts ClaudeOptions) core.Result { // Value: nil
	// Auto-boot if not running
	runningRes := d.IsRunning(ctx)
	if !runningRes.OK {
		return runningRes
	}
	if !core.MustCast[bool](runningRes) {
		core.Println("Dev environment not running, booting...")
		if r := d.Boot(ctx, DefaultBootOptions()); !r.OK {
			return core.Fail(core.E("DevOps.Claude", "failed to boot", r.Value.(error)))
		}
	}

	// Mount project
	if r := d.mountProject(ctx, projectDir); !r.OK {
		return core.Fail(core.E("DevOps.Claude", "failed to mount project", r.Value.(error)))
	}

	// Prepare environment variables to forward
	envVars := []string{}

	if !opts.NoAuth {
		authTypes := opts.Auth
		if len(authTypes) == 0 {
			authTypes = []string{"gh", "anthropic", "ssh", "git"}
		}

		for _, auth := range authTypes {
			switch auth {
			case "anthropic":
				if key := core.Env("ANTHROPIC_API_KEY"); key != "" {
					envVars = append(envVars, core.Concat("ANTHROPIC_API_KEY=", key))
				}
			case "git":
				// Forward git config
				name, _ := proc.NewCommand("git", "config", "user.name").Output()
				email, _ := proc.NewCommand("git", "config", "user.email").Output()
				if len(name) > 0 {
					trimmed := core.Trim(string(name))
					envVars = append(envVars, core.Concat("GIT_AUTHOR_NAME=", trimmed))
					envVars = append(envVars, core.Concat("GIT_COMMITTER_NAME=", trimmed))
				}
				if len(email) > 0 {
					trimmed := core.Trim(string(email))
					envVars = append(envVars, core.Concat("GIT_AUTHOR_EMAIL=", trimmed))
					envVars = append(envVars, core.Concat("GIT_COMMITTER_EMAIL=", trimmed))
				}
			}
		}
	}

	// Build SSH command with agent forwarding
	args := []string{
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=~/.core/known_hosts",
		"-o", "LogLevel=ERROR",
		"-A", // SSH agent forwarding
		"-p", core.Sprintf("%d", DefaultSSHPort),
	}

	args = append(args, "root@localhost")

	// Build command to run inside
	claudeCmd := "cd /app && claude"
	if opts.Model != "" {
		claudeCmd += " --model " + opts.Model
	}
	args = append(args, claudeCmd)

	// Set environment for SSH
	cmd := proc.NewCommandContext(ctx, "ssh", args...)
	cmd.Stdin = proc.Stdin
	cmd.Stdout = proc.Stdout
	cmd.Stderr = proc.Stderr

	// Pass environment variables through SSH
	if len(envVars) > 0 {
		cmd.Env = append(proc.Environ(), envVars...)
	}

	core.Println("Starting Claude in sandboxed environment...")
	core.Println("Project mounted at /app")
	core.Println(core.Concat("Auth forwarded: SSH agent", formatAuthList(opts)))
	core.Println()

	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("DevOps.Claude", "ssh claude", err))
	}
	return core.Ok(nil)
}

func formatAuthList(opts ClaudeOptions) string {
	if opts.NoAuth {
		return " (none)"
	}
	if len(opts.Auth) == 0 {
		return ", gh, anthropic, git"
	}
	return core.Concat(", ", core.Join(", ", opts.Auth...))
}

// CopyGHAuth copies GitHub CLI auth to the VM.
//
// Usage:
//
//	if r := dev.CopyGHAuth(ctx); !r.OK { return r }
func (d *DevOps) CopyGHAuth(ctx context.Context) core.Result { // Value: nil
	home := coreutil.HomeDir()
	if home == "" {
		return core.Fail(core.E("DevOps.CopyGHAuth", "home directory not available", nil))
	}

	ghConfigDir := coreutil.JoinPath(home, ".config", "gh")
	if !io.Local.IsDir(ghConfigDir) {
		return core.Ok(nil) // No gh config to copy
	}

	// Use scp to copy gh config
	cmd := proc.NewCommandContext(ctx, "scp",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=~/.core/known_hosts",
		"-o", "LogLevel=ERROR",
		"-P", core.Sprintf("%d", DefaultSSHPort),
		"-r", ghConfigDir,
		"root@localhost:/root/.config/",
	)
	if err := cmd.Run(); err != nil {
		return core.Fail(core.E("DevOps.CopyGHAuth", "scp gh config", err))
	}
	return core.Ok(nil)
}
