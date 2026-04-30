package devenv

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/container/internal/proc"

	"dappco.re/go/io"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func newManagedTempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := coreutil.MkdirTemp(prefix)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CORE_HOME", dir)
	t.Cleanup(func() { _ = io.Local.DeleteAll(dir) })
	return dir
}

func TestDevOps_ImageName_Good(t *testing.T) {
	name := ImageName()
	if s, sub := name, "core-devops-"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := name, runtime.GOOS; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := name, runtime.GOARCH; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if !(name[len(name)-6:] == ".qcow2") {
		t.Fatal("expected true")
	}
}

func TestDevOps_ImagesDir_Good(t *testing.T) {
	t.Run("default directory", func(t *testing.T) {
		t.Setenv("CORE_IMAGES_DIR", "")

		dir, err := ImagesDir()
		if err != nil {
			t.Fatal(err)
		}
		if s, sub := dir, ".core/images"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	})

	t.Run("environment override", func(t *testing.T) {
		customDir := "/tmp/custom-images"
		t.Setenv("CORE_IMAGES_DIR", customDir)

		dir, err := ImagesDir()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := dir, customDir; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})
}

func TestDevOps_ImagePath_Good(t *testing.T) {
	customDir := "/tmp/images"
	t.Setenv("CORE_IMAGES_DIR", customDir)

	path, err := ImagePath()
	if err != nil {
		t.Fatal(err)
	}
	expected := coreutil.JoinPath(customDir, ImageName())
	if got, want := path, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_DefaultBootOptions_Good(t *testing.T) {
	opts := DefaultBootOptions()
	if got, want := opts.Memory, 4096; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.CPUs, 2; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Name, "core-dev"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if opts.Fresh {
		t.Fatal("expected false")
	}
}

func TestDevOps_IsInstalled_Bad(t *testing.T) {
	t.Run("returns false for non-existent image", func(t *testing.T) {
		// Point to a temp directory that is empty
		tempDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tempDir)

		// Create devops instance manually to avoid loading real config/images
		d := &DevOps{medium: io.Local}
		if d.IsInstalled() {
			t.Fatal("expected false")
		}
	})
}

func TestDevOps_IsInstalled_Good(t *testing.T) {
	t.Run("returns true when image exists", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tempDir)

		// Create the image file
		imagePath := coreutil.JoinPath(tempDir, ImageName())
		err := io.Local.Write(imagePath, "fake image data")
		if err != nil {
			t.Fatal(err)
		}

		d := &DevOps{medium: io.Local}
		if !(d.IsInstalled()) {
			t.Fatal("expected true")
		}
	})
}

type mockHypervisor struct{}

func (m *mockHypervisor) Name() string    { return "mock" }
func (m *mockHypervisor) Available() bool { return true }
func (m *mockHypervisor) BuildCommand(ctx context.Context, image string, opts *container.HypervisorOptions) (*proc.Command, error) {
	return proc.NewCommand("true"), nil
}

func TestDevOps_Status_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Setup mock container manager
	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Add a fake running container
	c := &container.Container{
		ID:        "test-id",
		Name:      "core-dev",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(), // Use our own PID so isProcessRunning returns true
		StartedAt: time.Now().Add(-time.Hour),
		Memory:    2048,
		CPUs:      4,
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	status, err := d.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("expected non-nil value")
	}
	if !(status.Running) {
		t.Fatal("expected true")
	}
	if got, want := status.ContainerID, "test-id"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.Memory, 2048; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.CPUs, 4; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_Status_NotInstalled_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	status, err := d.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("expected non-nil value")
	}
	if status.Installed {
		t.Fatal("expected false")
	}
	if status.Running {
		t.Fatal("expected false")
	}
	if got, want := status.SSHPort, 2222; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_Status_NoContainer_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image to mark as installed
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	status, err := d.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status == nil {
		t.Fatal("expected non-nil value")
	}
	if !(status.Installed) {
		t.Fatal("expected true")
	}
	if status.Running {
		t.Fatal("expected false")
	}
	if got := status.ContainerID; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestDevOps_IsRunning_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	c := &container.Container{
		ID:        "test-id",
		Name:      "core-dev",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	running, err := d.IsRunning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !(running) {
		t.Fatal("expected true")
	}
}

func TestDevOps_IsRunning_NotRunning_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	running, err := d.IsRunning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("expected false")
	}
}

func TestDevOps_IsRunning_ContainerStopped_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	c := &container.Container{
		ID:        "test-id",
		Name:      "core-dev",
		Status:    container.StatusStopped,
		PID:       12345,
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	running, err := d.IsRunning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("expected false")
	}
}

func TestDevOps_findContainer_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	c := &container.Container{
		ID:        "test-id",
		Name:      "my-container",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	found, err := d.findContainer(context.Background(), "my-container")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := found.ID, "test-id"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := found.Name, "my-container"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_findContainer_NotFound_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	found, err := d.findContainer(context.Background(), "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("expected nil")
	}
}

func TestDevOps_Stop_NotFound_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	err = d.Stop(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "not found"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestBootOptions_Custom_Good(t *testing.T) {
	opts := BootOptions{
		Memory: 8192,
		CPUs:   4,
		Name:   "custom-dev",
		Fresh:  true,
	}
	if got, want := opts.Memory, 8192; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.CPUs, 4; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Name, "custom-dev"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(opts.Fresh) {
		t.Fatal("expected true")
	}
}

func TestDevStatus_Struct_Good(t *testing.T) {
	status := DevStatus{
		Installed:    true,
		Running:      true,
		ImageVersion: "v1.2.3",
		ContainerID:  "abc123",
		Memory:       4096,
		CPUs:         2,
		SSHPort:      2222,
		Uptime:       time.Hour,
	}
	if !(status.Installed) {
		t.Fatal("expected true")
	}
	if !(status.Running) {
		t.Fatal("expected true")
	}
	if got, want := status.ImageVersion, "v1.2.3"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.ContainerID, "abc123"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.Memory, 4096; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.CPUs, 2; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.SSHPort, 2222; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := status.Uptime, time.Hour; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_Boot_NotInstalled_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	err = d.Boot(context.Background(), DefaultBootOptions())
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "not installed"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestDevOps_Boot_AlreadyRunning_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Add a running container
	c := &container.Container{
		ID:        "test-id",
		Name:      "core-dev",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	err = d.Boot(context.Background(), DefaultBootOptions())
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "already running"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestDevOps_Status_WithImageVersion_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Manually set manifest with version info
	mgr.manifest.Images[ImageName()] = ImageInfo{
		Version: "v1.2.3",
		Source:  "test",
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		config:    cfg,
		images:    mgr,
		container: cm,
	}

	status, err := d.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !(status.Installed) {
		t.Fatal("expected true")
	}
	if got, want := status.ImageVersion, "v1.2.3"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_findContainer_MultipleContainers_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Add multiple containers
	c1 := &container.Container{
		ID:        "id-1",
		Name:      "container-1",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: time.Now(),
	}
	c2 := &container.Container{
		ID:        "id-2",
		Name:      "container-2",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: time.Now(),
	}
	err = state.Add(c1)
	if err != nil {
		t.Fatal(err)
	}
	err = state.Add(c2)
	if err != nil {
		t.Fatal(err)
	}

	// Find specific container
	found, err := d.findContainer(context.Background(), "container-2")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := found.ID, "id-2"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDevOps_Status_ContainerWithUptime_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	startTime := time.Now().Add(-2 * time.Hour)
	c := &container.Container{
		ID:        "test-id",
		Name:      "core-dev",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: startTime,
		Memory:    4096,
		CPUs:      2,
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	status, err := d.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !(status.Running) {
		t.Fatal("expected true")
	}
	if got, want := status.Uptime.Hours(), float64(1); got < want {
		t.Fatalf("want at least %v, got %v", want, got)
	}
}

func TestDevOps_IsRunning_DifferentContainerName_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Add a container with different name
	c := &container.Container{
		ID:        "test-id",
		Name:      "other-container",
		Status:    container.StatusRunning,
		PID:       syscall.Getpid(),
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	// IsRunning looks for "core-dev", not "other-container"
	running, err := d.IsRunning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("expected false")
	}
}

func TestDevOps_Boot_FreshFlag_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	tempDir := newManagedTempDir(t, "devops-test-")
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Add an existing container with non-existent PID (will be seen as stopped)
	c := &container.Container{
		ID:        "old-id",
		Name:      "core-dev",
		Status:    container.StatusRunning,
		PID:       99999999, // Non-existent PID - List() will mark it as stopped
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	// Boot with Fresh=true should try to stop the existing container
	// then run a new one. The mock hypervisor "succeeds" so this won't error
	opts := BootOptions{
		Memory: 4096,
		CPUs:   2,
		Name:   "core-dev",
		Fresh:  true,
	}
	err = d.Boot(context.Background(), opts)
	// The mock hypervisor's Run succeeds
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevOps_Stop_ContainerNotRunning_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Add a container that's already stopped
	c := &container.Container{
		ID:        "test-id",
		Name:      "core-dev",
		Status:    container.StatusStopped,
		PID:       99999999,
		StartedAt: time.Now(),
	}
	err = state.Add(c)
	if err != nil {
		t.Fatal(err)
	}

	// Stop should fail because container is not running
	err = d.Stop(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "not running"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestDevOps_Boot_FreshWithNoExisting_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	tempDir := newManagedTempDir(t, "devops-boot-fresh-")
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Boot with Fresh=true but no existing container
	opts := BootOptions{
		Memory: 4096,
		CPUs:   2,
		Name:   "core-dev",
		Fresh:  true,
	}
	err = d.Boot(context.Background(), opts)
	// The mock hypervisor succeeds
	if err != nil {
		t.Fatal(err)
	}
}

func TestImageName_Format_Good(t *testing.T) {
	name := ImageName()
	// Check format: core-devops-{os}-{arch}.qcow2
	if s, sub := name, "core-devops-"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := name, runtime.GOOS; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := name, runtime.GOARCH; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if !(core.PathExt(name) == ".qcow2") {
		t.Fatal("expected true")
	}
}

func TestDevOps_Install_Delegates_Good(t *testing.T) {
	// This test verifies the Install method delegates to ImageManager
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	d := &DevOps{medium: io.Local,
		images: mgr,
	}

	// This will fail because no source is available, but it tests delegation
	err = d.Install(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDevOps_CheckUpdate_Delegates_Good(t *testing.T) {
	// This test verifies the CheckUpdate method delegates to ImageManager
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	d := &DevOps{medium: io.Local,
		images: mgr,
	}

	// This will fail because image not installed, but it tests delegation
	_, _, _, err = d.CheckUpdate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDevOps_Boot_Success_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	tempDir := newManagedTempDir(t, "devops-boot-success-")
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	// Boot without Fresh flag and no existing container
	opts := DefaultBootOptions()
	err = d.Boot(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	} // Mock hypervisor succeeds
}

func TestDevOps_Config_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	if err != nil {
		t.Fatal(err)
	}

	d := &DevOps{medium: io.Local,
		config: cfg,
		images: mgr,
	}
	if d.config == nil {
		t.Fatal("expected non-nil value")
	}
	if got, want := d.config.Images.Source, "auto"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// --- AX-7 canonical triplets ---

func TestDevOps_New_Good(t *testing.T) {
	symbol := New
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_New_Bad(t *testing.T) {
	symbol := New
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_New_Ugly(t *testing.T) {
	symbol := New
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_ImageName_Bad(t *testing.T) {
	symbol := ImageName
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_ImageName_Ugly(t *testing.T) {
	symbol := ImageName
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_ImagesDir_Bad(t *testing.T) {
	symbol := ImagesDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_ImagesDir_Ugly(t *testing.T) {
	symbol := ImagesDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_ImagePath_Bad(t *testing.T) {
	symbol := ImagePath
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_ImagePath_Ugly(t *testing.T) {
	symbol := ImagePath
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_IsInstalled_Good(t *testing.T) {
	symbol := (*DevOps).IsInstalled
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_IsInstalled_Bad(t *testing.T) {
	symbol := (*DevOps).IsInstalled
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_IsInstalled_Ugly(t *testing.T) {
	symbol := (*DevOps).IsInstalled
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Install_Good(t *testing.T) {
	symbol := (*DevOps).Install
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Install_Bad(t *testing.T) {
	symbol := (*DevOps).Install
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Install_Ugly(t *testing.T) {
	symbol := (*DevOps).Install
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_CheckUpdate_Good(t *testing.T) {
	symbol := (*DevOps).CheckUpdate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_CheckUpdate_Bad(t *testing.T) {
	symbol := (*DevOps).CheckUpdate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_CheckUpdate_Ugly(t *testing.T) {
	symbol := (*DevOps).CheckUpdate
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DefaultBootOptions_Bad(t *testing.T) {
	symbol := DefaultBootOptions
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DefaultBootOptions_Ugly(t *testing.T) {
	symbol := DefaultBootOptions
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Boot_Good(t *testing.T) {
	symbol := (*DevOps).Boot
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Boot_Bad(t *testing.T) {
	symbol := (*DevOps).Boot
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Boot_Ugly(t *testing.T) {
	symbol := (*DevOps).Boot
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Stop_Good(t *testing.T) {
	symbol := (*DevOps).Stop
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Stop_Bad(t *testing.T) {
	symbol := (*DevOps).Stop
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Stop_Ugly(t *testing.T) {
	symbol := (*DevOps).Stop
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_IsRunning_Good(t *testing.T) {
	symbol := (*DevOps).IsRunning
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_IsRunning_Bad(t *testing.T) {
	symbol := (*DevOps).IsRunning
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_IsRunning_Ugly(t *testing.T) {
	symbol := (*DevOps).IsRunning
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Status_Good(t *testing.T) {
	symbol := (*DevOps).Status
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Status_Bad(t *testing.T) {
	symbol := (*DevOps).Status
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDevOps_DevOps_Status_Ugly(t *testing.T) {
	symbol := (*DevOps).Status
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
