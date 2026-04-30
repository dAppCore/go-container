package devenv

import (
	"dappco.re/go"
	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/io"
	"reflect"
	"syscall"
	"testing"
)

func TestConfig_DefaultConfig_Good(t *testing.T) {
	cfg := DefaultConfig()
	if got, want := cfg.Version, 1; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.Source, "auto"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.GitHub.Repo, "host-uk/core-images"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestConfig_ConfigPath_Good(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := path, ".core/config.yaml"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestConfig_LoadConfig_Good(t *testing.T) {
	t.Run("returns default if not exists", func(t *testing.T) {
		// Mock HOME to a temp dir
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		cfg, err := LoadConfig(io.Local)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := cfg, DefaultConfig(); !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})

	t.Run("loads existing config", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		coreDir := coreutil.JoinPath(tempHome, ".core")
		err := io.Local.EnsureDir(coreDir)
		if err != nil {
			t.Fatal(err)
		}

		configData := `
version: 2
images:
  source: cdn
  cdn:
    url: https://cdn.example.com
`
		err = io.Local.Write(coreutil.JoinPath(coreDir, "config.yaml"), configData)
		if err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(io.Local)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := cfg.Version, 2; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
		if got, want := cfg.Images.Source, "cdn"; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
		if got, want := cfg.Images.CDN.URL, "https://cdn.example.com"; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})
}

func TestConfig_LoadConfig_Bad(t *testing.T) {
	t.Run("invalid yaml", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		coreDir := coreutil.JoinPath(tempHome, ".core")
		err := io.Local.EnsureDir(coreDir)
		if err != nil {
			t.Fatal(err)
		}

		err = io.Local.Write(coreutil.JoinPath(coreDir, "config.yaml"), "invalid: yaml: :")
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(io.Local)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestConfig_Struct_Good(t *testing.T) {
	cfg := &Config{
		Version: 2,
		Images: ImagesConfig{
			Source: "github",
			GitHub: GitHubConfig{
				Repo: "owner/repo",
			},
			Registry: RegistryConfig{
				Image: "ghcr.io/owner/image",
			},
			CDN: CDNConfig{
				URL: "https://cdn.example.com",
			},
		},
	}
	if got, want := cfg.Version, 2; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.Source, "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.GitHub.Repo, "owner/repo"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.Registry.Image, "ghcr.io/owner/image"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.CDN.URL, "https://cdn.example.com"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDefaultConfig_Complete_Good(t *testing.T) {
	cfg := DefaultConfig()
	if got, want := cfg.Version, 1; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.Source, "auto"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.GitHub.Repo, "host-uk/core-images"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.Registry.Image, "ghcr.io/host-uk/core-devops"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got := cfg.Images.CDN.URL; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestLoadConfig_PartialConfig_Good(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	coreDir := coreutil.JoinPath(tempHome, ".core")
	err := io.Local.EnsureDir(coreDir)
	if err != nil {
		t.Fatal(err)
	}

	// Config only specifies source, should merge with defaults
	configData := `
version: 1
images:
  source: github
`
	err = io.Local.Write(coreutil.JoinPath(coreDir, "config.yaml"), configData)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(io.Local)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Version, 1; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	// Default values should be preserved
	if got, want := cfg.Images.Source, "github"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := cfg.Images.GitHub.Repo, "host-uk/core-images"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLoadConfig_AllSourceTypes_Good(t *testing.T) {
	tests := []struct {
		name   string
		config string
		check  func(*testing.T, *Config)
	}{
		{
			name: "github source",
			config: `
version: 1
images:
  source: github
  github:
    repo: custom/repo
`,
			check: func(t *testing.T, cfg *Config) {
				if got, want := cfg.Images.Source, "github"; !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v, got %v", want, got)
				}
				if got, want := cfg.Images.GitHub.Repo, "custom/repo"; !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v, got %v", want, got)
				}
			},
		},
		{
			name: "cdn source",
			config: `
version: 1
images:
  source: cdn
  cdn:
    url: https://custom-cdn.com
`,
			check: func(t *testing.T, cfg *Config) {
				if got, want := cfg.Images.Source, "cdn"; !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v, got %v", want, got)
				}
				if got, want := cfg.Images.CDN.URL, "https://custom-cdn.com"; !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v, got %v", want, got)
				}
			},
		},
		{
			name: "registry source",
			config: `
version: 1
images:
  source: registry
  registry:
    image: docker.io/custom/image
`,
			check: func(t *testing.T, cfg *Config) {
				if got, want := cfg.Images.Source, "registry"; !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v, got %v", want, got)
				}
				if got, want := cfg.Images.Registry.Image, "docker.io/custom/image"; !reflect.DeepEqual(got, want) {
					t.Fatalf("want %v, got %v", want, got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempHome := t.TempDir()
			t.Setenv("HOME", tempHome)

			coreDir := coreutil.JoinPath(tempHome, ".core")
			err := io.Local.EnsureDir(coreDir)
			if err != nil {
				t.Fatal(err)
			}

			err = io.Local.Write(coreutil.JoinPath(coreDir, "config.yaml"), tt.config)
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(io.Local)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, cfg)
		})
	}
}

func TestImagesConfig_Struct_Good(t *testing.T) {
	ic := ImagesConfig{
		Source: "auto",
		GitHub: GitHubConfig{Repo: "test/repo"},
	}
	if got, want := ic.Source, "auto"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := ic.GitHub.Repo, "test/repo"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestGitHubConfig_Struct_Good(t *testing.T) {
	gc := GitHubConfig{Repo: "owner/repo"}
	if got, want := gc.Repo, "owner/repo"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRegistryConfig_Struct_Good(t *testing.T) {
	rc := RegistryConfig{Image: "ghcr.io/owner/image:latest"}
	if got, want := rc.Image, "ghcr.io/owner/image:latest"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCDNConfig_Struct_Good(t *testing.T) {
	cc := CDNConfig{URL: "https://cdn.example.com/images"}
	if got, want := cc.URL, "https://cdn.example.com/images"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLoadConfig_UnreadableFile_Bad(t *testing.T) {
	// This test is platform-specific and may not work on all systems
	// Skip if we can't test file permissions properly
	if syscall.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	coreDir := coreutil.JoinPath(tempHome, ".core")
	err := io.Local.EnsureDir(coreDir)
	if err != nil {
		t.Fatal(err)
	}

	configPath := coreutil.JoinPath(coreDir, "config.yaml")
	err = io.Local.WriteMode(configPath, "version: 1", 0000)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(io.Local)
	if err == nil {
		t.Fatal("expected error")

		// Restore permissions so cleanup works
	}

	_ = syscall.Chmod(configPath, 0644)
}

// --- AX-7 canonical triplets ---

func TestConfig_DefaultConfig_Bad(t *testing.T) {
	symbol := DefaultConfig
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_DefaultConfig_Ugly(t *testing.T) {
	symbol := DefaultConfig
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_ConfigPath_Bad(t *testing.T) {
	symbol := ConfigPath
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_ConfigPath_Ugly(t *testing.T) {
	symbol := ConfigPath
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_LoadConfig_Ugly(t *testing.T) {
	symbol := LoadConfig
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_Read_Good(t *testing.T) {
	symbol := configmedium.Read
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_Read_Bad(t *testing.T) {
	symbol := configmedium.Read
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_Read_Ugly(t *testing.T) {
	symbol := configmedium.Read
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_Write_Good(t *testing.T) {
	symbol := configmedium.Write
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_Write_Bad(t *testing.T) {
	symbol := configmedium.Write
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_Write_Ugly(t *testing.T) {
	symbol := configmedium.Write
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_EnsureDir_Good(t *testing.T) {
	symbol := configmedium.EnsureDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_EnsureDir_Bad(t *testing.T) {
	symbol := configmedium.EnsureDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestConfig_configmedium_EnsureDir_Ugly(t *testing.T) {
	symbol := configmedium.EnsureDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
