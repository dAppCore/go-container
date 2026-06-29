package container

import (
	"context"
	goio "io" // Note: io.Reader/io.ReadCloser for external consumers; no core equivalent yet.
	"io/fs"
	"syscall" // Note: POSIX signal primitives; no core equivalent yet.
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"

	"dappco.re/go/container/internal/proc"
)

// LinuxKitManager implements the Manager interface for LinuxKit VMs.
type LinuxKitManager struct {
	state      *State
	hypervisor Hypervisor
	medium     coreio.Medium
}

// NewLinuxKitManager creates a new LinuxKit manager with auto-detected hypervisor.
//
// Usage:
//
//	manager := core.MustCast[*LinuxKitManager](NewLinuxKitManager(io.Local))
func NewLinuxKitManager(m coreio.Medium) core.Result { // Value: *LinuxKitManager
	statePathRes := DefaultStatePath()
	if !statePathRes.OK {
		return core.Fail(core.E("NewLinuxKitManager", "failed to determine state path", statePathRes.Value.(error)))
	}

	stateRes := LoadState(core.MustCast[string](statePathRes))
	if !stateRes.OK {
		return core.Fail(core.E("NewLinuxKitManager", "failed to load state", stateRes.Value.(error)))
	}

	hvRes := DetectHypervisor()
	if !hvRes.OK {
		return hvRes
	}

	return core.Ok(&LinuxKitManager{
		state:      core.MustCast[*State](stateRes),
		hypervisor: core.MustCast[Hypervisor](hvRes),
		medium:     m,
	})
}

// NewLinuxKitManagerWithHypervisor creates a manager with a specific hypervisor.
//
// Usage:
//
//	manager := NewLinuxKitManagerWithHypervisor(io.Local, state, hypervisor)
func NewLinuxKitManagerWithHypervisor(m coreio.Medium, state *State, hypervisor Hypervisor) *LinuxKitManager {
	return &LinuxKitManager{
		state:      state,
		hypervisor: hypervisor,
		medium:     m,
	}
}

// Run starts a new LinuxKit VM from the given image.
func (m *LinuxKitManager) Run(ctx context.Context, image string, opts RunOptions) core.Result { // Value: *Container
	// Validate image exists
	if !m.medium.IsFile(image) {
		return core.Fail(core.E("LinuxKitManager.Run", "image not found: "+image, nil))
	}

	// Detect image format
	format := DetectImageFormat(image)
	if format == FormatUnknown {
		return core.Fail(core.E("LinuxKitManager.Run", "unsupported image format: "+image, nil))
	}

	// Generate container ID
	idRes := GenerateID()
	if !idRes.OK {
		return core.Fail(core.E("LinuxKitManager.Run", "failed to generate container ID", idRes.Value.(error)))
	}
	id := core.MustCast[string](idRes)

	// Apply defaults
	if opts.Memory <= 0 {
		opts.Memory = 1024
	}
	if opts.CPUs <= 0 {
		opts.CPUs = 1
	}
	if opts.SSHPort <= 0 {
		opts.SSHPort = 2222
	}

	// Use name or generate from ID
	name := opts.Name
	if name == "" {
		name = id[:8]
	}

	// Ensure logs directory exists
	if r := EnsureLogsDir(); !r.OK {
		return core.Fail(core.E("LinuxKitManager.Run", "failed to create logs directory", r.Value.(error)))
	}

	// Get log file path
	logPathRes := LogPath(id)
	if !logPathRes.OK {
		return core.Fail(core.E("LinuxKitManager.Run", "failed to determine log path", logPathRes.Value.(error)))
	}
	logPath := core.MustCast[string](logPathRes)

	// Build hypervisor options
	hvOpts := &HypervisorOptions{
		Memory:  opts.Memory,
		CPUs:    opts.CPUs,
		LogFile: logPath,
		SSHPort: opts.SSHPort,
		Ports:   opts.Ports,
		Volumes: opts.Volumes,
		Detach:  opts.Detach,
	}

	// Build the command
	cmdRes := m.hypervisor.BuildCommand(ctx, image, hvOpts)
	if !cmdRes.OK {
		return core.Fail(core.E("LinuxKitManager.Run", "failed to build hypervisor command", cmdRes.Value.(error)))
	}
	cmd := core.MustCast[*proc.Command](cmdRes)

	// Create log file
	logFile, err := coreio.Local.Create(logPath)
	if err != nil {
		return core.Fail(core.E("LinuxKitManager.Run", "failed to create log file", err))
	}

	// Create container record
	container := &Container{
		ID:        id,
		Name:      name,
		Image:     image,
		Status:    StatusRunning,
		StartedAt: time.Now(),
		Ports:     opts.Ports,
		Memory:    opts.Memory,
		CPUs:      opts.CPUs,
		SSHPort:   opts.SSHPort,
		SSHKey:    opts.SSHKey,
	}

	if opts.Detach {
		// Run in background
		cmd.Stdout = logFile
		cmd.Stderr = logFile

		// Start the process
		if err := cmd.Start(); err != nil {
			if closeErr := logFile.Close(); closeErr != nil {
				return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", closeErr))
			}
			return core.Fail(core.E("LinuxKitManager.Run", "failed to start VM", err))
		}

		container.PID = cmd.Process.Pid

		// Save state
		if r := m.state.Add(container); !r.OK {
			// Try to kill the process we just started
			if killErr := cmd.Process.Kill(); killErr != nil {
				// Process may already have exited; return the state error below.
			}
			if closeErr := logFile.Close(); closeErr != nil {
				return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", closeErr))
			}
			return core.Fail(core.E("LinuxKitManager.Run", "failed to save state", r.Value.(error)))
		}

		// Close log file handle (process has its own)
		if err := logFile.Close(); err != nil {
			return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", err))
		}

		// Start a goroutine to wait for process exit and update state
		go m.waitForExit(container.ID, cmd)

		return core.Ok(container)
	}

	// Run in foreground
	// Tee output to both log file and stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", closeErr))
		}
		return core.Fail(core.E("LinuxKitManager.Run", "failed to get stdout pipe", err))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", closeErr))
		}
		return core.Fail(core.E("LinuxKitManager.Run", "failed to get stderr pipe", err))
	}

	if err := cmd.Start(); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", closeErr))
		}
		return core.Fail(core.E("LinuxKitManager.Run", "failed to start VM", err))
	}

	container.PID = cmd.Process.Pid

	// Save state before waiting
	if r := m.state.Add(container); !r.OK {
		if killErr := cmd.Process.Kill(); killErr != nil {
			// Process may already have exited; return the state error below.
		}
		if closeErr := logFile.Close(); closeErr != nil {
			return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", closeErr))
		}
		return core.Fail(core.E("LinuxKitManager.Run", "failed to save state", r.Value.(error)))
	}

	// Copy output to both log and stdout
	go func() {
		mw := goio.MultiWriter(logFile, proc.Stdout)
		_, _ = goio.Copy(mw, stdout)
	}()
	go func() {
		mw := goio.MultiWriter(logFile, proc.Stderr)
		_, _ = goio.Copy(mw, stderr)
	}()

	// Wait for the process to complete
	if err := cmd.Wait(); err != nil {
		container.Status = StatusError
	} else {
		container.Status = StatusStopped
	}

	if err := logFile.Close(); err != nil {
		return core.Fail(core.E("LinuxKitManager.Run", "failed to close log file", err))
	}
	if r := m.state.Update(container); !r.OK {
		return core.Fail(core.E("LinuxKitManager.Run", "update container state", r.Value.(error)))
	}

	return core.Ok(container)
}

// waitForExit monitors a detached process and updates state when it exits.
func (m *LinuxKitManager) waitForExit(id string, cmd *proc.Command) {
	err := cmd.Wait()

	container, ok := m.state.Get(id)
	if ok {
		if err != nil {
			container.Status = StatusError
		} else {
			container.Status = StatusStopped
		}
		if r := m.state.Update(container); !r.OK {
			// Detached monitor has no caller; List will repair stale state later.
		}
	}
}

// Stop stops a running container by sending SIGTERM.
func (m *LinuxKitManager) Stop(ctx context.Context, id string) core.Result { // Value: nil
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("LinuxKitManager.Stop", "context cancelled", err))
	}
	container, ok := m.state.Get(id)
	if !ok {
		return core.Fail(core.E("LinuxKitManager.Stop", "container not found: "+id, nil))
	}

	if container.Status != StatusRunning {
		return core.Fail(core.E("LinuxKitManager.Stop", "container is not running: "+id, nil))
	}

	// Find the process
	process := &proc.Process{Pid: container.PID}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be gone
		container.Status = StatusStopped
		if r := m.state.Update(container); !r.OK {
			return core.Fail(core.E("LinuxKitManager.Stop", "update container state", r.Value.(error)))
		}
		return core.Ok(nil)
	}

	// Honour already-cancelled contexts before waiting
	if err := ctx.Err(); err != nil {
		if killErr := process.Kill(); killErr != nil {
			// Process may already have exited; return the context error below.
		}
		return core.Fail(core.E("LinuxKitManager.Stop", "context cancelled", err))
	}

	deadline := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for isProcessRunning(container.PID) {
		select {
		case <-deadline:
			if err := process.Kill(); err != nil {
				// Process may already have exited.
			}
		case <-ctx.Done():
			if err := process.Kill(); err != nil {
				// Process may already have exited; return the context error below.
			}
			return core.Fail(core.E("LinuxKitManager.Stop", "context cancelled", ctx.Err()))
		case <-ticker.C:
		}
	}

	container.Status = StatusStopped
	return m.state.Update(container)
}

// List returns all known containers, verifying process state.
func (m *LinuxKitManager) List(ctx context.Context) core.Result { // Value: []*Container
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("LinuxKitManager.List", "context cancelled", err))
	}
	containers := m.state.All()

	// Verify each running container's process is still alive
	for _, c := range containers {
		if c.Status == StatusRunning {
			if !isProcessRunning(c.PID) {
				c.Status = StatusStopped
				if r := m.state.Update(c); !r.OK {
					return core.Fail(core.E("LinuxKitManager.List", "update container state", r.Value.(error)))
				}
			}
		}
	}

	return core.Ok(containers)
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	return (&proc.Process{Pid: pid}).Signal(syscall.Signal(0)) == nil
}

// Logs returns a reader for the container's log output.
func (m *LinuxKitManager) Logs(ctx context.Context, id string, follow bool) core.Result { // Value: ReadCloser
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("LinuxKitManager.Logs", "context cancelled", err))
	}
	_, ok := m.state.Get(id)
	if !ok {
		return core.Fail(core.E("LinuxKitManager.Logs", "container not found: "+id, nil))
	}

	logPathRes := LogPath(id)
	if !logPathRes.OK {
		return core.Fail(core.E("LinuxKitManager.Logs", "failed to determine log path", logPathRes.Value.(error)))
	}
	logPath := core.MustCast[string](logPathRes)

	if !m.medium.IsFile(logPath) {
		return core.Fail(core.E("LinuxKitManager.Logs", "no logs available for container: "+id, nil))
	}

	if !follow {
		// Simple case: just open and return the file
		file, err := m.medium.Open(logPath)
		if err != nil {
			return core.Fail(core.E("LinuxKitManager.Logs", "open log file", err))
		}
		return core.Ok(ReadCloser(file))
	}

	// Follow mode: create a reader that tails the file
	return newFollowReader(ctx, m.medium, logPath)
}

// followreader implements ReadCloser for following log files.
type followreader struct {
	file    fs.File
	ctx     context.Context
	cancel  context.CancelFunc
	medium  coreio.Medium
	path    string
	offset  int
	pending []byte
}

func newFollowReader(ctx context.Context, m coreio.Medium, path string) core.Result { // Value: *followreader
	file, err := m.Open(path)
	if err != nil {
		return core.Fail(core.E("newFollowReader", "open log file", err))
	}

	ctx, cancel := context.WithCancel(ctx)

	return core.Ok(&followreader{
		file:   file,
		ctx:    ctx,
		cancel: cancel,
		medium: m,
		path:   path,
	})
}

func (f *followreader) Read(p []byte) (
	int,
	error,
) {
	if len(p) == 0 {
		return 0, nil
	}

	for {
		if len(f.pending) > 0 {
			n := copy(p, f.pending)
			f.pending = f.pending[n:]
			return n, nil
		}

		select {
		case <-f.ctx.Done():
			return 0, goio.EOF
		default:
		}

		if _, err := f.file.Stat(); err != nil {
			return 0, err
		}

		content, err := f.medium.Read(f.path)
		if err != nil {
			return 0, err
		}
		if f.offset > len(content) {
			f.offset = 0
		}
		if f.offset < len(content) {
			f.pending = []byte(content[f.offset:])
			f.offset = len(content)
			continue
		}

		// No data available, wait a bit and try again
		select {
		case <-f.ctx.Done():
			return 0, goio.EOF
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (f *followreader) Close() (
	err error, // result
) {
	f.cancel()
	return f.file.Close()
}

// linuxkitSSHArgs builds the ssh argument list for running cmd inside container
// c. When tty is true it inserts -t to request a remote pseudo-terminal (for
// interactive sessions); otherwise the connection stays non-interactive. The
// SSH port falls back to the default 2222 when the container has none set.
func linuxkitSSHArgs(c *Container, cmd []string, tty bool) []string {
	sshPort := c.SSHPort
	if sshPort <= 0 {
		sshPort = 2222
	}
	args := []string{
		"-p", core.Sprintf("%d", sshPort),
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=~/.core/known_hosts",
		"-o", "LogLevel=ERROR",
	}
	if tty {
		args = append(args, "-t")
	}
	if c.SSHKey != "" {
		args = append(args, "-i", c.SSHKey)
	}
	args = append(args, "root@localhost")
	args = append(args, cmd...)
	return args
}

// runSSH executes ssh with args, wiring the child to the terminal (fds 0/1/2),
// and wraps any failure under op. Shared by Exec and ExecInteractive.
func runSSH(ctx context.Context, op string, args []string) core.Result { // Value: nil
	sshCmd := proc.NewCommandContext(ctx, "ssh", args...)
	sshCmd.Stdin = proc.Stdin
	sshCmd.Stdout = proc.Stdout
	sshCmd.Stderr = proc.Stderr
	if err := sshCmd.Run(); err != nil {
		return core.Fail(core.E(op, "ssh exec", err))
	}
	return core.Ok(nil)
}

// Exec executes a command inside the container via SSH (non-interactive).
func (m *LinuxKitManager) Exec(ctx context.Context, id string, cmd []string) core.Result { // Value: nil
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("LinuxKitManager.Exec", "context cancelled", err))
	}
	container, ok := m.state.Get(id)
	if !ok {
		return core.Fail(core.E("LinuxKitManager.Exec", "container not found: "+id, nil))
	}
	if container.Status != StatusRunning {
		return core.Fail(core.E("LinuxKitManager.Exec", "container is not running: "+id, nil))
	}
	return runSSH(ctx, "LinuxKitManager.Exec", linuxkitSSHArgs(container, cmd, false))
}

// ExecInteractive runs cmd inside the container over `ssh -t`, wiring the child
// to the terminal so the user gets a real interactive session. It blocks until
// the remote command exits.
func (m *LinuxKitManager) ExecInteractive(ctx context.Context, id string, cmd []string) core.Result { // Value: nil
	if err := ctx.Err(); err != nil {
		return core.Fail(core.E("LinuxKitManager.ExecInteractive", "context cancelled", err))
	}
	container, ok := m.state.Get(id)
	if !ok {
		return core.Fail(core.E("LinuxKitManager.ExecInteractive", "container not found: "+id, nil))
	}
	if container.Status != StatusRunning {
		return core.Fail(core.E("LinuxKitManager.ExecInteractive", "container is not running: "+id, nil))
	}
	return runSSH(ctx, "LinuxKitManager.ExecInteractive", linuxkitSSHArgs(container, cmd, true))
}

// State returns the manager's state (for testing).
func (m *LinuxKitManager) State() *State {
	return m.state
}

// Hypervisor returns the manager's hypervisor (for testing).
func (m *LinuxKitManager) Hypervisor() Hypervisor {
	return m.hypervisor
}
