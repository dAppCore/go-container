package devenv

import (
	"context"
	"runtime"
	"syscall"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/container"
	"dappco.re/go/core/container/internal/coreutil"
	"dappco.re/go/core/container/internal/proc"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManagedTempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := coreutil.MkdirTemp(prefix)
	require.NoError(t, err)
	t.Cleanup(func() { _ = io.Local.DeleteAll(dir) })
	return dir
}

func TestDevOps_ImageName_Good(t *testing.T) {
	name := ImageName()
	assert.Contains(t, name, "core-devops-")
	assert.Contains(t, name, runtime.GOOS)
	assert.Contains(t, name, runtime.GOARCH)
	assert.True(t, (name[len(name)-6:] == ".qcow2"))
}

func TestDevOps_ImagesDir_Good(t *testing.T) {
	t.Run("default directory", func(t *testing.T) {
		t.Setenv("CORE_IMAGES_DIR", "")

		dir, err := ImagesDir()
		assert.NoError(t, err)
		assert.Contains(t, dir, ".core/images")
	})

	t.Run("environment override", func(t *testing.T) {
		customDir := "/tmp/custom-images"
		t.Setenv("CORE_IMAGES_DIR", customDir)

		dir, err := ImagesDir()
		assert.NoError(t, err)
		assert.Equal(t, customDir, dir)
	})
}

func TestDevOps_ImagePath_Good(t *testing.T) {
	customDir := "/tmp/images"
	t.Setenv("CORE_IMAGES_DIR", customDir)

	path, err := ImagePath()
	assert.NoError(t, err)
	expected := coreutil.JoinPath(customDir, ImageName())
	assert.Equal(t, expected, path)
}

func TestDevOps_DefaultBootOptions_Good(t *testing.T) {
	opts := DefaultBootOptions()
	assert.Equal(t, 4096, opts.Memory)
	assert.Equal(t, 2, opts.CPUs)
	assert.Equal(t, "core-dev", opts.Name)
	assert.False(t, opts.Fresh)
}

func TestDevOps_IsInstalled_Bad(t *testing.T) {
	t.Run("returns false for non-existent image", func(t *testing.T) {
		// Point to a temp directory that is empty
		tempDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tempDir)

		// Create devops instance manually to avoid loading real config/images
		d := &DevOps{medium: io.Local}
		assert.False(t, d.IsInstalled())
	})
}

func TestDevOps_IsInstalled_Good(t *testing.T) {
	t.Run("returns true when image exists", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("CORE_IMAGES_DIR", tempDir)

		// Create the image file
		imagePath := coreutil.JoinPath(tempDir, ImageName())
		err := io.Local.Write(imagePath, "fake image data")
		require.NoError(t, err)

		d := &DevOps{medium: io.Local}
		assert.True(t, d.IsInstalled())
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
	require.NoError(t, err)

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
	require.NoError(t, err)

	status, err := d.Status(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.Running)
	assert.Equal(t, "test-id", status.ContainerID)
	assert.Equal(t, 2048, status.Memory)
	assert.Equal(t, 4, status.CPUs)
}

func TestDevOps_Status_NotInstalled_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	status, err := d.Status(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.False(t, status.Installed)
	assert.False(t, status.Running)
	assert.Equal(t, 2222, status.SSHPort)
}

func TestDevOps_Status_NoContainer_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image to mark as installed
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	status, err := d.Status(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.Installed)
	assert.False(t, status.Running)
	assert.Empty(t, status.ContainerID)
}

func TestDevOps_IsRunning_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	running, err := d.IsRunning(context.Background())
	assert.NoError(t, err)
	assert.True(t, running)
}

func TestDevOps_IsRunning_NotRunning_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	running, err := d.IsRunning(context.Background())
	assert.NoError(t, err)
	assert.False(t, running)
}

func TestDevOps_IsRunning_ContainerStopped_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	running, err := d.IsRunning(context.Background())
	assert.NoError(t, err)
	assert.False(t, running)
}

func TestDevOps_findContainer_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	found, err := d.findContainer(context.Background(), "my-container")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "test-id", found.ID)
	assert.Equal(t, "my-container", found.Name)
}

func TestDevOps_findContainer_NotFound_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	found, err := d.findContainer(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestDevOps_Stop_NotFound_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	err = d.Stop(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBootOptions_Custom_Good(t *testing.T) {
	opts := BootOptions{
		Memory: 8192,
		CPUs:   4,
		Name:   "custom-dev",
		Fresh:  true,
	}
	assert.Equal(t, 8192, opts.Memory)
	assert.Equal(t, 4, opts.CPUs)
	assert.Equal(t, "custom-dev", opts.Name)
	assert.True(t, opts.Fresh)
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
	assert.True(t, status.Installed)
	assert.True(t, status.Running)
	assert.Equal(t, "v1.2.3", status.ImageVersion)
	assert.Equal(t, "abc123", status.ContainerID)
	assert.Equal(t, 4096, status.Memory)
	assert.Equal(t, 2, status.CPUs)
	assert.Equal(t, 2222, status.SSHPort)
	assert.Equal(t, time.Hour, status.Uptime)
}

func TestDevOps_Boot_NotInstalled_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	statePath := coreutil.JoinPath(tempDir, "containers.json")
	state := container.NewState(statePath)
	h := &mockHypervisor{}
	cm := container.NewLinuxKitManagerWithHypervisor(io.Local, state, h)

	d := &DevOps{medium: io.Local,
		images:    mgr,
		container: cm,
	}

	err = d.Boot(context.Background(), DefaultBootOptions())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestDevOps_Boot_AlreadyRunning_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	err = d.Boot(context.Background(), DefaultBootOptions())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestDevOps_Status_WithImageVersion_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	assert.NoError(t, err)
	assert.True(t, status.Installed)
	assert.Equal(t, "v1.2.3", status.ImageVersion)
}

func TestDevOps_findContainer_MultipleContainers_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)
	err = state.Add(c2)
	require.NoError(t, err)

	// Find specific container
	found, err := d.findContainer(context.Background(), "container-2")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "id-2", found.ID)
}

func TestDevOps_Status_ContainerWithUptime_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	status, err := d.Status(context.Background())
	assert.NoError(t, err)
	assert.True(t, status.Running)
	assert.GreaterOrEqual(t, status.Uptime.Hours(), float64(1))
}

func TestDevOps_IsRunning_DifferentContainerName_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	// IsRunning looks for "core-dev", not "other-container"
	running, err := d.IsRunning(context.Background())
	assert.NoError(t, err)
	assert.False(t, running)
}

func TestDevOps_Boot_FreshFlag_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	tempDir := newManagedTempDir(t, "devops-test-")
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	assert.NoError(t, err)
}

func TestDevOps_Stop_ContainerNotRunning_Bad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	// Stop should fail because container is not running
	err = d.Stop(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestDevOps_Boot_FreshWithNoExisting_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	tempDir := newManagedTempDir(t, "devops-boot-fresh-")
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	assert.NoError(t, err)
}

func TestImageName_Format_Good(t *testing.T) {
	name := ImageName()
	// Check format: core-devops-{os}-{arch}.qcow2
	assert.Contains(t, name, "core-devops-")
	assert.Contains(t, name, runtime.GOOS)
	assert.Contains(t, name, runtime.GOARCH)
	assert.True(t, core.PathExt(name) == ".qcow2")
}

func TestDevOps_Install_Delegates_Good(t *testing.T) {
	// This test verifies the Install method delegates to ImageManager
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	d := &DevOps{medium: io.Local,
		images: mgr,
	}

	// This will fail because no source is available, but it tests delegation
	err = d.Install(context.Background(), nil)
	assert.Error(t, err)
}

func TestDevOps_CheckUpdate_Delegates_Good(t *testing.T) {
	// This test verifies the CheckUpdate method delegates to ImageManager
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	d := &DevOps{medium: io.Local,
		images: mgr,
	}

	// This will fail because image not installed, but it tests delegation
	_, _, _, err = d.CheckUpdate(context.Background())
	assert.Error(t, err)
}

func TestDevOps_Boot_Success_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	tempDir := newManagedTempDir(t, "devops-boot-success-")
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	// Create fake image
	imagePath := coreutil.JoinPath(tempDir, ImageName())
	err := io.Local.Write(imagePath, "fake")
	require.NoError(t, err)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

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
	assert.NoError(t, err) // Mock hypervisor succeeds
}

func TestDevOps_Config_Good(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("CORE_IMAGES_DIR", tempDir)

	cfg := DefaultConfig()
	mgr, err := NewImageManager(io.Local, cfg)
	require.NoError(t, err)

	d := &DevOps{medium: io.Local,
		config: cfg,
		images: mgr,
	}

	assert.NotNil(t, d.config)
	assert.Equal(t, "auto", d.config.Images.Source)
}
