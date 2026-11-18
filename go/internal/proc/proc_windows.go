//go:build windows

// proc_windows.go — the same process API as the unix half, over os/exec.
//
// The unix implementation is built on raw syscalls: fork, pipe(2), Wait4 and
// integer file descriptors. None of those exist on Windows, where process
// creation is CreateProcess with inheritable HANDLEs and a different model
// throughout. Reimplementing that by hand would mean writing — and shipping
// untested — a second process runtime.
//
// os/exec already is that runtime, maintained by the Go team and correct on
// Windows by construction. cmd/vzagent in this module reaches for the stdlib
// for the same reason. So Windows gets a working implementation rather than a
// stub that compiles and refuses, and callers get the behaviour they asked for
// on the platform they asked for it on.

package proc

import (
	"context"
	goio "io"
	"os"
	"os/exec"
	"syscall"

	core "github.com/dappcore/go"
)

// Process is a started child, identified the same way as on unix.
type Process struct {
	Pid int

	// handle is the os/exec view of the same child. Kill and Signal go
	// through it because a Pid alone is not a handle on Windows.
	handle *os.Process
}

// Kill ends the process. A nil or unstarted Process is not an error: callers
// clean up on paths where the child may never have started.
func (p *Process) Kill() (
	err error, // result
) {
	if p == nil || p.Pid <= 0 || p.handle == nil {
		return nil
	}
	return p.handle.Kill()
}

// Signal delivers a signal.
//
// Windows has no signals. Go maps os.Kill onto TerminateProcess and rejects
// the rest, so anything that is not a request to end the process is refused
// here rather than passed down to fail obscurely — the caller learns the
// platform cannot do what it asked.
func (p *Process) Signal(sig syscall.Signal) (
	err error, // result
) {
	if p == nil || p.Pid <= 0 || p.handle == nil {
		return nil
	}
	if sig == syscall.SIGKILL || sig == syscall.SIGTERM {
		return p.handle.Kill()
	}
	return core.E(
		"proc.Process.Signal",
		"Windows has no signals; only SIGKILL and SIGTERM map to ending the process",
		nil,
	)
}

// Command mirrors the unix Command field for field, so callers compile
// unchanged across platforms.
type Command struct {
	Path   string
	Args   []string
	Dir    string
	Env    []string
	Stdin  goio.Reader
	Stdout goio.Writer
	Stderr goio.Writer

	Process *Process

	ctx     context.Context
	cmd     *exec.Cmd
	started bool
	waited  bool
}

var (
	Stdin  goio.ReadCloser  = os.Stdin
	Stdout goio.WriteCloser = os.Stdout
	Stderr goio.WriteCloser = os.Stderr
)

func Environ() []string { return os.Environ() }

func NewCommandContext(ctx context.Context, name string, args ...string) *Command {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Command{
		Path: name,
		Args: append([]string{name}, args...),
		ctx:  ctx,
	}
}

func NewCommand(name string, args ...string) *Command {
	return NewCommandContext(context.Background(), name, args...)
}

// LookPath resolves a command against PATH, honouring PATHEXT — which is what
// makes `docker` find `docker.exe`, and why this cannot be the unix version
// with a different separator.
func LookPath(name string) (
	string,
	error,
) {
	if name == "" {
		return "", core.E("proc.LookPath", "empty command", nil)
	}
	resolved, err := exec.LookPath(name)
	if err != nil {
		return "", core.E("proc.LookPath", core.Concat("executable not found: ", name), err)
	}
	return resolved, nil
}

// build materialises the exec.Cmd once, so the pipe methods and Start agree
// about which child they are configuring.
func (c *Command) build() *exec.Cmd {
	if c.cmd != nil {
		return c.cmd
	}
	args := c.Args
	if len(args) > 0 {
		args = args[1:]
	}
	cmd := exec.CommandContext(c.ctx, c.Path, args...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	c.cmd = cmd
	return cmd
}

func (c *Command) StdoutPipe() (
	goio.ReadCloser,
	error,
) {
	if c.started {
		return nil, core.E("proc.Command.StdoutPipe", "command already started", nil)
	}
	return c.build().StdoutPipe()
}

func (c *Command) StderrPipe() (
	goio.ReadCloser,
	error,
) {
	if c.started {
		return nil, core.E("proc.Command.StderrPipe", "command already started", nil)
	}
	return c.build().StderrPipe()
}

func (c *Command) Start() (
	err error, // result
) {
	if c.started {
		return core.E("proc.Command.Start", "command already started", nil)
	}
	cmd := c.build()
	// Re-read the stream fields: a caller may have set them after the pipe
	// methods built the command, and on unix Start is where they are read.
	if cmd.Stdin == nil {
		cmd.Stdin = c.Stdin
	}
	if cmd.Stdout == nil {
		cmd.Stdout = c.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = c.Stderr
	}
	if err := cmd.Start(); err != nil {
		return core.E("proc.Command.Start", core.Concat("cannot start: ", c.Path), err)
	}
	c.started = true
	c.Process = &Process{Pid: cmd.Process.Pid, handle: cmd.Process}
	return nil
}

func (c *Command) Run() (
	err error, // result
) {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Command) Output() (
	[]byte,
	error,
) {
	if c.Stdout != nil {
		return nil, core.E("proc.Command.Output", "stdout already configured", nil)
	}
	out, err := c.build().Output()
	if err != nil {
		return out, core.E("proc.Command.Output", core.Concat("command failed: ", c.Path), err)
	}
	return out, nil
}

// Wait blocks until the child exits. Waiting twice is not an error, matching
// the unix half: callers defer a Wait and also call it on the happy path.
func (c *Command) Wait() (
	err error, // result
) {
	if !c.started {
		return core.E("proc.Command.Wait", "command not started", nil)
	}
	if c.waited {
		return nil
	}
	c.waited = true
	if err := c.cmd.Wait(); err != nil {
		return core.E("proc.Command.Wait", core.Concat("command failed: ", c.Path), err)
	}
	return nil
}
