# CLAUDE.md

## Project Overview

`core/go-container` provides container runtime management, LinuxKit image building, and portable dev environments. Three packages: root (container runtime + LinuxKit), devenv (dev environment orchestration), sources (image download from CDN/GitHub).

## Build & Development

```bash
go test ./...
go test -race ./...
```

## Architecture

Three packages:

- Root (`container`) — Container lifecycle (start/stop/status), hypervisor abstraction (QEMU/Hyperkit), LinuxKit YAML config builder with embedded templates, state persistence
- `devenv/` — Dev environment management: config loading, image management, shell access, test runner, serve mode, Claude integration. Depends on container/ and sources/
- `sources/` — Image download sources: CDN and GitHub release fetchers with checksum verification

## Dependencies

- `go-io` — File/process utilities
- `go-config` — Configuration loading (devenv only)
- `testify` — Test assertions

## Coding Standards

- UK English
- All functions have typed params/returns
- Tests use testify
- Test naming: `_Good`, `_Bad`, `_Ugly` suffixes
- License: EUPL-1.2
