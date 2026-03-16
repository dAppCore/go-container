# Hypervisors

Module: `forge.lthn.ai/core/go-container`

## Interface

```go
type Hypervisor interface {
    Name() string
    Available() bool
    BuildCommand(ctx, image string, opts *HypervisorOptions) (*exec.Cmd, error)
}
```

## QEMU

`NewQemuHypervisor()` — Default binary: `qemu-system-x86_64`.

Features:
- KVM acceleration on Linux (`/dev/kvm`)
- HVF acceleration on macOS (`-accel hvf`)
- Nographic mode with serial console on stdio
- User networking with port forwarding (`hostfwd`)
- Virtio-9p filesystem shares for volumes
- Supports all image formats (ISO as `-cdrom`, others as `-drive`)

## Hyperkit

`NewHyperkitHypervisor()` — macOS-only, default binary: `hyperkit`.

Features:
- ACPI support
- Virtio-blk for disk images
- Slirp networking with port forwarding
- Serial console on stdio
- PCI slot-based device assignment

## Detection

`DetectHypervisor()` priority:
1. Hyperkit (macOS only, if installed)
2. QEMU (all platforms)
3. Error if neither available

`GetHypervisor(name)` returns a specific hypervisor by name ("qemu" or "hyperkit").
