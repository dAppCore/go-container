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
