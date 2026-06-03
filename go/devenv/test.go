package devenv

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/io"
	"gopkg.in/yaml.v3"

	"dappco.re/go/container/internal/coreutil"
)

// TestConfig holds test configuration from .core/test.yaml.
type TestConfig struct {
	Version  int               `yaml:"version"`
	Command  string            `yaml:"command,omitempty"`
	Commands []TestCommand     `yaml:"commands,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
}

// TestCommand is a named test command.
type TestCommand struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// TestOptions configures test execution.
type TestOptions struct {
	Name    string   // Run specific named command from .core/test.yaml
	Command []string // Override command (from -- args)
}

// Test runs tests in the dev environment.
//
// Usage:
//
//	if r := dev.Test(ctx, ".", devenv.TestOptions{}); !r.OK { return r }
func (d *DevOps) Test(ctx context.Context, projectDir string, opts TestOptions) core.Result { // Value: nil
	runningRes := d.IsRunning(ctx)
	if !runningRes.OK {
		return runningRes
	}
	if !core.MustCast[bool](runningRes) {
		return core.Fail(core.E("DevOps.Test", "dev environment not running (run 'core dev boot' first)", nil))
	}

	var cmd string

	// Priority: explicit command > named command > auto-detect
	if len(opts.Command) > 0 {
		cmd = core.Join(" ", opts.Command...)
	} else if opts.Name != "" {
		cfgRes := LoadTestConfig(d.medium, projectDir)
		if !cfgRes.OK {
			return cfgRes
		}
		cfg := core.MustCast[*TestConfig](cfgRes)
		for _, c := range cfg.Commands {
			if c.Name == opts.Name {
				cmd = c.Run
				break
			}
		}
		if cmd == "" {
			return core.Fail(core.E("DevOps.Test", "test command "+opts.Name+" not found in .core/test.yaml", nil))
		}
	} else {
		cmd = DetectTestCommand(d.medium, projectDir)
		if cmd == "" {
			return core.Fail(core.E("DevOps.Test", "could not detect test command (create .core/test.yaml)", nil))
		}
	}

	// Run via SSH - construct command as single string for shell execution
	return d.sshShell(ctx, []string{"cd", "/app", "&&", cmd})
}

// DetectTestCommand auto-detects the test command for a project.
//
// Usage:
//
//	cmd := DetectTestCommand(io.Local, ".")
func DetectTestCommand(m io.Medium, projectDir string) string {
	// 1. Check .core/test.yaml
	if cfgRes := LoadTestConfig(m, projectDir); cfgRes.OK {
		if cfg := core.MustCast[*TestConfig](cfgRes); cfg.Command != "" {
			return cfg.Command
		}
	}

	// 2. Check composer.json for test script
	if hasFile(m, projectDir, "composer.json") {
		if hasComposerScript(m, projectDir, "test") {
			return "composer test"
		}
	}

	// 3. Check package.json for test script
	if hasFile(m, projectDir, "package.json") {
		if hasPackageScript(m, projectDir, "test") {
			return "npm test"
		}
	}

	// 4. Check go.mod
	if hasFile(m, projectDir, "go.mod") {
		return "go test ./..."
	}

	// 5. Check pytest
	if hasFile(m, projectDir, "pytest.ini") || hasFile(m, projectDir, "pyproject.toml") {
		return "pytest"
	}

	// 6. Check Taskfile
	if hasFile(m, projectDir, "Taskfile.yaml") || hasFile(m, projectDir, "Taskfile.yml") {
		return "task test"
	}

	return ""
}

// LoadTestConfig loads .core/test.yaml.
//
// Usage:
//
//	cfg := core.MustCast[*TestConfig](LoadTestConfig(io.Local, "."))
func LoadTestConfig(m io.Medium, projectDir string) core.Result { // Value: *TestConfig
	absPath := coreutil.AbsPath(coreutil.JoinPath(projectDir, ".core", "test.yaml"))

	content, err := m.Read(absPath)
	if err != nil {
		return core.Fail(core.E("LoadTestConfig", "read test config", err))
	}

	var cfg TestConfig
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return core.Fail(core.E("LoadTestConfig", "parse test config", err))
	}

	return core.Ok(&cfg)
}

func hasFile(m io.Medium, dir, name string) bool {
	absPath := coreutil.AbsPath(coreutil.JoinPath(dir, name))
	return m.IsFile(absPath)
}

func hasPackageScript(m io.Medium, projectDir, script string) bool {
	absPath := coreutil.AbsPath(coreutil.JoinPath(projectDir, "package.json"))

	content, err := m.Read(absPath)
	if err != nil {
		return false
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	result := core.JSONUnmarshalString(content, &pkg)
	if !result.OK {
		return false
	}

	_, ok := pkg.Scripts[script]
	return ok
}

func hasComposerScript(m io.Medium, projectDir, script string) bool {
	absPath := coreutil.AbsPath(coreutil.JoinPath(projectDir, "composer.json"))

	content, err := m.Read(absPath)
	if err != nil {
		return false
	}

	var pkg struct {
		Scripts map[string]any `json:"scripts"`
	}
	result := core.JSONUnmarshalString(content, &pkg)
	if !result.OK {
		return false
	}

	_, ok := pkg.Scripts[script]
	return ok
}
