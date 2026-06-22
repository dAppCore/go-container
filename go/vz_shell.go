//go:build darwin

package container

// VZProvider interactive shell (SP4 / RFC.vz.md §5): the host side of the
// vzproto VerbShell session over the vsock control channel. Unlike Exec (one
// bounded round-trip), a shell session has no read deadline — it stays open
// while the user works — so only the initial control dial is timed.

import (
	"io"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container/internal/vzproto"

	vzvsock "github.com/tmc/apple/x/vzkit/vsock"
)

// vzShellConnectTimeout bounds the vsock control dial for an interactive shell.
// Only the connect is bounded; the session itself runs without a deadline.
const vzShellConnectTimeout = 30 * time.Second

// WinSize is a terminal window size for VZProvider.Shell — the initial size and
// each resize event. It mirrors the control protocol's window size in public,
// framework-free terms so callers need not reach the internal vsock protocol.
type WinSize struct {
	// Cols is the window width in columns.
	Cols int
	// Rows is the window height in rows.
	Rows int
}

// Shell drops an interactive shell into a running VZ guest over the vsock
// control channel: it dials the guest agent, runs the VerbShell session
// (forwarding stdin/stdout and relaying resize events) and returns the shell's
// exit code. stdin/stdout are the host terminal streams and the caller owns
// terminal raw mode; resize delivers SIGWINCH window changes (nil for none);
// initial is the starting size; shell names the guest shell (empty uses the
// guest default, /bin/sh).
//
//	code := core.MustCast[int](p.Shell(id, os.Stdin, os.Stdout, resizes, container.WinSize{Cols: 80, Rows: 24}, ""))
func (p *VZProvider) Shell(id string, stdin io.Reader, stdout io.Writer, resize <-chan WinSize, initial WinSize, shell string) core.Result { // Value: int
	if !p.Available() {
		return vzUnavailable()
	}
	if id == "" {
		return core.Fail(core.E("VZProvider.Shell", "container id is required", nil))
	}
	entry := p.entry(id)
	if entry == nil {
		return core.Fail(core.E("VZProvider.Shell", "container not tracked: "+id, nil))
	}
	if status := p.status(entry); status != StatusRunning {
		return core.Fail(core.E("VZProvider.Shell", "container not running: "+id+" ("+string(status)+")", nil))
	}

	mgrRes := p.vsockManager(entry)
	if !mgrRes.OK {
		return mgrRes
	}
	connRes := vzConnectControl(core.MustCast[*vzvsock.Manager](mgrRes), vzShellConnectTimeout)
	if !connRes.OK {
		return connRes
	}
	conn := core.MustCast[vzVsockConn](connRes)
	defer func() { _ = conn.Close() }()

	code, err := vzproto.ShellClient(conn, stdin, stdout, vzShellResize(resize), vzproto.WinSize{Cols: initial.Cols, Rows: initial.Rows}, shell)
	if err != nil {
		return core.Fail(core.E("VZProvider.Shell", "interactive shell session", err))
	}
	return core.Ok(code)
}

// vzShellResize bridges the public WinSize resize channel onto the internal
// protocol's, so callers stay framework-free. A nil input yields a nil channel
// (no resize). The goroutine ends when the caller closes the input channel.
func vzShellResize(in <-chan WinSize) <-chan vzproto.WinSize {
	if in == nil {
		return nil
	}
	out := make(chan vzproto.WinSize)
	go func() {
		defer close(out)
		for ws := range in {
			out <- vzproto.WinSize{Cols: ws.Cols, Rows: ws.Rows}
		}
	}()
	return out
}
