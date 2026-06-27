package proc

import (
	"bytes"
	"context"
	goio "io"
	"syscall"
	"testing"

	core "dappco.re/go"
)

// shellPath resolves a POSIX shell, skipping the test if no /bin/sh is present
// (which would only happen on a non-POSIX host the package never targets).
func shellPath(t *testing.T) string {
	t.Helper()
	path, err := LookPath("sh")
	if err != nil {
		t.Skipf("no sh on PATH: %v", err)
	}
	return path
}

// TestProcBehaviour_LookPath_Good resolves a bare command name via PATH.
//
//	path, err := LookPath("sh") // "/bin/sh", nil
func TestProcBehaviour_LookPath_Good(t *testing.T) {
	path, err := LookPath("sh")
	if err != nil {
		t.Fatalf("LookPath(sh) error: %v", err)
	}
	if path == "" {
		t.Fatal("LookPath(sh) returned empty path with nil error")
	}
	if !isExecutable(path) {
		t.Fatalf("LookPath(sh) = %q which is not executable", path)
	}
}

// TestProcBehaviour_LookPath_Bad returns an error for an empty command name.
func TestProcBehaviour_LookPath_Bad(t *testing.T) {
	if _, err := LookPath(""); err == nil {
		t.Fatal("LookPath(\"\") returned nil error")
	}
}

// TestProcBehaviour_LookPath_Ugly reports not-found for a command absent from PATH.
func TestProcBehaviour_LookPath_Ugly(t *testing.T) {
	if _, err := LookPath("definitely-not-a-real-binary-xyz"); err == nil {
		t.Fatal("LookPath of a missing command returned nil error")
	}
}

// TestProcBehaviour_LookPathAbsolute_Good accepts an absolute path directly when
// it points at an executable, skipping the PATH search.
func TestProcBehaviour_LookPath_Absolute_Good(t *testing.T) {
	abs := shellPath(t)
	got, err := LookPath(abs)
	if err != nil {
		t.Fatalf("LookPath(%q) error: %v", abs, err)
	}
	if got != abs {
		t.Fatalf("LookPath(%q) = %q, want the path unchanged", abs, got)
	}
}

// TestProcBehaviour_LookPathAbsolute_Bad rejects an absolute path that is not an
// executable file.
func TestProcBehaviour_LookPath_Absolute_Bad(t *testing.T) {
	if _, err := LookPath("/nonexistent/path/binary"); err == nil {
		t.Fatal("LookPath of a missing absolute path returned nil error")
	}
}

// TestProcBehaviour_NewCommand_Good populates Path and Args from the constructor.
//
//	cmd := NewCommand("sh", "-c", "exit 0")
func TestProcBehaviour_NewCommand_Good(t *testing.T) {
	cmd := NewCommand("sh", "-c", "exit 0")
	if cmd.Path != "sh" {
		t.Fatalf("NewCommand Path = %q, want %q", cmd.Path, "sh")
	}
	if len(cmd.Args) != 3 || cmd.Args[0] != "sh" {
		t.Fatalf("NewCommand Args = %v, want [sh -c exit 0]", cmd.Args)
	}
}

// TestProcBehaviour_NewCommandContext_Good defaults a nil context to Background.
func TestProcBehaviour_NewCommandContext_Good(t *testing.T) {
	cmd := NewCommandContext(nil, "sh")
	if cmd.ctx == nil {
		t.Fatal("NewCommandContext(nil, ...) left ctx nil")
	}
}

// TestProcBehaviour_Run_Good runs a command that exits zero.
//
//	err := NewCommand("sh", "-c", "exit 0").Run() // nil
func TestProcBehaviour_Run_Good(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run of `exit 0` returned error: %v", err)
	}
}

// TestProcBehaviour_Run_Bad surfaces a non-zero exit status as an error.
func TestProcBehaviour_Run_Bad(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "exit 7")
	if err := cmd.Run(); err == nil {
		t.Fatal("Run of `exit 7` returned nil error")
	}
}

// TestProcBehaviour_Run_Ugly fails to start when the command does not exist.
func TestProcBehaviour_Run_Ugly(t *testing.T) {
	cmd := NewCommand("definitely-not-a-real-binary-xyz")
	if err := cmd.Run(); err == nil {
		t.Fatal("Run of a missing command returned nil error")
	}
}

// TestProcBehaviour_Output_Good captures stdout from the child process.
//
//	out, err := NewCommand("sh", "-c", "printf hello").Output() // "hello", nil
func TestProcBehaviour_Output_Good(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "printf hello")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output returned error: %v", err)
	}
	if string(out) != "hello" {
		t.Fatalf("Output = %q, want %q", string(out), "hello")
	}
}

// TestProcBehaviour_Output_Bad rejects Output when Stdout is already configured.
func TestProcBehaviour_Output_Bad(t *testing.T) {
	cmd := NewCommand("sh", "-c", "true")
	cmd.Stdout = Stdout
	if _, err := cmd.Output(); err == nil {
		t.Fatal("Output with Stdout pre-set returned nil error")
	}
}

// TestProcBehaviour_Output_Ugly returns the partial output alongside the wait
// error when the child writes then exits non-zero.
func TestProcBehaviour_Output_Ugly(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "printf partial; exit 3")
	out, err := cmd.Output()
	if err == nil {
		t.Fatal("Output of `printf partial; exit 3` returned nil error")
	}
	if string(out) != "partial" {
		t.Fatalf("Output = %q, want the partial bytes %q", string(out), "partial")
	}
}

// TestProcBehaviour_StderrPipe_Good streams stderr through the requested pipe.
func TestProcBehaviour_StderrPipe_Good(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "printf oops 1>&2")
	pipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe error: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	data, _ := goio.ReadAll(pipe)
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if string(data) != "oops" {
		t.Fatalf("stderr = %q, want %q", string(data), "oops")
	}
}

// TestProcBehaviour_StderrPipe_Bad refuses duplicate and after-start requests.
func TestProcBehaviour_StderrPipe_Bad(t *testing.T) {
	cmd := NewCommand("sh", "-c", "true")
	if _, err := cmd.StderrPipe(); err != nil {
		t.Fatalf("first StderrPipe error: %v", err)
	}
	if _, err := cmd.StderrPipe(); err == nil {
		t.Fatal("second StderrPipe returned nil error")
	}

	sh := shellPath(t)
	started := NewCommand(sh, "-c", "true")
	if err := started.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer func() { _ = started.Wait() }()
	if _, err := started.StderrPipe(); err == nil {
		t.Fatal("StderrPipe after Start returned nil error")
	}
}

// TestProcBehaviour_StdoutPipe_Bad refuses a second pipe request after one is made.
func TestProcBehaviour_StdoutPipe_Bad(t *testing.T) {
	cmd := NewCommand("sh", "-c", "true")
	if _, err := cmd.StdoutPipe(); err != nil {
		t.Fatalf("first StdoutPipe error: %v", err)
	}
	if _, err := cmd.StdoutPipe(); err == nil {
		t.Fatal("second StdoutPipe returned nil error")
	}
}

// TestProcBehaviour_StdoutPipe_Ugly refuses a pipe request after the command starts.
func TestProcBehaviour_StdoutPipe_Ugly(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer func() { _ = cmd.Wait() }()
	if _, err := cmd.StdoutPipe(); err == nil {
		t.Fatal("StdoutPipe after Start returned nil error")
	}
}

// TestProcBehaviour_Start_Bad refuses to start a command twice.
func TestProcBehaviour_Start_Bad(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "true")
	if err := cmd.Start(); err != nil {
		t.Fatalf("first Start error: %v", err)
	}
	defer func() { _ = cmd.Wait() }()
	if err := cmd.Start(); err == nil {
		t.Fatal("second Start returned nil error")
	}
}

// TestProcBehaviour_Wait_Bad refuses to wait on a command that never started.
func TestProcBehaviour_Wait_Bad(t *testing.T) {
	cmd := NewCommand("sh", "-c", "true")
	if err := cmd.Wait(); err == nil {
		t.Fatal("Wait before Start returned nil error")
	}
}

// TestProcBehaviour_Wait_Ugly is idempotent: a second Wait returns the cached result.
func TestProcBehaviour_Wait_Ugly(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "exit 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	first := cmd.Wait()
	second := cmd.Wait()
	if first == nil || second == nil {
		t.Fatalf("Wait results = (%v, %v), want both non-nil for exit 5", first, second)
	}
}

// TestProcBehaviour_StartCancelledContext_Ugly refuses to start once the context
// is already cancelled.
func TestProcBehaviour_Start_CancelledContext_Ugly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := NewCommandContext(ctx, "sh", "-c", "true")
	if err := cmd.Start(); err == nil {
		t.Fatal("Start with a cancelled context returned nil error")
	}
}

// TestProcBehaviour_ProcessKill_Good signals SIGKILL to a live child.
func TestProcBehaviour_Process_Kill_Good(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill error: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Wait after Kill returned nil error, want signalled status")
	}
}

// TestProcBehaviour_ProcessKill_Bad is a no-op on a nil/zero process.
func TestProcBehaviour_Process_Kill_Bad(t *testing.T) {
	var p *Process
	if err := p.Kill(); err != nil {
		t.Fatalf("Kill on nil Process = %v, want nil", err)
	}
	zero := &Process{Pid: 0}
	if err := zero.Kill(); err != nil {
		t.Fatalf("Kill on zero-pid Process = %v, want nil", err)
	}
}

// TestProcBehaviour_ProcessSignal_Bad is a no-op on a nil/zero process.
func TestProcBehaviour_Process_Signal_Bad(t *testing.T) {
	var p *Process
	if err := p.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Signal on nil Process = %v, want nil", err)
	}
	zero := &Process{Pid: 0}
	if err := zero.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Signal on zero-pid Process = %v, want nil", err)
	}
}

// TestProcBehaviour_ContextCancel_Ugly kills a running child when its context is
// cancelled mid-flight.
func TestProcBehaviour_Wait_ContextCancel_Ugly(t *testing.T) {
	sh := shellPath(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := NewCommandContext(ctx, sh, "-c", "sleep 30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	cancel()
	if err := cmd.Wait(); err == nil {
		t.Fatal("Wait after context cancel returned nil error, want killed status")
	}
}

// TestProcBehaviour_Environ_Good returns a non-empty environment slice.
func TestProcBehaviour_Environ_Good(t *testing.T) {
	if len(Environ()) == 0 {
		t.Fatal("Environ returned an empty slice")
	}
}

// TestProcBehaviour_CustomEnv_Good passes an explicit environment to the child.
func TestProcBehaviour_Env_Custom_Good(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "printf %s \"$PROC_FIXTURE\"")
	cmd.Env = []string{"PROC_FIXTURE=fixture-value"}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}
	if string(out) != "fixture-value" {
		t.Fatalf("child $PROC_FIXTURE = %q, want %q", string(out), "fixture-value")
	}
}

// TestProcBehaviour_isExecutable_Bad reports false for a missing path.
func TestProcBehaviour_isExecutable_Bad(t *testing.T) {
	if isExecutable("/nonexistent/path/binary") {
		t.Fatal("isExecutable reported true for a missing path")
	}
}

// TestProcBehaviour_isExecutable_Ugly reports false for a non-regular file (a
// directory is never an executable command).
func TestProcBehaviour_isExecutable_Ugly(t *testing.T) {
	dir := t.TempDir()
	if isExecutable(dir) {
		t.Fatalf("isExecutable reported true for directory %q", dir)
	}
}

// TestProcBehaviour_StdioWriter_Good writes through a stdiowriter bound to a
// real file descriptor and reads the bytes back via a stdioreader.
func TestProcBehaviour_stdiowriter_Good(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/stdio.txt"
	wfd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open for write: %v", err)
	}
	w := &stdiowriter{fd: wfd}
	if w.Fd() != uintptr(wfd) {
		t.Fatalf("stdiowriter.Fd() = %d, want %d", w.Fd(), wfd)
	}
	n, err := w.Write([]byte("payload"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len("payload") {
		t.Fatalf("Write n = %d, want %d", n, len("payload"))
	}
	if err := w.Close(); err != nil {
		t.Fatalf("stdiowriter.Close error: %v", err)
	}
	_ = syscall.Close(wfd)

	rfd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open for read: %v", err)
	}
	defer func() { _ = syscall.Close(rfd) }()
	r := &stdioreader{fd: rfd}
	if r.Fd() != uintptr(rfd) {
		t.Fatalf("stdioreader.Fd() = %d, want %d", r.Fd(), rfd)
	}
	buf := make([]byte, 32)
	got, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if string(buf[:got]) != "payload" {
		t.Fatalf("Read = %q, want %q", string(buf[:got]), "payload")
	}
	if err := r.Close(); err != nil {
		t.Fatalf("stdioreader.Close error: %v", err)
	}
}

func TestProcBehaviour_stdiowriter_Bad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/closed.txt"
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open for write: %v", err)
	}
	w := &stdiowriter{fd: fd}
	if err := syscall.Close(fd); err != nil {
		t.Fatalf("close fd: %v", err)
	}
	if _, err := w.Write([]byte("payload")); err == nil {
		t.Fatal("Write to a closed fd returned nil error")
	}
}

// TestProcBehaviour_StdioReader_Ugly returns io.EOF once the descriptor is drained.
func TestProcBehaviour_stdioreader_Ugly(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/empty.txt"
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = syscall.Close(fd) }()
	r := &stdioreader{fd: fd}
	if _, err := r.Read(make([]byte, 8)); err != goio.EOF {
		t.Fatalf("Read on empty file = %v, want io.EOF", err)
	}
}

// TestProcBehaviour_ProcessSignal_Good delivers a signal to a live child, which
// terminates it and surfaces a signalled status from Wait.
func TestProcBehaviour_Process_Signal_Good(t *testing.T) {
	sh := shellPath(t)
	cmd := NewCommand(sh, "-c", "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Signal error: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Wait after SIGTERM returned nil error, want signalled status")
	}
}

// TestProcBehaviour_StdinFD_Good feeds the child stdin through an fdProvider so
// inputFD takes its descriptor rather than /dev/null.
func TestProcBehaviour_Stdin_FD_Good(t *testing.T) {
	sh := shellPath(t)
	dir := t.TempDir()
	path := dir + "/in.txt"
	wfd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open for write: %v", err)
	}
	if _, err := syscall.Write(wfd, []byte("from-stdin")); err != nil {
		t.Fatalf("seed stdin file: %v", err)
	}
	_ = syscall.Close(wfd)

	rfd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open for read: %v", err)
	}
	defer func() { _ = syscall.Close(rfd) }()

	cmd := NewCommand(sh, "-c", "cat")
	cmd.Stdin = &stdioreader{fd: rfd}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}
	if string(out) != "from-stdin" {
		t.Fatalf("child stdin echo = %q, want %q", string(out), "from-stdin")
	}
}

func TestProcBehaviour_Stdout_FD_Good(t *testing.T) {
	sh := shellPath(t)
	dir := t.TempDir()
	path := dir + "/out.txt"
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open for write: %v", err)
	}

	cmd := NewCommand(sh, "-c", "printf redirected")
	cmd.Stdout = &stdiowriter{fd: fd}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	_ = syscall.Close(fd)

	rfd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open for read: %v", err)
	}
	defer func() { _ = syscall.Close(rfd) }()
	buf := make([]byte, 32)
	n, err := syscall.Read(rfd, buf)
	if err != nil {
		t.Fatalf("read redirected stdout: %v", err)
	}
	if string(buf[:n]) != "redirected" {
		t.Fatalf("stdout file = %q, want redirected", string(buf[:n]))
	}
}

func TestProcBehaviour_Stdout_NonFD_Ugly(t *testing.T) {
	sh := shellPath(t)
	var buf bytes.Buffer
	cmd := NewCommand(sh, "-c", "printf discarded")
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("non-fd writer received %q, want output discarded through /dev/null", buf.String())
	}
}

// TestProcBehaviour_Dir_Good runs the child in a chosen working directory.
func TestProcBehaviour_Dir_Good(t *testing.T) {
	sh := shellPath(t)
	dir := t.TempDir()
	cmd := NewCommand(sh, "-c", "pwd")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}
	if !core.Contains(string(out), dir) {
		// macOS resolves /tmp via /private; accept a suffix match on the leaf.
		leaf := core.Split(dir, "/")
		last := leaf[len(leaf)-1]
		if !core.Contains(string(out), last) {
			t.Fatalf("pwd = %q, want it rooted at %q", string(out), dir)
		}
	}
}
