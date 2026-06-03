<!-- SPDX-License-Identifier: EUPL-1.2 -->

# `core vm` Image + Lifecycle Commands — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (inline). Steps use `- [ ]` tracking.

**Goal:** Expose the 8 existing `AppleProvider` methods as flat `core vm` subcommands — `build/pull/push/images/rmi` (Apple-only) and `kill/rm/inspect` (dispatch by owner).

**Architecture:** New `cmd/vm/cmd_images.go` holds the Apple-only image group + a `requireApple` guard. `kill/rm/inspect` join `cmd/vm/cmd_container.go` and reuse `resolveContainerOwner`. All register through `AddVMCommands`.

**Tech Stack:** Go 1.26, `dappco.re/go` (`core.Result`, `core.E`, `core.JSONMarshalIndent`, tabwriter via `proc.Stdout`), plain stdlib `testing`.

**Spec:** `docs/superpowers/specs/2026-06-03-vm-image-lifecycle-cli-design.md`

---

### Task 1: `requireApple` guard (TDD'd seam)

**Files:** Create `go/cmd/vm/cmd_images.go`; create `go/cmd/vm/cmd_images_test.go`.

- [ ] **Step 1 — failing test** (`cmd_images_test.go`):
```go
package vm

import (
	"testing"

	core "dappco.re/go"
)

func TestCmdImages_requireApple_Bad(t *testing.T) {
	// On a host without the apple runtime, requireApple must fail with an
	// actionable message rather than returning a provider.
	r := requireApple()
	if r.OK {
		// Apple runtime present (dev mac): must yield a usable provider.
		if core.MustCast[*appleProviderT](r) == nil {
			t.Fatal("OK requireApple returned nil provider")
		}
		return
	}
	if r.Error() == "" {
		t.Fatal("failed requireApple must carry a message")
	}
}
```
(`appleProviderT` is an alias note — use the real type `*container.AppleProvider`; the test imports `container`. Final test uses `core.MustCast[*container.AppleProvider](r)`.)
- [ ] **Step 2 — verify RED:** `go test ./cmd/vm/ -run TestCmdImages_requireApple` build fail "undefined: requireApple".
- [ ] **Step 3 — implement** (`cmd_images.go`):
```go
package vm

import (
	// Note: AX-6 — text/tabwriter is structural for CLI table formatting; no core primitive.
	"text/tabwriter"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/container/internal/proc"
)

// requireApple returns an available AppleProvider, or a Fail explaining that
// the macOS Containerisation runtime is required.
//
// Usage:
//
//	r := requireApple(); if !r.OK { return resultFromError(r.Value.(error)) }
func requireApple() core.Result { // Value: *container.AppleProvider
	p := container.NewAppleProvider()
	if !p.Available() {
		return core.Fail(core.E("vm", "the apple container runtime is not available on this host (requires macOS 26+ and the `container` CLI; run `container system start`)", nil))
	}
	return core.Ok(p)
}
```
Fix the test to `core.MustCast[*container.AppleProvider](r)` and `import "dappco.re/go/container"`.
- [ ] **Step 4 — verify GREEN:** `go test ./cmd/vm/ -run TestCmdImages_requireApple` PASS.
- [ ] **Step 5 — commit:** `feat(cmd/vm): requireApple guard for image commands`.

### Task 2: image group commands

**Files:** Modify `go/cmd/vm/cmd_images.go`; `go/cmd/vm/cmd_images_test.go`.

- [ ] **Step 1 — failing test** (empty-arg guards + table formatter, no binary):
```go
func TestCmdImages_pullEmpty_Bad(t *testing.T) {
	if pullImage("").OK {
		t.Fatal("expected error for empty ref")
	}
}
func TestCmdImages_rmiEmpty_Bad(t *testing.T) {
	if removeImage("").OK {
		t.Fatal("expected error for empty ref")
	}
}
func TestCmdImages_pushEmpty_Bad(t *testing.T) {
	if pushImage("").OK {
		t.Fatal("expected error for empty ref")
	}
}
func TestCmdImages_formatImages_Good(t *testing.T) {
	out := formatImages([]*container.Image{
		{Name: "docker.io/library/alpine:latest", Digest: "sha256:deadbeefcafef00d"},
	})
	if !core.Contains(out, "alpine:latest") || !core.Contains(out, "sha256:deadbeef") {
		t.Fatalf("formatImages missing fields:\n%s", out)
	}
}
```
- [ ] **Step 2 — verify RED:** build fail (pullImage/removeImage/pushImage/formatImages undefined).
- [ ] **Step 3 — implement.** In `cmd_images.go`, add command registrars (`addVMBuildCommand/addVMPullCommand/addVMPushCommand/addVMImagesCommand/addVMRmiCommand`) and the thin handlers returning `core.Result`:
```go
func pullImage(ref string) core.Result {
	if ref == "" {
		return core.Fail(core.E("vm pull", "image reference is required", nil))
	}
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	pr := p.Pull(ref)
	if !pr.OK {
		return pr
	}
	img := core.MustCast[*container.Image](pr)
	core.Print(nil, "%s %s %s", successStyle.Render(vmT("common.status.ok")), img.Name, img.Digest)
	return core.Ok(nil)
}

func pushImage(ref string) core.Result {
	if ref == "" {
		return core.Fail(core.E("vm push", "image reference is required", nil))
	}
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	return p.Push(&container.Image{Path: ref}, ref)
}

func removeImage(ref string) core.Result {
	if ref == "" {
		return core.Fail(core.E("vm rmi", "image reference is required", nil))
	}
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	return p.RemoveImage(ref)
}

func buildImage(context, file, tag string) core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	source := context
	if file != "" {
		source = file
	}
	br := p.Build(container.ContainerConfig{Source: source, Name: tag})
	if !br.OK {
		return br
	}
	img := core.MustCast[*container.Image](br)
	core.Print(nil, "%s %s", successStyle.Render(vmT("common.status.ok")), img.Path)
	return core.Ok(nil)
}

func listImages() core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	lr := p.ListImages()
	if !lr.OK {
		return lr
	}
	imgs := core.MustCast[[]*container.Image](lr)
	if len(imgs) == 0 {
		core.Println(vmT("cmd.vm.images.none"))
		return core.Ok(nil)
	}
	core.Print(nil, "%s", formatImages(imgs))
	return core.Ok(nil)
}

// formatImages renders images as a REPOSITORY/DIGEST table.
func formatImages(imgs []*container.Image) string {
	var b core.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	core.Print(w, "%s", "REPOSITORY\tDIGEST")
	for _, img := range imgs {
		digest := img.Digest
		if len(digest) > 19 {
			digest = digest[:19]
		}
		core.Print(w, "%s\t%s", img.Name, digest)
	}
	_ = w.Flush()
	return b.String()
}
```
(Confirm `core.Builder` exists during build; else use a small `proc`-backed buffer or render line-by-line into a `[]string` joined with `\n`. The command registrars mirror `addVMPsCommand`/`addVMStopCommand`, calling these handlers via `resultFromError`-style wrapping or returning the `core.Result` directly.)
- [ ] **Step 4 — verify GREEN:** the unit tests PASS; `go build ./...`.
- [ ] **Step 5 — commit:** `feat(cmd/vm): vm build/pull/push/images/rmi (Apple image management)`.

### Task 3: lifecycle group commands (kill/rm/inspect)

**Files:** Modify `go/cmd/vm/cmd_container.go`; `go/cmd/vm/cmd_container_test.go`.

- [ ] **Step 1 — failing test** (empty-arg guards + inspect render, no binary):
```go
func TestCmdContainer_killEmpty_Bad(t *testing.T) {
	if killContainer("").OK {
		t.Fatal("expected error for empty id")
	}
}
func TestCmdContainer_rmEmpty_Bad(t *testing.T) {
	if removeContainer("").OK {
		t.Fatal("expected error for empty id")
	}
}
func TestCmdContainer_inspectEmpty_Bad(t *testing.T) {
	if inspectContainer("").OK {
		t.Fatal("expected error for empty id")
	}
}
```
- [ ] **Step 2 — verify RED:** build fail (killContainer/removeContainer/inspectContainer undefined).
- [ ] **Step 3 — implement.** In `cmd_container.go`, add registrars (`addVMKillCommand/addVMRmCommand/addVMInspectCommand`) and handlers (each: empty-arg guard, `resolveContainerOwner`, dispatch):
```go
func killContainer(id string) core.Result {
	if id == "" {
		return core.Fail(core.E("vm kill", vmT("cmd.vm.error.id_required"), nil))
	}
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return core.Fail(err)
	}
	if apple != nil {
		return apple.Kill(fullID)
	}
	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return mgrRes
	}
	return core.MustCast[*container.LinuxKitManager](mgrRes).Stop(context.Background(), fullID)
}

func removeContainer(id string) core.Result {
	if id == "" {
		return core.Fail(core.E("vm rm", vmT("cmd.vm.error.id_required"), nil))
	}
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return core.Fail(err)
	}
	if apple != nil {
		return apple.Remove(fullID)
	}
	mgrRes := container.NewLinuxKitManager(io.Local)
	if !mgrRes.OK {
		return mgrRes
	}
	manager := core.MustCast[*container.LinuxKitManager](mgrRes)
	_ = manager.Stop(context.Background(), fullID) // best-effort stop before remove
	return manager.State().Remove(fullID)
}

func inspectContainer(id string) core.Result {
	if id == "" {
		return core.Fail(core.E("vm inspect", vmT("cmd.vm.error.id_required"), nil))
	}
	apple, fullID, err := resolveContainerOwner(id)
	if err != nil {
		return core.Fail(err)
	}
	var c *container.Container
	if apple != nil {
		ir := apple.Inspect(fullID)
		if !ir.OK {
			return ir
		}
		c = core.MustCast[*container.Container](ir)
	} else {
		mgrRes := container.NewLinuxKitManager(io.Local)
		if !mgrRes.OK {
			return mgrRes
		}
		lr := core.MustCast[*container.LinuxKitManager](mgrRes).List(context.Background())
		if !lr.OK {
			return lr
		}
		for _, x := range core.MustCast[[]*container.Container](lr) {
			if x.ID == fullID {
				c = x
			}
		}
		if c == nil {
			return core.Fail(core.E("vm inspect", "container not found: "+fullID, nil))
		}
	}
	jr := core.JSONMarshalIndent(c, "", "  ")
	if !jr.OK {
		return jr
	}
	core.Println(string(core.MustCast[[]byte](jr)))
	return core.Ok(nil)
}
```
(`State().Remove` returns `core.Result` post-migration; confirm during build. Manager's `Stop` returns `core.Result`.)
- [ ] **Step 4 — verify GREEN:** unit tests PASS; `go build ./...`.
- [ ] **Step 5 — commit:** `feat(cmd/vm): vm kill/rm/inspect lifecycle commands`.

### Task 4: registration + docs

**Files:** Modify `go/cmd/vm/cmd_vm.go` (`AddVMCommands`), `go/cmd/vm/cmd_commands.go`, `specs/RFC.commands.md`.

- [ ] **Step 1:** In `AddVMCommands`, add the 8 `addVM*Command(c)` calls.
- [ ] **Step 2:** Update `cmd_commands.go` docstring + `RFC.commands.md` command tree to list build/pull/push/images/rmi/kill/rm/inspect.
- [ ] **Step 3 — verify:** `go build ./... && go test ./cmd/vm/... .` green; `go vet ./...`. Add `{Good,Bad,Ugly}` triplets + `Example*` for any new EXPORTED symbols (the `addVM*Command`/handlers are lowercase/private → no ax7 requirement; confirm audit). 
- [ ] **Step 4 — commit:** `feat(cmd/vm): register image+lifecycle commands; update command docs`.

### Task 5: live e2e + final gate

**Files:** Modify `go/cmd/vm/cmd_images_test.go` (or apple_test.go) for a gated live smoke.

- [ ] **Step 1 — gated live smoke:** with `CORE_APPLE_E2E=1`: `pullImage(alpine)` → `listImages()` succeeds + (assert via a provider `ListImages` find) → `removeImage(alpine)`. (Driving the handlers directly; they print, so assert the underlying provider round-trip rather than stdout.) Practically, certify via `AppleProvider` directly in apple_test.go if handler stdout makes assertions awkward — the handlers are thin pass-throughs already unit-guarded.
- [ ] **Step 2 — run:** `CORE_APPLE_E2E=1 go test ./... -run 'E2E' -count=1` PASS.
- [ ] **Step 3 — final gate:** `go test ./... -race` green; `go vet ./...`; `bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .` → COMPLIANT.
- [ ] **Step 4 — commit:** `test(cmd/vm): live smoke for image command round-trip`.

---

## Self-Review

- **Spec coverage:** requireApple (T1) ✓; build/pull/push/images/rmi (T2) ✓; kill/rm/inspect dispatch (T3) ✓; registration+docs (T4) ✓; image-Apple-only via requireApple ✓; lifecycle-both via resolveContainerOwner ✓; error handling (guards in T2/T3) ✓; tests + audit (T1-T3, T5) ✓.
- **Placeholders:** the `core.Builder`/`State().Remove`/`core.JSONMarshalIndent` notes are explicit build-time confirmations of existing API, not TBDs — verify each during build and adjust to the real symbol.
- **Type consistency:** handlers return `core.Result`; `requireApple→*container.AppleProvider`; `formatImages([]*container.Image) string`; lifecycle handlers `(string) core.Result`; consistent across tasks.
