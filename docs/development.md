---
title: Development
description: How to build, test, and contribute to go-container.
---

# Development

## Prerequisites

- **Go 1.26+** -- The module uses Go 1.26 features.
- **Go workspace** -- This module is part of a Go workspace at `~/Code/go.work`. Local development of sibling modules (go-io, config, go-i18n, cli) requires the workspace file.

Optional (for actually running VMs):

- **QEMU** -- `qemu-system-x86_64` for running LinuxKit images on any platform.
- **Hyperkit** -- macOS-only alternative hypervisor.
- **LinuxKit** -- For building images from templates (`linuxkit build`).
- **GitHub CLI** (`gh`) -- For the GitHub image source.


## Running tests

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# Single test by name
go test -run TestState_Add_Good ./...

# Single package
go test ./sources/
go test ./devenv/
```

Tests use `testify` for assertions. Most tests are self-contained and do not require a running hypervisor -- they test command construction, state management, template parsing, and configuration loading in isolation.


## Test naming convention

Tests follow a `_Good`, `_Bad`, `_Ugly` suffix pattern:

| Suffix | Meaning |
|--------|---------|
| `_Good` | Happy path -- valid inputs, expected success |
| `_Bad` | Expected error conditions -- invalid inputs, missing resources |
| `_Ugly` | Edge cases, panics, and boundary conditions |

Examples from the codebase:

```go
func TestNewState_Good(t *testing.T)           { /* creates state successfully */ }
func TestLoadState_Bad_InvalidJSON(t *testing.T) { /* handles corrupt state file */ }
func TestGetHypervisor_Bad_Unknown(t *testing.T) { /* rejects unknown hypervisor name */ }
```


## Project structure

```
go-container/
  container.go          # Container struct, Manager interface, Status, RunOptions, ImageFormat
  hypervisor.go         # Hypervisor interface, QemuHypervisor, HyperkitHypervisor, DetectHypervisor
  hypervisor_test.go
  linuxkit.go           # LinuxKitManager (Manager implementation), followReader for log tailing
  linuxkit_test.go
  state.go              # State persistence (containers.json), log paths
  state_test.go
  templates.go          # Template listing, loading, variable substitution, user template scanning
  templates_test.go
  templates/            # Embedded LinuxKit YAML templates
    core-dev.yml
    server-php.yml
  sources/
    source.go           # ImageSource interface, SourceConfig
    source_test.go
    cdn.go              # CDNSource implementation
    cdn_test.go
    github.go           # GitHubSource implementation
    github_test.go
  devenv/
    devops.go           # DevOps orchestrator, Boot, Stop, Status, ImageName, ImagePath
    devops_test.go
    config.go           # Config, ImagesConfig, LoadConfig from ~/.core/config.yaml
    config_test.go
    images.go           # ImageManager, Manifest, Install, CheckUpdate
    images_test.go
    shell.go            # Shell (SSH and serial console)
    shell_test.go
    serve.go            # Serve (mount project, auto-detect serve command)
    serve_test.go
    test.go             # Test (auto-detect test command, .core/test.yaml)
    test_test.go
    claude.go           # Claude sandbox session with auth forwarding
    claude_test.go
    ssh_utils.go        # Host key scanning for ~/.core/known_hosts
  cmd/vm/
    cmd_vm.go           # CLI registration (init + AddVMCommands)
    cmd_commands.go     # Package doc
    cmd_container.go    # run, ps, stop, logs, exec commands
    cmd_templates.go    # templates, templates show, templates vars commands
```


## Coding standards

- **UK English** in all strings, comments, and documentation (colour, organisation, honour).
- **Strict typing** -- All function parameters and return values are typed. No `interface{}` without justification.
- **Error wrapping** -- Use `fmt.Errorf("context: %w", err)` for all error returns.
- **`io.Medium` abstraction** -- File system operations go through `io.Medium` (from `go-io`) rather than directly calling `os` functions. This enables testing with mock file systems. The `io.Local` singleton is used for real file system access.
- **Compile-time interface checks** -- Use `var _ Interface = (*Impl)(nil)` to verify implementations at compile time (see `sources/cdn.go` and `sources/github.go`).
- **Context propagation** -- All operations that might block accept a `context.Context` as their first parameter.


## Adding a new hypervisor

1. Create a new struct implementing the `Hypervisor` interface in `hypervisor.go`:

```go
type MyHypervisor struct {
    Binary string
}

func (h *MyHypervisor) Name() string { return "my-hypervisor" }
func (h *MyHypervisor) Available() bool { /* check if binary exists */ }
func (h *MyHypervisor) BuildCommand(ctx context.Context, image string, opts *HypervisorOptions) (*exec.Cmd, error) {
    // Build and return exec.Cmd
}
```

2. Register it in `DetectHypervisor()` and `GetHypervisor()` in `hypervisor.go`.

3. Add tests following the `_Good`/`_Bad` naming convention.


## Adding a new image source

1. Create a new struct implementing `ImageSource` in the `sources/` package:

```go
type MySource struct {
    config SourceConfig
}

var _ ImageSource = (*MySource)(nil)  // Compile-time check

func (s *MySource) Name() string { return "my-source" }
func (s *MySource) Available() bool { /* check prerequisites */ }
func (s *MySource) LatestVersion(ctx context.Context) (string, error) { /* fetch version */ }
func (s *MySource) Download(ctx context.Context, m io.Medium, dest string, progress func(downloaded, total int64)) error {
    // Download image to dest
}
```

2. Wire it into `NewImageManager()` in `devenv/images.go` under the appropriate source selector.


## Adding a new LinuxKit template

### Built-in template

1. Create a `.yml` file in the `templates/` directory.
2. Add an entry to `builtinTemplates` in `templates.go`.
3. The file will be embedded via the `//go:embed templates/*.yml` directive.

### User template

Place a `.yml` file in either:

- `.core/linuxkit/` relative to your project root (workspace-scoped)
- `~/.core/linuxkit/` in your home directory (global)

The first comment line in the YAML file is extracted as the template description.


## File system paths

All persistent data lives under `~/.core/`:

| Path | Purpose |
|------|---------|
| `~/.core/containers.json` | Container state registry |
| `~/.core/logs/` | Per-container log files |
| `~/.core/images/` | Downloaded dev environment images |
| `~/.core/images/manifest.json` | Image version manifest |
| `~/.core/config.yaml` | Global configuration |
| `~/.core/known_hosts` | SSH host keys for dev VMs |
| `~/.core/linuxkit/` | User-defined LinuxKit templates |

The `CORE_IMAGES_DIR` environment variable overrides the default images directory.


## Licence

EUPL-1.2
