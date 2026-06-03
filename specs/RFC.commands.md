---
Status: Aspirational
---

# go-container — Commands

> CLI command registrations and their implementation files in `cmd/vm/`.

## Command tree

```
core vm
├── run         # Run from image/OCI-ref or template (--publish/--volume/--env)
├── ps          # List running/stopped containers (aggregated across runtimes)
├── stop        # Stop a running container by ID
├── kill        # Kill a running container (SIGKILL)
├── rm          # Remove a container
├── logs        # View container log output (--follow for tail)
├── exec        # Execute command inside container
├── inspect     # Show detailed container information (JSON)
├── build       # Build an OCI image from a Containerfile (apple runtime)
├── pull        # Pull an image from a registry (apple runtime)
├── push        # Push a locally-tagged image to a registry (apple runtime)
├── images      # List images (apple runtime)
├── rmi         # Remove an image (apple runtime)
├── templates   # List templates
│   ├── show    # Display a template's YAML content
│   └── vars    # Show required/optional variables for a template
└── system      # Manage the Apple container system (apple runtime)
    ├── start    # Start apiserver + kernel (--no-kernel-install to skip)
    ├── status   # Show system status
    └── stop     # Stop system services
```

## File map

| File | Purpose |
|------|---------|
| `cmd/vm/cmd_vm.go` | Command root (`AddVMCommands`), style vars, CLI helpers (`optionArgs`, `optionStrings`, `resultFromError`, `vmT`) |
| `cmd/vm/cmd_commands.go` | Package docstring listing all 6 subcommands |
| `cmd/vm/cmd_container.go` | `run`, `ps`, `stop`, `kill`, `rm`, `logs`, `exec`, `inspect` — container lifecycle. `resolveRuntime`, `runContainer`/`runContainerApple` (RunOptions + `--publish`/`--volume`/`--env`), `resolveContainerOwner` (Apple-first dispatch), `shortID`, port/volume/env parse helpers; kill/rm/inspect dispatch by owner |
| `cmd/vm/cmd_images.go` | `build`, `pull`, `push`, `images`, `rmi` — Apple-only OCI image management (`requireApple` guard, 1:1 over AppleProvider methods, `formatImages` table) |
| `cmd/vm/cmd_system.go` | `system start/status/stop` — Apple-only system service management over `AppleProvider.SystemStart/SystemStop/SystemStatus` |
| `cmd/vm/cmd_templates.go` | `templates`, `templates show`, `templates vars` — LinuxKit template management. Includes `RunFromTemplate` (apply → build → run), `buildLinuxKitImage`, `findBuiltImage`, `lookupLinuxKit`, `ParseVarFlags` |

## Command registration

All commands register via `cli.RegisterCommands(AddVMCommands)` in `cmd_vm.go` init(). The `AddVMCommands` function registers the 'vm' root and its 6 subcommands on a `*core.Core` instance.

## Runtime resolution

The `--runtime` flag accepts: `auto` (default, uses `container.Detect()`), `apple`, `docker`, `podman`, `linuxkit`, `tim` (routes to LinuxKit). Runtime auto-detection follows the priority chain: Apple → Docker → Podman → LinuxKit → None.

## Template flow

```
core vm run --template <name> --var KEY=VALUE
  → ApplyTemplate(name, vars)    # template substitution
  → linuxkit build --format iso-bios --name <out> <yml>
  → findBuiltImage(base)         # locate .iso/.qcow2/.raw/.vmdk
  → manager.Run(image, opts)     # boot via hypervisor
```
