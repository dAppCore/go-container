<!-- SPDX-License-Identifier: EUPL-1.2 -->

# GOAL — Apple Containers completion

**Working dir:** `/Users/snider/Code/core/go-container`
**Branch:** `dev`
**Reference spec:** `plans/code/core/go/container/RFC.apple.md` (in host-uk plans tree)
**Reference impl:** `go/apple.go` — 326 LOC, signatures present, many bodies stubbed

CoreGO integration (Result migration, banned-imports cleanup, AX-7 triplets) comes later — this GOAL is pure feature work to make Apple Containers actually do what its method names promise. Keep the existing `coreerr.E` / `(T, error)` style for now; matching it is fine.

## Tasks

### Task 1 — Build actually invokes `container build`

- File: `go/apple.go::AppleProvider.Build` (line 91)
- Today: returns an Image struct pointing at `config.Source` with `Format: FormatUnknown` and never invokes the CLI
- Required: `proc.NewCommandContext(ctx, a.Binary, "build", ...)` with CLI args from `ContainerConfig`:
  - `--tag <config.Name>` if Name set
  - `--file <config.Source>` if Source is a Containerfile
  - build context = directory of Source (or `.` if Source is just a tag)
- Capture image digest from CLI stdout; populate `Image.Digest` (add to Image struct in models.go if missing)
- Set `Image.Format` to `FormatOCI` on success

### Task 2 — Encrypt actually encrypts

- File: `go/apple.go::AppleProvider.Encrypt` (line 273)
- Today: returns an EncryptedImage struct with `.stim` path suffix but no actual encryption
- Required: read `image.Path`, encrypt with key using AES-256-GCM, write to `image.Path + ".stim"`
- Format: 12-byte nonce prefix + ciphertext + 16-byte GCM tag
- Set `EncryptedImage.Size` to the actual written file size

### Task 3 — Decrypt actually decrypts

- File: `go/apple.go::AppleProvider.Decrypt` (line 301)
- Today: mirror stub of Encrypt
- Required: inverse of Task 2 — read `encrypted.Path`, parse 12-byte nonce + ciphertext + 16-byte tag, decrypt with key, write plaintext to path with `.stim` trimmed
- Set `Image.Format` via `DetectImageFormat(path)` (already imported)
- Set `Image.Size` to plaintext size

### Task 4 — Stop method

- New: `func (a *AppleProvider) Stop(id string) error`
- Required: `a.Binary, "stop", id`. Update tracked entry `Container.Status` to `StatusStopped`.

### Task 5 — Kill method

- New: `func (a *AppleProvider) Kill(id string) error`
- Required: `a.Binary, "kill", id`. SIGKILL equivalent. Status → `StatusKilled` (add constant to models.go if missing).

### Task 6 — Remove method

- New: `func (a *AppleProvider) Remove(id string) error`
- Required: `a.Binary, "rm", id`. Delete from `a.tracked` map after success, holding `appleProviderLock`.

### Task 7 — Logs method

- New: `func (a *AppleProvider) Logs(id string, tail int) (string, error)`
- Required: `a.Binary, "logs", "--tail", strconv.Itoa(tail), id`. Return combined stdout/stderr.
- `tail <= 0` defaults to 200.

### Task 8 — Exec method

- New: `func (a *AppleProvider) Exec(id, command string, args ...string) (string, error)`
- Required: `a.Binary, "exec", id, command, args...`. Return stdout.

### Task 9 — List method (all containers)

- New: `func (a *AppleProvider) List() ([]*Container, error)`
- Required: `a.Binary, "ls", "--format", "json"`. Parse JSON into `[]*Container`.
- Schema: array of `{id, name, image, status, created_at, ports}`. Map each into `Container`.

### Task 10 — Inspect method

- New: `func (a *AppleProvider) Inspect(id string) (*Container, error)`
- Required: `a.Binary, "inspect", id`. Parse single-container JSON into `Container`.

### Task 11 — Pull image

- New: `func (a *AppleProvider) Pull(ref string) (*Image, error)`
- Required: `a.Binary, "pull", ref`. Return `Image` with reference + digest.

### Task 12 — Push image

- New: `func (a *AppleProvider) Push(image *Image, ref string) error`
- Required: `a.Binary, "push", image.Path, ref`.

### Task 13 — Remove image

- New: `func (a *AppleProvider) RemoveImage(id string) error`
- Required: `a.Binary, "rmi", id`.

### Task 14 — List images

- New: `func (a *AppleProvider) ListImages() ([]*Image, error)`
- Required: `a.Binary, "images", "--format", "json"`. Parse into `[]*Image`.

### Task 15 — Version detection

- File: `go/apple.go::AppleProvider.Available` (line 73)
- Today: `Version` field declared but never populated
- Required: when Available returns true, shell out `a.Binary, "--version"` once, store parsed version in `a.Version`. Idempotent: skip if already set.

### Task 16 — Metal GPU passthrough

- File: `go/apple.go::AppleProvider.Run` (line 127) — extend GPU handling
- Today: `if ro.GPU { args = append(args, "--gpu") }` at line 169-171
- Required: when `ro.GPU == true` AND running on Apple Silicon (`runtime.GOARCH == "arm64" && runtime.GOOS == "darwin"`), additionally append `--device metal` (verify the actual flag name against Apple's `container` CLI docs — placeholder, the macOS 26+ CLI flag may differ)
- If `ro.GPU == true` AND host is Intel macOS, return error: "Metal GPU passthrough requires Apple Silicon"
- Add helper `func isAppleSilicon() bool`

### Task 17 — tracked map GC

- File: `go/apple.go::AppleProvider.track` (line 200)
- Today: tracked entries stay in the map after container exits
- Required: when the goroutine at line 212-222 closes `entry.Done`, delete the entry from `a.tracked` after a retention window (default 5 minutes). Add `RetentionWindow time.Duration` field to `AppleProvider`.

## Success condition

```bash
cd /Users/snider/Code/core/go-container
go build ./...
go test ./...
```

Both exit 0. The 17 tasks above all show real implementations (no stub returns).

## Discipline

- Commit per task, conventional prefix `feat(container):` for new methods, `fix(container):` for stub-to-real conversions.
- Co-Authored-By: Virgil <virgil@lethean.io>
- Match existing `apple.go` style — `coreerr.E(scope, msg, cause)` for errors, `(T, error)` returns. CoreGO migration is a separate later pass.
- Never spawn the real `container` CLI as a test side-effect — use the mocking pattern from existing `apple_test.go`.
- No `replace` directives in `go.mod`. Workspace mode handles deps.

## Stop conditions — write GOAL-STATUS.md and exit

- Apple `container` CLI flag names differ from this GOAL's assumptions. Capture the actual flag, surface to Snider.
- 6 hours elapsed.
- Any task introduces a build break the same task can't fix.

## After the 17 tasks land — AX polish

Run the audit and fix every dimension it reports until the verdict is COMPLIANT:

```bash
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

The audit IS the spec for AX cleanup. Each dimension's preamble inside the script explains the rule + the canonical fix. No additional guidance needed here.

## Everything else missing in go-container

- TODO.md at repo root is stale (single migration task superseded by audit script) — delete it
- RFC.md §16 Implementation Priority table is outdated — §13 Runtime Detection is done (move to ✅), §12 Apple Containers becomes ✅ after the 17 tasks above
- RFC.commands.md is a 13-line stub — thicken to match the actual cmd/vm/ tree (cmd_container, cmd_vm, cmd_templates, cmd_commands)
- RFC.imports.md is a 24-line stub — thicken to match the real import surface across go/*.go
- runtime.go detectApple() does not set capGPU on Apple Silicon — wire `capGPU` when GOARCH=arm64 && GOOS=darwin
- RFC.md §3.3 LinuxKit output formats: verify ISO / raw / qcow2 / VMDK / AMI / GCP image all build through linuxkit.go and have triplet+Example tests for each format path
- RFC.md §3.4 dm-crypt encrypted storage: verify implementation present and tested
- RFC.md §3.5 Networking: verify WireGuard / VPNKit / Vsock / static IP all wired in linuxkit.go
- RFC.md §6 Commands: cross-check every CLI subcommand described in §6 has a counterpart in cmd/vm/
- RFC.md §8 DevOps Portable Environment: verify devenv/ package has Boot / Stop / Status / image mgmt / test detection / serve detection / SSH shell / serial console / Claude workspace mount
- RFC.md §9 Hypervisor Selection: verify DetectHypervisor returns QEMU on Linux/KVM and Hyperkit on macOS with VPNKit fallback; verify DetectImageFormat handles iso/raw/qcow2/vmdk
- RFC.md §10 State Persistence: verify container registry at ~/.core/containers.json with thread-safe Add/Update/Remove
- RFC.md §11 Logging: verify per-container logs land at ~/.core/logs/{id}.log
- RFC.md §15 Metal GPU passthrough: §15.2 WithGPU RunOption (already exists), §15.3 HasGPU detection (capability bit exists but not wired to Apple Silicon detection — see runtime.go bullet above), §15.4 go-mlx integration smoke test inside an Apple container
- RFC.tim.md §3 rootfs three-layer convention (base/app/data): verify TIMBundle layout actually writes this structure
- RFC.tim.md §6 Key Hierarchy: workspace_key → container_key → layer_keys[] derivation already in tim.go — verify each step has triplet+Example tests
- RFC.tim.md §7 Borg.DataNode integration: DataNode wraps TIM container — verify NewDataNode + Start + Seal cover the full lifecycle described in §7
- AGENTS.md, GEMINI.md, PROMPT.md, UPGRADE.md, RECENT.md at repo root — audit each for stale content; delete or refresh
- docs/architecture.md, docs/development.md, docs/index.md — verify these match current code (not the pre-extraction shape from go-devops)
- README.md — verify it reflects the current single-repo state (not the legacy go-devops bundle history)
- v0.9.0 audit findings (snapshot 2026-05-13, will be re-checked after the 17 feature tasks land):
  - legacy-log-package: 19 sites — `import coreerr "dappco.re/go/log"` → use `core.E` from `dappco.re/go`
  - ax7-triplet-gaps: 2 — script reports which symbols
  - example-gaps: 2 — script reports which symbols
  - missing-example-files: 1 — script reports which source file
- All RFC.md / RFC.apple.md / RFC.tim.md / RFC.commands.md / RFC.imports.md / RFC.models.md cross-references must resolve: every `code/core/go/io/RFC.md §Medium`-style link must point at a file that exists at the named anchor
