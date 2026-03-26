package devenv

import (
	"testing"

	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/io"
)

func TestDetectTestCommand_ComposerJSON_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"scripts":{"test":"pest"}}`)

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "composer test" {
		t.Errorf("expected 'composer test', got %q", cmd)
	}
}

func TestDetectTestCommand_PackageJSON_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), `{"scripts":{"test":"vitest"}}`)

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "npm test" {
		t.Errorf("expected 'npm test', got %q", cmd)
	}
}

func TestDetectTestCommand_GoMod_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "go.mod"), "module example")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "go test ./..." {
		t.Errorf("expected 'go test ./...', got %q", cmd)
	}
}

func TestDetectTestCommand_CoreTestYaml_Good(t *testing.T) {
	tmpDir := t.TempDir()
	coreDir := coreutil.JoinPath(tmpDir, ".core")
	_ = io.Local.EnsureDir(coreDir)
	_ = io.Local.Write(coreutil.JoinPath(coreDir, "test.yaml"), "command: custom-test")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "custom-test" {
		t.Errorf("expected 'custom-test', got %q", cmd)
	}
}

func TestDetectTestCommand_Pytest_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "pytest.ini"), "[pytest]")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "pytest" {
		t.Errorf("expected 'pytest', got %q", cmd)
	}
}

func TestDetectTestCommand_Taskfile_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "Taskfile.yaml"), "version: '3'")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "task test" {
		t.Errorf("expected 'task test', got %q", cmd)
	}
}

func TestDetectTestCommand_NoFiles_Bad(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "" {
		t.Errorf("expected empty string, got %q", cmd)
	}
}

func TestDetectTestCommand_Priority_Good(t *testing.T) {
	// .core/test.yaml should take priority over other detection methods
	tmpDir := t.TempDir()
	coreDir := coreutil.JoinPath(tmpDir, ".core")
	_ = io.Local.EnsureDir(coreDir)
	_ = io.Local.Write(coreutil.JoinPath(coreDir, "test.yaml"), "command: my-custom-test")
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "go.mod"), "module example")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "my-custom-test" {
		t.Errorf("expected 'my-custom-test' (from .core/test.yaml), got %q", cmd)
	}
}

func TestTest_LoadTestConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	coreDir := coreutil.JoinPath(tmpDir, ".core")
	_ = io.Local.EnsureDir(coreDir)

	configYAML := `version: 1
command: default-test
commands:
  - name: unit
    run: go test ./...
  - name: integration
    run: go test -tags=integration ./...
env:
  CI: "true"
`
	_ = io.Local.Write(coreutil.JoinPath(coreDir, "test.yaml"), configYAML)

	cfg, err := LoadTestConfig(io.Local, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Command != "default-test" {
		t.Errorf("expected command 'default-test', got %q", cfg.Command)
	}
	if len(cfg.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cfg.Commands))
	}
	if cfg.Commands[0].Name != "unit" {
		t.Errorf("expected first command name 'unit', got %q", cfg.Commands[0].Name)
	}
	if cfg.Env["CI"] != "true" {
		t.Errorf("expected env CI='true', got %q", cfg.Env["CI"])
	}
}

func TestLoadTestConfig_NotFound_Bad(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadTestConfig(io.Local, tmpDir)
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
}

func TestTest_HasPackageScript_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), `{"scripts":{"test":"jest","build":"webpack"}}`)

	if !hasPackageScript(io.Local, tmpDir, "test") {
		t.Error("expected to find 'test' script")
	}
	if !hasPackageScript(io.Local, tmpDir, "build") {
		t.Error("expected to find 'build' script")
	}
}

func TestHasPackageScript_MissingScript_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), `{"scripts":{"build":"webpack"}}`)

	if hasPackageScript(io.Local, tmpDir, "test") {
		t.Error("expected not to find 'test' script")
	}
}

func TestTest_HasComposerScript_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"scripts":{"test":"pest","post-install-cmd":"@php artisan migrate"}}`)

	if !hasComposerScript(io.Local, tmpDir, "test") {
		t.Error("expected to find 'test' script")
	}
}

func TestHasComposerScript_MissingScript_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"scripts":{"build":"@php build.php"}}`)

	if hasComposerScript(io.Local, tmpDir, "test") {
		t.Error("expected not to find 'test' script")
	}
}

func TestTestConfig_Struct_Good(t *testing.T) {
	cfg := &TestConfig{
		Version:  2,
		Command:  "my-test",
		Commands: []TestCommand{{Name: "unit", Run: "go test ./..."}},
		Env:      map[string]string{"CI": "true"},
	}
	if cfg.Version != 2 {
		t.Errorf("expected version 2, got %d", cfg.Version)
	}
	if cfg.Command != "my-test" {
		t.Errorf("expected command 'my-test', got %q", cfg.Command)
	}
	if len(cfg.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(cfg.Commands))
	}
	if cfg.Env["CI"] != "true" {
		t.Errorf("expected CI=true, got %q", cfg.Env["CI"])
	}
}

func TestTestCommand_Struct_Good(t *testing.T) {
	cmd := TestCommand{
		Name: "integration",
		Run:  "go test -tags=integration ./...",
	}
	if cmd.Name != "integration" {
		t.Errorf("expected name 'integration', got %q", cmd.Name)
	}
	if cmd.Run != "go test -tags=integration ./..." {
		t.Errorf("expected run command, got %q", cmd.Run)
	}
}

func TestTestOptions_Struct_Good(t *testing.T) {
	opts := TestOptions{
		Name:    "unit",
		Command: []string{"go", "test", "-v"},
	}
	if opts.Name != "unit" {
		t.Errorf("expected name 'unit', got %q", opts.Name)
	}
	if len(opts.Command) != 3 {
		t.Errorf("expected 3 command parts, got %d", len(opts.Command))
	}
}

func TestDetectTestCommand_TaskfileYml_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "Taskfile.yml"), "version: '3'")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "task test" {
		t.Errorf("expected 'task test', got %q", cmd)
	}
}

func TestDetectTestCommand_Pyproject_Good(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "pyproject.toml"), "[tool.pytest]")

	cmd := DetectTestCommand(io.Local, tmpDir)
	if cmd != "pytest" {
		t.Errorf("expected 'pytest', got %q", cmd)
	}
}

func TestHasPackageScript_NoFile_Bad(t *testing.T) {
	tmpDir := t.TempDir()

	if hasPackageScript(io.Local, tmpDir, "test") {
		t.Error("expected false for missing package.json")
	}
}

func TestHasPackageScript_InvalidJSON_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), `invalid json`)

	if hasPackageScript(io.Local, tmpDir, "test") {
		t.Error("expected false for invalid JSON")
	}
}

func TestHasPackageScript_NoScripts_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), `{"name":"test"}`)

	if hasPackageScript(io.Local, tmpDir, "test") {
		t.Error("expected false for missing scripts section")
	}
}

func TestHasComposerScript_NoFile_Bad(t *testing.T) {
	tmpDir := t.TempDir()

	if hasComposerScript(io.Local, tmpDir, "test") {
		t.Error("expected false for missing composer.json")
	}
}

func TestHasComposerScript_InvalidJSON_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `invalid json`)

	if hasComposerScript(io.Local, tmpDir, "test") {
		t.Error("expected false for invalid JSON")
	}
}

func TestHasComposerScript_NoScripts_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"name":"test/pkg"}`)

	if hasComposerScript(io.Local, tmpDir, "test") {
		t.Error("expected false for missing scripts section")
	}
}

func TestLoadTestConfig_InvalidYAML_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	coreDir := coreutil.JoinPath(tmpDir, ".core")
	_ = io.Local.EnsureDir(coreDir)
	_ = io.Local.Write(coreutil.JoinPath(coreDir, "test.yaml"), "invalid: yaml: :")

	_, err := LoadTestConfig(io.Local, tmpDir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadTestConfig_MinimalConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	coreDir := coreutil.JoinPath(tmpDir, ".core")
	_ = io.Local.EnsureDir(coreDir)
	_ = io.Local.Write(coreutil.JoinPath(coreDir, "test.yaml"), "version: 1")

	cfg, err := LoadTestConfig(io.Local, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Command != "" {
		t.Errorf("expected empty command, got %q", cfg.Command)
	}
}

func TestDetectTestCommand_ComposerWithoutScript_Good(t *testing.T) {
	tmpDir := t.TempDir()
	// composer.json without test script should not return composer test
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"name":"test/pkg"}`)

	cmd := DetectTestCommand(io.Local, tmpDir)
	// Falls through to empty (no match)
	if cmd != "" {
		t.Errorf("expected empty string, got %q", cmd)
	}
}

func TestDetectTestCommand_PackageJSONWithoutScript_Good(t *testing.T) {
	tmpDir := t.TempDir()
	// package.json without test or dev script
	_ = io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), `{"name":"test"}`)

	cmd := DetectTestCommand(io.Local, tmpDir)
	// Falls through to empty
	if cmd != "" {
		t.Errorf("expected empty string, got %q", cmd)
	}
}
