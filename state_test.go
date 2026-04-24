package container

import (
	"dappco.re/go/container/internal/coreutil"
	core "dappco.re/go/core"
	"dappco.re/go/core/io"
	"reflect"
	"testing"
	"time"
)

func TestState_NewState_Good(t *testing.T) {
	state := NewState("/tmp/test-state.json")
	if state == nil {
		t.Fatal("expected non-nil value")
	}
	if state.Containers == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := state.FilePath(), "/tmp/test-state.json"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLoadState_NewFile_Good(t *testing.T) {
	// Test loading from non-existent file
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("expected non-nil value")
	}
	if got := state.Containers; len(got) != 0 {
		t.Fatal("expected empty value")
	}
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
	if err != nil {
		t.Fatal(err)
	}

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(state.Containers), 1; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}

	c, ok := state.Get("abc12345")
	if !(ok) {
		t.Fatal("expected true")
	}
	if got, want := c.Name, "test-container"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := c.Status, StatusRunning; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLoadState_InvalidJSON_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")

	// Create invalid JSON
	err := io.Local.Write(statePath, "invalid json{")
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadState(statePath)
	if err == nil {
		t.Fatal("expected error")
	}
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
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's in memory
	c, ok := state.Get("abc12345")
	if !(ok) {
		t.Fatal("expected true")
	}
	if got, want := c.Name, container.Name; !reflect.DeepEqual(

		// Verify file was created
		got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(io.Local.IsFile(statePath)) {
		t.Fatal("expected true")
	}
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
	if err != nil {
		t.Fatal(err)
	}

	// Verify update
	c, ok := state.Get("abc12345")
	if !(ok) {
		t.Fatal("expected true")
	}
	if got, want := c.Status, StatusStopped; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	_, ok := state.Get("abc12345")
	if ok {
		t.Fatal("expected false")
	}
}

func TestState_Get_NotFound_Bad(t *testing.T) {
	state := NewState("/tmp/test-state.json")

	_, ok := state.Get("nonexistent")
	if ok {
		t.Fatal("expected false")
	}
}

func TestState_All_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state := NewState(statePath)

	_ = state.Add(&Container{ID: "aaa11111"})
	_ = state.Add(&Container{ID: "bbb22222"})
	_ = state.Add(&Container{ID: "ccc33333"})

	all := state.All()
	if got, want := len(all), 3; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

func TestState_SaveState_CreatesDirectory_Good(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := coreutil.JoinPath(tmpDir, "nested", "dir", "containers.json")
	state := NewState(nestedPath)

	_ = state.Add(&Container{ID: "abc12345"})

	err := state.SaveState()
	if err != nil {
		t.Fatal(err)
	}

	// Verify directory was created
	if !io.Local.IsDir(core.PathDir(nestedPath)) {
		t.Fatal("expected true")
	}
}

func TestState_DefaultStateDir_Good(t *testing.T) {
	dir, err := DefaultStateDir()
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := dir, ".core"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestState_DefaultStatePath_Good(t *testing.T) {
	path, err := DefaultStatePath()
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := path, "containers.json"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestState_DefaultLogsDir_Good(t *testing.T) {
	dir, err := DefaultLogsDir()
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := dir, "logs"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestState_LogPath_Good(t *testing.T) {
	path, err := LogPath("abc12345")
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := path, "abc12345.log"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestState_EnsureLogsDir_Good(t *testing.T) {
	// This test creates real directories - skip in CI if needed
	err := EnsureLogsDir()
	if err != nil {
		t.Fatal(err)
	}

	logsDir, _ := DefaultLogsDir()
	if !(io.Local.IsDir(logsDir)) {
		t.Fatal("expected true")
	}
}

func TestState_GenerateID_Good(t *testing.T) {
	id1, err := GenerateID()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(id1), 8; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}

	id2, err := GenerateID()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(id2), 8; got !=

		// IDs should be different
		want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := id2, id1; reflect.DeepEqual(got, want) {
		t.Fatalf("did not expect %v", got)
	}
}
