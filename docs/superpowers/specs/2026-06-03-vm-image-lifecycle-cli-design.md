<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — `core vm` image + lifecycle commands

**Date:** 2026-06-03
**Module:** `dappco.re/go/container` (code under `go/`)
**Status:** Approved (decisions locked with Snider 2026-06-03)
**Roadmap:** Feature ② of the macOS-user series (① run flags ✅ → ② image/lifecycle CLI → ③ system cmd + interactive shell).

## 1. Goal

The `AppleProvider` already implements `Build / Pull / Push / ListImages /
RemoveImage / Kill / Remove / Inspect` (all live-certified), but none are exposed
on the CLI — `core vm` offers only run/ps/stop/logs/exec/templates. macOS users
must drop to the raw `container` binary for image management and for
kill/rm/inspect. This feature wires those 8 methods to flat `core vm`
subcommands.

## 2. Locked decisions

| Decision | Choice |
|---|---|
| Scope | All 8 commands in one cycle. |
| Image group (build/pull/push/images/rmi) | **Apple-only** (LinuxKit has no OCI image management; its build path is `templates`/`linuxkit build`). |
| Lifecycle group (kill/rm/inspect) | **Both runtimes**, dispatched by owner via `resolveContainerOwner` (like stop/logs/exec). |
| Naming | Flat, matching `run/ps/stop/logs/exec`. |
| File layout | New `cmd/vm/cmd_images.go` for the image group; kill/rm/inspect added to `cmd/vm/cmd_container.go`. |

## 3. Command surface

| Command | Args/flags | Backend |
|---|---|---|
| `vm build` | `[--tag NAME] [--file Containerfile] [context]` | Apple `Build(ContainerConfig{Source, Name})` |
| `vm pull` | `<ref>` | Apple `Pull(ref)` |
| `vm push` | `<ref>` | Apple `Push(&Image{Path: ref}, ref)` (ref must be tagged locally) |
| `vm images` | — | Apple `ListImages()` → table |
| `vm rmi` | `<ref>` | Apple `RemoveImage(ref)` |
| `vm kill` | `<id>` | dispatch: Apple `Kill(id)` / LinuxKit `Manager.Stop` |
| `vm rm` | `<id>` | dispatch: Apple `Remove(id)` / LinuxKit `Stop`-if-running + `State().Remove(id)` |
| `vm inspect` | `<id>` | dispatch: Apple `Inspect(id)` / LinuxKit `List`+filter |

## 4. Image group (`cmd/vm/cmd_images.go`, Apple-only)

`requireApple()` returns an available provider or a clear failure:
```go
// requireApple returns an available AppleProvider, or a Fail explaining that
// the macOS Containerisation runtime is required.
func requireApple() core.Result { // Value: *container.AppleProvider
	p := container.NewAppleProvider()
	if !p.Available() {
		return core.Fail(core.E("vm", "the apple container runtime is not available on this host (requires macOS 26+ and the `container` CLI; run `container system start`)", nil))
	}
	return core.Ok(p)
}
```
Each command: `r := requireApple(); if !r.OK { return resultFromError(...) }; p := core.MustCast[*container.AppleProvider](r)` then call the method, surfacing failures via the command's `core.Result`.

- **build:** positional `context` (default `.`), `--tag` → `Name`, `--file` → goes into `ContainerConfig.Source` when set (else context). Map to `Build(ContainerConfig{Source: file-or-context, Name: tag})`; on success print the image id/digest.
- **pull:** empty-ref guard; print pulled image ref + digest.
- **push:** empty-ref guard; `Push(&Image{Path: ref}, ref)`.
- **images:** `ListImages()` → tabwriter table with header `REPOSITORY  DIGEST` (digest shortened); "no images" message when empty. Mirrors `listContainers` styling.
- **rmi:** empty-ref guard; `RemoveImage(ref)`; success message.

## 5. Lifecycle group (`cmd/vm/cmd_container.go`, dispatch by owner)

Each reads `args[0]` (empty-arg guard), calls `resolveContainerOwner(id)` →
`(apple *AppleProvider, fullID string, err error)`:
- **kill:** apple → `apple.Kill(fullID)`; else `manager.Stop(ctx, fullID)` (LinuxKit Stop escalates to SIGKILL). Status message.
- **rm:** apple → `apple.Remove(fullID)`; else: if the LinuxKit container is running, `manager.Stop(ctx, fullID)` first, then `manager.State().Remove(fullID)`. Success message.
- **inspect:** apple → `core.MustCast[*Container](apple.Inspect(fullID))`; else find `fullID` in `manager.List(ctx)`. Render the `*Container` with `core.JSONMarshalIndent(c, "", "  ")` to stdout.

`resolveContainerOwner` already builds a LinuxKit manager in its fallback; the
lifecycle handlers re-resolve the manager when they need it (small, local — no
new Manager interface methods).

## 6. Registration & docs

- `AddVMCommands` (cmd_vm.go) registers the 8 new commands (add `addVMBuildCommand`, `addVMPullCommand`, `addVMPushCommand`, `addVMImagesCommand`, `addVMRmiCommand` in cmd_images.go; `addVMKillCommand`, `addVMRmCommand`, `addVMInspectCommand` in cmd_container.go).
- `cmd/vm/cmd_commands.go` package docstring + `specs/RFC.commands.md` command tree updated to the full surface.

## 7. Error handling

Image ops: `requireApple` failure when the runtime is absent. Empty-arg guards on pull/push/rmi/kill/rm/inspect → `core.Fail(core.E("vm <cmd>", "...required", nil))`. Lifecycle: existing `resolveContainerOwner` no-match / multiple-match errors. Provider/Manager failures wrapped with `core.E`.

## 8. Testing

- **Unit (no binary):** `requireApple` returns a failed Result when `Available()` is false (the common CI case → also lets us assert the message); empty-arg guards on each command's handler; the images-table formatter and the inspect JSON render (pure helpers fed a `*Container`/`[]*Image`). Mirror the repo's plain-stdlib test style; full `{Good,Bad,Ugly}` triplets + `Example*` for any new public symbols (keep audit COMPLIANT).
- **Live (CORE_APPLE_E2E=1):** `pull alpine → images (find it) → rmi`; and `run (sleep) → inspect (id matches) → kill → rm`. Assert via the parsed results.
- Full module `go test ./... -race` + `go vet` green; `audit.sh` → COMPLIANT (0).

## 9. Out of scope / follow-ups

- `vm registry login` (private registries) — Feature ④ territory.
- Interactive `exec -it` / `vm shell` — Feature ③.
- `vm image ls`-style subgroup naming — flat naming chosen for consistency.
- Build args / multi-stage / platform flags on `vm build` — minimal `--tag`/`--file` only here.
