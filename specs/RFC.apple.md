---
Status: Aspirational
parent: code/core/go/container/RFC.md
sections: [12, 13]
feature: Apple Containers Provider + Runtime Detection
module: dappco.re/go/container
repo: core/go-container
lang: go
tags:
  - apple
  - containers
  - runtime
  - detection
  - macos
---
# go-container Sub-spec — Apple Containers + Runtime Detection

> An agent should be able to implement Apple Containers provider and runtime detection from this document alone.

**Parent:** [RFC.md](RFC.md) §12-13
**Feature:** Apple Containerisation framework provider and multi-runtime detection interface

---

## 1. Apple Containers Provider

Apple's Containerisation framework (macOS 26+, Apache 2.0, Swift, open source) provides hardware-isolated containers via lightweight VMs with sub-second startup. Each container runs in its own VM — genuine hardware-level isolation, not namespace tricks.

Reference: `github.com/apple/container`

### 1.1 Why Apple Containers

1. **Native** — no daemon, no Docker Desktop, built into macOS
2. **Fast** — sub-second cold start via lightweight VMs
3. **Isolated** — each container is a hardware VM, not a namespace
4. **Open** — Apache 2.0 licence, Swift, community contributions welcome
5. **Borg alignment** — Borg.DataNode maps directly to an Apple Container; each Borg node IS a container with hardware isolation

### 1.2 Provider Implementation

```go
// AppleProvider implements the Provider interface using Apple's
// Containerisation framework. Available on macOS 26+ only.
//
//   provider := container.NewAppleProvider()
//   img, err := provider.Build(config)
//   ctr, err := provider.Run(img, container.WithMemory("4G"))
type AppleProvider struct {
    runtime  string   // Path to container CLI binary
    version  string   // Detected framework version
}

func (a *AppleProvider) Build(config ContainerConfig) (*Image, error)          { }
func (a *AppleProvider) Run(image *Image, opts ...RunOption) (*Container, error) { }
func (a *AppleProvider) Encrypt(image *Image, key []byte) (*EncryptedImage, error) { }
func (a *AppleProvider) Decrypt(encrypted *EncryptedImage, key []byte) (*Image, error) { }
```

### 1.3 CLI Integration

```bash
core run app.yml                          # Auto-detected (Apple on macOS 26+)
core run app.yml --runtime=apple          # Explicit Apple runtime
core run app.yml --runtime=docker         # Force Docker instead
core run app.yml --runtime=podman         # Force Podman instead
```

### 1.4 Detection

```go
// IsAppleAvailable checks whether Apple's Containerisation framework
// is present on the current system (macOS 26+).
//
//   if container.IsAppleAvailable() {
//       provider = container.NewAppleProvider()
//   }
func IsAppleAvailable() bool { }
```

---

## 2. Runtime Detection Interface

Automatic runtime detection with priority ordering and capability reporting. The system probes for available container runtimes and selects the best one.

### 2.1 Detection API

```go
// Detect probes the system for available container runtimes.
// Returns the highest-priority runtime found.
// Priority: Apple Containers (native) → Docker → Podman → None.
//
//   rt := container.Detect()
//   fmt.Println(rt.Type)  // "apple", "docker", "podman", or "none"
func Detect() ContainerRuntime { }
```

### 2.2 ContainerRuntime Type

```go
// ContainerRuntime describes a detected container runtime and its capabilities.
//
//   rt := container.Detect()
//   if rt.HasGPU() {
//       opts = append(opts, container.WithGPU(true))
//   }
type ContainerRuntime struct {
    Type     string   // "apple", "docker", "podman", "none"
    Version  string   // Runtime version string
    Path     string   // Path to runtime binary
    caps     uint32   // Capability bitfield
}

func (r ContainerRuntime) HasGPU() bool            { }  // GPU passthrough available
func (r ContainerRuntime) HasNetworkIsolation() bool { } // Network namespace isolation
func (r ContainerRuntime) HasVolumeMounts() bool     { } // Host volume mounting
func (r ContainerRuntime) HasEncryption() bool       { } // Native encrypted storage
func (r ContainerRuntime) IsHardwareIsolated() bool  { } // True VM isolation (not namespaces)
```

### 2.3 Capability Matrix

| Capability | Apple | Docker | Podman |
|------------|-------|--------|--------|
| GPU passthrough | Planned (Metal) | NVIDIA (Linux) | NVIDIA (Linux) |
| Network isolation | Yes (VM) | Yes (namespace) | Yes (namespace) |
| Volume mounts | Yes | Yes | Yes |
| Native encryption | No (use STIM) | No (use dm-crypt) | No |
| Hardware isolation | Yes (VM) | No (namespace) | No (namespace) |
| Sub-second start | Yes | No | No |

### 2.4 All Runtimes

```go
// DetectAll returns every container runtime found on the system,
// ordered by priority (highest first).
//
//   runtimes := container.DetectAll()
//   for _, rt := range runtimes {
//       fmt.Printf("%s %s at %s\n", rt.Type, rt.Version, rt.Path)
//   }
func DetectAll() []ContainerRuntime { }
```

---

## 3. Cross-References

| Spec | Relationship |
|------|-------------|
| `code/core/go/container/RFC.md` §15 | Metal GPU passthrough (depends on Apple runtime) |
| `code/core/go/container/RFC.tim.md` | TIM containers run on Apple provider |
| `code/core/go/mlx/RFC.md` | MLX inference inside Apple containers |

---

## Changelog

| Date | Change |
|------|--------|
| 2026-04-08 | Extracted from RFC.md §12-13 into standalone sub-spec |
