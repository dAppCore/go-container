<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — CoreGO v0.10.3 foundations + Apple Linux-container reconciliation

**Date:** 2026-06-03
**Module:** `dappco.re/go/container` (code under `go/`)
**Status:** Approved direction (decisions locked with Snider 2026-06-03)

## 1. Goal

Two workstreams, sequenced **W2 → W1**:

- **W2 — CoreGO v0.10.3 foundations.** Bring the whole module onto the modern
  CoreGO idiom so the v0.9.0 compliance audit reports **0 findings**. Headline:
  `core.Result` (panic-recovering, error-checking-for-you) replaces the
  `func … error` shape everywhere, including the public interfaces.
- **W1 — Apple Linux-container reconciliation.** Make the `AppleProvider`
  actually drive Apple's `container` runtime end-to-end, reconciled against the
  **live `container 0.12.3` CLI** (now installed on this host), per `RFC.apple.md`.
  macOS *guest* VMs are explicitly **out of scope** (Snider's separate research
  task — that path is Virtualization.framework, not `apple/container`).

## 2. Locked decisions

| Decision | Choice |
|---|---|
| Sequence | W2 first (audit → 0), then W1 features |
| `core.Result` depth | Convert public `Provider`/`Manager`/`Hypervisor` interfaces to return `core.Result` (full adoption, ripples to all callers) |
| Parallelism | 2 subagents in isolated git worktrees, partitioned by package seam |
| macOS guest VMs | Out of scope |
| Verification | W1 certified against the live `container 0.12.3` binary |

## 3. W2 — foundations (audit baseline: 98 findings)

Audit: `bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .`
(run from repo root). Baseline 2026-06-03:

| Finding | Count | Fix |
|---|---|---|
| legacy-log-package | 19 | drop `coreerr "dappco.re/go/log"`; use `core.E(scope,msg,cause)` from `dappco.re/go` |
| unreferenced-tests | 37 | each `Test…` body must name its target symbol (real exercise, not reflect/dispatcher theatre) |
| ax7-triplet-gaps | 13 | every public symbol → `Test<File>_<Symbol>_{Good,Bad,Ugly}` in `<file>_test.go` |
| example-gaps | 13 | every public symbol → `Example<Symbol>` in `<file>_example_test.go` |
| err-shape-funcs | 8 | `func … error` → `func … core.Result` |
| banned-imports | 4 | `os` / `encoding/json` → core wrappers (`c.Fs()`, core JSON helpers) |
| tuple-result-shape | 3 | `(*T, error)` → single `core.Result`, value in `r.Value` |
| missing-example-files | 1 | add the missing `<file>_example_test.go` |

`core.Result` shape (from CoreGO reference + audit canon): carries `OK`,
`Value`, `Error`; auto-recovers panics at boundary. Constructors `core.Ok(v)`,
`core.Fail(err)`, `core.ResultOf(v, err)`. Caller idiom: `r := f(); if !r.OK
{ return r }; v := r.Value.(*T)`. Error construction: `core.E("Op","message",cause)`.
Interface-contract `func … error` survivors are allowed only where CoreGO itself
keeps them (Unwrap, error constructors) — target floor, not zero-at-any-cost.

### 3.1 Parallelisation (the package-seam partition)

The root package `dappco.re/go/container` (all `go/*.go`) is one compile unit and
holds the interfaces — its production conversion is the **coherent spine** and
cannot be split mid-package. Subpackages are separate compile units downstream of
root. Plan:

1. **Lead** lands the new `core.Result` interface contract (`provider.go`,
   `container.go`, `hypervisor.go`) on a green baseline, and sets up 2 worktrees.
2. **Agent A (worktree)** owns the **root package** `go/*.go`: convert all
   production impls + tests + triplets/examples to the new idiom. Keeps
   `go build .` / `go test .` green.
3. **Agent B (worktree)** owns **`go/devenv/`, `go/cmd/vm/`, `go/sources/`**:
   convert callers to the new interfaces + their tests/triplets/examples.
   Codes against the frozen root contract.
4. **Lead** merges (file ownership is disjoint → clean), owns `go.mod`/`go.sum`,
   drives `audit.sh` → 0 and `go test ./...` green.

Agents follow TDD and the v0.9.0 standard; the audit is the acceptance gate.

## 4. W1 — Apple reconciliation (verified ground truth, `container 0.12.3`)

Captured live on this host (Apple Silicon, macOS 26.5) so it survives compaction.
`container system start --enable-kernel-install` is **required** before run/build
(kata kernel installed; apiserver registered with launchd).

### 4.1 CLI-surface corrections (`apple.go`)

| Concern | Current (wrong) | Real CLI 0.12.3 |
|---|---|---|
| pull | `container pull <ref>` | `container image pull <ref>` |
| push | `container push <path> <ref>` | `container image push <ref>` (single tagged ref) |
| list images | `container images --format json` | `container image ls --format json` (top-level `images` → "plugin not found") |
| remove image | `container rmi <id>` | `container image delete <id>` (alias `rm`) |
| remove container | `container rm <id>` | OK — `rm` is a real alias of `delete` |
| list / inspect | `ls --format json` ok | `list/ls --format json` (json/table/yaml); `inspect` takes **many ids → JSON array**, no `--format` |
| logs tail | `logs --tail <n>` | `logs -n <n>` (`--boot`, `-f/--follow` also exist) |
| GPU | `run --gpu --device metal` | **no such flags** — return "unsupported" (RFC §15 blocked-on-Apple) |
| build | `build --tag --file <ctx>` | OK (`-t/--tag`, `-f/--file`, context positional) |
| run mem/cpu/ports/vols | `--memory NM --cpus N --publish --volume` | OK (`-m` accepts K/M/G/T/P suffix; `-p`, `-v`, `--mount`) |

Extra `run` flags worth mapping from `ContainerConfig`: `-e/--env`, `-w/--workdir`,
`--cap-add/--cap-drop`, `--read-only`, `--entrypoint`.

### 4.2 Real JSON schema (`list --format json` and `inspect` — identical object)

```json
[{
  "status": "running",
  "startedDate": 802181959.432204,           // CFAbsoluteTime: Unix = v + 978307200
  "networks": [{"ipv4Address":"192.168.64.2/24","hostname":"…","macAddress":"…","mtu":1280}],
  "configuration": {
    "id": "coreprobe",
    "image": {"reference":"docker.io/library/alpine:latest","descriptor":{"digest":"sha256:…","size":9218}},
    "resources": {"cpus":4,"memoryInBytes":1073741824},
    "platform": {"os":"linux","architecture":"arm64"},
    "initProcess": {"executable":"sleep","arguments":["300"],"environment":["PATH=…"],"workingDirectory":"/"},
    "publishedPorts": [], "labels": {}, "mounts": [], "capAdd": [], "capDrop": []
  }
}]
```

`image ls --format json`:
```json
[{"reference":"docker.io/library/alpine:latest","fullSize":"4.2 MB","descriptor":{"digest":"sha256:…","size":9218}}]
```

`appleContainerJSON`/`appleImageJSON` must be rewritten to this nested shape
(id at `configuration.id`, ports at `configuration.publishedPorts` array,
`startedDate` float → time via CFAbsoluteTime epoch, image via `descriptor.digest`).

### 4.3 Lifecycle wiring

`vm ps/stop/logs/exec` (`cmd_container.go:228/321/391/433`) hardcode
`NewLinuxKitManager`, so an Apple container can be started but never listed,
stopped, logged or exec'd. Route lifecycle through the resolved runtime/provider.
Forward ports & volumes through `runContainerApple` (currently dropped).

### 4.4 Bootstrap

`AppleProvider.Available()` only checks binary + `--version`. Add a
`container system status` check / `system start` ensure-step (or a clear,
actionable error) so Build/Run don't fail cryptically on a cold system.

## 5. Testing

- W2: TDD per the v0.9.0 standard; `audit.sh` → 0 + `go test ./... ` (and `-race`)
  green are the acceptance gates.
- W1: unit tests mock the `proc.Command` boundary (real CLI args asserted against
  §4.1); at least one **live** end-to-end smoke against the installed
  `container 0.12.3` (pull alpine, run, ls, logs, exec, stop, delete) to certify.

## 6. Risks

- **Interface ripple** (Result conversion) is wide; mitigated by lead-lands-contract-first + disjoint package ownership.
- **Merge conflicts** between agents; mitigated by package-seam file ownership and lead-owned `go.mod`.
- **CFAbsoluteTime** epoch easy to get wrong (off by 978307200s / ~31 years) — explicit test.
- **Live-CLI variance** across `container` patch versions; pin behaviour to 0.12.3 and assert arg construction in unit tests, not just live runs.
