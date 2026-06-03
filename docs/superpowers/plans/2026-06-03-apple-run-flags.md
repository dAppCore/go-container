<!-- SPDX-License-Identifier: EUPL-1.2 -->

# `core vm run` -p/-v/-e Flags — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (inline) or subagent-driven-development. Steps use `- [ ]` tracking.

**Goal:** Add repeatable `-p/--publish`, `-v/--volume`, `-e/--env` flags to `core vm run`, forwarding ports/volumes to both runtimes and env to Apple.

**Architecture:** Pure parse helpers in `cmd/vm` turn flag strings into `RunOptions.{Ports,Volumes,Env}`; `runContainer`/`runContainerApple` are refactored to take a `RunOptions` (collapsing their unwieldy param lists) so the new fields thread through both paths. `appleRunArgs` gains `-e` emission; LinuxKit already consumes Ports/Volumes via the hypervisor.

**Tech Stack:** Go 1.26, `dappco.re/go` (`core.Result`, `core.E`, `core.Split`, `core.Sprintf`), `strconv` (not banned), plain stdlib `testing`.

**Spec:** `docs/superpowers/specs/2026-06-03-apple-run-flags-design.md`

---

### Task 1: `RunOptions.Env` + `WithEnv`

**Files:** Modify `go/container.go` (RunOptions), `go/provider.go` (WithEnv), `go/provider_test.go`, `go/provider_example_test.go`.

- [ ] **Step 1 — failing test** (`provider_test.go`, after the `WithArgs` triplet):
```go
func TestProvider_WithEnv_Good(t *testing.T) {
	o := ApplyRunOptions(WithEnv("FOO=bar", "BAZ=qux"))
	if len(o.Env) != 2 || o.Env[0] != "FOO=bar" || o.Env[1] != "BAZ=qux" {
		t.Fatalf("WithEnv => %v, want [FOO=bar BAZ=qux]", o.Env)
	}
}
func TestProvider_WithEnv_Bad(t *testing.T) {
	o := ApplyRunOptions(WithEnv()) // degenerate but valid
	if len(o.Env) != 0 {
		t.Fatalf("WithEnv() => %v, want empty", o.Env)
	}
}
func TestProvider_WithEnv_Ugly(t *testing.T) {
	// Values may contain '=' and be empty; order preserved.
	o := ApplyRunOptions(WithEnv("URL=https://x?a=b", "EMPTY="))
	want := []string{"URL=https://x?a=b", "EMPTY="}
	if len(o.Env) != len(want) {
		t.Fatalf("WithEnv len = %d, want %d (%v)", len(o.Env), len(want), o.Env)
	}
	for i := range want {
		if o.Env[i] != want[i] {
			t.Fatalf("WithEnv[%d] = %q, want %q", i, o.Env[i], want[i])
		}
	}
}
```
- [ ] **Step 2 — verify RED:** `cd go && go test . -run TestProvider_WithEnv` → build fail "undefined: WithEnv / o.Env".
- [ ] **Step 3 — implement.** `container.go` RunOptions, after `Args []string`:
```go
	// Env is the container environment in KEY=VALUE form (Apple runtime only;
	// LinuxKit bakes env into the image at build time).
	Env []string
```
`provider.go`, before `ApplyRunOptions` (mirror `WithArgs`):
```go
// WithEnv sets container environment variables (KEY=VALUE), e.g.
// WithEnv("PORT=8080"). Apple runtime only; LinuxKit env is image-baked.
//
// Usage:
//
//	p.Run(img, container.WithEnv("ENV=prod"))
func WithEnv(env ...string) RunOption {
	return func(o *RunOptions) {
		o.Env = append(o.Env, env...)
	}
}
```
`provider_example_test.go`, after `ExampleWithArgs`:
```go
func ExampleWithEnv() {
	// WithEnv sets container environment variables (KEY=VALUE), Apple-only.
	_ = ApplyRunOptions(WithEnv("PORT=8080"))
}
```
- [ ] **Step 4 — verify GREEN:** `go test . -run TestProvider_WithEnv` PASS; `go test .` green.
- [ ] **Step 5 — commit:** `feat(container): add RunOptions.Env + WithEnv option`.

### Task 2: `appleRunArgs` emits `-e KEY=VALUE`

**Files:** Modify `go/apple.go` (`appleRunArgs`), `go/apple_test.go`.

- [ ] **Step 1 — failing test** (`apple_test.go`, after `TestApple_appleRunArgs_ContainerArgs_Good`):
```go
func TestApple_appleRunArgs_Env_Good(t *testing.T) {
	r := appleRunArgs("web", &Image{Path: "alpine:latest"},
		RunOptions{Env: []string{"FOO=bar"}, Volumes: map[string]string{"/h": "/c"}, Args: []string{"sleep", "1"}})
	if !r.OK {
		t.Fatal(r.Error())
	}
	args := core.MustCast[[]string](r)
	joined := core.Join(" ", args...)
	if !core.Contains(joined, "-e FOO=bar") {
		t.Fatalf("args %q missing `-e FOO=bar`", joined)
	}
	// env must come before the image, container args after it.
	ePos, imgPos := indexOf(args, "FOO=bar"), indexOf(args, "alpine:latest")
	if ePos < 0 || imgPos < 0 || ePos > imgPos {
		t.Fatalf("env must precede image: %v", args)
	}
}

// indexOf returns the first index of v in s, or -1.
func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
```
- [ ] **Step 2 — verify RED:** `go test . -run TestApple_appleRunArgs_Env_Good` FAIL (no `-e` emitted).
- [ ] **Step 3 — implement.** In `appleRunArgs`, after the `ro.Volumes` loop and BEFORE `args = append(args, image.Path)`:
```go
	for _, e := range ro.Env {
		args = append(args, "-e", e)
	}
```
- [ ] **Step 4 — verify GREEN:** test PASS; `go test .` green.
- [ ] **Step 5 — commit:** `feat(apple): emit -e env flags in appleRunArgs`.

### Task 3: parse helpers in `cmd/vm`

**Files:** Modify `go/cmd/vm/cmd_container.go`, `go/cmd/vm/cmd_container_test.go`.

- [ ] **Step 1 — failing tests** (`cmd_container_test.go`):
```go
func TestCmdContainer_parsePublish_Good(t *testing.T) {
	got, err := parsePublish([]string{"8080:80", "127.0.0.1:5432:5432/tcp"})
	if err != nil {
		t.Fatal(err)
	}
	if got[8080] != 80 || got[5432] != 5432 {
		t.Fatalf("parsePublish => %v, want 8080->80, 5432->5432", got)
	}
}
func TestCmdContainer_parsePublish_Bad(t *testing.T) {
	if _, err := parsePublish([]string{"8080"}); err == nil {
		t.Fatal("expected error for missing colon")
	}
	if _, err := parsePublish([]string{"http:80"}); err == nil {
		t.Fatal("expected error for non-numeric host port")
	}
}
func TestCmdContainer_parseVolumes_Good(t *testing.T) {
	got, err := parseVolumes([]string{"/data:/app", "./cfg:/etc/app"})
	if err != nil {
		t.Fatal(err)
	}
	if got["/data"] != "/app" || got["./cfg"] != "/etc/app" {
		t.Fatalf("parseVolumes => %v", got)
	}
}
func TestCmdContainer_parseVolumes_Bad(t *testing.T) {
	if _, err := parseVolumes([]string{"/data"}); err == nil {
		t.Fatal("expected error for missing colon")
	}
}
func TestCmdContainer_parseEnv_Good(t *testing.T) {
	got, err := parseEnv([]string{"FOO=bar", "URL=https://x?a=b", "EMPTY="})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "FOO=bar" || got[2] != "EMPTY=" {
		t.Fatalf("parseEnv => %v", got)
	}
}
func TestCmdContainer_parseEnv_Bad(t *testing.T) {
	if _, err := parseEnv([]string{"NOEQUALS"}); err == nil {
		t.Fatal("expected error for missing '='")
	}
}
```
- [ ] **Step 2 — verify RED:** `go test ./cmd/vm/ -run 'TestCmdContainer_parse'` build fail (undefined).
- [ ] **Step 3 — implement** (in `cmd_container.go`; add `"strconv"` import):
```go
// parsePublish parses docker-style "[host-ip:]host:container[/proto]" port
// specs into a host→container map. host-ip and /proto are dropped (tcp assumed;
// RunOptions.Ports is map[int]int).
func parsePublish(specs []string) (map[int]int, error) {
	ports := make(map[int]int, len(specs))
	for _, s := range specs {
		spec := s
		if core.Contains(spec, "/") {
			spec = core.Split(spec, "/")[0]
		}
		parts := core.Split(spec, ":")
		if len(parts) < 2 {
			return nil, core.E("vm run", core.Sprintf("invalid --publish %q: want host:container", s), nil)
		}
		host, e1 := strconv.Atoi(parts[len(parts)-2])
		ctr, e2 := strconv.Atoi(parts[len(parts)-1])
		if e1 != nil || e2 != nil {
			return nil, core.E("vm run", core.Sprintf("invalid --publish %q: non-numeric port", s), nil)
		}
		ports[host] = ctr
	}
	return ports, nil
}

// parseVolumes parses "host:container" mount specs into a host→container map.
func parseVolumes(specs []string) (map[string]string, error) {
	vols := make(map[string]string, len(specs))
	for _, s := range specs {
		i := core.Index(s, ":")
		if i <= 0 || i == len(s)-1 {
			return nil, core.E("vm run", core.Sprintf("invalid --volume %q: want host:container", s), nil)
		}
		vols[s[:i]] = s[i+1:]
	}
	return vols, nil
}

// parseEnv validates KEY=VALUE env specs (value may be empty or contain '=').
func parseEnv(specs []string) ([]string, error) {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		if !core.Contains(s, "=") {
			return nil, core.E("vm run", core.Sprintf("invalid --env %q: want KEY=VALUE", s), nil)
		}
		out = append(out, s)
	}
	return out, nil
}
```
(If `core.Index` is unavailable, use `core.Split(s, ":")` with a 2-cap join; confirm the helper name during build.)
- [ ] **Step 4 — verify GREEN:** `go test ./cmd/vm/ -run 'TestCmdContainer_parse'` PASS.
- [ ] **Step 5 — commit:** `feat(cmd/vm): port/volume/env flag parse helpers`.

### Task 4: wire flags + thread `RunOptions` through both run paths

**Files:** Modify `go/cmd/vm/cmd_container.go` (`addVMRunCommand`, `runContainer`, `runContainerApple`).

- [ ] **Step 1 — add flags** to `addVMRunCommand`'s `core.NewOptions(...)`: `core.Option{Key: "publish", Value: ""}`, `{Key: "volume", Value: ""}`, `{Key: "env", Value: ""}` (repeatable, read via `optionStrings`, mirroring `--var`).
- [ ] **Step 2 — refactor signatures** (collapse the unwieldy param lists):
```go
func runContainer(image, runtimeFlag string, opts container.RunOptions) (err error) // result
func runContainerApple(image string, opts container.RunOptions) (err error)        // result
```
`runContainer` reads `opts.Name/Detach/Memory/CPUs/GPU/Args` (no destructured params); dispatches to `runContainerApple(image, opts)` for Apple, else builds the LinuxKit `RunOptions` from `opts` INCLUDING `Ports: opts.Ports, Volumes: opts.Volumes`. `runContainerApple` builds its `[]RunOption` adding `container.WithPorts(opts.Ports)`, `container.WithVolumes(opts.Volumes)`, `container.WithEnv(opts.Env...)` alongside the existing Name/Memory/CPUs/Detach/Args/GPU options.
- [ ] **Step 3 — build `opts` in `addVMRunCommand`:** parse the new flags, fail on error, set fields:
```go
ports, err := parsePublish(optionStrings(opts, "publish"))
if err != nil { return core.Fail(err) }
volumes, err := parseVolumes(optionStrings(opts, "volume"))
if err != nil { return core.Fail(err) }
env, err := parseEnv(optionStrings(opts, "env"))
if err != nil { return core.Fail(err) }
runOpts := container.RunOptions{
	Name: opts.String("name"), Detach: opts.Bool("detach"),
	Memory: opts.Int("memory"), CPUs: opts.Int("cpus"), SSHPort: opts.Int("ssh-port"),
	GPU: opts.Bool("gpu"), Ports: ports, Volumes: volumes, Env: env,
	Args: args[1:],
}
return resultFromError(runContainer(image, opts.String("runtime"), runOpts))
```
(The template branch keeps building its own `runOpts` for `RunFromTemplate` — unchanged.)
- [ ] **Step 4 — verify:** `go build ./... && go test ./cmd/vm/... .` green; `go vet ./...`.
- [ ] **Step 5 — commit:** `feat(cmd/vm): wire -p/-v/-e through run to both runtimes`.

### Task 5: live e2e + final gate

**Files:** Modify `go/apple_test.go` (extend a CORE_APPLE_E2E smoke).

- [ ] **Step 1 — failing/gated test** (`apple_test.go`): a `CORE_APPLE_E2E`-gated test that runs a container with a published port via `p.Run(WithName, WithDetach, WithPorts(map[int]int{18080: 80}), WithArgs("sleep","60"))`, polls `List`, and asserts the parsed `Container.Ports[18080] == 80` (proves -p reached the runtime via the real `publishedPorts` JSON).
```go
func TestApple_E2E_RunPublish_Smoke(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" { t.Skip("set CORE_APPLE_E2E=1") }
	p := NewAppleProvider()
	if !p.Available() { t.Skip("apple container runtime not available") }
	const name = "core-publish-e2e"
	const ref = "docker.io/library/alpine:latest"
	ctx := context.Background()
	_ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run()
	defer func() { _ = proc.NewCommandContext(ctx, "container", "delete", "--force", name).Run() }()
	if r := p.Pull(ref); !r.OK { t.Fatalf("Pull: %v", r.Error()) }
	if r := p.Run(&Image{Path: ref}, WithName(name), WithDetach(true),
		WithPorts(map[int]int{18080: 80}), WithArgs("sleep", "60")); !r.OK {
		t.Fatalf("Run: %v", r.Error())
	}
	var got *Container
	for i := 0; i < 30 && got == nil; i++ {
		if lr := p.List(); lr.OK {
			for _, c := range core.MustCast[[]*Container](lr) {
				if c.ID == name && c.Status == StatusRunning { got = c }
			}
		}
		if got == nil { time.Sleep(500 * time.Millisecond) }
	}
	if got == nil { t.Fatal("container not running") }
	if got.Ports[18080] != 80 {
		t.Fatalf("published ports = %v, want 18080->80", got.Ports)
	}
}
```
- [ ] **Step 2 — run it:** `CORE_APPLE_E2E=1 go test ./ -run TestApple_E2E_RunPublish_Smoke -count=1` → PASS.
- [ ] **Step 3 — final gate:** `go test ./... -race` green; `go vet ./...`; `bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .` → COMPLIANT.
- [ ] **Step 4 — commit:** `test(apple): live e2e for -p published-port round-trip`.

---

## Self-Review

- **Spec coverage:** flags (Task 4) ✓; parse helpers (Task 3) ✓; RunOptions.Env + WithEnv (Task 1) ✓; appleRunArgs -e (Task 2) ✓; both-paths threading + the two latent bugs (Task 4) ✓; error handling (Task 3 + Task 4 Step 3) ✓; unit + live tests (Tasks 1-3, 5) ✓; audit COMPLIANT (Task 5) ✓.
- **Placeholders:** none — real code + commands. The one `core.Index` caveat is an explicit build-time confirmation, not a TBD.
- **Type consistency:** `RunOptions.Env []string` (Task 1) used by `WithEnv` (1), `appleRunArgs` (2), threading (4); `parsePublish→map[int]int`, `parseVolumes→map[string]string`, `parseEnv→[]string` consistent across Tasks 3-4; `runContainer(image, runtimeFlag, opts)` / `runContainerApple(image, opts)` consistent in Task 4.
