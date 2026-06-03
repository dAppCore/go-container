<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — `core vm tim` (Borg-backed TIM/STIM CLI)

**Date:** 2026-06-03
**Module:** `dappco.re/go/container` (code under `go/`)
**Status:** Approved (decisions locked with Snider 2026-06-03)
**RFC:** RFC.md §6 (Commands) + RFC.tim.md (TIM/STIM format). Feature ④ — the next RFC gap after the macOS-user roadmap (①–③b).

## 1. Goal

Expose the **authoritative Borg** TIM/STIM container format on the CLI: pack a
directory into a `.tim` bundle, encrypt it to `.stim`, decrypt it back, and
inspect either. Back the commands with `forge.lthn.ai/Snider/Borg/pkg/tim`
**in-process** (no shell-out), and **retire this repo's divergent native TIM
format** so the tree carries one — Borg-compatible — implementation.

## 2. Background — the divergence this resolves

Before this work the repo held its own `go/tim.go` (a directory layout
`config.json` + `rootfs/{base,app,data}/`, AES-256-GCM, hierarchical sha256
keys). That format is **byte-incompatible** with Borg, the authoritative
implementation:

| | this repo's old `go/tim.go` | **Borg** (`forge.lthn.ai/Snider/Borg`) |
|---|---|---|
| TIM | a *directory* | a single *tar file* (`.tim`) |
| STIM | many `rootfs/<layer>.stim` files | one Trix-wrapped file (`.stim`) |
| crypto | AES-256-GCM, hierarchical keys | XChaCha20-Poly1305 via `Enchantrix`, `sha256(password)` |
| pack | none (no laydown) | a Borgfile (`ADD src dest`) / directory tree |

Decision (Snider, 2026-06-03): **import Borg + Enchantrix** and call them
in-process; **remove** the native format. One format, the canonical one.

## 3. Locked decisions

| Decision | Choice |
|---|---|
| Strategy | Import `forge.lthn.ai/Snider/Borg/pkg/tim` (+ `Enchantrix`) and call `ToTar`/`ToSigil`/`FromTar`/`FromSigil` in-process. No shelling out to a `borg` binary. |
| Key source | `--key-file <path>` (env fallback `CORE_TIM_KEY`). The file's trimmed contents are the **passphrase** for `ToSigil`/`FromSigil`; Borg derives the AEAD key via `sha256`. The file holds a secret of any length (not a raw 32-byte key — Borg's API is passphrase-based). |
| `pack` input | A source **directory** (`RootFS.AddPath(<dir>)`), matching RFC §6's `tim pack <dir>`. No Borgfile (`-f`) input this round. **No config flags** — Borg's TIM config is a minimal placeholder (`defaultConfig()` is an empty trix header; `borg compile` sets no entrypoint/env), so pack does not invent a config schema. Entrypoint/env customization is a follow-up for when Borg's config model grows. |
| Placement | `core vm tim …` under the module's existing `vm` namespace (the RFC's top-level `core tim` is `core vm tim` here, like `core vm run`). |
| Native format | **Removed** (`tim.go`, `datacube.go`, `datanode.go` + their tests/examples). One TIM format in the tree. |

## 4. Borg API surface used

From `forge.lthn.ai/Snider/Borg/pkg/tim` (pure format ops — no runc/cgo/GUI):

```go
func New() (*TerminalIsolationMatrix, error)               // empty TIM
func FromTar(data []byte) (*TerminalIsolationMatrix, error) // load .tim
func FromSigil(data []byte, password string) (*TerminalIsolationMatrix, error) // load+decrypt .stim
func (m *TerminalIsolationMatrix) ToTar() ([]byte, error)   // serialise .tim
func (m *TerminalIsolationMatrix) ToSigil(password string) ([]byte, error) // encrypt .stim
// m.Config []byte (raw JSON); m.RootFS *datanode.DataNode with AddData(name,[]byte) / AddPath(dir, opts)
```

Inspect uses `forge.lthn.ai/Snider/Enchantrix/pkg/trix.Decode(data, "STIM", nil)`
to read the `.stim` header without a key. Transitive deps pulled in: only
`Enchantrix v0.0.4` and `golang.org/x/crypto`.

## 5. Commands (`go/cmd/vm/cmd_tim.go`)

Registered as `vm/tim` (group) + `vm/tim/{pack,encrypt,decrypt,inspect}`,
mirroring the `vm/system/*` nesting from ③a. Each `Action` returns `core.Result`.

- **`vm tim pack <src-dir> <out.tim>`**
  - `tim.New()` → `m.RootFS.AddPath(<src-dir>, datanode.AddPathOptions{})` →
    `m.ToTar()` → write `<out.tim>`. No config flags — see §3 (Borg's config is a
    placeholder; pack stores the directory tree into a Borg-default `.tim`).
- **`vm tim encrypt <in.tim> <out.stim> --key-file <path>`**
  - read `<in.tim>` → `tim.FromTar(data)` → `m.ToSigil(passphrase)` → write `<out.stim>`.
- **`vm tim decrypt <in.stim> <out.tim> --key-file <path>`**
  - read `<in.stim>` → `tim.FromSigil(data, passphrase)` → `m.ToTar()` → write `<out.tim>`.
- **`vm tim inspect <file>`**
  - If the file begins with the `STIM` magic → `trix.Decode(data,"STIM",nil)`,
    print the header fields (algorithm, sizes, version). Otherwise treat as a
    `.tim` → `tim.FromTar(data)` and print the decoded `config.json`. Read-only,
    no key required.

Shared helpers in `cmd_tim.go`:
- `timKeyphrase(opts core.Options) core.Result` — resolve the passphrase from
  `--key-file` (read+trim) or `CORE_TIM_KEY`; Fail if neither is set/non-empty.
- `timIsSTIM(data []byte) bool` — magic-byte sniff (`"STIM"` prefix).

File I/O uses `io.Local` (the `io.Medium` abstraction), consistent with the rest
of the module; reads/writes go through `medium.Read`/`medium.Write`.

## 6. Key handling

`--key-file <path>`: `io.Local.Read(path)`, `core.TrimSpace` the contents, use
the result as the `password string` for `ToSigil`/`FromSigil`. If `--key-file`
is absent, fall back to `core.Env("CORE_TIM_KEY")`. Empty/missing on both →
`core.Fail`. The passphrase never appears in argv (file/env only).

## 7. Dependency wiring

- **`/Users/snider/Code/go.work`**: add `./snider/Borg` and `./snider/Enchantrix`
  to the `use (…)` block (the canonical, audit-excluded mechanism for local
  resolution).
- **`go/go.mod`**: add `require forge.lthn.ai/Snider/Borg <release>` and
  `require forge.lthn.ai/Snider/Enchantrix v0.0.4` (exact Borg tag resolved at
  build time). `golang.org/x/crypto` arrives as an indirect dep.
- **No `replace`** unless the forge proxy cannot resolve the modules outside the
  workspace; if needed, an external `replace forge.lthn.ai/Snider/… => …` is
  audit-safe (the `replace-directives` check is scoped to `dappco.re/go` only).
- go-container's `go` directive must be ≥ the highest imported module's (Enchantrix
  is `go 1.26`).

## 8. Removal — one TIM format

- **Delete:** `go/tim.go`, `go/tim_test.go`, `go/tim_example_test.go`,
  `go/datacube.go`, `go/datacube_test.go`, `go/datacube_example_test.go`,
  `go/datanode.go`, `go/datanode_test.go` (and `go/datanode_example_test.go` if
  present). `datacube`/`datanode` depend on `tim.go` (`DataNode.Seal` →
  `EncryptTIM`; `DataCube` → `EncryptLayer`), so they leave together.
- **Edit `go/container_behaviour_test.go`:** remove the 5 `DataCube`/`DataNode`
  test functions (`DataCubeDelegation_Good`, `DataCubeRename_Good`,
  `DataCubeDelete_Good`, `DataCubeStreaming_Good`, `DataNodeUptime_Good/_Bad`).
- **Edit `go/provider.go`:** drop the stale "TIM" doc-comment mentions; keep
  `Encrypt`/`Decrypt`/`EncryptedImage` (AppleProvider implements them).
- Verified non-importers (stay untouched): `linuxkit.go`, `apple.go`,
  `runtime.go`, `cmd/vm/*`, `devenv/*`, `sources/*`.
- After deletion the audit's matched-set rules stay satisfied (no source without
  test, no orphan test/example files).

## 9. Error handling

`core.Result` everywhere. Guards: missing positional args → Fail with
`vmT("cmd.vm.error.…")`; missing/empty key → Fail; `inspect` on an unreadable or
unrecognised file → Fail. Borg `(…, error)` returns are wrapped with
`core.E("vm tim …", msg, err)`. A wrong key surfaces as a failed `FromSigil`.

## 10. Testing

Borg's `pkg/tim` is pure Go (filesystem + crypto), so the happy paths run under
`go test` with no external runtime:

- **pack → inspect:** pack a temp dir (a real `t.TempDir()` with a file written
  via `io.Local`) to `<out.tim>`; `FromTar` round-trips the packed file, and
  `inspect` reports the bundle's (default) config.
- **encrypt → decrypt round-trip:** pack → encrypt with a key → decrypt with the
  same key → the recovered `.tim` round-trips to the original tar bytes / config.
- **wrong key:** `decrypt` with a different key → failed Result.
- **guards:** missing key-file → Fail; missing args → Fail; `inspect` bad magic
  → Fail.
- Pure-helper triplets where they add value: `timIsSTIM_{Good,Bad}`,
  `timKeyphrase_{Good,Bad}`. Triplets + `Example*` for the new exported cmd/vm
  symbols (audit COMPLIANT).
- Gate: `go test ./... -race` green, `go vet ./...` clean, `audit.sh` (repo root)
  COMPLIANT.

## 11. Docs

`go/cmd/vm/cmd_commands.go` package docstring and `specs/RFC.commands.md` command
tree gain the `tim` group (`pack`/`encrypt`/`decrypt`/`inspect`). The file map
row gains `cmd_tim.go`.

## 12. Out of scope / follow-ups

- `borg run` (container execution) — format ops only.
- Borgfile (`-f`) input to `pack` — directory input only this round.
- One-shot `pack --key-file` (pack-and-encrypt) — the four commands compose.
- A top-level `core tim` alias (we register under `vm`).
