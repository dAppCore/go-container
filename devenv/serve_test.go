package devenv

import (
	"testing"

	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestDetectServeCommand_Laravel_Good(t *testing.T) {
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "artisan"), "#!/usr/bin/env php")
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "php artisan octane:start --host=0.0.0.0 --port=8000", cmd)
}

func TestDetectServeCommand_NodeDev_Good(t *testing.T) {
	tmpDir := t.TempDir()
	packageJSON := `{"scripts":{"dev":"vite","start":"node index.js"}}`
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), packageJSON)
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "npm run dev -- --host 0.0.0.0", cmd)
}

func TestDetectServeCommand_NodeStart_Good(t *testing.T) {
	tmpDir := t.TempDir()
	packageJSON := `{"scripts":{"start":"node server.js"}}`
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "package.json"), packageJSON)
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "npm start", cmd)
}

func TestDetectServeCommand_PHP_Good(t *testing.T) {
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"require":{}}`)
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "frankenphp php-server -l :8000", cmd)
}

func TestDetectServeCommand_GoMain_Good(t *testing.T) {
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "go.mod"), "module example")
	assert.NoError(t, err)
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "main.go"), "package main")
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "go run .", cmd)
}

func TestDetectServeCommand_GoWithoutMain_Good(t *testing.T) {
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "go.mod"), "module example")
	assert.NoError(t, err)

	// No main.go, so falls through to fallback
	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "python3 -m http.server 8000", cmd)
}

func TestDetectServeCommand_Django_Good(t *testing.T) {
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "manage.py"), "#!/usr/bin/env python")
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "python manage.py runserver 0.0.0.0:8000", cmd)
}

func TestDetectServeCommand_Fallback_Good(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "python3 -m http.server 8000", cmd)
}

func TestDetectServeCommand_Priority_Good(t *testing.T) {
	// Laravel (artisan) should take priority over PHP (composer.json)
	tmpDir := t.TempDir()
	err := io.Local.Write(coreutil.JoinPath(tmpDir, "artisan"), "#!/usr/bin/env php")
	assert.NoError(t, err)
	err = io.Local.Write(coreutil.JoinPath(tmpDir, "composer.json"), `{"require":{}}`)
	assert.NoError(t, err)

	cmd := DetectServeCommand(io.Local, tmpDir)
	assert.Equal(t, "php artisan octane:start --host=0.0.0.0 --port=8000", cmd)
}

func TestServeOptions_Default_Good(t *testing.T) {
	opts := ServeOptions{}
	assert.Equal(t, 0, opts.Port)
	assert.Equal(t, "", opts.Path)
}

func TestServeOptions_Custom_Good(t *testing.T) {
	opts := ServeOptions{
		Port: 3000,
		Path: "public",
	}
	assert.Equal(t, 3000, opts.Port)
	assert.Equal(t, "public", opts.Path)
}

func TestServe_HasFile_Good(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := coreutil.JoinPath(tmpDir, "test.txt")
	err := io.Local.Write(testFile, "content")
	assert.NoError(t, err)

	assert.True(t, hasFile(io.Local, tmpDir, "test.txt"))
}

func TestServe_HasFile_Bad(t *testing.T) {
	tmpDir := t.TempDir()

	assert.False(t, hasFile(io.Local, tmpDir, "nonexistent.txt"))
}

func TestHasFile_Directory_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := coreutil.JoinPath(tmpDir, "subdir")
	err := io.Local.EnsureDir(subDir)
	assert.NoError(t, err)

	// hasFile correctly returns false for directories (only true for regular files)
	assert.False(t, hasFile(io.Local, tmpDir, "subdir"))
}
