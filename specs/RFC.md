---
Status: Aspirational
module: dappco.re/go/container
repo: core/go-container
lang: go
tier: lib
depends:
  - code/core/go
tags:
  - container
  - docker
  - isolation
  - execution
  - vm
---
# go-container RFC — Docker-Free Container Runtime

> An agent should be able to build and run containers from this document alone.

**Module:** `dappco.re/go/container`
**Repository:** `core/go-container`
**Files:** 34
**Sub-specs:** [Models](RFC.models.md) | [Commands](RFC.commands.md) | [Apple + Runtime Detection](RFC.apple.md) | [TIM Format](RFC.tim.md)

---

## 1. Overview

Container runtime without the daemon overhead. Immutable images with LinuxKit (stable, default) or TIM bundles (experimental). Used by core/dev for portable dev environments and by LEM for model isolation.

**Philosophy:** Default to trusted, battle-tested tech. Our stuff is experimental.

```
┌─────────────────────────────────────────────────────────┐
│                     core run (abstraction)               │
├────────────────────────────┬────────────────────────────┤
│  LinuxKit (default)        │  TIM (experimental)        │
│  ✓ Trusted, battle-tested  │  ⚠ Homegrown              │
│  ✓ Multi-format output     │  ✓ Lightweight             │
│  ✓ dm-crypt encryption     │  ✓ Sigil encryption        │
│  ✓ Community support       │  ⚠ Just us                 │
├────────────────────────────┴────────────────────────────┤
│  containerd / runc (OCI runtime)                        │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Provider Interface

```go
type Provider interface {
    Build(config) → Image
    Run(image, opts) → Container
    Encrypt(image, key) → EncryptedImage
    Decrypt(encrypted, key) → Image
}
```

```bash
core run app.yml                        # LinuxKit (default)
core run app.tim --provider tim         # TIM (experimental)
core run app.yml --provider linuxkit    # Explicit
```

---

## 3. LinuxKit — Default Provider

Builds minimal, immutable Linux distributions. YAML-based, declarative.

### 3.1 Why LinuxKit First

1. **Trust** — Docker Inc + community backed, years of production use
2. **Formats** — ISO, raw disk, AWS AMI, GCP, Azure, qemu, VMware
3. **dm-crypt** — built-in encrypted volume support
4. **Immutable** — read-only base, explicit writable mounts
5. **Composable** — YAML config, pluggable components

### 3.2 LinuxKit YAML Structure

```yaml
kernel:
  image: linuxkit/kernel:6.6.13
  cmdline: "console=tty0"

init:
  - linuxkit/init:v1.0.0
  - linuxkit/runc:v1.0.0
  - linuxkit/containerd:v1.0.0

onboot:           # Sequential boot services
  - name: dhcpcd
    image: linuxkit/dhcpcd:v1.0.0
  - name: dm-crypt
    image: linuxkit/dm-crypt:v1.0.0
    command: ["/usr/bin/crypto", "crypt_dev", "/dev/sda1"]

services:         # Long-running services
  - name: sshd
    image: linuxkit/sshd:v1.0.0

files:            # Baked-in configs (immutable)
  - path: root/.ssh/authorized_keys
    contents: "${SSH_PUBLIC_KEY}"
```

### 3.3 Output Formats

| Format | Use Case |
|--------|----------|
| ISO | Bare metal, USB boot |
| raw | Generic disk image |
| qcow2 | QEMU/KVM |
| AWS AMI | Amazon EC2 |
| GCP image | Google Compute |
| VMware VMDK | VMware ESXi |

### 3.4 dm-crypt Encrypted Storage

```yaml
onboot:
  - name: dm-crypt
    image: linuxkit/dm-crypt:v1.0.0
    command: ["/usr/bin/crypto", "crypt_dev", "/dev/sda1"]
files:
  - path: etc/dm-crypt/key
    contents: "${DM_CRYPT_KEY}"
```

### 3.5 Networking

- **WireGuard VPN** — isolated network namespaces, services run inside VPN
- **VPNKit** — Docker Desktop-style host↔VM networking
- **Vsock** — VM-host communication (expose containerd via vsock)
- **Static IP** — deterministic network config

---

## 4. Use Cases

### Immutable Dev Environment (core-dev)
Baked-in SSH keys, Claude API key, git config. Read-only base, writable workspace mount via overlay.

### Secure Edge Node
dm-crypt for secrets, minimal attack surface. No shell, no package manager.

### Air-Gapped Deployment
Build offline, deploy via USB/ISO. All dependencies baked in.

### Writable Overlay on Immutable Base
Read-only base, ephemeral writable layer. Changes discarded on shutdown. Perfect for safe experimentation.

### Persistent + Ephemeral Hybrid
dm-crypt (persistent secrets) + overlay (ephemeral workspace). Secrets survive reboots, workspace is disposable.

---

## 5. TIM Format (Experimental)

TIM (Terminal Isolation Matrix) — lightweight container format from Borg:

- `config.json` — OCI runtime spec
- `rootfs/` — distroless filesystem
- STIM = encrypted TIM (Sigil/Enchantrix encryption)

---

## 6. Commands

```bash
# Running
core run <config.yml>              # Start from LinuxKit config
core run <app.tim>                 # Start TIM bundle
core ps                            # List running containers
core stop <name>                   # Stop container
core logs <name>                   # View logs
core exec <name> <command>         # Execute in container

# Building
core build --format iso app.yml    # Build ISO
core build --format qcow2 app.yml  # Build QEMU image

# TIM
core tim pack <dir>                # Pack directory into TIM
core tim encrypt <file.tim>        # Encrypt to STIM
core tim decrypt <file.stim>       # Decrypt STIM
core tim inspect <file.tim>        # Show OCI spec
```

---

## 7. Package Structure

```
go-container/
├── container.go        # Manager interface, Container/Status/RunOptions types
├── hypervisor.go       # Hypervisor interface (QEMU, Hyperkit detection)
├── linuxkit.go         # LinuxKitManager implementation
├── state.go            # State persistence (JSON-backed container registry)
├── tim.go              # TIM provider (experimental)
├── templates.go        # Template management for reusable configs
├── cmd/vm/             # CLI commands
│   ├── cmd_vm.go       # VM command root
│   ├── cmd_container.go # run, ps, stop, logs, exec commands
│   ├── cmd_templates.go # template listing, instantiation
│   └── cmd_commands.go  # command registration
├── devenv/             # Portable dev environment (Claude devenv)
│   ├── devops.go       # DevOps orchestration (Boot, Stop, Status)
│   ├── config.go       # Configuration loading
│   ├── images.go       # Image management + version detection
│   ├── test.go         # Test command detection (npm, go, composer)
│   ├── serve.go        # Serve command detection (dev servers)
│   ├── shell.go        # SSH shell + serial console access
│   ├── claude.go       # Claude API integration (auth copying)
│   └── ssh_utils.go    # SSH key management
├── sources/            # Image source management
│   ├── source.go       # Source interface
│   ├── cdn.go          # CDN-based downloads
│   └── github.go       # GitHub release downloads
├── internal/
│   ├── proc/           # Process execution wrapper
│   └── coreutil/       # Utilities (paths, home dir)
```

---

## 8. DevOps Portable Environment

The `devenv` package provides a complete portable development environment using LinuxKit images. Core features:

### Installation & Lifecycle
- `Boot()` — starts the dev VM with configurable memory/CPU
- `Stop()` — graceful VM shutdown
- `IsRunning()` — status check
- `Status()` — detailed DevStatus (IP, SSH port, running services)

### Image Management
- Platform-specific naming: `core-devops-{os}-{arch}.qcow2`
- Manifest-based version tracking
- CDN + GitHub release source support
- `IsInstalled()` / `CheckUpdate()` / `Install()` methods

### Development Commands
- `Shell()` — SSH shell or serial console access
- `Serve()` — detect and run dev servers (npm dev, go run, etc.)
- `Test()` — detect and execute project tests
- `Claude()` — mount Claude workspace + copy GitHub auth

### Test Command Detection
Auto-detects test runners from package files:
- npm: `npm test` or custom `test` script
- Go: `go test ./...`
- PHP: `composer test`

### Serve Command Detection
Auto-detects dev servers:
- npm: `npm run dev` or `npm start`
- Go: `go run .`
- Python: `python manage.py runserver`

---

## 9. Hypervisor Selection

Automatic hypervisor detection with fallback support:

```go
// Detects system capabilities
hv, err := DetectHypervisor()  // QEMU on Linux, Hyperkit on macOS

// Explicit selection
hv, err := GetHypervisor("qemu")
hv, err := GetHypervisor("hyperkit")
```

| Hypervisor | Platform | Features |
|------------|----------|----------|
| QEMU | Linux | KVM acceleration, 9p filesystem sharing, full feature set |
| Hyperkit | macOS | Native hypervisor, VPNKit networking, Vsock |

### Image Format Detection

Auto-detects image format from file extension or magic bytes:
- ISO, qcow2, raw, vmdk, ami

---

## 10. State Persistence

Container registry persisted to `~/.core/containers.json`. Thread-safe operations:

```go
state, err := LoadState(path)
state.Add(container)           // Atomic add + persist
state.Update(container)        // Update + persist
state.Remove(id)               // Remove + persist
containers := state.All()      // Get all copies
```

---

## 11. Logging

Each running container writes to `~/.core/logs/{id}.log`. Hypervisor output captured per container.

---

## 12. Apple Containers Provider

AppleProvider implements the Provider interface using macOS 26+ Containerisation framework (Apache 2.0, Swift). Hardware-isolated VMs with sub-second startup. Includes `IsAppleAvailable()` detection, CLI `--runtime=apple` flag, and Borg.DataNode alignment.

**Full spec:** [RFC.apple.md](RFC.apple.md) §1

---

## 13. Runtime Detection Interface

Automatic runtime detection with priority ordering (Apple → Docker → Podman → None). ContainerRuntime type with capability bitfield (GPU, network isolation, volume mounts, encryption, hardware isolation). `Detect()` and `DetectAll()` functions.

**Full spec:** [RFC.apple.md](RFC.apple.md) §2

---

## 14. TIM Format Expansion

Full expansion of §5. TIMConfig (OCI-compatible config.json subset), three-layer rootfs convention (base/app/data), DataCube as io.Medium, STIM encryption via Borg sigil chain (workspace → container → layer key hierarchy), and Borg.DataNode integration.

**Full spec:** [RFC.tim.md](RFC.tim.md)

---

## 15. Metal GPU Passthrough

Apple Silicon Metal GPU access from within containers. Currently not supported by Apple's framework but architecturally expected.

### 15.1 Current State

Apple Containers run Linux guests in lightweight VMs. The guest does not have access to the host Metal GPU. Apple's roadmap suggests GPU passthrough is a planned feature.

### 15.2 Design (When Available)

```go
// WithGPU requests Metal GPU access for the container.
// Returns an error if the runtime does not support GPU passthrough.
//
//   ctr, err := provider.Run(image, container.WithGPU(true))
//   // Inside the container, go-mlx can access Metal directly
func WithGPU(enabled bool) RunOption { }
```

### 15.3 GPU Capability Detection

```go
// HasGPU reports whether the container runtime supports GPU passthrough.
//
//   rt := container.Detect()
//   if rt.HasGPU() {
//       opts = append(opts, container.WithGPU(true))
//   } else {
//       // Fall back to CPU-only inference
//   }
```

### 15.4 go-mlx Integration

When GPU passthrough is available, go-mlx inside a container accesses Metal directly:

```
┌──────────────────────────────────────┐
│  Apple Container (lightweight VM)     │
│                                       │
│  ┌──────────────┐  ┌──────────────┐  │
│  │  Application  │  │  go-mlx      │  │
│  │              │  │  (MLX/Metal)  │  │
│  └──────────────┘  └──────┬───────┘  │
│                           │ Metal    │
└───────────────────────────┼──────────┘
                            │ GPU passthrough
┌───────────────────────────┼──────────┐
│  macOS Host               ▼          │
│  Apple Silicon GPU (M-series)        │
└──────────────────────────────────────┘
```

Use cases:
- **LEM inference** — model runs inside an isolated container, accesses Metal for fast inference
- **Training** — fine-tuning LoRA adapters inside containers with full GPU access
- **Batch processing** — multiple containers share GPU via Metal's scheduling

---

## 16. Implementation Priority

| Priority | Section | Feature |
|----------|---------|---------|
| P1 | §2, §3, §4 | Provider interface, LinuxKit, use cases (existing) |
| P1 | §13 | Runtime detection interface ✅ |
| P2 | §12 | Apple Containers provider ✅ |
| P2 | §14 | TIM format expansion (OCI config, rootfs, STIM) ✅ |
| P3 | §15 | Metal GPU passthrough (capGPU wired, blocked on Apple) |
| P3 | §9 | Hypervisor selection (existing) |

---

## 17. Changelog

| Date | Change |
|------|--------|
| 2026-04-08 | Extracted §12-13 into RFC.apple.md, §14 into RFC.tim.md; main RFC retains summaries + links |
| 2026-04-08 | Added §14.3 DataCube as I/O Medium (io.Cube wraps Medium with Enchantrix encryption) |
| 2026-04-08 | Added §12 Apple Containers provider, §13 runtime detection, §14 TIM expansion, §15 Metal GPU passthrough, §16 priority, §17 changelog |

---

## 18. Integration

| Package | Purpose |
|---------|---------|
| `core/dev` | Portable dev environments via `core dev` |
| `code/core/go/build` | LinuxKit image builder for releases |
| `LEM` | Model isolation (TIM containers) |
| `Borg` | Encrypted container storage (STIM), DataNode wraps TIM |
| `go-process` | Process lifecycle management |
| `go-mlx` | Metal GPU inference inside containers |

---

## 19. Reference Material

| Resource | Location |
|----------|----------|
| Core framework spec | `code/core/go/RFC.md` |
| I/O Medium interface | `code/core/go/io/RFC.md` |
| Process primitives | `code/core/go/process/RFC.md` |
| Dev environments | `code/core/dev/RFC.md` |
| Apple Containers | `github.com/apple/container` |
