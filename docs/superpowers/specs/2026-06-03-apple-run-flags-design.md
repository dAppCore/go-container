<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — `core vm run` networking / mount / env flags

**Date:** 2026-06-03
**Module:** `dappco.re/go/container` (code under `go/`)
**Status:** Approved (decisions locked with Snider 2026-06-03)
**Roadmap:** Feature ① of the macOS-user series (① run flags → ② image/lifecycle CLI → ③ system cmd + interactive shell).

## 1. Goal

`core vm run` cannot publish a port, mount a volume, or set an environment
variable today — its only flags are name/detach/memory/cpus/ssh-port/template/
runtime/gpu. That makes it impossible to run a real service or dev workload on
an Apple container from the CLI, even though `AppleProvider` (via `appleRunArgs`)
already emits `--publish`/`--volume`. This feature adds the missing flags and
wires them through.

## 2. Locked decisions

| Decision | Choice |
|---|---|
| `-p/--publish`, `-v/--volume` scope | **Both runtimes** — populate `RunOptions.Ports`/`Volumes`, honoured by Apple (`appleRunArgs`) and LinuxKit (QEMU/Hyperkit `hostfwd`/`9p`). |
| `-e/--env` scope | **Apple-only** — adds `RunOptions.Env`, emitted by `appleRunArgs`; LinuxKit bakes env into the image at build time (no runtime injection), so it's ignored there (documented). |
| Flag syntax | docker/`container`-style, repeatable. |
| Parsing | pure helper functions in `cmd/vm` (testable), following the existing `--var` repeated-flag pattern. |

## 3. Flags (on `core vm run`)

| Flag | Form | Target |
|---|---|---|
| `-p, --publish` | `host:container[/proto]` (repeatable) | `RunOptions.Ports` |
| `-v, --volume` | `host:container` (repeatable) | `RunOptions.Volumes` |
| `-e, --env` | `KEY=VALUE` (repeatable) | `RunOptions.Env` (new) |

Read via the existing `optionStrings(opts, <flag>)` repeated-flag mechanism
(as `--var` already is).

## 4. Parse helpers (pure, `cmd/vm`, TDD'd)

```go
// parsePublish turns ["8080:80", "127.0.0.1:5432:5432/tcp"] into a host→container
// port map. An optional /proto suffix is accepted and ignored (tcp assumed —
// RunOptions.Ports is map[int]int; UDP support is a noted follow-up). Errors on
// a missing colon or non-numeric port.
func parsePublish(specs []string) (map[int]int, error)

// parseVolumes turns ["/data:/app", "./cfg:/etc/app"] into a host→container
// path map. Errors on a missing colon.
func parseVolumes(specs []string) (map[string]string, error)

// parseEnv validates and passes through ["KEY=VALUE", ...]. Errors on a missing '='.
func parseEnv(specs []string) ([]string, error)
```

Error shape: `core.E("vm run", "invalid --publish \"x\": want host:container", nil)`.
A `[host-ip:]host-port:container-port[/proto]` form is accepted by taking the
last two colon-separated numeric fields as host:container; the host-ip prefix
and `/proto` suffix are dropped (tcp assumed), since `RunOptions.Ports` is
`map[int]int`. Preserving host-ip/proto is a noted follow-up (§9).

## 5. Core additions (root package)

- `RunOptions.Env []string` — container environment in `KEY=VALUE` form (in `container.go`).
- `WithEnv(env ...string) RunOption` — appends to `RunOptions.Env` (in `provider.go`), mirroring `WithArgs`. Gets a full `{Good,Bad,Ugly}` triplet in `provider_test.go` + `ExampleWithEnv` in `provider_example_test.go` (keep audit COMPLIANT).
- `appleRunArgs` — after the existing `--publish`/`--volume` loops, emit `-e KEY=VALUE` for each `ro.Env` entry (matches `container run -e/--env`). Env tokens go before the image, like the other flags; container `Args` stay last.

## 6. Threading (fix both run paths)

`addVMRunCommand`: read `--publish`/`--volume`/`--env` → parse via the §4 helpers → on error, `return core.Fail(...)` (no container started) → set `runOpts.Ports/Volumes/Env` → pass through `runContainer`.

Two existing bugs to fix in the same change:
- **`runContainerApple`** currently builds its `[]RunOption` without ports/volumes (a stale "handled via the CLI directly" comment). Add `container.WithPorts(ports)`, `container.WithVolumes(volumes)`, and `container.WithEnv(env...)`.
- **`runContainer`** (LinuxKit path) builds its `RunOptions` literal without `Ports`/`Volumes`. Add them so the QEMU/Hyperkit `hostfwd`/`9p` plumbing receives them. `Env` is left unset for LinuxKit (image-baked; documented).

Note: `Ports` is `map[int]int`, so a `host-ip`/`proto` is not represented in the
shared option; the Apple path's richer `[host-ip:]…[/proto]` form is a follow-up
if needed. For now host:container (tcp) covers the common case on both runtimes.

## 7. Error handling

Malformed `-p`/`-v`/`-e` fails the command with a clear `core.E` message naming
the offending spec; no container is created. Empty flag list = no change (today's
behaviour preserved).

## 8. Testing

- **Unit (no binary):** TDD `parsePublish` / `parseVolumes` / `parseEnv` (Good = typical, Bad = malformed → error, Ugly = edge: host-ip prefix / `/proto` / `KEY=` empty value / multiple entries). `WithEnv` triplet. `appleRunArgs` emits `-e KEY=VALUE` after `--publish`/`--volume` and before the image; ports/volumes still emitted.
- **Live (CORE_APPLE_E2E=1):** `run -p 8080:80 -e FOO=bar … sleep 60`, then assert `List`'s `publishedPorts` shows `8080→80` (the existing schema parser already maps `publishedPorts`).
- Full module `go test ./... -race` + `go vet` green; `audit.sh` stays COMPLIANT (0).

## 9. Out of scope / follow-ups

- UDP / host-ip in the shared `Ports` map (would need a richer port type).
- `--mount type=…` long form (only `-v` short form here).
- Features ② (image/lifecycle CLI) and ③ (system cmd + interactive shell) — separate specs.
