<!-- SPDX-License-Identifier: EUPL-1.2 -->

# CoreGO v0.10.3 Foundations (W2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring `dappco.re/go/container` onto the modern CoreGO idiom so `audit.sh` reports **0 findings**, with public interfaces returning `core.Result`.

**Architecture:** Convert in layers following the import graph. The interface conversion is the cross-cutting spine; `sources`+`internal/*` are independent of the root package and convert concurrently; `devenv`+`cmd/vm` are downstream and convert after root. Two subagents work disjoint compile units in isolated worktrees; the lead owns interface decisions, `go.mod`, merges, and the final audit gate.

**Tech Stack:** Go 1.26, `dappco.re/go` v0.10.3 (`core.Result`, `core.E`, `core.JSON*`), `dappco.re/go/io` (`io.Medium`/`io.Local`), CoreGO AssertX/RequireX test helpers, `audit.sh` as the acceptance gate.

---

## Reference — verified CoreGO v0.10.3 API (use these exact signatures)

```go
// Result — dappco.re/go/result.go + options.go
type Result struct { Value any; OK bool }
func Ok(v any) Result              // Result{Value:v, OK:true}
func Fail(err error) Result        // Result{Value:err, OK:false}
func ResultOf(v any, err error) Result   // adapts (value, error)
func Cast[T any](r Result) (T, bool)      // typed extract, ok=false if !OK or wrong type
func MustCast[T any](r Result) T          // typed extract, panics on !OK/mismatch (init/test only)
func (r Result) Error() string            // "" when OK
func (r Result) Code() string             // stable code e.g. "fs.notfound" (only for *core.Err)

// Errors — dappco.re/go/error.go
func E(op, msg string, err error) error   // returns *core.Err

// JSON — dappco.re/go/json.go  (Value carries the result; check .OK)
func JSONMarshal(v any) Result            // Value: []byte
func JSONUnmarshal(data []byte, target any) Result   // unmarshals into target ptr

// Filesystem — dappco.re/go/io (io.Local is a Medium); replaces os.*
io.Local.Read(path string) (string, error)
io.Local.Write(path, content string) error
io.Local.WriteMode(path, content string, mode fs.FileMode) error
io.Local.Stat(path string) (fs.FileInfo, error)
io.Local.Exists(path string) bool
io.Local.IsFile(path string) bool
```

Caller idiom: `r := p.Build(cfg); if !r.OK { return r }; img := core.MustCast[*Image](r)`
(or `img, ok := core.Cast[*Image](r)` when a soft failure is wanted).

---

## Execution model (layered, 2-agent concurrent Round 1)

Import graph (verified): `coreutil`(leaf) ← `proc` ← {`sources`} and ← `container`(root) ← {`devenv`,`cmd/vm`}. `sources` does **not** import root.

| Round | Owner | Compile units | Concurrency |
|---|---|---|---|
| 0 | Lead | branch, baseline audit, worktrees | — |
| 1 | **Agent A** | root package (`go/*.go`) | concurrent with B |
| 1 | **Agent B** | `go/internal/coreutil`, `go/internal/proc`, `go/sources` | concurrent with A |
| 2 | Lead (or 2 agents) | `go/devenv`, `go/cmd/vm` (against converted root) | after Round 1 merge |
| 3 | Lead | merge, `audit.sh` → 0, `go test ./... -race` | — |

Worktrees via `superpowers:using-git-worktrees`. Disjoint file ownership → clean merges. Lead solely edits `go.mod`/`go.sum` (`go mod tidy` once at the end).

---

## Transform recipes (the canon — apply across owned files; TDD each change)

### R1 — legacy-log → `core.E` (19 sites)
```go
// BEFORE
import coreerr "dappco.re/go/log"
return nil, coreerr.E("AppleProvider.Build", "msg", err)
// AFTER  (drop the coreerr import; core is already imported as `core "dappco.re/go"`)
return core.Fail(core.E("AppleProvider.Build", "msg", err))   // when the func now returns Result
```
The op/msg/cause arguments are unchanged — only the package (`coreerr.` → `core.`) and, where the function is being Result-converted, the wrapping (`core.Fail(core.E(...))`).

### R2 — `func … error` → `func … core.Result` (err-shape, 8 sites + AppleProvider methods)
```go
// BEFORE
func (a *AppleProvider) Stop(id string) error {
    if id == "" { return coreerr.E("AppleProvider.Stop", "container id is required", nil) }
    if err := cmd.Run(); err != nil { return coreerr.E("AppleProvider.Stop", "stop container", err) }
    return nil
}
// AFTER
func (a *AppleProvider) Stop(id string) core.Result {
    if id == "" { return core.Fail(core.E("AppleProvider.Stop", "container id is required", nil)) }
    if err := cmd.Run(); err != nil { return core.Fail(core.E("AppleProvider.Stop", "stop container", err)) }
    return core.Ok(nil)
}
```
Predicates that return `bool` (`Available`, `IsAppleAvailable`, `isAppleSilicon`) and pure getters (`Tracked() []*Container`, `Name() string`) **stay as-is** — Result conversion applies only to `… error` / `(…, error)` shapes.

### R3 — tuple `(*T, error)` → single `core.Result` (3 sites + interface methods)
```go
// BEFORE
func (a *AppleProvider) Build(config ContainerConfig) (*Image, error) {
    ...
    return &Image{...}, nil
}
// AFTER
func (a *AppleProvider) Build(config ContainerConfig) core.Result {
    ...
    return core.Ok(&Image{...})
}
// CALLER BEFORE:  img, err := p.Build(cfg); if err != nil { return ..., err }
// CALLER AFTER:   r := p.Build(cfg); if !r.OK { return r }; img := core.MustCast[*Image](r)
```

### R4 — banned imports → wrappers (4 sites, all in apple.go/apple_test.go)
```go
// os.ReadFile(path)            → r := io.Local.Read(path);  []byte(core.MustCast[string](r)) after !r... use (s,err):
content, err := io.Local.Read(path)                      // returns string
// os.WriteFile(path, data, 0600) →
err := io.Local.WriteMode(path, string(data), 0600)
// os.Stat(path) (exists+isdir test) → io.Local.Exists(path) / io.Local.IsFile(path) / io.Local.Stat(path)
// encoding/json.Unmarshal(data,&v) → r := core.JSONUnmarshal(data, &v); if !r.OK { return core.Fail(...) }
// encoding/json.Marshal(v)         → r := core.JSONMarshal(v); bytes := core.MustCast[[]byte](r)
```
Drop the `"os"`, `"encoding/json"`, `"path/filepath"` imports. `path/filepath` in apple_test.go → `core.PathJoin`/`core.PathDir` (already used in apple.go).

### R5 — AX-7 triplets (ax7-triplet-gaps 13, unreferenced-tests 37)
Every public symbol `X` in `<file>.go` needs `Test<File>_X_Good`, `_Bad`, `_Ugly` in `<file>_test.go`, and **each body must name `X`** (call it / reference the receiver). No reflect/dispatcher indirection. Good = happy path, Bad = error path, Ugly = edge case. Fix the 37 unreferenced tests by making the body actually invoke its named symbol.

### R6 — examples (example-gaps 13, missing-example-files 1)
Every public symbol needs `Example<Symbol>` (or `ExampleType_Method`) in `<file>_example_test.go`, runnable, with `// Output:` where deterministic.

---

## Task 0 — Lead: baseline + worktrees

- [ ] **Step 1: Branch + record baseline**

```bash
cd /Users/snider/Code/core/go-container
git checkout -b feat/corego-v0103-foundations
git add docs/superpowers/specs docs/superpowers/plans && git commit -m "docs: corego v0.10.3 foundations design + plan"
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh . | tee /tmp/audit-baseline.txt   # expect 98 findings
cd go && go build ./... && go test ./... 2>&1 | tail -5   # expect green baseline
```

- [ ] **Step 2: Create two worktrees** (via superpowers:using-git-worktrees)

```bash
git worktree add ../gc-root   feat/corego-root
git worktree add ../gc-leaf   feat/corego-leaf
```

## Task A — Agent A: root package conversion (worktree `gc-root`)

**Files (own all):** `go/provider.go go/container.go go/hypervisor.go go/runtime.go go/apple.go go/linuxkit.go go/state.go go/tim.go go/templates.go go/datacube.go go/datanode.go go/gpu.go go/service.go` + each `*_test.go` + each `*_example_test.go`.

**New interface signatures (verbatim):**
```go
// provider.go
type Provider interface {
    Build(config ContainerConfig) core.Result            // Value: *Image
    Run(image *Image, opts ...RunOption) core.Result     // Value: *Container
    Encrypt(image *Image, key []byte) core.Result        // Value: *EncryptedImage
    Decrypt(encrypted *EncryptedImage, key []byte) core.Result // Value: *Image
}
// container.go
type Manager interface {
    Run(ctx context.Context, image string, opts RunOptions) core.Result // Value: *Container
    Stop(ctx context.Context, id string) core.Result                    // Value: nil
    List(ctx context.Context) core.Result                               // Value: []*Container
    Logs(ctx context.Context, id string, follow bool) core.Result       // Value: ReadCloser
    Exec(ctx context.Context, id string, cmd []string) core.Result      // Value: nil
}
// hypervisor.go
type Hypervisor interface {
    Name() string
    Available() bool
    BuildCommand(ctx context.Context, image string, opts *HypervisorOptions) core.Result // Value: *proc.Command
}
```

- [ ] **Step A1:** Apply R2/R3 to the three interfaces + every impl in the file list. Apply R1 (drop `coreerr`). Apply R4 in apple.go.
- [ ] **Step A2:** Update all intra-root callers to the Result idiom (`if !r.OK { return r }` + `core.MustCast`).
- [ ] **Step A3:** `cd ../gc-root/go && go build .` → green (root package only).
- [ ] **Step A4:** Convert root `*_test.go` to the new shape; apply R5/R6 to close root triplet/example/unreferenced gaps. `go test .` green.
- [ ] **Step A5:** `bash …/audit.sh .` from gc-root root — root-package findings should be 0 (downstream subpackage findings remain until Round 2). Commit per logical group.

## Task B — Agent B: internal + sources (worktree `gc-leaf`)

**Files (own all):** `go/internal/coreutil/*.go`, `go/internal/proc/*.go`, `go/sources/*.go` + tests + examples.

- [ ] **Step B1:** Apply R1 (legacy-log), R2/R3 (error→Result on these packages' public funcs), R5/R6 (triplets/examples) across the three packages. `sources` does not import root, so no interface-ripple wait.
- [ ] **Step B2:** `cd ../gc-leaf/go && go build ./internal/... ./sources/... && go test ./internal/... ./sources/...` → green.
- [ ] **Step B3:** Commit per package.

## Task C — Lead: merge Round 1 + convert downstream

- [ ] **Step C1:** Merge `feat/corego-root` and `feat/corego-leaf` into `feat/corego-v0103-foundations` (disjoint files → no conflicts; resolve `go.mod` solely here).
- [ ] **Step C2:** Convert `go/devenv/*.go` and `go/cmd/vm/*.go` callers to the new Result interfaces (R1/R2/R3 + R5/R6). These depend on the now-converted root.
- [ ] **Step C3:** `cd go && go build ./... && go test ./...` → green.

## Task D — Lead: final gate

- [ ] **Step D1:** `cd go && go mod tidy` (drop now-unused `dappco.re/go/log`; bump `cli/config/io` to current if needed for the Result API).
- [ ] **Step D2:** `bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .` → **verdict: COMPLIANT (0 findings)**.
- [ ] **Step D3:** `cd go && go test ./... -race` green; `go vet ./...` clean.
- [ ] **Step D4:** Commit; ready for W1 (Apple reconciliation — separate plan).

---

## Self-Review

- **Spec coverage:** every audit dimension with a non-zero baseline maps to a recipe (R1 legacy-log, R2 err-shape, R3 tuple, R4 banned-imports, R5 triplets+unreferenced, R6 examples). Findings that were already 0 need no task.
- **Placeholder scan:** recipes carry real before/after code + real API signatures; no TBD.
- **Type consistency:** interface signatures defined once in Task A and referenced (not redefined) in Task C; `core.Result` value types annotated per method.
- **Scope:** W2 only (audit → 0). W1 (Apple features) is a separate plan; its ground truth is preserved in the design doc.
- **Known accepted cost:** apple.go is converted here (idiom) and rewritten again in W1 (behaviour) — W2 touch is light/idiom-only.
