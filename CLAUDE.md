# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
go test ./...                              # all tests
go test -race ./...                        # with race detector
go test -run TestState_Add_Good ./...      # single test by name
go test ./sources/                         # single package
go test ./devenv/                          # single package
```

No build step required -- this is a library. The `cmd/vm/` package registers CLI commands via `init()` into the parent `core/cli` binary.

**Go workspace**: This module is part of a Go workspace (`~/Code/go.work`). Sibling modules (`go-io`, `go-config`, `go-i18n`, `cli`) are resolved via the workspace file during local development.

## Architecture

Three packages with a clear dependency direction: `devenv` -> `container` (root) -> `sources`.

- **Root (`container`)** -- Container lifecycle (`Manager` interface, `LinuxKitManager` implementation), hypervisor abstraction (`Hypervisor` interface with QEMU and Hyperkit implementations), JSON-persisted state (`~/.core/containers.json`), and LinuxKit template engine with embedded YAML templates and `${VAR:-default}` variable substitution.

- **`devenv/`** -- `DevOps` orchestrator composing container and sources into a dev environment workflow: boot/stop/status, SSH shell and serial console access, project mounting via reverse SSHFS at `/app`, auto-detection of serve/test commands by project type, and sandboxed Claude sessions with auth forwarding.

- **`sources/`** -- `ImageSource` interface with CDN (HTTP GET) and GitHub (`gh release download`) implementations. `ImageManager` in devenv maintains a manifest tracking installed versions.

## Key Patterns

- **`io.Medium` abstraction** -- File system operations use `io.Medium` (from `go-io`) rather than `os` directly. Use `io.Local` for real file access. This enables test mocking.
- **Compile-time interface checks** -- `var _ Interface = (*Impl)(nil)` (see `sources/cdn.go`, `sources/github.go`).
- **Copy-on-read state** -- `State.Get()` and `State.All()` return copies to prevent data races.
- **All persistent data** lives under `~/.core/` (containers.json, logs, images, config.yaml, known_hosts, linuxkit templates). `CORE_IMAGES_DIR` env var overrides the images directory.

## Coding Standards

- UK English (colour, organisation, honour)
- Tests use testify; naming convention: `_Good` (happy path), `_Bad` (expected errors), `_Ugly` (edge cases)
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Context propagation: all blocking operations take `context.Context` as first parameter
- Licence: EUPL-1.2
