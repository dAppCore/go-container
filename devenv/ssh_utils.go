package devenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// ensureHostKey ensures that the host key for the dev environment is in the known hosts file.
// This is used after boot to allow StrictHostKeyChecking=yes to work.
func ensureHostKey(ctx context.Context, port int) error {
	// Skip if requested (used in tests)
	if os.Getenv("CORE_SKIP_SSH_SCAN") == "true" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return coreerr.E("ensureHostKey", "get home dir", err)
	}

	knownHostsPath := filepath.Join(home, ".core", "known_hosts")

	// Ensure directory exists
	if err := coreio.Local.EnsureDir(filepath.Dir(knownHostsPath)); err != nil {
		return coreerr.E("ensureHostKey", "create known_hosts dir", err)
	}

	// Get host key using ssh-keyscan
	cmd := exec.CommandContext(ctx, "ssh-keyscan", "-p", fmt.Sprintf("%d", port), "localhost")
	out, err := cmd.Output()
	if err != nil {
		return coreerr.E("ensureHostKey", "ssh-keyscan failed", err)
	}

	if len(out) == 0 {
		return coreerr.E("ensureHostKey", "ssh-keyscan returned no keys", nil)
	}

	// Read existing known_hosts to avoid duplicates
	existingStr, _ := coreio.Local.Read(knownHostsPath)

	// Append new keys that aren't already there
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return coreerr.E("ensureHostKey", "open known_hosts", err)
	}
	defer f.Close()

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(existingStr, line) {
			if _, err := f.WriteString(line + "\n"); err != nil {
				return coreerr.E("ensureHostKey", "write known_hosts", err)
			}
		}
	}

	return nil
}
