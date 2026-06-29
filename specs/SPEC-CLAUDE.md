# go-container — Docker-Free Container Runtime

> Agent context summary for `plans/code/core/go/container/`. Read this first, then dive into RFC.md.

## What go-container Is

Container runtime without the daemon overhead. Builds and runs immutable Linux images via LinuxKit (default, trusted) or TIM bundles (experimental, from Borg). Used by core/dev for portable dev environments and by LEM for model isolation. Also provides Apple Containers (macOS 26+) and automatic runtime detection.

## Key Facts

- **Module**: `dappco.re/go/container`
- **Repo**: `core/go-container`
- **Files**: 34
- **Providers**: LinuxKit (default), TIM (experimental), Apple Containers (macOS 26+)
- **Hypervisors**: QEMU (Linux, KVM), Hyperkit (macOS, VPNKit)
- **Output formats**: ISO, raw, qcow2, VMware VMDK, AWS AMI, GCP image

## Architecture

### Provider Interface

`Build(config) → Image`, `Run(image, opts) → Container`, `Encrypt(image, key) → EncryptedImage`. CLI: `core run app.yml` (LinuxKit default) or `core run app.tim --provider tim`.

### LinuxKit (Default)

YAML-based declarative config. Minimal immutable Linux distributions. dm-crypt encrypted storage. Networking: WireGuard VPN, VPNKit, Vsock, static IP. Composable init/onboot/services/files sections.

### TIM Format (Experimental)

Terminal Isolation Matrix from Borg. `config.json` (OCI runtime spec) + `rootfs/` (distroless). STIM = encrypted TIM (Sigil/Enchantrix encryption). Three-layer rootfs (base/app/data). DataCube as io.Medium.

### Apple Containers (macOS 26+)

Hardware-isolated VMs with sub-second startup. `IsAppleAvailable()` detection. Priority ordering: Apple → Docker → Podman → None. Capability bitfield (GPU, network isolation, volume mounts, encryption). Metal GPU passthrough architecturally expected.

### DevOps Portable Environment

`devenv` package: Boot/Stop/Status lifecycle, platform-specific images, auto-detect test runners and dev servers, SSH shell + serial console, Claude workspace mounting.

### State Persistence

Container registry at `~/.core/containers.json`. Thread-safe Add/Update/Remove. Logs at `~/.core/logs/{id}.log`.

## Critical Rules

- **This is a code/ spec** — self-contained, no project/ references
- **LinuxKit is default, TIM is experimental** — "default to trusted, battle-tested tech"
- **Provider interface abstracts everything** — same CLI regardless of backend
- **Runtime auto-detection** — Apple → Docker → Podman → None priority chain

## Spec Index

| File | Scope |
|------|-------|
| [RFC.md](RFC.md) | **Main spec** — providers, LinuxKit, TIM, commands, package structure, devenv, hypervisors, state (~350 lines) |
| [RFC.apple.md](RFC.apple.md) | Apple Containers provider + runtime detection interface |
| [RFC.commands.md](RFC.commands.md) | CLI command tree |
| [RFC.models.md](RFC.models.md) | Type definitions |
| [RFC.imports.md](RFC.imports.md) | Import map |
| [RFC.tim.md](RFC.tim.md) | TIM format expansion — config, rootfs, DataCube, STIM encryption |
