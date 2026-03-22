# go-container

Module: `dappco.re/go/core/container`

Container runtime for managing LinuxKit VMs as lightweight containers. Supports running LinuxKit images (ISO, qcow2, vmdk, raw) via QEMU or Hyperkit hypervisors. Includes a dev environment system for Claude Code agents and development workflows.

## Architecture

| File/Dir | Purpose |
|----------|---------|
| `container.go` | `Container`, `Manager` interface, `Status`, `RunOptions`, `ImageFormat` types |
| `hypervisor.go` | `Hypervisor` interface, `QemuHypervisor`, `HyperkitHypervisor`, `DetectHypervisor()` |
| `linuxkit.go` | LinuxKit image building |
| `state.go` | Container state persistence |
| `templates.go` | LinuxKit YAML template management |
| `devenv/` | Development environment: Claude agent, config, Docker, images, shell, SSH, serve, test |
| `sources/` | Image sources: CDN, GitHub, generic source interface |
| `cmd/vm/` | CLI commands: container, templates, VM management |

## Key Types

### Container Runtime

- **`Container`** — Running instance: `ID` (8 hex chars), `Name`, `Image`, `Status`, `PID`, `StartedAt`, `Ports`, `Memory`, `CPUs`
- **`Manager`** interface — `Run()`, `Stop()`, `List()`, `Logs()`, `Exec()`
- **`RunOptions`** — `Name`, `Detach`, `Memory` (MB), `CPUs`, `Ports`, `Volumes`, `SSHPort`, `SSHKey`
- **`Status`** — `StatusRunning`, `StatusStopped`, `StatusError`
- **`ImageFormat`** — `FormatISO`, `FormatQCOW2`, `FormatVMDK`, `FormatRaw`, `FormatUnknown`

### Hypervisors

- **`Hypervisor`** interface — `Name()`, `Available()`, `BuildCommand()`
- **`QemuHypervisor`** — QEMU with KVM (Linux) or HVF (macOS) acceleration, virtio networking, 9p volume shares
- **`HyperkitHypervisor`** — macOS-only Hyperkit with ACPI, virtio-blk, slirp networking
- **`HypervisorOptions`** — `Memory`, `CPUs`, `LogFile`, `SSHPort`, `Ports`, `Volumes`, `Detach`
- **`DetectHypervisor()`** — Auto-selects best available (prefers Hyperkit on macOS, falls back to QEMU)
- **`DetectImageFormat()`** — Determines format from file extension

### Dev Environment (`devenv/`)

- **Claude agent** configuration and management
- **Docker** container operations
- **Image** management (pull, build, cache)
- **Shell** access to containers
- **SSH** utilities for container access
- **Serve** — development server management
- **Test** — container-based test execution

### Image Sources (`sources/`)

- **Source** interface for fetching LinuxKit images
- **CDN source** — Download from CDN
- **GitHub source** — Download from GitHub releases

## Usage

```go
import "dappco.re/go/core/container"

// Auto-detect hypervisor
hv, _ := container.DetectHypervisor()

// Detect image format
format := container.DetectImageFormat("vm.qcow2") // FormatQCOW2

// Generate container ID
id, _ := container.GenerateID() // e.g. "a1b2c3d4"
```

## Dependencies

- No core ecosystem dependencies in root package
- `devenv/` imports SSH and Docker tooling
