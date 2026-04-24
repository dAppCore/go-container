package proc

import (
	"context"
	goio "io"
	"sync"
	"syscall"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"

	"dappco.re/go/container/internal/coreutil"
)

type fdProvider interface {
	Fd() uintptr
}

type Process struct {
	Pid int
}

func (p *Process) Kill() error {
	if p == nil || p.Pid <= 0 {
		return nil
	}
	return syscall.Kill(p.Pid, syscall.SIGKILL)
}

func (p *Process) Signal(sig syscall.Signal) error {
	if p == nil || p.Pid <= 0 {
		return nil
	}
	return syscall.Kill(p.Pid, sig)
}

type Command struct {
	Path   string
	Args   []string
	Dir    string
	Env    []string
	Stdin  goio.Reader
	Stdout goio.Writer
	Stderr goio.Writer

	Process *Process

	ctx context.Context

	started bool
	done    chan struct{}
	waitErr error
	waited  bool
	waitMu  sync.Mutex

	stdoutPipe *pipeReader
	stderrPipe *pipeReader
}

type pipeReader struct {
	fd      int
	childFD int
}

func (p *pipeReader) Read(data []byte) (int, error) {
	n, err := syscall.Read(p.fd, data)
	if err != nil {
		return n, err
	}
	if n == 0 {
		return 0, goio.EOF
	}
	return n, nil
}

func (p *pipeReader) Close() error {
	var first error
	if p.fd >= 0 {
		if err := syscall.Close(p.fd); err != nil {
			first = err
		}
		p.fd = -1
	}
	if p.childFD >= 0 {
		if err := syscall.Close(p.childFD); err != nil && first == nil {
			first = err
		}
		p.childFD = -1
	}
	return first
}

type stdioReader struct {
	fd int
}

func (s *stdioReader) Read(data []byte) (int, error) {
	n, err := syscall.Read(s.fd, data)
	if err != nil {
		return n, err
	}
	if n == 0 {
		return 0, goio.EOF
	}
	return n, nil
}

func (s *stdioReader) Close() error { return nil }

func (s *stdioReader) Fd() uintptr { return uintptr(s.fd) }

type stdioWriter struct {
	fd int
}

func (s *stdioWriter) Write(data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		n, err := syscall.Write(s.fd, data)
		total += n
		if err != nil {
			return total, err
		}
		data = data[n:]
	}
	return total, nil
}

func (s *stdioWriter) Close() error { return nil }

func (s *stdioWriter) Fd() uintptr { return uintptr(s.fd) }

var (
	Stdin  goio.ReadCloser  = &stdioReader{fd: 0}
	Stdout goio.WriteCloser = &stdioWriter{fd: 1}
	Stderr goio.WriteCloser = &stdioWriter{fd: 2}
)

var (
	nullFD   int
	nullOnce sync.Once
	nullErr  error
)

func Environ() []string {
	return syscall.Environ()
}

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

func LookPath(name string) (string, error) {
	if name == "" {
		return "", core.E("proc.LookPath", "empty command", nil)
	}
	if core.Contains(name, "/") || core.Contains(name, "\\") {
		if isExecutable(name) {
			return name, nil
		}
		return "", core.E("proc.LookPath", core.Concat("executable not found: ", name), nil)
	}

	pathEnv := core.Env("PATH")
	sep := core.Env("PS")
	if sep == "" {
		sep = ":"
	}

	for _, dir := range core.Split(pathEnv, sep) {
		if dir == "" {
			dir = "."
		}
		candidate := coreutil.JoinPath(dir, name)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}

	return "", core.E("proc.LookPath", core.Concat("executable not found: ", name), nil)
}

func (c *Command) StdoutPipe() (goio.ReadCloser, error) {
	if c.started {
		return nil, core.E("proc.Command.StdoutPipe", "command already started", nil)
	}
	if c.stdoutPipe != nil {
		return nil, core.E("proc.Command.StdoutPipe", "stdout pipe already requested", nil)
	}
	fds := make([]int, 2)
	if err := syscall.Pipe(fds); err != nil {
		return nil, err
	}
	c.stdoutPipe = &pipeReader{fd: fds[0], childFD: fds[1]}
	return c.stdoutPipe, nil
}

func (c *Command) StderrPipe() (goio.ReadCloser, error) {
	if c.started {
		return nil, core.E("proc.Command.StderrPipe", "command already started", nil)
	}
	if c.stderrPipe != nil {
		return nil, core.E("proc.Command.StderrPipe", "stderr pipe already requested", nil)
	}
	fds := make([]int, 2)
	if err := syscall.Pipe(fds); err != nil {
		return nil, err
	}
	c.stderrPipe = &pipeReader{fd: fds[0], childFD: fds[1]}
	return c.stderrPipe, nil
}

func (c *Command) Start() error {
	if c.started {
		return core.E("proc.Command.Start", "command already started", nil)
	}
	if c.ctx != nil {
		if err := c.ctx.Err(); err != nil {
			return err
		}
	}

	path, err := LookPath(c.Path)
	if err != nil {
		return err
	}

	files := []uintptr{
		c.inputFD(),
		c.outputFD(c.stdoutPipe, c.Stdout),
		c.outputFD(c.stderrPipe, c.Stderr),
	}

	env := c.Env
	if env == nil {
		env = Environ()
	}

	pid, _, err := syscall.StartProcess(path, c.Args, &syscall.ProcAttr{
		Dir:   c.Dir,
		Env:   env,
		Files: files,
	})
	if err != nil {
		return err
	}

	c.Process = &Process{Pid: pid}
	c.done = make(chan struct{})
	c.started = true
	c.closeChildPipeEnds()
	c.watchContext()

	return nil
}

func (c *Command) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Command) Output() ([]byte, error) {
	if c.Stdout != nil {
		return nil, core.E("proc.Command.Output", "stdout already configured", nil)
	}
	reader, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	if err := c.Start(); err != nil {
		return nil, err
	}

	data, readErr := goio.ReadAll(reader)
	waitErr := c.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		return data, waitErr
	}
	return data, nil
}

func (c *Command) Wait() error {
	c.waitMu.Lock()
	defer c.waitMu.Unlock()

	if !c.started {
		return core.E("proc.Command.Wait", "command not started", nil)
	}
	if c.waited {
		return c.waitErr
	}

	var status syscall.WaitStatus
	for {
		_, err := syscall.Wait4(c.Process.Pid, &status, 0, nil)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			c.waitErr = err
			break
		}
		if status.Exited() && status.ExitStatus() != 0 {
			c.waitErr = core.E("proc.Command.Wait", core.Sprintf("exit status %d", status.ExitStatus()), nil)
		}
		if status.Signaled() {
			c.waitErr = core.E("proc.Command.Wait", core.Sprintf("signal %d", status.Signal()), nil)
		}
		break
	}

	c.waited = true
	close(c.done)
	return c.waitErr
}

func (c *Command) inputFD() uintptr {
	if c.Stdin == nil {
		return uintptr(openNull())
	}
	if file, ok := c.Stdin.(fdProvider); ok {
		return file.Fd()
	}
	return uintptr(openNull())
}

func (c *Command) outputFD(pipe *pipeReader, writer goio.Writer) uintptr {
	if pipe != nil {
		return uintptr(pipe.childFD)
	}
	if writer == nil {
		return uintptr(openNull())
	}
	if file, ok := writer.(fdProvider); ok {
		return file.Fd()
	}
	return uintptr(openNull())
}

func (c *Command) closeChildPipeEnds() {
	if c.stdoutPipe != nil && c.stdoutPipe.childFD >= 0 {
		_ = syscall.Close(c.stdoutPipe.childFD)
		c.stdoutPipe.childFD = -1
	}
	if c.stderrPipe != nil && c.stderrPipe.childFD >= 0 {
		_ = syscall.Close(c.stderrPipe.childFD)
		c.stderrPipe.childFD = -1
	}
}

func (c *Command) watchContext() {
	if c.ctx == nil || c.done == nil || c.Process == nil {
		return
	}
	go func() {
		select {
		case <-c.ctx.Done():
			_ = c.Process.Kill()
		case <-c.done:
		}
	}()
}

func isExecutable(path string) bool {
	info, err := coreio.Local.Stat(path)
	if err != nil {
		return false
	}
	if !info.Mode().IsRegular() {
		return false
	}
	return info.Mode()&0111 != 0
}

func openNull() int {
	nullOnce.Do(func() {
		nullFD, nullErr = syscall.Open("/dev/null", syscall.O_RDWR, 0)
	})
	if nullErr != nil {
		return 2
	}
	return nullFD
}
