package devenv

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"

	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/container/internal/proc"
)

// ensureHostKey ensures that the host key for the dev environment is in the known hosts file.
// This is used after boot to allow StrictHostKeyChecking=yes to work.
func ensureHostKey(ctx context.Context, port int) (
	err error, // result
) {
	// Skip if requested (used in tests)
	if core.Env("CORE_SKIP_SSH_SCAN") == "true" {
		return nil
	}

	home := coreutil.HomeDir()
	if home == "" {
		return coreerr.E("ensureHostKey", "get home dir", nil)
	}

	knownHostsPath := coreutil.JoinPath(home, ".core", "known_hosts")

	// Ensure directory exists
	if err := coreio.Local.EnsureDir(core.PathDir(knownHostsPath)); err != nil {
		return coreerr.E("ensureHostKey", "create known_hosts dir", err)
	}

	// Get host key using ssh-keyscan
	cmd := proc.NewCommandContext(ctx, "ssh-keyscan", "-p", core.Sprintf("%d", port), "localhost")
	out, err := cmd.Output()
	if err != nil {
		return coreerr.E("ensureHostKey", "ssh-keyscan failed", err)
	}

	if len(out) == 0 {
		return coreerr.E("ensureHostKey", "ssh-keyscan returned no keys", nil)
	}

	// Read existing known_hosts to avoid duplicates
	existingStr, _ := coreio.Local.Read(knownHostsPath)

	if !coreio.Local.Exists(knownHostsPath) {
		if err := coreio.Local.WriteMode(knownHostsPath, "", 0600); err != nil {
			return coreerr.E("ensureHostKey", "create known_hosts", err)
		}
	}

	// Append new keys that aren't already there
	f, err := coreio.Local.Append(knownHostsPath)
	if err != nil {
		return coreerr.E("ensureHostKey", "open known_hosts", err)
	}
	defer f.Close()

	lines := core.Split(string(out), "\n")
	for _, line := range lines {
		line = core.Trim(line)
		if line == "" || core.HasPrefix(line, "#") {
			continue
		}
		if !core.Contains(existingStr, line) {
			if _, err := f.Write([]byte(core.Concat(line, "\n"))); err != nil {
				return coreerr.E("ensureHostKey", "write known_hosts", err)
			}
		}
	}

	return nil
}
