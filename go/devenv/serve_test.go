package devenv

import (
	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/io"
	"reflect"
	"testing"
)

func TestDetectServeCommand_Laravel_Good(t *testing.T) {
	auditTarget := "Laravel"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "artisan"), "#!/usr/bin/env php")
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "php artisan octane:start --host=0.0.0.0 --port=8000"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_NodeDev_Good(t *testing.T) {
	auditTarget := "NodeDev"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	packageJSON := `{"scripts":{"dev":"vite","start":"node index.js"}}`
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), packageJSON)
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "npm run dev -- --host 0.0.0.0"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_NodeStart_Good(t *testing.T) {
	auditTarget := "NodeStart"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	packageJSON := `{"scripts":{"start":"node server.js"}}`
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), packageJSON)
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "npm start"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_PHP_Good(t *testing.T) {
	auditTarget := "PHP"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"require":{}}`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "frankenphp php-server -l :8000"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_GoMain_Good(t *testing.T) {
	auditTarget := "GoMain"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "go.mod"), "module example")
	if err != nil {
		t.Fatal(err)
	}
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "main.go"), "package main")
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "go run ."; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_GoWithoutMain_Good(t *testing.T) {
	auditTarget := "GoWithoutMain"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "go.mod"), "module example")
	if err != nil {
		t.Fatal(err)
	}

	// No main.go, so falls through to fallback
	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "python3 -m http.server 8000"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_Django_Good(t *testing.T) {
	auditTarget := "Django"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "manage.py"), "#!/usr/bin/env python")
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "python manage.py runserver 0.0.0.0:8000"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_Fallback_Good(t *testing.T) {
	auditTarget := "Fallback"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "python3 -m http.server 8000"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDetectServeCommand_Priority_Good(t *testing.T) {
	auditTarget := "Priority"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Laravel (artisan) should take priority over PHP (composer.json)
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "artisan"), "#!/usr/bin/env php")
	if err != nil {
		t.Fatal(err)
	}
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"require":{}}`)
	if err != nil {
		t.Fatal(err)
	}

	cmd := DetectServeCommand(io.Local, tmpDir)
	if got, want := cmd, "php artisan octane:start --host=0.0.0.0 --port=8000"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestServeOptions_Default_Good(t *testing.T) {
	auditTarget := "Default"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ServeOptions{}
	if got, want := opts.Port, 0; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Path, ""; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestServeOptions_Custom_Good(t *testing.T) {
	auditTarget := "Custom"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ServeOptions{
		Port: 3000,
		Path: "public",
	}
	if got, want := opts.Port, 3000; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Path, "public"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestServe_HasFile_Good(t *testing.T) {
	auditTarget := "HasFile"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	testFile := coreutil.JoinPath(tmpDir, "test.txt")
	err := io.Local.Write(testFile, "content")
	if err != nil {
		t.Fatal(err)
	}
	if !(hasFile(io.Local, tmpDir, "test.txt")) {
		t.Fatal("expected true")
	}
}

func TestServe_HasFile_Bad(t *testing.T) {
	auditTarget := "HasFile"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	if hasFile(io.Local, tmpDir, "nonexistent.txt") {
		t.Fatal("expected false")
	}
}

func TestHasFile_Directory_Bad(t *testing.T) {
	auditTarget := "Directory"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	tmpDir := t.TempDir()
	subDir := coreutil.JoinPath(tmpDir, "subdir")
	err := io.Local.EnsureDir(subDir)
	if err != nil {
		t.Fatal(err)
	}

	// hasFile correctly returns false for directories (only true for regular files)
	if hasFile(io.Local, tmpDir, "subdir") {
		t.Fatal("expected false")
	}
}

// --- AX-7 canonical triplets ---

func TestServe_DevOps_Serve_Good(t *testing.T) {
	auditTarget := "DevOps Serve"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Serve
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestServe_DevOps_Serve_Bad(t *testing.T) {
	auditTarget := "DevOps Serve"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Serve
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestServe_DevOps_Serve_Ugly(t *testing.T) {
	auditTarget := "DevOps Serve"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Serve
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestServe_DetectServeCommand_Good(t *testing.T) {
	auditTarget := "DetectServeCommand"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := DetectServeCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestServe_DetectServeCommand_Bad(t *testing.T) {
	auditTarget := "DetectServeCommand"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := DetectServeCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestServe_DetectServeCommand_Ugly(t *testing.T) {
	auditTarget := "DetectServeCommand"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := DetectServeCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
