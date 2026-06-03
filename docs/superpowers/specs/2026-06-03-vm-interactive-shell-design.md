<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — `core vm` interactive shell / `exec -it`

**Date:** 2026-06-03
**Module:** `dappco.re/go/container` (code under `go/`)
**Status:** Approved (decisions locked with Snider 2026-06-03)
**Roadmap:** Feature ③b — the final macOS-user feature (① run flags ✅ → ② image/lifecycle CLI ✅ → ③a system commands ✅ → ③b interactive shell).

## 1. Goal

Give macOS users a real terminal inside a running container:
`core vm shell <id>` (and `core vm exec -it <id> <cmd>`). The existing `vm exec`
captures output non-interactively; this adds a TTY-wired path.

## 2. Locked decisions

| Decision | Choice |
|---|---|
| Surface | Both `vm shell <id> [cmd…]` (default `/bin/sh`) and `-i/--interactive` + `-t/--tty` flags on `vm exec`. |
| Runtime scope | Both, full TTY. Apple: `container exec -i -t`. LinuxKit: `ssh -t`. Dispatch by owner (like the lifecycle commands). |
| Mechanism | Concrete `ExecInteractive` methods on `*AppleProvider` and `*LinuxKitManager` — **not** a `Manager` interface change (cmd/vm already uses the concrete `*LinuxKitManager`; avoids rippling `Exec`'s callers). |
| stdio | Wire the child's `Stdin/Stdout/Stderr` to `proc.{Stdin,Stdout,Stderr}` (the terminal fds 0/1/2; no `os` import). |

## 3. Provider / Manager methods

```go
// AppleProvider.ExecInteractive runs an interactive command in a container with
// a TTY, wiring the process to the terminal (blocking until it exits).
func (a *AppleProvider) ExecInteractive(id string, cmd ...string) core.Result // Value: nil

// appleExecInteractiveArgs builds `container exec -i -t <id> <cmd…>`.
func appleExecInteractiveArgs(id string, cmd []string) []string

// (*LinuxKitManager).ExecInteractive runs an interactive command over `ssh -t`,
// wiring the process to the terminal.
func (m *LinuxKitManager) ExecInteractive(ctx context.Context, id string, cmd []string) core.Result // Value: nil
```

- `AppleProvider.ExecInteractive`: empty-id guard → Fail; build args via `appleExecInteractiveArgs`; `cmd := proc.NewCommandContext(...); cmd.Stdin = proc.Stdin; cmd.Stdout = proc.Stdout; cmd.Stderr = proc.Stderr; cmd.Run()`; wrap failures with `core.E`.
- `appleExecInteractiveArgs(id, cmd)` → `["exec", "-i", "-t", id] + cmd` (pure, TDD).
- **LinuxKit refactor:** extract the ssh-arg building currently inline in `Exec` into a pure `linuxkitSSHArgs(c *Container, cmd []string, tty bool) []string` (inserts `-t` when `tty`). `Exec` calls it with `tty=false` (unchanged behaviour); `ExecInteractive` with `tty=true`. Both wire `proc` stdio + `Run` (a shared private `runSSH` helper is fine). No `Manager` interface change.

## 4. cmd/vm commands (`cmd_container.go`)

- `vm shell <id> [cmd…]`: id required (guard); `cmd` defaults to `["/bin/sh"]`; calls `execInteractive(id, cmd)`.
- `vm exec`: add `core.Option{Key:"interactive", Value:false}` (`-i`) + `{Key:"tty", Value:false}` (`-t`). In `execInContainer`, if `interactive || tty`, route to `execInteractive(id, cmd)`; else the existing capture path. (Documented idiom: `vm exec -i -t <id> <cmd>`.)
- Shared `execInteractive(id string, cmd []string) core.Result`: `resolveContainerOwner(id)` → Apple `p.ExecInteractive(id, cmd...)` / LinuxKit `manager.ExecInteractive(ctx, id, cmd)`.

## 5. Error handling

Empty-id guard on `shell`/`exec`. `resolveContainerOwner` no-match/ambiguous errors. Provider/Manager failures wrapped via `core.E`. A non-zero exit from the interactive command surfaces as a failed Result (the user's shell exit code isn't specially mapped — out of scope).

## 6. Testing

TTYs resist `go test`, so coverage is layered:
- **Unit (no binary):** `appleExecInteractiveArgs(id, cmd)` → `["exec","-i","-t",id,...]`; `linuxkitSSHArgs(c, cmd, true)` includes `-t` and `linuxkitSSHArgs(c, cmd, false)` omits it (and both carry the port/known-hosts/key args). Empty-id guards on `ExecInteractive` + the `shell`/`exec` handlers. Triplets + `Example*` for the new public methods (audit COMPLIANT).
- **Live (CORE_APPLE_E2E=1):** `ExecInteractive` "Good" runs a NON-interactive command (`echo interactive-ok`) through the interactive path against a real running container and asserts it does not error — exercising the real `container exec -i -t` wiring. (If the runtime rejects `-t` without a real TTY under `go test`, the Good degrades to a documented skip; discovered at build time.)
- **Manual smoke (documented):** a genuine interactive shell — `core vm shell <id>` — is verified by hand, not automated.
- Full module `go test ./... -race` + `go vet` green; `audit.sh` → COMPLIANT.

## 7. Docs

`cmd/vm/cmd_commands.go` docstring + `specs/RFC.commands.md` tree gain `shell` and the `exec -i/-t` flags.

## 8. Out of scope / follow-ups

- SIGWINCH window-resize propagation; explicit raw-mode handling beyond what `container`/`ssh` do.
- Mapping the inner command's exit code to `core`'s process exit code.
- Roadmap complete after this: ④ registry login and other items remain ad-hoc, not part of this series.
