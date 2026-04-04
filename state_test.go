package container

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/io"

	"dappco.re/go/core/container/internal/coreutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_NewState_Good(t *testing.T) {
	state := NewState("/tmp/test-state.json")

	assert.NotNil(t, state)
	assert.NotNil(t, state.Containers)
	assert.Equal(t, "/tmp/test-state.json", state.FilePath())
}

func TestLoadState_NewFile_Good(t *testing.T) {
	// Test loading from non-existent file
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")

	state, err := LoadState(statePath)

	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.Empty(t, state.Containers)
}

func TestLoadState_ExistingFile_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")

	// Create a state file with data
	content := `{
		"containers": {
			"abc12345": {
				"id": "abc12345",
				"name": "test-container",
				"image": "/path/to/image.iso",
				"status": "running",
				"pid": 12345,
				"started_at": "2024-01-01T00:00:00Z"
			}
		}
	}`
	err := io.Local.Write(statePath, content)
	require.NoError(t, err)

	state, err := LoadState(statePath)

	require.NoError(t, err)
	assert.Len(t, state.Containers, 1)

	c, ok := state.Get("abc12345")
	assert.True(t, ok)
	assert.Equal(t, "test-container", c.Name)
	assert.Equal(t, StatusRunning, c.Status)
}

func TestLoadState_InvalidJSON_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")

	// Create invalid JSON
	err := io.Local.Write(statePath, "invalid json{")
	require.NoError(t, err)

	_, err = LoadState(statePath)
	assert.Error(t, err)
}

func TestState_Add_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state := NewState(statePath)

	container := &Container{
		ID:        "abc12345",
		Name:      "test",
		Image:     "/path/to/image.iso",
		Status:    StatusRunning,
		PID:       12345,
		StartedAt: time.Now(),
	}

	err := state.Add(container)
	require.NoError(t, err)

	// Verify it's in memory
	c, ok := state.Get("abc12345")
	assert.True(t, ok)
	assert.Equal(t, container.Name, c.Name)

	// Verify file was created
	assert.True(t, io.Local.IsFile(statePath))
}

func TestState_Update_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state := NewState(statePath)

	container := &Container{
		ID:     "abc12345",
		Status: StatusRunning,
	}
	_ = state.Add(container)

	// Update status
	container.Status = StatusStopped
	err := state.Update(container)
	require.NoError(t, err)

	// Verify update
	c, ok := state.Get("abc12345")
	assert.True(t, ok)
	assert.Equal(t, StatusStopped, c.Status)
}

func TestState_Remove_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state := NewState(statePath)

	container := &Container{
		ID: "abc12345",
	}
	_ = state.Add(container)

	err := state.Remove("abc12345")
	require.NoError(t, err)

	_, ok := state.Get("abc12345")
	assert.False(t, ok)
}

func TestState_Get_NotFound_Bad(t *testing.T) {
	state := NewState("/tmp/test-state.json")

	_, ok := state.Get("nonexistent")
	assert.False(t, ok)
}

func TestState_All_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state := NewState(statePath)

	_ = state.Add(&Container{ID: "aaa11111"})
	_ = state.Add(&Container{ID: "bbb22222"})
	_ = state.Add(&Container{ID: "ccc33333"})

	all := state.All()
	assert.Len(t, all, 3)
}

func TestState_SaveState_CreatesDirectory_Good(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := coreutil.JoinPath(tmpDir, "nested", "dir", "containers.json")
	state := NewState(nestedPath)

	_ = state.Add(&Container{ID: "abc12345"})

	err := state.SaveState()
	require.NoError(t, err)

	// Verify directory was created
	assert.True(t, io.Local.IsDir(core.PathDir(nestedPath)))
}

func TestState_DefaultStateDir_Good(t *testing.T) {
	dir, err := DefaultStateDir()
	require.NoError(t, err)
	assert.Contains(t, dir, ".core")
}

func TestState_DefaultStatePath_Good(t *testing.T) {
	path, err := DefaultStatePath()
	require.NoError(t, err)
	assert.Contains(t, path, "containers.json")
}

func TestState_DefaultLogsDir_Good(t *testing.T) {
	dir, err := DefaultLogsDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "logs")
}

func TestState_LogPath_Good(t *testing.T) {
	path, err := LogPath("abc12345")
	require.NoError(t, err)
	assert.Contains(t, path, "abc12345.log")
}

func TestState_EnsureLogsDir_Good(t *testing.T) {
	// This test creates real directories - skip in CI if needed
	err := EnsureLogsDir()
	assert.NoError(t, err)

	logsDir, _ := DefaultLogsDir()
	assert.True(t, io.Local.IsDir(logsDir))
}

func TestState_GenerateID_Good(t *testing.T) {
	id1, err := GenerateID()
	require.NoError(t, err)
	assert.Len(t, id1, 8)

	id2, err := GenerateID()
	require.NoError(t, err)
	assert.Len(t, id2, 8)

	// IDs should be different
	assert.NotEqual(t, id1, id2)
}
