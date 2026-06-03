<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — `core vm system` commands

**Date:** 2026-06-03
**Module:** `dappco.re/go/container` (code under `go/`)
**Status:** Approved (decisions locked with Snider 2026-06-03)
**Roadmap:** Feature ③a of the macOS-user series (① run flags ✅ → ② image/lifecycle CLI ✅ → ③a system commands → ③b interactive shell).

## 1. Goal

The Apple `container` runtime needs its system services started
(`container system start`) before build/run work; on a cold Mac that's a manual
step (the #16 preflight only *reports* "run `container system start`"). Expose
the bring-up/teardown/status from `core` so macOS users never touch the raw
binary: `core vm system start | status | stop`.

## 2. Locked decisions

| Decision | Choice |
|---|---|
| Command set | `start`, `status`, `stop` |
| Structure | `vm system <action>` subgroup (mirrors `vm templates`) |
| Kernel install | `start` passes `--enable-kernel-install` by default; `--no-kernel-install` opts out (`--disable-kernel-install`). The raw CLI prompts interactively, which can't work from `core`. |
| Runtime | Apple-only (reuses the `requireApple` guard from ②). |
| File | New `cmd/vm/cmd_system.go`. |

## 3. Provider methods (`apple.go`)

Three new public methods (each gets a `{Good,Bad,Ugly}` triplet in `apple_test.go` + `Example*` in `apple_example_test.go` — audit stays COMPLIANT):

```go
// SystemStart brings up the apiserver + background services. installKernel
// chooses --enable-kernel-install vs --disable-kernel-install (the CLI would
// otherwise prompt, which is impossible non-interactively).
func (a *AppleProvider) SystemStart(installKernel bool) core.Result // Value: nil

// SystemStop stops all `container` services.
func (a *AppleProvider) SystemStop() core.Result // Value: nil

// SystemStatus returns the raw `container system status` output.
func (a *AppleProvider) SystemStatus() core.Result // Value: string
```

Private arg builders (no ax7 requirement): `appleSystemStartArgs(installKernel bool) []string` → `["system","start","--enable-kernel-install"]` or `["system","start","--disable-kernel-install"]`; `appleSystemStopArgs() []string` → `["system","stop"]`. `appleSystemStatusArgs()` already exists.

`systemRunning()` (the #16 internal preflight) is refactored to call
`SystemStatus()` and check the output contains `"running"` (DRY; one source of
truth for "is the system up").

## 4. Commands (`cmd/vm/cmd_system.go`, Apple-only)

`addVMSystemCommand(c)` registers the parent `vm/system` plus three actions
(mirroring how `addVMTemplatesCommand` registers `vm/templates` + show/vars):

| Command | Flags | Handler |
|---|---|---|
| `vm system start` | `--no-kernel-install` (bool) | `requireApple` → `SystemStart(!noKernel)` → "started" line |
| `vm system status` | — | `requireApple` → `SystemStatus()` → print raw output |
| `vm system stop` | — | `requireApple` → `SystemStop()` → "stopped" line |

Handlers return `core.Result` and call directly (no `resultFromError` wrapper),
matching the ② image handlers.

## 5. Error handling

`requireApple` failure when the runtime/binary is absent (clear "requires macOS
26+ … `container system start`" message). Provider failures wrapped via
`core.E` carrying the CLI's stderr where available.

## 6. Testing

- **Unit (no binary):** `appleSystemStartArgs(true)` → includes `--enable-kernel-install`; `appleSystemStartArgs(false)` → `--disable-kernel-install`; `appleSystemStopArgs()` → `["system","stop"]`. Triplets/examples for `SystemStart`/`SystemStop`/`SystemStatus` (the `Bad`/guard cases use a bogus `Binary` so the call fails without the real runtime; `Good` is `CORE_APPLE_E2E`-gated). Handler guards via `requireApple` returning Fail when unavailable.
- **Live (CORE_APPLE_E2E=1):** `SystemStatus().OK` and its Value contains `"running"`; `SystemStart(true).OK` (idempotent — the daemon is already up). **Not** exercising `SystemStop` live (it would tear down the runtime the other e2e tests depend on).
- Full module `go test ./... -race` + `go vet` green; `audit.sh` → COMPLIANT.

## 7. Docs

`cmd/vm/cmd_commands.go` docstring + `specs/RFC.commands.md` command tree gain the `system` subgroup (start/status/stop).

## 8. Out of scope / follow-ups

- ③b interactive `exec -it` / `vm shell` — separate spec (needs a TTY exec path in the provider).
- `container system kernel` / `dns` / `property` subcommands — not exposed here (YAGNI).
- Auto-starting the system from build/run (vs the #16 clear-error preflight) — kept as an explicit user action.
