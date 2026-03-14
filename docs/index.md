---
title: go-container
description: Container runtime, LinuxKit image builder, and portable development environment management for Go.
---

# go-container

`forge.lthn.ai/core/go-container` provides a container runtime built on LinuxKit and lightweight hypervisors. It manages the full lifecycle of LinuxKit virtual machines -- from building images with embedded templates, to running them via QEMU or Hyperkit, to offering a portable development environment with shell access, project mounting, test execution, and Claude AI integration.

This is **not** a Docker wrapper. It runs real VMs from LinuxKit images (ISO, qcow2, VMDK, raw) using platform-native acceleration (KVM on Linux, HVF on macOS, Hyperkit where available).


## Module path

```
forge.lthn.ai/core/go-container
```

Requires **Go 1.26+**.


## Quick start

### Run a VM from an image

```go
import (
    "context"
    container "forge.lthn.ai/core/go-container"
    "forge.lthn.ai/core/go-io"
)

manager, err := container.NewLinuxKitManager(io.Local)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
c, err := manager.Run(ctx, "/path/to/image.qcow2", container.RunOptions{
    Name:    "my-vm",
    Memory:  2048,
    CPUs:    2,
    SSHPort: 2222,
    Detach:  true,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Started container %s (PID %d)\n", c.ID, c.PID)
```

### Use the development environment

```go
import (
    "forge.lthn.ai/core/go-container/devenv"
    "forge.lthn.ai/core/go-io"
)

dev, err := devenv.New(io.Local)
if err != nil {
    log.Fatal(err)
}

// Boot the dev environment (downloads image if needed)
ctx := context.Background()
err = dev.Boot(ctx, devenv.DefaultBootOptions())

// Open an SSH shell
err = dev.Shell(ctx, devenv.ShellOptions{})

// Run tests inside the VM
err = dev.Test(ctx, "/path/to/project", devenv.TestOptions{})
```

### Build and run from a LinuxKit template

```go
import container "forge.lthn.ai/core/go-container"

// List available templates (built-in + user-defined)
templates := container.ListTemplates()

// Apply variables to a template
content, err := container.ApplyTemplate("core-dev", map[string]string{
    "SSH_KEY":  "ssh-ed25519 AAAA...",
    "MEMORY":   "4096",
    "HOSTNAME": "my-dev-box",
})
```


## Package layout

| Package | Import path | Purpose |
|---------|-------------|---------|
| `container` (root) | `forge.lthn.ai/core/go-container` | Container struct, Manager interface, hypervisor abstraction, LinuxKit manager, state persistence, template engine |
| `devenv` | `forge.lthn.ai/core/go-container/devenv` | Portable dev environment orchestration: boot, shell, serve, test, Claude sandbox, image management |
| `sources` | `forge.lthn.ai/core/go-container/sources` | Image download backends: CDN and GitHub Releases with progress reporting |
| `cmd/vm` | `forge.lthn.ai/core/go-container/cmd/vm` | CLI commands (`core vm run`, `core vm ps`, `core vm stop`, `core vm logs`, `core vm exec`, `core vm templates`) |


## Dependencies

| Module | Purpose |
|--------|---------|
| `forge.lthn.ai/core/go-io` | File system abstraction (`Medium` interface), process utilities |
| `forge.lthn.ai/core/config` | Configuration loading (used by `devenv` for `~/.core/config.yaml`) |
| `forge.lthn.ai/core/go-i18n` | Internationalised UI strings (used by `cmd/vm`) |
| `forge.lthn.ai/core/cli` | CLI framework (used by `cmd/vm` for command registration) |
| `github.com/stretchr/testify` | Test assertions |
| `gopkg.in/yaml.v3` | YAML parsing for test configuration |

The root `container` package has only two direct dependencies: `go-io` and the standard library. The `devenv` and `cmd/vm` packages pull in the heavier dependencies.


## CLI commands

When registered via `cmd/vm`, the following commands become available under `core vm`:

| Command | Description |
|---------|-------------|
| `core vm run [image]` | Run a VM from an image file or `--template` |
| `core vm ps` | List running VMs (`-a` for all including stopped) |
| `core vm stop <id>` | Stop a running VM by ID or name (supports partial matching) |
| `core vm logs <id>` | View VM logs (`-f` to follow) |
| `core vm exec <id> <cmd>` | Execute a command inside the VM via SSH |
| `core vm templates` | List available LinuxKit templates |
| `core vm templates show <name>` | Display a template's full YAML |
| `core vm templates vars <name>` | Show a template's required and optional variables |


## Built-in templates

Two LinuxKit templates are embedded in the binary:

- **core-dev** -- Full development environment with Go, Node.js, PHP, Docker-in-LinuxKit, and SSH access
- **server-php** -- Production PHP server with FrankenPHP, Caddy reverse proxy, and health checks

User-defined templates can be placed in `.core/linuxkit/` (workspace-relative) or `~/.core/linuxkit/` (global). They are discovered automatically and merged with the built-in set.


## Licence

EUPL-1.2. See [LICENSE](../LICENSE) for the full text.
