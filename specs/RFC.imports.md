---
Status: Aspirational
---

# container — Imports

> Ecosystem dependencies extracted from source code.

## github.com/dappcore/go (core framework)

| Module | Import path | Used in |
|--------|-------------|---------|
| Core | `github.com/dappcore/go` | All packages (`core` alias) |
| Log | `github.com/dappcore/go-log/go` | apple.go, container.go, gpu.go, hypervisor.go, linuxkit.go, runtime.go, tim.go, state.go, datacube.go, datanode.go, devenv/, sources/, cmd/vm/ (`coreerr` alias) |
| I/O | `github.com/dappcore/go-io/go` | apple.go, linuxkit.go, state.go, templates.go, tim.go, datacube.go, datanode.go, devenv/, sources/, cmd/vm/ (`io` alias for `github.com/dappcore/go-io/go`) |
| CLI | `github.com/dappcore/cli/go/pkg/cli` | cmd/vm/cmd_vm.go |
| Config | `github.com/dappcore/config/go` | devenv/config.go |

## Internal packages

| Package | Import path | Purpose |
|---------|-------------|---------|
| proc | `github.com/dappcore/go-container/go/internal/proc` | Process execution (StartProcess, Wait4, pipes). Used by: apple.go, hypervisor.go, linuxkit.go, runtime.go, devenv/shell.go, devenv/ssh_utils.go, devenv/claude.go, devenv/serve.go, sources/github.go, cmd/vm/ |
| coreutil | `github.com/dappcore/go-container/go/internal/coreutil` | Path utilities (HomeDir, TempDir, JoinPath, MkdirTemp, etc.). Used by: state.go, templates.go, devenv/, sources/cdn.go, cmd/vm/cmd_templates.go |

## Standard library

| Package | Used in | Note |
|---------|---------|------|
| `context` | container.go, apple.go, hypervisor.go, linuxkit.go, runtime.go, tim.go, datacube.go, datanode.go, state.go, devenv/, sources/, cmd/vm/ | AX-6 structural exemption |
| `crypto/aes` | tim.go, apple.go, datacube.go | AES encryption primitives |
| `crypto/cipher` | tim.go, apple.go, datacube.go | GCM mode |
| `crypto/rand` | container.go, tim.go, apple.go | ID generation, nonces |
| `crypto/sha256` | tim.go, apple.go, datacube.go | Key derivation |
| `encoding/hex` | container.go | Container ID encoding |
| `encoding/json` | apple.go, state.go | CLI output parsing, state persistence |
| `os` | apple.go | File stat (Containerfile detection) |
| `time` | container.go, apple.go, linuxkit.go, datanode.go, devenv/, cmd/vm/ | Timestamps, durations, GC timers |
| `runtime` | runtime.go, apple.go | GOOS/GOARCH detection |
| `strconv` | apple.go | Numeric CLI flag formatting |
| `embed` | templates.go | Embedded YAML templates |
| `iter` | templates.go | Lazy template iteration |
| `regexp` | templates.go | Variable extraction from templates |
| `net/http` | sources/cdn.go | CDN download client |
| `text/tabwriter` | cmd/vm/ | CLI table formatting |
| `io` (stdlib) | cmd/vm/cmd_container.go, devenv/ | `io.Copy` for streaming log output |

## Third-party

| Module | Version | Used in |
|--------|---------|---------|
| `gopkg.in/yaml.v3` | v3.0.1 | devenv/test.go (package manager config parsing) |

## AX-6 exemptions

The following stdlib imports are structural exemptions (no `github.com/dappcore/go` equivalent exists):
- `context.Context` — process cancellation and timeout primitives
- `encoding/hex` — stable hex string encoding for container IDs
- `crypto/*` — no core crypto primitives
- `text/tabwriter` — CLI table formatting
- `io.Copy` — streaming process output
- `runtime.GOOS` / `runtime.GOARCH` — build-time platform constants
- `embed` / `regexp` — no core equivalents for template embedding / regex
