---
title: Architecture
description: Internal design of go-container -- types, data flow, hypervisor abstraction, state management, and template engine.
---

# Architecture

go-container is organised into three packages with clear responsibilities. The root `container` package owns the core abstractions. The `devenv` package composes those abstractions into a higher-level development environment. The `sources` package provides pluggable image download backends.

```
container (root)
  |-- Manager interface + LinuxKitManager implementation
  |-- Hypervisor interface (QEMU, Hyperkit)
  |-- State (persistent container registry)
  |-- Template engine (embedded + user templates)
  |
  +-- devenv/
  |     |-- DevOps orchestrator
  |     |-- ImageManager (download, manifest, update checks)
  |     |-- Shell, Serve, Test, Claude sessions
  |     +-- Config (from ~/.core/config.yaml)
  |
  +-- sources/
        |-- ImageSource interface
        |-- CDNSource
        +-- GitHubSource
```


## Key types

### Container

The central data structure representing a running or stopped VM instance.

```go
type Container struct {
    ID        string            `json:"id"`
    Name      string            `json:"name,omitempty"`
    Image     string            `json:"image"`
    Status    Status            `json:"status"`
    PID       int               `json:"pid"`
    StartedAt time.Time         `json:"started_at"`
    Ports     map[int]int       `json:"ports,omitempty"`
    Memory    int               `json:"memory,omitempty"`
    CPUs      int               `json:"cpus,omitempty"`
}
```

Each container gets a unique 8-character hex ID generated from `crypto/rand`. Status transitions follow `running -> stopped | error`.


### Manager interface

The `Manager` interface defines the contract for container lifecycle management:

```go
type Manager interface {
    Run(ctx context.Context, image string, opts RunOptions) (*Container, error)
    Stop(ctx context.Context, id string) error
    List(ctx context.Context) ([]*Container, error)
    Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error)
    Exec(ctx context.Context, id string, cmd []string) error
}
```

The only implementation is `LinuxKitManager`, which delegates VM execution to a `Hypervisor` and tracks state in a `State` store.


### Hypervisor interface

Abstracts the underlying virtualisation technology:

```go
type Hypervisor interface {
    Name() string
    Available() bool
    BuildCommand(ctx context.Context, image string, opts *HypervisorOptions) (*exec.Cmd, error)
}
```

Two implementations exist:

| Implementation | Platform | Acceleration | Binary |
|----------------|----------|-------------|--------|
| `QemuHypervisor` | All | KVM (Linux), HVF (macOS) | `qemu-system-x86_64` |
| `HyperkitHypervisor` | macOS only | Native macOS hypervisor | `hyperkit` |

`DetectHypervisor()` auto-selects the best available hypervisor. On macOS it prefers Hyperkit, falling back to QEMU. On Linux it uses QEMU with KVM if `/dev/kvm` is present.


## Data flow: running a container

When `LinuxKitManager.Run()` is called, the following sequence occurs:

1. **Validate** -- Checks the image file exists via `io.Medium` and detects its format from the file extension (`.iso`, `.qcow2`, `.vmdk`, `.raw`, `.img`).

2. **Generate ID** -- Creates an 8-character hex identifier using `crypto/rand`.

3. **Apply defaults** -- Memory defaults to 1024 MB, CPUs to 1, SSH port to 2222.

4. **Build command** -- Delegates to the `Hypervisor.BuildCommand()` method, which constructs the full command line including:
   - Memory and CPU allocation
   - Disk image attachment (format-specific flags)
   - Network with port forwarding (SSH + user-defined ports)
   - 9p volume shares (QEMU only)
   - Hardware acceleration flags

5. **Start process** -- In **detached** mode, stdout/stderr are redirected to a log file under `~/.core/logs/<id>.log`, and a background goroutine monitors the process for exit. In **foreground** mode, output is tee'd to both the log file and the terminal.

6. **Persist state** -- The container record is written to `~/.core/containers.json` via the `State` store.

7. **Monitor** -- For detached containers, `waitForExit()` runs in a goroutine, updating the container status to `stopped` or `error` when the process terminates.


## State management

The `State` struct provides a thread-safe, JSON-persisted container registry:

```go
type State struct {
    Containers map[string]*Container `json:"containers"`
    mu         sync.RWMutex
    filePath   string
}
```

Key design decisions:

- **Copy-on-read**: `Get()` and `All()` return copies of container structs to prevent data races when callers modify the returned values.
- **Write-through**: Every mutation (`Add`, `Update`, `Remove`) immediately persists to disk via `SaveState()`.
- **Auto-create**: `LoadState()` returns an empty state if the file does not exist, and `SaveState()` creates parent directories as needed.

Default paths:

| Path | Purpose |
|------|---------|
| `~/.core/containers.json` | Container state file |
| `~/.core/logs/<id>.log` | Per-container log files |


## Stopping a container

`LinuxKitManager.Stop()` performs a graceful shutdown:

1. Sends `SIGTERM` to the hypervisor process.
2. Waits up to 10 seconds for the process to exit.
3. If the process does not exit in time, sends `SIGKILL`.
4. Respects context cancellation throughout -- if the context is cancelled, the process is killed immediately.


## Template engine

LinuxKit templates are YAML files that define a complete VM image configuration (kernel, init, services, files). The template engine adds variable substitution on top.

### Variable syntax

Two forms are supported:

- `${VAR}` -- Required variable. Produces an error if not provided.
- `${VAR:-default}` -- Optional variable with a default value.

### Resolution order

`GetTemplate(name)` searches for templates in this order:

1. **Embedded templates** -- Compiled into the binary via `//go:embed templates/*.yml`. These always take precedence.
2. **Workspace templates** -- `.core/linuxkit/` relative to the current working directory.
3. **User templates** -- `~/.core/linuxkit/` in the user's home directory.

User-defined templates that share a name with a built-in template are ignored (built-ins win).

### Variable extraction

`ExtractVariables(content)` parses a template and returns two collections:

- A sorted slice of required variable names (those using `${VAR}` syntax with no default).
- A map of optional variable names to their default values (those using `${VAR:-default}` syntax).

This powers the `core vm templates vars <name>` command.


## Development environment (devenv)

The `DevOps` struct in the `devenv` package composes the lower-level primitives into a complete development workflow.

### Boot sequence

1. Checks the dev image is installed (platform-specific qcow2 file: `core-devops-{os}-{arch}.qcow2`).
2. Launches the image via `LinuxKitManager.Run()` in detached mode with 4096 MB RAM, 2 CPUs, and SSH on port 2222.
3. Polls for up to 60 seconds until the VM's SSH host key can be scanned, then writes it to `~/.core/known_hosts`.

### Shell access

Two modes are available:

- **SSH** (default) -- Connects via `ssh -A -p 2222 root@localhost` with agent forwarding and strict host key checking against `~/.core/known_hosts`.
- **Serial console** -- Attaches to the QEMU serial console socket via `socat`.

### Project mounting

Projects are mounted into the VM at `/app` using a reverse SSHFS tunnel. The VM opens an SSH reverse tunnel back to the host (port 10000) and mounts the host directory via SSHFS.

### Auto-detection

Several operations auto-detect the project type by inspecting files on disk:

| File detected | Serve command | Test command |
|---------------|---------------|--------------|
| `artisan` | `php artisan octane:start` | -- |
| `package.json` with `dev` script | `npm run dev -- --host 0.0.0.0` | `npm test` |
| `composer.json` with `test` script | `frankenphp php-server` | `composer test` |
| `go.mod` | `go run .` | `go test ./...` |
| `manage.py` | `python manage.py runserver` | -- |
| `pytest.ini` or `pyproject.toml` | -- | `pytest` |

Auto-detection can be overridden with `.core/test.yaml` for tests or explicit `--command` flags.

### Claude integration

`DevOps.Claude()` starts a sandboxed Claude session inside the VM:

1. Auto-boots the dev environment if not running.
2. Mounts the project directory at `/app`.
3. Forwards authentication credentials (Anthropic API key, GitHub CLI config, SSH agent, git identity) based on configurable options.
4. Launches `claude` inside the VM via SSH with agent forwarding.


## Image sources (sources)

The `ImageSource` interface defines how dev environment images are downloaded:

```go
type ImageSource interface {
    Name() string
    Available() bool
    LatestVersion(ctx context.Context) (string, error)
    Download(ctx context.Context, m io.Medium, dest string, progress func(downloaded, total int64)) error
}
```

| Source | Backend | Availability check |
|--------|---------|--------------------|
| `CDNSource` | HTTP GET from a configured CDN URL | CDN URL is configured |
| `GitHubSource` | `gh release download` via the GitHub CLI | `gh` is installed and authenticated |

The `ImageManager` in `devenv` maintains a `manifest.json` in `~/.core/images/` that tracks installed image versions, SHA256 checksums, download timestamps, and source names. When the source is set to `"auto"` (the default), it tries GitHub first, then CDN.


## Configuration

The `devenv` package reads `~/.core/config.yaml` via the `go-config` library:

```yaml
version: 1
images:
  source: auto          # auto | github | cdn
  github:
    repo: host-uk/core-images
  registry:
    image: ghcr.io/host-uk/core-devops
  cdn:
    url: https://cdn.example.com/images
```

If the file does not exist, sensible defaults are used.


## Licence

EUPL-1.2
