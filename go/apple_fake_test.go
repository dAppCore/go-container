// SPDX-License-Identifier: EUPL-1.2

package container

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A fake `container` CLI, so the AppleProvider tests run everywhere.
//
// They did not. Every one of them opened with
//
//	if !p.Available() { t.Skip("apple container runtime not available") }
//
// and CI is golang:1.26-alpine on Linux, so the skip was unconditional there:
// roughly twenty-five tests that had never executed once in the pipeline they
// were written for. A suite reports green just as loudly for a test that
// skipped as for one that passed.
//
// Nothing had to be redesigned to fix that, because the seams were already
// public. AppleProvider.Binary is a field, and discoverHostOS reads $GOOS — so
// pointing Binary at a script and setting GOOS=darwin drives the real provider
// code down the real code paths, and the only thing that was ever missing is a
// binary at the other end.
//
// The fake is a shell script rather than a Go double on purpose: the seam under
// test is process execution — argv construction, exit codes, stdout parsing —
// and a Go double would replace exactly the part worth exercising.
type fakeAppleCLI struct {
	t   *testing.T
	dir string
	bin string
	log string
}

// newFakeAppleCLI writes the script and returns a handle to it.
//
// `container system status` answers "apiserver is running" by default, because
// Available() gates on it and every test would otherwise begin by saying so.
// Everything else exits 0 with empty stdout until a test says otherwise.
func newFakeAppleCLI(t *testing.T) *fakeAppleCLI {
	t.Helper()

	dir := t.TempDir()
	f := &fakeAppleCLI{
		t:   t,
		dir: dir,
		bin: filepath.Join(dir, "container"),
		log: filepath.Join(dir, "invocations.log"),
	}

	// Responses are looked up by the leading words of argv, longest first, so
	// `image ls` can be scripted separately from `image pull` while a bare
	// `image` still matches both if that is all a test cares to say.
	script := `#!/bin/sh
printf '%s\n' "$*" >> "` + f.log + `"
dir="` + dir + `"
key3="$1_$2_$3"
key2="$1_$2"
key1="$1"
for k in "$key3" "$key2" "$key1"; do
  if [ -f "$dir/resp_$k.out" ]; then
    cat "$dir/resp_$k.out"
    if [ -f "$dir/resp_$k.code" ]; then exit "$(cat "$dir/resp_$k.code")"; fi
    exit 0
  fi
done
exit 0
`
	if err := os.WriteFile(f.bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake container CLI: %v", err)
	}

	f.Respond("--version", appleVersionOutput, 0)
	f.Respond("system status", appleStatusRunning, 0)

	return f
}

// Verbatim output from the real Apple container CLI, version 0.12.3 (f989901),
// captured on macOS from /usr/local/bin/container. Copied rather than composed,
// because the entire value of these fixtures is that nobody invented them —
// the defect they pin existed precisely because the stopped-state wording was
// guessed at.
//
// Note both states exit 0. The exit code cannot tell them apart.
const (
	// appleStatusRunning is a FIELD/VALUE table, not a sentence.
	appleStatusRunning = "FIELD              VALUE\n" +
		"status             running\n" +
		"appRoot            /Users/snider/Library/Application Support/com.apple.container/\n"

	// appleStatusStopped is prose, and it contains the substring "running".
	appleStatusStopped = "apiserver is not running and not registered with launchd\n"

	appleVersionOutput = "container CLI version 0.12.3 (f989901)\n"
)

// Respond scripts what the fake prints, and exits with, for a subcommand.
//
// The subcommand is matched on its leading words: "image ls" is distinct from
// "image pull", and "image" alone catches either.
func (f *fakeAppleCLI) Respond(subcommand string, stdout string, exitCode int) {
	f.t.Helper()

	key := strings.ReplaceAll(strings.TrimSpace(subcommand), " ", "_")
	if err := os.WriteFile(filepath.Join(f.dir, "resp_"+key+".out"), []byte(stdout), 0o644); err != nil {
		f.t.Fatalf("script %q: %v", subcommand, err)
	}
	if exitCode != 0 {
		codeFile := filepath.Join(f.dir, "resp_"+key+".code")
		if err := os.WriteFile(codeFile, []byte(strconv.Itoa(exitCode)), 0o644); err != nil {
			f.t.Fatalf("script exit code for %q: %v", subcommand, err)
		}
	}
}

// Provider returns an AppleProvider wired to the fake, with the host reported
// as darwin so the runtime is reachable on a Linux CI box.
//
// t.Setenv, so the override is undone when the test ends and the next one is
// not silently running on a machine that claims to be a Mac.
func (f *fakeAppleCLI) Provider() *AppleProvider {
	f.t.Helper()
	f.t.Setenv("GOOS", "darwin")

	p := NewAppleProvider()
	p.Binary = f.bin

	return p
}

// Invocations returns every argv the provider has executed, in order.
//
// This is the half that makes a fake worth more than a stub: a test can assert
// the flags the provider actually built, which is where a CLI wrapper's bugs
// live.
func (f *fakeAppleCLI) Invocations() []string {
	f.t.Helper()

	data, err := os.ReadFile(f.log)
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}

	return lines
}

// CalledWith reports whether any invocation contained the given fragment.
func (f *fakeAppleCLI) CalledWith(fragment string) bool {
	f.t.Helper()

	for _, line := range f.Invocations() {
		if strings.Contains(line, fragment) {
			return true
		}
	}

	return false
}
