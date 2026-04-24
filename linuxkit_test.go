package container

import (
	"context"

	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/container/internal/proc"
	core "dappco.re/go/core"

	"dappco.re/go/io"
	"errors"
	"reflect"
	"slices"
	"syscall"
	"testing"
	"time"
)

// MockHypervisor is a mock implementation for testing.
type MockHypervisor struct {
	name         string
	available    bool
	buildErr     error
	lastImage    string
	lastOpts     *HypervisorOptions
	commandToRun string
}

func NewMockHypervisor() *MockHypervisor {
	return &MockHypervisor{
		name:         "mock",
		available:    true,
		commandToRun: "echo",
	}
}

func (m *MockHypervisor) Name() string {
	return m.name
}

func (m *MockHypervisor) Available() bool {
	return m.available
}

func (m *MockHypervisor) BuildCommand(ctx context.Context, image string, opts *HypervisorOptions) (*proc.Command, error) {
	m.lastImage = image
	m.lastOpts = opts
	if m.buildErr != nil {
		return nil, m.buildErr
	}
	// Return a simple command that exits quickly
	return proc.NewCommandContext(ctx, m.commandToRun, "test"), nil
}

// newTestManager creates a LinuxKitManager with mock hypervisor for testing.
// Uses manual temp directory management to avoid race conditions with t.TempDir cleanup.
func newTestManager(t *testing.T) (*LinuxKitManager, *MockHypervisor, string) {
	tmpDir, err := coreutil.MkdirTemp("linuxkit-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORE_HOME", tmpDir)

	// Manual cleanup that handles race conditions with state file writes
	t.Cleanup(func() {
		// Give any pending file operations time to complete
		time.Sleep(10 * time.Millisecond)
		_ = io.Local.DeleteAll(tmpDir)
	})

	statePath := coreutil.JoinPath(tmpDir, "containers.json")

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}

	mock := NewMockHypervisor()
	manager := NewLinuxKitManagerWithHypervisor(io.Local, state, mock)

	return manager, mock, tmpDir
}

func TestLinuxKit_NewLinuxKitManagerWithHypervisor_Good(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state, _ := LoadState(statePath)
	mock := NewMockHypervisor()

	manager := NewLinuxKitManagerWithHypervisor(io.Local, state, mock)
	if manager == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := manager.State(), state; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := manager.Hypervisor(), mock; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLinuxKitManager_Run_Detached_Good(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	// Create a test image file
	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Use a command that runs briefly then exits
	mock.commandToRun = "sleep"

	ctx := context.Background()
	opts := RunOptions{
		Name:   "test-vm",
		Detach: true,
		Memory: 512,
		CPUs:   2,
	}

	container, err := manager.Run(ctx, imagePath, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := container.ID; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
	if got, want := container.Name, "test-vm"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := container.Image, imagePath; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := container.Status, StatusRunning; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := container.PID, 0; got <= want {
		t.Fatalf("want greater than %v, got %v", want, got)
	}
	if got, want := container.Memory, 512; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := container.CPUs, 2; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Verify hypervisor was called with correct options
	if got, want := mock.lastImage, imagePath; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := mock.lastOpts.Memory, 512; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := mock.lastOpts.CPUs, 2; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Clean up - stop the container
	time.Sleep(100 * time.Millisecond)
}

func TestLinuxKitManager_Run_DefaultValues_Good(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	imagePath := coreutil.JoinPath(tmpDir, "test.qcow2")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	opts := RunOptions{Detach: true}

	container, err := manager.Run(ctx, imagePath, opts)
	if err != nil {
		t.Fatal(err)
	}
	// Check defaults were applied
	if got, want := mock.lastOpts.Memory, 1024; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := mock.lastOpts.CPUs, 1; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := mock.lastOpts.SSHPort, 2222; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Name should default to first 8 chars of ID
	if got, want := container.Name, container.ID[:8]; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Wait for the mock process to complete to avoid temp dir cleanup issues
	time.Sleep(50 * time.Millisecond)
}

func TestLinuxKitManager_Run_ImageNotFound_Bad(t *testing.T) {
	manager, _, _ := newTestManager(t)

	ctx := context.Background()
	opts := RunOptions{Detach: true}

	_, err := manager.Run(ctx, "/nonexistent/image.iso", opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "image not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Run_UnsupportedFormat_Bad(t *testing.T) {
	manager, _, tmpDir := newTestManager(t)

	imagePath := coreutil.JoinPath(tmpDir, "test.txt")
	err := io.Local.Write(imagePath, "not an image")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	opts := RunOptions{Detach: true}

	_, err = manager.Run(ctx, imagePath, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "unsupported image format"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Stop_Good(t *testing.T) {
	manager, _, _ := newTestManager(t)

	// Add a fake running container with a non-existent PID
	// The Stop function should handle this gracefully
	container := &Container{
		ID:        "abc12345",
		Status:    StatusRunning,
		PID:       999999, // Non-existent PID
		StartedAt: time.Now(),
	}
	_ = manager.State().Add(container)

	ctx := context.Background()
	err := manager.Stop(ctx, "abc12345")
	// Stop should succeed (process doesn't exist, so container is marked stopped)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the container status was updated
	c, ok := manager.State().Get("abc12345")
	if !(ok) {
		t.Fatal("expected true")
	}
	if got, want := c.Status, StatusStopped; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLinuxKitManager_Stop_NotFound_Bad(t *testing.T) {
	manager, _, _ := newTestManager(t)

	ctx := context.Background()
	err := manager.Stop(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "container not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Stop_NotRunning_Bad(t *testing.T) {
	_, _, tmpDir := newTestManager(t)
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewLinuxKitManagerWithHypervisor(io.Local, state, NewMockHypervisor())

	container := &Container{
		ID:     "abc12345",
		Status: StatusStopped,
	}
	_ = state.Add(container)

	ctx := context.Background()
	err = manager.Stop(ctx, "abc12345")
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "not running"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_List_Good(t *testing.T) {
	_, _, tmpDir := newTestManager(t)
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewLinuxKitManagerWithHypervisor(io.Local, state, NewMockHypervisor())

	_ = state.Add(&Container{ID: "aaa11111", Status: StatusStopped})
	_ = state.Add(&Container{ID: "bbb22222", Status: StatusStopped})

	ctx := context.Background()
	containers, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(containers), 2; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

func TestLinuxKitManager_List_VerifiesRunningStatus_Good(t *testing.T) {
	_, _, tmpDir := newTestManager(t)
	statePath := coreutil.JoinPath(tmpDir, "containers.json")
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewLinuxKitManagerWithHypervisor(io.Local, state, NewMockHypervisor())

	// Add a "running" container with a fake PID that doesn't exist
	_ = state.Add(&Container{
		ID:     "abc12345",
		Status: StatusRunning,
		PID:    999999, // PID that almost certainly doesn't exist
	})

	ctx := context.Background()
	containers, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(containers), 1; got !=
		// Status should have been updated to stopped since PID doesn't exist
		want {
		t.Fatalf("want len %v, got %v", want, got)
	}
	if got, want := containers[0].Status, StatusStopped; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLinuxKitManager_Logs_Good(t *testing.T) {
	manager, _, tmpDir := newTestManager(t)

	// Create a log file manually
	logsDir := coreutil.JoinPath(tmpDir, "logs")
	if err := io.Local.EnsureDir(logsDir); err != nil {
		t.Fatal(err)
	}

	container := &Container{ID: "abc12345"}
	_ = manager.State().Add(container)

	// Override the default logs dir for testing by creating the log file
	// at the expected location
	logContent := "test log content\nline 2\n"
	logPath, err := LogPath("abc12345")
	if err != nil {
		t.Fatal(err)
	}
	if err := io.Local.EnsureDir(core.PathDir(logPath)); err != nil {
		t.Fatal(err)
	}
	if err := io.Local.Write(logPath, logContent); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader, err := manager.Logs(ctx, "abc12345", false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()

	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	if got, want := string(buf[:n]), logContent; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLinuxKitManager_Logs_NotFound_Bad(t *testing.T) {
	manager, _, _ := newTestManager(t)

	ctx := context.Background()
	_, err := manager.Logs(ctx, "nonexistent", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "container not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Logs_NoLogFile_Bad(t *testing.T) {
	manager, _, _ := newTestManager(t)

	// Use a unique ID that won't have a log file
	uniqueID, err := GenerateID()
	if err != nil {
		t.Fatal(err)
	}
	container := &Container{ID: uniqueID}
	_ = manager.State().Add(container)

	ctx := context.Background()
	reader, err := manager.Logs(ctx, uniqueID, false)

	// If logs existed somehow, clean up the reader
	if reader != nil {
		_ = reader.Close()
	}
	if err == nil {
		t.Fatal("expected error")
	}
	if err != nil {
		if s, sub := err.Error(), "no logs available"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	}
}

func TestLinuxKitManager_Exec_NotFound_Bad(t *testing.T) {
	manager, _, _ := newTestManager(t)

	ctx := context.Background()
	err := manager.Exec(ctx, "nonexistent", []string{"ls"})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "container not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Exec_NotRunning_Bad(t *testing.T) {
	manager, _, _ := newTestManager(t)

	container := &Container{ID: "abc12345", Status: StatusStopped}
	_ = manager.State().Add(container)

	ctx := context.Background()
	err := manager.Exec(ctx, "abc12345", []string{"ls"})
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "not running"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKit_DetectImageFormat_Good(t *testing.T) {
	tests := []struct {
		path   string
		format ImageFormat
	}{
		{"/path/to/image.iso", FormatISO},
		{"/path/to/image.ISO", FormatISO},
		{"/path/to/image.qcow2", FormatQCOW2},
		{"/path/to/image.QCOW2", FormatQCOW2},
		{"/path/to/image.vmdk", FormatVMDK},
		{"/path/to/image.raw", FormatRaw},
		{"/path/to/image.img", FormatRaw},
		{"image.iso", FormatISO},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got, want := DetectImageFormat(tt.path), tt.format; !reflect.DeepEqual(got, want) {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}

func TestDetectImageFormat_Unknown_Bad(t *testing.T) {
	tests := []string{
		"/path/to/image.txt",
		"/path/to/image",
		"noextension",
		"/path/to/image.docx",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			if got, want := DetectImageFormat(path), FormatUnknown; !reflect.DeepEqual(got, want) {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}

func TestQemuHypervisor_Name_Good(t *testing.T) {
	q := NewQemuHypervisor()
	if got, want := q.Name(), "qemu"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestQemuHypervisor_BuildCommand_Good(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{
		Memory:  2048,
		CPUs:    4,
		SSHPort: 2222,
		Ports:   map[int]int{8080: 80},
		Detach:  true,
	}

	cmd, err := q.BuildCommand(ctx, "/path/to/image.iso", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")

		// Check command path
	}
	if s, sub := cmd.Path, "qemu"; !core.

		// Check that args contain expected values
		Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}

	args := cmd.Args
	if s, sub := args, "-m"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "2048"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "-smp"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "4"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "-nographic"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Logs_Follow_Good(t *testing.T) {
	manager, _, _ := newTestManager(t)

	// Create a unique container ID
	uniqueID, err := GenerateID()
	if err != nil {
		t.Fatal(err)
	}
	container := &Container{ID: uniqueID}
	_ = manager.State().Add(container)

	// Create a log file at the expected location
	logPath, err := LogPath(uniqueID)
	if err != nil {
		t.Fatal(err)
	}
	if err := io.Local.EnsureDir(core.PathDir(logPath)); err !=

		// Write initial content
		nil {
		t.Fatal(err)
	}

	err = io.Local.Write(logPath, "initial log content\n")
	if err != nil {
		t.Fatal(err)
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Get the follow reader
	reader, err := manager.Logs(ctx, uniqueID, true)
	if err != nil {
		t.Fatal(err)
	}

	// Cancel the context to stop the follow
	cancel()

	// Read should return EOF after context cancellation
	buf := make([]byte, 1024)
	_, readErr := reader.Read(buf)

	// After context cancel, Read should return EOF
	if got, want := readErr.Error(), "EOF"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Close the reader
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFollowReader_Read_WithData_Good(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := coreutil.JoinPath(tmpDir, "test.log")

	// Create log file with content
	content := "test log line 1\ntest log line 2\n"
	err := io.Local.Write(logPath, content)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reader, err := newFollowReader(ctx, io.Local, logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()

	// The followReader seeks to end, so we need to append more content
	f, err := io.Local.Append(logPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write([]byte("new line\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err !=

		// Give the reader time to poll
		nil {
		t.Fatal(err)
	}

	time.Sleep(150 * time.Millisecond)

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err == nil {
		if got, want := n, 0; got <= want {
			t.Fatalf("want greater than %v, got %v", want, got)
		}
	}
}

func TestFollowReader_Read_ContextCancel_Good(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := coreutil.JoinPath(tmpDir, "test.log")

	// Create log file
	err := io.Local.Write(logPath, "initial content\n")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	reader, err := newFollowReader(ctx, io.Local, logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Cancel the context
	cancel()

	// Read should return EOF
	buf := make([]byte, 1024)
	_, readErr := reader.Read(buf)
	if got, want := readErr.Error(), "EOF"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	_ = reader.Close()
}

func TestFollowReader_Close_Good(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := coreutil.JoinPath(tmpDir, "test.log")

	err := io.Local.Write(logPath, "content\n")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader, err := newFollowReader(ctx, io.Local, logPath)
	if err != nil {
		t.Fatal(err)
	}

	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Reading after close should fail or return EOF
	buf := make([]byte, 1024)
	_, readErr := reader.Read(buf)
	if readErr == nil {
		t.Fatal("expected error")
	}
}

func TestNewFollowReader_FileNotFound_Bad(t *testing.T) {
	ctx := context.Background()
	_, err := newFollowReader(ctx, io.Local, "/nonexistent/path/to/file.log")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLinuxKitManager_Run_BuildCommandError_Bad(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	// Create a test image file
	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Configure mock to return an error
	mock.buildErr = errors.New("test error")

	ctx := context.Background()
	opts := RunOptions{Detach: true}

	_, err = manager.Run(ctx, imagePath, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "failed to build hypervisor command"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Run_Foreground_Good(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	// Create a test image file
	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Use echo which exits quickly
	mock.commandToRun = "echo"

	ctx := context.Background()
	opts := RunOptions{
		Name:   "test-foreground",
		Detach: false, // Run in foreground
		Memory: 512,
		CPUs:   1,
	}

	container, err := manager.Run(ctx, imagePath, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := container.ID; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
	if got, want := container.Name, "test-foreground"; !reflect.DeepEqual(
		// Foreground process should have completed
		got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := container.Status, StatusStopped; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLinuxKitManager_Stop_ContextCancelled_Good(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	// Create a test image file
	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Use a command that takes a long time
	mock.commandToRun = "sleep"

	// Start a container
	ctx := context.Background()
	opts := RunOptions{
		Name:   "test-cancel",
		Detach: true,
	}

	container, err := manager.Run(ctx, imagePath, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure cleanup happens regardless of test outcome
	t.Cleanup(func() {
		_ = manager.Stop(context.Background(), container.ID)
	})

	// Create a context that's already cancelled
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Stop with cancelled context
	err = manager.Stop(cancelCtx, container.ID)
	// Should return context error
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err, context.Canceled; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestIsProcessRunning_ExistingProcess_Good(t *testing.T) {
	// Use our own PID which definitely exists
	running := isProcessRunning(syscall.Getpid())
	if !(running) {
		t.Fatal("expected true")
	}
}

func TestIsProcessRunning_NonexistentProcess_Bad(t *testing.T) {
	// Use a PID that almost certainly doesn't exist
	running := isProcessRunning(999999)
	if running {
		t.Fatal("expected false")
	}
}

func TestLinuxKitManager_Run_WithPortsAndVolumes_Good(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	opts := RunOptions{
		Name:    "test-ports",
		Detach:  true,
		Memory:  512,
		CPUs:    1,
		SSHPort: 2223,
		Ports:   map[int]int{8080: 80, 443: 443},
		Volumes: map[string]string{"/host/data": "/container/data"},
	}

	container, err := manager.Run(ctx, imagePath, opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := container.ID; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
	if got, want := container.Ports, map[int]int{8080: 80, 443: 443}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := mock.lastOpts.SSHPort, 2223; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := mock.lastOpts.Volumes, map[string]string{"/host/data": "/container/data"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	time.Sleep(50 * time.Millisecond)
}

func TestFollowReader_Read_ReaderError_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := coreutil.JoinPath(tmpDir, "test.log")

	// Create log file
	err := io.Local.Write(logPath, "content\n")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader, err := newFollowReader(ctx, io.Local, logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Close the underlying file to cause read errors
	_ = reader.file.Close()

	// Read should return an error
	buf := make([]byte, 1024)
	_, readErr := reader.Read(buf)
	if readErr == nil {
		t.Fatal("expected error")
	}
}

func TestLinuxKitManager_Run_StartError_Bad(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Use a command that doesn't exist to cause Start() to fail
	mock.commandToRun = "/nonexistent/command/that/does/not/exist"

	ctx := context.Background()
	opts := RunOptions{
		Name:   "test-start-error",
		Detach: true,
	}

	_, err = manager.Run(ctx, imagePath, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "failed to start VM"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Run_ForegroundStartError_Bad(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Use a command that doesn't exist to cause Start() to fail
	mock.commandToRun = "/nonexistent/command/that/does/not/exist"

	ctx := context.Background()
	opts := RunOptions{
		Name:   "test-foreground-error",
		Detach: false,
	}

	_, err = manager.Run(ctx, imagePath, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "failed to start VM"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestLinuxKitManager_Run_ForegroundWithError_Good(t *testing.T) {
	manager, mock, tmpDir := newTestManager(t)

	imagePath := coreutil.JoinPath(tmpDir, "test.iso")
	err := io.Local.Write(imagePath, "fake image")
	if err != nil {
		t.Fatal(err)
	}

	// Use a command that exits with error
	mock.commandToRun = "false" // false command exits with code 1

	ctx := context.Background()
	opts := RunOptions{
		Name:   "test-foreground-exit-error",
		Detach: false,
	}

	container, err := manager.Run(ctx, imagePath, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Container should be in error state since process exited with error
	if got, want := container.Status, StatusError; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestLinuxKitManager_Stop_ProcessExitedWhileRunning_Good(t *testing.T) {
	manager, _, _ := newTestManager(t)

	// Add a "running" container with a process that has already exited
	// This simulates the race condition where process exits between status check
	// and signal send
	container := &Container{
		ID:        "test1234",
		Status:    StatusRunning,
		PID:       999999, // Non-existent PID
		StartedAt: time.Now(),
	}
	_ = manager.State().Add(container)

	ctx := context.Background()
	err := manager.Stop(ctx, "test1234")
	// Stop should succeed gracefully
	if err != nil {
		t.Fatal(err)
	}

	// Container should be stopped
	c, ok := manager.State().Get("test1234")
	if !(ok) {
		t.Fatal("expected true")
	}
	if got, want := c.Status, StatusStopped; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
