// SPDX-License-Identifier: EUPL-1.2

package container

import (
	"strings"
	"testing"
	"time"

	core "github.com/dappcore/go"
)

// AppleProvider against a fake `container` CLI — the paths CI has never run.
//
// The existing apple_test.go covers the same surface but every case is behind
// `if !p.Available() { t.Skip(...) }`, which on the Linux CI image is every
// case. These are the same questions asked so that the answer does not depend
// on what the host happens to have installed. {@see fakeAppleCLI}
//
// What is asserted here is mostly argv. A CLI wrapper's bugs are in the flags
// it builds, and the fake records every invocation, so a test can say the
// provider asked for the right thing rather than only that it did not error.

func TestAppleHermetic_Available_Good(t *testing.T) {
	p := newFakeAppleCLI(t).Provider()

	if !p.Available() {
		t.Fatal("Available() = false against a fake CLI reporting a running apiserver")
	}
	if p.Version == "" {
		t.Fatal("Available() did not capture the version the CLI reported")
	}
}

// The system probe is not a formality — a binary on PATH with the services down
// selects a runtime whose every call exits 1, which is why Available() asks.
func TestAppleHermetic_Available_SystemStopped_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("system status", appleStatusStopped, 0)

	if fake.Provider().Available() {
		t.Fatal("Available() = true while the apiserver reports itself stopped")
	}
}

func TestAppleHermetic_Available_StatusFails_Ugly(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("system status", "", 1)

	if fake.Provider().Available() {
		t.Fatal("Available() = true when `container system status` exited non-zero")
	}
}

// Not darwin is not available, whatever is on PATH.
func TestAppleHermetic_Available_NotDarwin_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)
	p := fake.Provider()
	t.Setenv("GOOS", "linux")

	if p.Available() {
		t.Fatal("Available() = true on a host reporting itself as linux")
	}
}

func TestAppleHermetic_SystemStatus_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("system status", appleStatusRunning, 0)

	r := fake.Provider().SystemStatus()
	if !r.OK {
		t.Fatalf("SystemStatus() not OK: %v", r)
	}
	if !strings.Contains(core.MustCast[string](r), "running") {
		t.Fatalf("SystemStatus() = %q; want the CLI's own words back", core.MustCast[string](r))
	}
	if !fake.CalledWith("system status") {
		t.Fatalf("provider did not run `system status`; invocations: %v", fake.Invocations())
	}
}

func TestAppleHermetic_SystemStop_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("system status", appleStatusStopped, 0)
	p := fake.Provider()

	p.SystemStop()

	if !fake.CalledWith("system stop") {
		t.Fatalf("SystemStop() did not run `system stop`; invocations: %v", fake.Invocations())
	}
}

// Build refuses an empty config before it reaches the CLI.
func TestAppleHermetic_Build_MissingSource_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)

	r := fake.Provider().Build(ContainerConfig{})
	if r.OK {
		t.Fatal("Build(empty config) succeeded; want a refusal")
	}
	if fake.CalledWith("build") {
		t.Fatalf("Build reached the CLI with no source; invocations: %v", fake.Invocations())
	}
}

// Run refuses a nil image rather than building argv around it.
func TestAppleHermetic_Run_NilImage_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)

	r := fake.Provider().Run(nil)
	if r.OK {
		t.Fatal("Run(nil) succeeded; want a refusal")
	}
	if fake.CalledWith("run") {
		t.Fatalf("Run reached the CLI with a nil image; invocations: %v", fake.Invocations())
	}
}

// GPU passthrough is unsupported, and saying so is the documented contract —
// HasGPU()->WithGPU->Run must fail honestly rather than run without the GPU.
func TestAppleHermetic_Run_GPU_Unsupported_Ugly(t *testing.T) {
	fake := newFakeAppleCLI(t)

	r := fake.Provider().Run(&Image{ID: "img", Name: "local/test"}, WithGPU(true))
	if r.OK {
		t.Fatal("Run(WithGPU) succeeded; the Apple runtime has no Metal passthrough")
	}
	if fake.CalledWith("run") {
		t.Fatalf("Run reached the CLI with an unsupported GPU request; invocations: %v", fake.Invocations())
	}
}

func TestAppleHermetic_Pull_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)

	fake.Provider().Pull("docker.io/library/alpine:latest")

	if !fake.CalledWith("image pull docker.io/library/alpine:latest") {
		t.Fatalf("Pull built the wrong argv; invocations: %v", fake.Invocations())
	}
}

func TestAppleHermetic_Pull_CLIFails_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("image pull", "no such image", 1)

	if r := fake.Provider().Pull("nope:latest"); r.OK {
		t.Fatal("Pull() OK though the CLI exited 1")
	}
}

func TestAppleHermetic_RemoveImage_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)

	fake.Provider().RemoveImage("local/test")

	if !fake.CalledWith("image delete local/test") {
		t.Fatalf("RemoveImage built the wrong argv; invocations: %v", fake.Invocations())
	}
}

func TestAppleHermetic_Logs_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("logs", "hello from the container\n", 0)

	r := fake.Provider().Logs("ctr-1", 25)
	if !r.OK {
		t.Fatalf("Logs() not OK: %v", r)
	}
	// The tail count is a flag the wrapper builds, so it is worth pinning.
	if !fake.CalledWith("logs -n 25 ctr-1") {
		t.Fatalf("Logs built the wrong argv; invocations: %v", fake.Invocations())
	}
}

func TestAppleHermetic_Logs_CLIFails_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("logs", "", 1)

	if r := fake.Provider().Logs("gone", 10); r.OK {
		t.Fatal("Logs() OK though the CLI exited 1")
	}
}

// --- appleSystemRunning, against the CLI's own words ---

// The regression this file exists for: the stopped output contains "running".
//
// A substring check answers yes to it, which is how a stopped runtime came to
// report itself available. These fixtures are verbatim CLI 0.12.3 output, so
// the test cannot drift back into asserting something nobody has ever seen.
func TestAppleHermetic_SystemRunning_Table_Good(t *testing.T) {
	if !appleSystemRunning(appleStatusRunning) {
		t.Fatal("appleSystemRunning(running table) = false")
	}
}

func TestAppleHermetic_SystemRunning_StoppedProse_Bad(t *testing.T) {
	if appleSystemRunning(appleStatusStopped) {
		t.Fatalf("appleSystemRunning(%q) = true — the stopped prose contains the word \"running\"", appleStatusStopped)
	}
}

// Anything it cannot read is not-running: believing the runtime is up when it
// is down costs a caller every subsequent verb, believing it is down when it is
// up costs them one `container system start`.
func TestAppleHermetic_SystemRunning_Unparseable_Ugly(t *testing.T) {
	for _, out := range []string{
		"",
		"\n\n",
		"FIELD              VALUE\n", // header, no rows
		"FIELD              VALUE\nappRoot            /tmp/x\n", // rows, no status
		"status",                        // key with no value
		"status             degraded\n", // a value that is not running
	} {
		if appleSystemRunning(out) {
			t.Fatalf("appleSystemRunning(%q) = true; want false", out)
		}
	}
}

// --- SystemStop waits, and is bounded ---

// `container system stop` is asynchronous and two-phase: it returns while the
// CLI is still stopping containers, and status keeps reporting running for
// seconds. SystemStop must not tell its caller the system is down until it is.
func TestAppleHermetic_SystemStop_WaitsForDown_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)
	p := fake.Provider()
	p.StopPollInterval = time.Millisecond

	// Still running when stop is issued; the fake flips to stopped underneath
	// the poll, the way the real teardown does.
	go func() {
		time.Sleep(5 * time.Millisecond)
		fake.Respond("system status", appleStatusStopped, 0)
	}()

	r := p.SystemStop()
	if !r.OK {
		t.Fatalf("SystemStop() failed though the system went down: %v", r)
	}
	if !fake.CalledWith("system stop") {
		t.Fatalf("SystemStop did not issue the stop; invocations: %v", fake.Invocations())
	}
}

// The bail. A waiter that can outlive what it waits for is a bug, so a system
// that never goes down must produce a named timeout rather than a hang.
func TestAppleHermetic_SystemStop_NeverStops_Bad(t *testing.T) {
	fake := newFakeAppleCLI(t)
	p := fake.Provider()
	p.StopPolls = 3
	p.StopPollInterval = time.Millisecond

	done := make(chan core.Result, 1)
	go func() { done <- p.SystemStop() }()

	select {
	case r := <-done:
		if r.OK {
			t.Fatal("SystemStop() OK though status never stopped reporting running")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SystemStop() did not bail — it is unbounded")
	}
}

// A stop the CLI itself refuses fails immediately, without polling.
func TestAppleHermetic_SystemStop_CLIFails_Ugly(t *testing.T) {
	fake := newFakeAppleCLI(t)
	fake.Respond("system stop", "permission denied", 1)
	p := fake.Provider()
	p.StopPolls = 100
	p.StopPollInterval = time.Second // never reached; the command failed

	start := time.Now()
	if r := p.SystemStop(); r.OK {
		t.Fatal("SystemStop() OK though `container system stop` exited 1")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("SystemStop polled after the command failed (%s)", elapsed)
	}
}

// The escape hatch keeps the old semantics for callers that do not care.
func TestAppleHermetic_SystemStopNoWait_DoesNotPoll_Good(t *testing.T) {
	fake := newFakeAppleCLI(t)
	p := fake.Provider()
	p.StopPolls = 100
	p.StopPollInterval = time.Second // would be 100s if it polled

	start := time.Now()
	if r := p.SystemStopNoWait(); !r.OK {
		t.Fatalf("SystemStopNoWait() failed: %v", r)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("SystemStopNoWait waited (%s); it is meant to return at once", elapsed)
	}
	if !fake.CalledWith("system stop") {
		t.Fatalf("SystemStopNoWait did not issue the stop; invocations: %v", fake.Invocations())
	}
}
