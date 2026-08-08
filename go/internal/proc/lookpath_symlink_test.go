//go:build unix

// SPDX-License-Identifier: EUPL-1.2

package proc

import (
	"os"
	"path/filepath"
	"testing"
)

// A PATH entry is very often a symlink pointing somewhere else entirely.
//
// This is how every package manager and app bundle installs a command:
// /usr/local/bin/docker is a link into /Applications/Docker.app, brew links
// into ../Cellar, and /usr/bin is full of links into /etc/alternatives. If
// LookPath cannot see through one it reports that a runtime is absent on a
// machine where it is installed and working, and go-container decides there is
// no Docker to sandbox agents in.
//
// Hermetic on purpose — a real docker would make this test a statement about
// the machine rather than about LookPath.
func TestLookPath_ResolvesSymlinkedExecutable_Good(t *testing.T) {
	target := filepath.Join(t.TempDir(), "realtool")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// The link lives in its own directory, and points out of it: an os.Root
	// rooted on the link's directory is entitled to refuse to follow it, which
	// is exactly how this regressed.
	binDir := t.TempDir()
	link := filepath.Join(binDir, "symtool")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}

	t.Setenv("PATH", binDir)

	got, err := LookPath("symtool")
	if err != nil {
		t.Fatalf("LookPath(symtool) = %v; want the link at %s", err, link)
	}
	if got != link {
		t.Fatalf("LookPath(symtool) = %q; want %q", got, link)
	}
}

// The same link, probed directly.
//
// Separated from the PATH walk so a failure says which half broke: whether
// isExecutable cannot see through the link, or LookPath is not consulting it.
func TestLookPath_SymlinkIsExecutable_Good(t *testing.T) {
	target := filepath.Join(t.TempDir(), "realtool")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(t.TempDir(), "symtool")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}

	if !isExecutable(link) {
		t.Fatalf("isExecutable(%s) = false; the link resolves to a 0755 regular file", link)
	}
}

// A link whose target has been removed is not an executable.
//
// The counterpart to the above: following the link must not become believing
// it. A dangling link resolves to nothing, and reporting it as a command would
// hand the caller a path that fails on exec.
func TestLookPath_DanglingSymlink_Bad(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "gone")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "dangling")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}

	if isExecutable(link) {
		t.Fatal("isExecutable(dangling link) = true; want false")
	}
}

// A link to something that is not executable stays not executable.
func TestLookPath_SymlinkToNonExecutable_Ugly(t *testing.T) {
	target := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(target, []byte("not a program\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(t.TempDir(), "looks-like-a-tool")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this filesystem: %v", err)
	}

	if isExecutable(link) {
		t.Fatal("isExecutable(link to 0644 file) = true; want false")
	}
}
