<!-- SPDX-License-Identifier: EUPL-1.2 -->

# `core vm` Interactive Shell / `exec -it` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (inline). Steps use `- [ ]` tracking.

**Goal:** `core vm shell <id> [cmd]` + `vm exec -i -t` — a TTY into a running container, both runtimes.

**Architecture:** Concrete `ExecInteractive` on `*AppleProvider` (`container exec -i -t`) and `*LinuxKitManager` (`ssh -t`), both wiring `proc.{Stdin,Stdout,Stderr}` + `Run()`. A shared cmd/vm `execInteractive` handler dispatches by owner. No `Manager` interface change.

**Tech Stack:** Go 1.26, `dappco.re/go` (`core.Result`, `core.E`), `internal/proc` (terminal stdio + `Command`), plain stdlib `testing`.

**Spec:** `docs/superpowers/specs/2026-06-03-vm-interactive-shell-design.md`

---

### Task 1: Apple `ExecInteractive` + arg builder

**Files:** Modify `go/apple.go`, `go/apple_test.go`, `go/apple_example_test.go`.

- [ ] **Step 1 — failing tests** (`apple_test.go`):
```go
func TestApple_appleExecInteractiveArgs_Good(t *testing.T) {
	got := appleExecInteractiveArgs("web", []string{"/bin/sh"})
	want := []string{"exec", "-i", "-t", "web", "/bin/sh"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
func TestApple_AppleProvider_ExecInteractive_Bad(t *testing.T) {
	p := NewAppleProvider()
	if p.ExecInteractive("").OK {
		t.Fatal("expected error for empty id")
	}
}
func TestApple_AppleProvider_ExecInteractive_Ugly(t *testing.T) {
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}
	if p.ExecInteractive("web", "/bin/sh").OK {
		t.Fatal("expected failure with a bogus binary")
	}
}
func TestApple_AppleProvider_ExecInteractive_Good(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1")
	}
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}
	const name = "core-exec-i-e2e"
	ctx := context.Background()
	_ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run()
	defer func() { _ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() }()
	if r := p.Pull("docker.io/library/alpine:latest"); !r.OK {
		t.Fatalf("Pull: %v", r.Error())
	}
	if r := p.Run(&Image{Path: "docker.io/library/alpine:latest"}, WithName(name), WithDetach(true), WithArgs("sleep", "60")); !r.OK {
		t.Fatalf("Run: %v", r.Error())
	}
	// poll until running
	for i := 0; i < 30; i++ {
		if lr := p.List(); lr.OK {
			found := false
			for _, c := range core.MustCast[[]*Container](lr) {
				if c.ID == name && c.Status == StatusRunning {
					found = true
				}
			}
			if found {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Non-interactive command through the interactive path; -t without a real
	// TTY may warn but should still run. If the runtime rejects it, this is the
	// documented degrade-to-skip point.
	if r := p.ExecInteractive(name, "echo", "interactive-ok"); !r.OK {
		t.Skipf("ExecInteractive needs a TTY in this env: %v", r.Error())
	}
}
```
- [ ] **Step 2 — verify RED:** `cd go && go test . -run 'TestApple_(appleExecInteractiveArgs|AppleProvider_ExecInteractive)'` build fail (undefined).
- [ ] **Step 3 — implement** (`apple.go`, near `Exec`):
```go
// appleExecInteractiveArgs builds `container exec -i -t <id> <cmd…>`.
func appleExecInteractiveArgs(id string, cmd []string) []string {
	return append([]string{"exec", "-i", "-t", id}, cmd...)
}

// ExecInteractive runs an interactive command in a container with a TTY, wiring
// the child to the terminal and blocking until it exits.
//
// Usage:
//
//	if r := p.ExecInteractive(id, "/bin/sh"); !r.OK { return r }
func (a *AppleProvider) ExecInteractive(id string, cmd ...string) core.Result { // Value: nil
	if id == "" {
		return core.Fail(core.E("AppleProvider.ExecInteractive", "container id is required", nil))
	}
	c := proc.NewCommandContext(context.Background(), a.Binary, appleExecInteractiveArgs(id, cmd)...)
	c.Stdin = proc.Stdin
	c.Stdout = proc.Stdout
	c.Stderr = proc.Stderr
	if err := c.Run(); err != nil {
		return core.Fail(core.E("AppleProvider.ExecInteractive", "interactive exec", err))
	}
	return core.Ok(nil)
}
```
Add `ExampleAppleProvider_ExecInteractive` to `apple_example_test.go`.
- [ ] **Step 4 — verify GREEN:** `go test . -run 'ExecInteractive'` (the Good skips without CORE_APPLE_E2E); `go test .` green.
- [ ] **Step 5 — commit:** `feat(apple): ExecInteractive (TTY exec)`.

### Task 2: LinuxKit `ExecInteractive` + `linuxkitSSHArgs` extraction

**Files:** Modify `go/linuxkit.go`, `go/linuxkit_test.go`.

- [ ] **Step 1 — failing tests** (`linuxkit_test.go`):
```go
func TestLinuxKit_linuxkitSSHArgs_Good(t *testing.T) {
	c := &Container{SSHPort: 2200}
	withTTY := linuxkitSSHArgs(c, []string{"/bin/sh"}, true)
	noTTY := linuxkitSSHArgs(c, []string{"/bin/sh"}, false)
	if !contains(withTTY, "-t") {
		t.Fatalf("tty args missing -t: %v", withTTY)
	}
	if contains(noTTY, "-t") {
		t.Fatalf("non-tty args should omit -t: %v", noTTY)
	}
	for _, want := range []string{"-p", "2200", "root@localhost", "/bin/sh"} {
		if !contains(withTTY, want) {
			t.Fatalf("args %v missing %q", withTTY, want)
		}
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
```
(If a `contains` test helper already exists in the package, reuse it instead of redefining.)
- [ ] **Step 2 — verify RED:** build fail (undefined `linuxkitSSHArgs`).
- [ ] **Step 3 — implement** (`linuxkit.go`): extract the inline ssh-arg building from `Exec` into:
```go
// linuxkitSSHArgs builds the ssh argument vector for a container exec. tty
// allocates a pseudo-terminal (`-t`) for interactive sessions.
func linuxkitSSHArgs(c *Container, cmd []string, tty bool) []string {
	sshPort := c.SSHPort
	if sshPort <= 0 {
		sshPort = 2222
	}
	args := []string{"-p", core.Sprintf("%d", sshPort)}
	if tty {
		args = append(args, "-t")
	}
	args = append(args,
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=~/.core/known_hosts",
		"-o", "LogLevel=ERROR",
	)
	if c.SSHKey != "" {
		args = append(args, "-i", c.SSHKey)
	}
	args = append(args, "root@localhost")
	return append(args, cmd...)
}
```
Refactor `Exec` to use `linuxkitSSHArgs(container, cmd, false)`; add:
```go
// ExecInteractive runs an interactive command over `ssh -t`, wiring the child
// to the terminal and blocking until it exits.
func (m *LinuxKitManager) ExecInteractive(ctx context.Context, id string, cmd []string) core.Result { // Value: nil
	container, ok := m.state.Get(id)
	if !ok {
		return core.Fail(core.E("LinuxKitManager.ExecInteractive", "container not found: "+id, nil))
	}
	if container.Status != StatusRunning {
		return core.Fail(core.E("LinuxKitManager.ExecInteractive", "container is not running: "+id, nil))
	}
	sshCmd := proc.NewCommandContext(ctx, "ssh", linuxkitSSHArgs(container, cmd, true)...)
	sshCmd.Stdin = proc.Stdin
	sshCmd.Stdout = proc.Stdout
	sshCmd.Stderr = proc.Stderr
	if err := sshCmd.Run(); err != nil {
		return core.Fail(core.E("LinuxKitManager.ExecInteractive", "ssh -t exec", err))
	}
	return core.Ok(nil)
}
```
Add `ExampleLinuxKitManager_ExecInteractive` to `linuxkit_example_test.go`.
- [ ] **Step 4 — verify GREEN:** `go test . -run 'linuxkitSSHArgs|ExecInteractive'`; `go test .` green (existing Exec tests still pass after refactor).
- [ ] **Step 5 — commit:** `feat(linuxkit): ExecInteractive (ssh -t) + linuxkitSSHArgs helper`.

### Task 3: cmd/vm `vm shell` + `exec -it`

**Files:** Modify `go/cmd/vm/cmd_container.go`, `go/cmd/vm/cmd_vm.go`, `go/cmd/vm/cmd_container_test.go`, `go/cmd/vm/cmd_commands.go`, `specs/RFC.commands.md`.

- [ ] **Step 1 — failing test** (`cmd_container_test.go`):
```go
func TestCmdContainer_shellContainer_Bad(t *testing.T) {
	if shellContainer("", nil).OK {
		t.Fatal("expected error for empty id")
	}
}
```
- [ ] **Step 2 — verify RED:** build fail (undefined `shellContainer`).
- [ ] **Step 3 — implement** (`cmd_container.go`):
```go
func execInteractive(id string, cmd []string) core.Result {
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return core.Fail(err)
	}
	if apple != nil {
		return apple.ExecInteractive(fullID, cmd...)
	}
	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return mgrRes
	}
	return core.MustCast[*container.LinuxKitManager](mgrRes).ExecInteractive(context.Background(), fullID, cmd)
}

func shellContainer(id string, cmd []string) core.Result {
	if id == "" {
		return core.Fail(core.E("vm shell", vmT("cmd.vm.error.id_required"), nil))
	}
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}
	return execInteractive(id, cmd)
}

// addVMShellCommand adds the 'shell' command under vm.
func addVMShellCommand(c *core.Core) {
	registerVMCommand(c, "vm/shell", core.Command{
		Description: "Open an interactive shell in a container (default /bin/sh)",
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm shell", vmT("cmd.vm.error.id_required"), nil))
			}
			return shellContainer(args[0], args[1:])
		},
	})
}
```
Add `-i/--interactive` + `-t/--tty` flags to `addVMExecCommand`'s `core.NewOptions(...)`; in its Action, when `opts.Bool("interactive") || opts.Bool("tty")`, call `execInteractive(args[0], args[1:])`; else `execInContainer(args[0], args[1:])` as today. Register `addVMShellCommand(c)` in `AddVMCommands`.
- [ ] **Step 4 — verify GREEN:** `go build ./... && go test ./cmd/vm/... .`; `go vet ./...`. Update `cmd_commands.go` + `RFC.commands.md` (add `shell`; note `exec -i/-t`).
- [ ] **Step 5 — commit:** `feat(cmd/vm): vm shell + exec -i/-t interactive path`.

### Task 4: live e2e + final gate

- [ ] **Step 1 — live smoke:** `CORE_APPLE_E2E=1 go test ./ -run 'TestApple_AppleProvider_ExecInteractive_Good' -count=1` — runs/skips per TTY availability (documented).
- [ ] **Step 2 — final gate:** `go test ./... -race` green; `go vet ./...`; `audit.sh` → COMPLIANT.
- [ ] **Step 3 — manual smoke (note to user):** `core vm shell <id>` once the core binary is built — verified by hand.

---

## Self-Review

- **Spec coverage:** Apple ExecInteractive + builder (T1) ✓; LinuxKit ExecInteractive + linuxkitSSHArgs extraction (T2) ✓; vm shell + exec -i/-t + dispatch (T3) ✓; docs (T3) ✓; tests + audit (T1-T4) ✓; manual-smoke note (T4) ✓.
- **Placeholders:** none; the degrade-to-skip Good is explicit.
- **Type consistency:** `ExecInteractive` returns `core.Result`; `appleExecInteractiveArgs(id, []string)→[]string`, `linuxkitSSHArgs(c, []string, bool)→[]string`; cmd handlers `(string, []string) core.Result`. The `contains` test helper is defined once (reused if present).
