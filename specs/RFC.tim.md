---
Status: Aspirational
parent: code/core/go/container/RFC.md
section: 14
feature: TIM Format Expansion
module: dappco.re/go/container
repo: core/go-container
lang: go
tags:
  - tim
  - stim
  - oci
  - encryption
  - borg
  - sigil
---
# go-container Sub-spec — TIM Format Expansion

> An agent should be able to implement TIM/STIM container format from this document alone.

**Parent:** [RFC.md](RFC.md) §14
**Feature:** OCI config, rootfs structure, STIM encryption, key hierarchy, Borg.DataNode integration

---

## 1. Purpose

TIM (Terminal Isolation Matrix) is the Borg-native container format. This spec expands the overview in the parent RFC §5 into full implementation detail: OCI-compatible config, three-layer rootfs convention, Enchantrix encryption (STIM), sigil key hierarchy, and Borg.DataNode wrapping.

---

## 2. OCI Config

TIM uses a subset of the OCI runtime specification for `config.json`:

```go
// TIMConfig defines the OCI-compatible configuration for a TIM container.
//
//   tim := container.TIMConfig{
//       EntryPoint: []string{"/app/server"},
//       Env:        []string{"CORE_ENV=production"},
//   }
type TIMConfig struct {
    EntryPoint   []string          `json:"entrypoint"`    // Process to run
    Env          []string          `json:"env"`           // Environment variables
    WorkDir      string            `json:"workdir"`       // Working directory
    Mounts       []TIMMount        `json:"mounts"`        // Filesystem mounts
    Capabilities []string          `json:"capabilities"`  // Linux capabilities
    ReadOnly     bool              `json:"readonly"`      // Read-only rootfs
}

// TIMMount defines a filesystem mount point within the container.
//
//   mount := container.TIMMount{Source: "/data", Target: "/app/data", ReadOnly: true}
type TIMMount struct {
    Source   string `json:"source"`    // Host path
    Target   string `json:"target"`    // Container path
    ReadOnly bool   `json:"readonly"`  // Mount as read-only
}
```

---

## 3. Rootfs Structure

TIM rootfs follows a three-layer convention:

```
rootfs/
├── base/       # Minimal distroless layer (libc, ca-certs, tzdata)
├── app/        # Application layer (binary, static assets)
└── data/       # Data layer (config, state — often a mount point)
```

- **base/** — distroless, no shell, no package manager, minimal attack surface
- **app/** — application binary and any bundled assets (immutable after build)
- **data/** — runtime state, config files, databases (writable, often a volume mount)

---

## 4. DataCube as I/O Medium

DataCube IS an `io.Medium` implementation — `io.Cube()` wraps any Medium with Enchantrix encryption. See `code/core/go/io/RFC.md §Medium` for the interface.

---

## 5. STIM Encryption Flow

STIM (Secure TIM) encrypts the rootfs using the Borg sigil chain:

```
TIM Container
    │
    ▼
┌────────────────────────────┐
│  Sigil Key Derivation       │
│  workspace_key               │
│    └─► container_key         │
│         └─► layer_keys[]     │
└────────────────────────────┘
    │
    ▼
STIM Bundle (encrypted rootfs + cleartext config.json)
```

```go
// EncryptTIM encrypts a TIM bundle into a STIM bundle using the Borg sigil chain.
// The key hierarchy derives container-specific keys from the workspace key.
//
//   stim, err := container.EncryptTIM(tim, workspaceKey)
func EncryptTIM(tim *TIMBundle, workspaceKey []byte) (*STIMBundle, error) { }

// DecryptSTIM decrypts a STIM bundle back into a TIM bundle.
//
//   tim, err := container.DecryptSTIM(stim, workspaceKey)
func DecryptSTIM(stim *STIMBundle, workspaceKey []byte) (*TIMBundle, error) { }
```

---

## 6. Key Hierarchy

| Level | Key | Derived From | Encrypts |
|-------|-----|-------------|----------|
| 1 | Workspace key | Borg master sigil | Container keys |
| 2 | Container key | Workspace key + container ID | Layer keys |
| 3 | Layer keys | Container key + layer name | Individual rootfs layers |

Each layer is encrypted independently. This permits selective decryption — the `base/` layer can be shared across containers without decrypting `app/` or `data/`.

---

## 7. Borg.DataNode Integration

Each Borg.DataNode wraps a TIM container. The DataNode provides:

- **Identity** — TIM authenticates the node via Borg sigil (not SSH keys)
- **Isolation** — hardware VM isolation (Apple Containers) or namespace isolation (Linux)
- **Storage** — STIM-encrypted rootfs with per-layer key hierarchy
- **Lifecycle** — DataNode start/stop maps to container start/stop

```go
// DataNode wraps a TIM container with Borg identity and lifecycle.
//
//   node := borg.NewDataNode("worker-01", container.NewTIMProvider())
//   node.Start()  // boots the TIM container with Borg identity
type DataNode struct {
    ID        string
    Provider  container.Provider
    Sigil     borg.Sigil
    Container *container.Container
}
```

---

## 8. Cross-References

| Spec | Relationship |
|------|-------------|
| `code/core/go/io/RFC.md` §Medium | io.Cube() wraps Medium with Enchantrix |
| `code/core/go/container/RFC.apple.md` | Apple provider runs TIM containers |
| `rfc/snider/RFC-BORG-*.md` | Borg sigil chain, DataNode, Enchantrix |

---

## Changelog

| Date | Change |
|------|--------|
| 2026-04-08 | Extracted from RFC.md §14 into standalone sub-spec |
