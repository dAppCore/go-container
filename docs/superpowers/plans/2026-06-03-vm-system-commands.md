<!-- SPDX-License-Identifier: EUPL-1.2 -->

# `core vm system` Commands — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (inline). Steps use `- [ ]` tracking.

**Goal:** `core vm system start | status | stop` over the Apple `container` runtime.

**Architecture:** 3 new public `AppleProvider` methods (`SystemStart/SystemStop/SystemStatus`) + private arg builders; a new `cmd/vm/cmd_system.go` registering the `vm system` subgroup (Apple-only, `requireApple`-gated, mirrors `vm templates`). `systemRunning()` (the #16 preflight) is DRY-refactored onto `SystemStatus()`.

**Tech Stack:** Go 1.26, `dappco.re/go` (`core.Result`, `core.E`, `core.Contains`, `core.Lower`), `internal/proc`, plain stdlib `testing`.

**Spec:** `docs/superpowers/specs/2026-06-03-vm-system-commands-design.md`

---

### Task 1: arg builders (pure, TDD)

**Files:** Modify `go/apple.go`; `go/apple_test.go`.

- [ ] **Step 1 — failing tests** (`apple_test.go`, near `appleSystemStatusArgs` test):
```go
func TestApple_appleSystemStartArgs_Good(t *testing.T) {
	if got, want := appleSystemStartArgs(true), []string{"system", "start", "--enable-kernel-install"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("install: got %v, want %v", got, want)
	}
	if got, want := appleSystemStartArgs(false), []string{"system", "start", "--disable-kernel-install"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("no-install: got %v, want %v", got, want)
	}
}
func TestApple_appleSystemStopArgs_Good(t *testing.T) {
	if got, want := appleSystemStopArgs(), []string{"system", "stop"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
```
- [ ] **Step 2 — verify RED:** `cd go && go test . -run 'TestApple_appleSystem(Start|Stop)Args'` build fail (undefined).
- [ ] **Step 3 — implement** (`apple.go`, near `appleSystemStatusArgs`):
```go
// appleSystemStartArgs builds the `container system start` argument vector. The
// kernel-install flag is forced (the CLI otherwise prompts interactively).
func appleSystemStartArgs(installKernel bool) []string {
	flag := "--enable-kernel-install"
	if !installKernel {
		flag = "--disable-kernel-install"
	}
	return []string{"system", "start", flag}
}

// appleSystemStopArgs builds the `container system stop` argument vector.
func appleSystemStopArgs() []string {
	return []string{"system", "stop"}
}
```
- [ ] **Step 4 — verify GREEN:** tests PASS.
- [ ] **Step 5 — commit:** `feat(apple): system start/stop arg builders`.

### Task 2: provider methods + systemRunning refactor + triplets/examples

**Files:** Modify `go/apple.go`, `go/apple_test.go`, `go/apple_example_test.go`.

- [ ] **Step 1 — failing tests** (triplets; Bad uses a bogus binary so the call fails without the runtime; Good is e2e-gated):
```go
func TestApple_AppleProvider_SystemStatus_Bad(t *testing.T) {
	p := &AppleProvider{Binary: "nonexistent-container-xyz"}
	if p.SystemStatus().OK {
		t.Fatal("expected failure with a bogus binary")
	}
}
func TestApple_AppleProvider_SystemStatus_Good(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple runtime not available")
	}
	r := p.SystemStatus()
	if !r.OK || !core.Contains(core.Lower(core.MustCast[string](r)), "running") {
		t.Fatalf("SystemStatus = %v / %q", r.OK, r.Error())
	}
}
func TestApple_AppleProvider_SystemStatus_Ugly(t *testing.T) {
	// Empty binary name also yields a failed Result rather than a panic.
	p := &AppleProvider{Binary: ""}
	_ = p.SystemStatus() // must not panic; result may be OK or Fail by host
}
```
(Same Bad/Good/Ugly shape for `SystemStart` — Bad: bogus binary `SystemStart(true)` fails; Good: e2e-gated idempotent `SystemStart(true).OK`; Ugly: `SystemStart(false)` with bogus binary fails — and for `SystemStop` — Bad/Ugly bogus-binary failure; Good e2e-gated is skipped to avoid tearing down the runtime, so `SystemStop_Good` asserts the bogus-binary path is a clean Fail, documented.)
- [ ] **Step 2 — verify RED:** build fail (undefined methods).
- [ ] **Step 3 — implement** (`apple.go`):
```go
// SystemStart brings up the apiserver + background services.
func (a *AppleProvider) SystemStart(installKernel bool) core.Result { // Value: nil
	if err := proc.NewCommand(a.Binary, appleSystemStartArgs(installKernel)...).Run(); err != nil {
		return core.Fail(core.E("AppleProvider.SystemStart", "start container system", err))
	}
	return core.Ok(nil)
}

// SystemStop stops all `container` services.
func (a *AppleProvider) SystemStop() core.Result { // Value: nil
	if err := proc.NewCommand(a.Binary, appleSystemStopArgs()...).Run(); err != nil {
		return core.Fail(core.E("AppleProvider.SystemStop", "stop container system", err))
	}
	return core.Ok(nil)
}

// SystemStatus returns the raw `container system status` output.
func (a *AppleProvider) SystemStatus() core.Result { // Value: string
	out, err := proc.NewCommand(a.Binary, appleSystemStatusArgs()...).Output()
	if err != nil {
		return core.Fail(core.E("AppleProvider.SystemStatus", "container system status", err))
	}
	return core.Ok(string(out))
}
```
Refactor `systemRunning()`:
```go
func (a *AppleProvider) systemRunning() bool {
	r := a.SystemStatus()
	return r.OK && core.Contains(core.Lower(core.MustCast[string](r)), "running")
}
```
Add `ExampleAppleProvider_SystemStart/SystemStop/SystemStatus` to `apple_example_test.go`.
- [ ] **Step 4 — verify GREEN:** `go test .` green; `-race` on the new triplets.
- [ ] **Step 5 — commit:** `feat(apple): SystemStart/SystemStop/SystemStatus provider methods`.

### Task 3: `cmd/vm/cmd_system.go` + registration + docs

**Files:** Create `go/cmd/vm/cmd_system.go`, `go/cmd/vm/cmd_system_test.go`; modify `go/cmd/vm/cmd_vm.go`, `go/cmd/vm/cmd_commands.go`, `specs/RFC.commands.md`.

- [ ] **Step 1 — failing test** (handler guards; on a non-apple host `requireApple` fails — here it passes, so assert the call returns a Result either way without panic, plus a deterministic guard via the e2e gate):
```go
package vm

import (
	"testing"

	core "dappco.re/go"
)

func TestCmdSystem_systemStatus_Good(t *testing.T) {
	// Returns a Result without panicking regardless of host; on a mac with the
	// runtime up it is OK, on CI it fails via requireApple with a message.
	r := systemStatus()
	if !r.OK && r.Error() == "" {
		t.Fatal("failed systemStatus must carry a message")
	}
}
```
- [ ] **Step 2 — verify RED:** build fail (undefined `systemStatus`).
- [ ] **Step 3 — implement** (`cmd_system.go`): `addVMSystemCommand(c)` registering `vm/system` + `vm/system/start` (`--no-kernel-install` bool), `vm/system/status`, `vm/system/stop`; handlers `systemStart(installKernel bool) core.Result`, `systemStatus() core.Result`, `systemStop() core.Result` — each `requireApple` → provider method → print:
```go
func systemStart(installKernel bool) core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	sr := core.MustCast[*container.AppleProvider](r).SystemStart(installKernel)
	if !sr.OK {
		return sr
	}
	core.Print(nil, "%s", successStyle.Render("system started"))
	core.Println()
	return core.Ok(nil)
}
func systemStatus() core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	sr := core.MustCast[*container.AppleProvider](r).SystemStatus()
	if !sr.OK {
		return sr
	}
	core.Println(core.MustCast[string](sr))
	return core.Ok(nil)
}
func systemStop() core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	sr := core.MustCast[*container.AppleProvider](r).SystemStop()
	if !sr.OK {
		return sr
	}
	core.Print(nil, "%s", successStyle.Render("system stopped"))
	core.Println()
	return core.Ok(nil)
}
```
Register in `AddVMCommands` (`addVMSystemCommand(c)`); update `cmd_commands.go` docstring + `RFC.commands.md` tree (`system` subgroup).
- [ ] **Step 4 — verify GREEN:** `go build ./... && go test ./cmd/vm/... .` green; `go vet ./...`.
- [ ] **Step 5 — commit:** `feat(cmd/vm): vm system start/status/stop commands`.

### Task 4: final gate

- [ ] **Step 1 — live smoke:** `CORE_APPLE_E2E=1 go test ./ -run 'SystemStatus_Good' -count=1` PASS (status running). The provider triplets cover the rest.
- [ ] **Step 2 — final gate:** `go test ./... -race` green; `go vet ./...`; `audit.sh` → COMPLIANT.
- [ ] **Step 3 — commit (if any straggler):** part of Task 2/3 commits.

---

## Self-Review

- **Spec coverage:** arg builders (T1) ✓; SystemStart/Stop/Status + systemRunning refactor + examples (T2) ✓; commands + registration + docs (T3) ✓; tests + audit (T1-T4) ✓.
- **Placeholders:** none; the test bodies + impl are concrete.
- **Type consistency:** `SystemStatus→core.Result (Value string)`, `SystemStart(bool)/SystemStop()→core.Result`; `appleSystemStartArgs(bool)/appleSystemStopArgs()→[]string`; handlers `(...) core.Result`. Consistent.
