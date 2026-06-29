// vzagent interactive-shell support (SP4): OpenShell makes agentHandler a
// vzproto.ShellHandler, so the serve loop routes a VerbShell request to a shell
// running on a pseudo-terminal. creack/pty is pure Go, so the guest binary stays
// static (CGO_ENABLED=0) and free of host framework dependencies.
package main

import (
	"os"
	"os/exec"

	"github.com/creack/pty"

	"dappco.re/go/container/internal/vzproto"
)

const (
	// defaultGuestShell is the shell vzagent launches for a VerbShell request
	// that names none. /bin/sh is the one shell a minimal LinuxKit guest is
	// guaranteed to ship.
	defaultGuestShell = "/bin/sh"
	// defaultShellCols / defaultShellRows size the PTY when the host omits a
	// window size in the handshake.
	defaultShellCols = 80
	defaultShellRows = 24
)

// ptySession is the guest side of an interactive session (vzproto.ShellSession):
// a shell attached to a pseudo-terminal. Read/Write are the PTY master (terminal
// output to the host / host keystrokes), Resize applies a window size, Wait
// blocks on the shell's exit, and Close tears both down.
type ptySession struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

func (s *ptySession) Read(b []byte) (int, error)  { return s.ptmx.Read(b) }
func (s *ptySession) Write(b []byte) (int, error) { return s.ptmx.Write(b) }

// Resize applies a new window size so guest TUIs redraw to the host terminal.
func (s *ptySession) Resize(cols, rows int) error {
	return pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Wait blocks until the shell exits and reports its code (1 for a non-exit
// failure, mirroring a shell that died on a signal).
func (s *ptySession) Wait() int {
	err := s.cmd.Wait()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// Close stops the shell and releases the PTY.
func (s *ptySession) Close() error {
	_ = s.ptmx.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}

// OpenShell starts a shell on a new PTY sized to the request, satisfying
// vzproto.ShellHandler. The command defaults to /bin/sh and the size to 80x24
// when the host omits them.
func (h *agentHandler) OpenShell(req vzproto.Request) (vzproto.ShellSession, error) {
	shell := req.Command
	if shell == "" {
		shell = defaultGuestShell
	}
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	cols, rows := req.Cols, req.Rows
	if cols == 0 {
		cols = defaultShellCols
	}
	if rows == 0 {
		rows = defaultShellRows
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	return &ptySession{ptmx: ptmx, cmd: cmd}, nil
}
