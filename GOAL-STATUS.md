<!-- SPDX-License-Identifier: EUPL-1.2 -->

# GOAL-STATUS — Apple Containers completion + spec/doc audit

**Date:** 2026-05-13
**Status:** All 17 tasks + post-review fixes implemented, build + test pass

## 17 Tasks completion

| Task | Status | File | Notes |
|------|--------|------|-------|
| 1 — Build invokes CLI | ✅ | `go/apple.go:110` | Shells out `container build --tag <name> [--file <src>] <ctx>`. |
| 2 — Encrypt | ✅ | `go/apple.go:363` | AES-256-GCM (key derived via SHA-256), writes `.stim`. |
| 3 — Decrypt | ✅ | `go/apple.go:420` | Inverse of Encrypt. |
| 4 — Stop | ✅ | `go/apple.go:484` | `container stop <id>`, StatusStopped. |
| 5 — Kill | ✅ | `go/apple.go:505` | `container kill <id>`, StatusKilled. |
| 6 — Remove | ✅ | `go/apple.go:527` | `container rm <id>`, delete from tracked map. |
| 7 — Logs | ✅ | `go/apple.go:547` | `container logs --tail <n> <id>`, default 200. |
| 8 — Exec | ✅ | `go/apple.go:571` | `container exec <id> <command> [args...]`. |
| 9 — List | ✅ | `go/apple.go:598` | `container ls --format json` → `[]*Container`. |
| 10 — Inspect | ✅ | `go/apple.go:616` | `container inspect <id>` → `*Container`. |
| 11 — Pull | ✅ | `go/apple.go:636` | `container pull <ref>`. |
| 12 — Push | ✅ | `go/apple.go:670` | `container push <path> <ref>`. |
| 13 — RemoveImage | ✅ | `go/apple.go:689` | `container rmi <id>`. |
| 14 — ListImages | ✅ | `go/apple.go:708` | `container images --format json`. |
| 15 — Version detection | ✅ | `go/apple.go:83` | `Available()` shells out `--version` once. |
| 16 — Metal GPU | ✅ | `go/apple.go:246` | `isAppleSilicon()`. `--gpu --device metal`. |
| 17 — Tracked GC | ✅ | `go/apple.go:280` | `RetentionWindow` field, `time.AfterFunc` cleanup. |

## Post-review fixes (GOAL §Everything else missing)

| Item | Action | Details |
|------|--------|---------|
| `capGPU` on Apple Silicon | ✅ wired | `go/runtime.go:166` — `detectApple()` now sets `capGPU` when `GOARCH=arm64 && GOOS=darwin` |
| Stale TODO.md | ✅ deleted | Root TODO.md was a single migration task, superseded |
| RFC.md §16 priority table | ✅ updated | §13 Runtime Detection → ✅, §12 Apple → ✅, §14 TIM → ✅ |
| RFC.commands.md | ✅ thickened | 13→85 lines. Full command tree, file map, runtime resolution, template flow |
| RFC.imports.md | ✅ thickened | 24→67 lines. All deps categorised: core, internal, stdlib, third-party, AX-6 exemptions |
| UPGRADE.md | ✅ deleted | Stale historical report with forge.lthn.ai paths |
| CONSUMERS.md | ✅ fixed | Old path `dappco.re/go/core/container` → `dappco.re/go/container`; added go-mlx, LEM, Borg |
| PROMPT.md | ✅ fixed | `src/` → `go/`; TODO.md → GOAL.md; removed PERSONA.md reference |
| RECENT.md | ✅ updated | Added Apple provider 17-task entry |
| docs/index.md | ✅ fixed | Module path, added Apple + TIM mention |
| docs/architecture.md | ✅ updated | Added Provider tree (AppleProvider, TIMProvider), DataCube, DataNode, cmd/vm |
| README.md | ✅ expanded | 3→5 lines with module path + feature summary |

## Verified existing (already implemented, confirmed during review)

| Feature | Location | Status |
|---------|----------|--------|
| TIMConfig, TIMMount, TIMBundle, STIMBundle | `go/tim.go` | ✅ |
| EncryptTIM, DecryptSTIM | `go/tim.go` | ✅ |
| Three-layer rootfs (base/app/data) | `go/tim.go:47-54` | ✅ |
| DataCube (encrypted io.Medium) | `go/datacube.go` | ✅ |
| DataNode (TIM + Borg identity) | `go/datanode.go` | ✅ |
| State persistence (~/.core/containers.json) | `go/state.go` | ✅ |
| Logging (~/.core/logs/{id}.log) | `go/state.go:157-176` | ✅ |
| DetectHypervisor (QEMU/Hyperkit) | `go/hypervisor.go` | ✅ |
| DetectImageFormat (iso/qcow2/vmdk/raw) | `go/hypervisor.go:245` | ✅ |
| devenv/ (22 files, full lifecycle) | `go/devenv/` | ✅ |
| WithGPU RunOption | `go/gpu.go` | ✅ |
| cmd/vm/ (run/ps/stop/logs/exec/templates) | `go/cmd/vm/` | ✅ |

## Build & test

```bash
cd go && go build ./...  # exit 0
cd go && go test ./...   # exit 0, all pass
cd go && go vet ./...    # exit 0
```

## Remaining for separate passes

- AX polish audit: `bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .`
- macOS 26+ CLI flag verification (GPU flag, JSON schema, digest format)
- RFC.md §3.3 AMI/GCP formats (not in source — LinuxKit consumer-side only)
- v0.9.0 audit findings (legacy-log-package, ax7-triplet-gaps, example-gaps)
- RFC cross-reference link resolution
