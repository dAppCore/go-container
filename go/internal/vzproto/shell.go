// Package vzproto interactive-session protocol (SP4 / RFC.vz.md §5).
//
// The batch protocol in vzproto.go is one Request → one buffered Response.
// An interactive shell needs bidirectional streaming and a PTY, so a VerbShell
// request is acked with a Response and then switches the connection to
// ShellFrame streaming until an exit frame. The batch verbs (exec/status/stop)
// are wholly unchanged — a VerbShell connection is handed to the shell pump and
// the batch loop returns, so a pre-shell host/agent pair keeps working
// byte-for-byte. Like the rest of vzproto this file is stdlib-only: it is
// compiled into the tiny static cmd/vzagent guest binary.
package vzproto

import (
	"errors"
	"io"
)

// VerbShell opens an interactive PTY session. It is acked with a Response, then
// both sides exchange ShellFrames until a ShellExit frame ends the session.
const VerbShell = "shell"

// ProtocolVersion is the vzproto wire version. The host and the vzagent guest
// binary ship together (RFC.vz.md §5), so this is informational — bumped when
// the frame set changes. v2 added the interactive ShellFrame family on top of
// the v1 batch Request/Response.
const ProtocolVersion = 2

// Interactive ShellFrame kinds.
const (
	// ShellStdin carries host keystrokes to the guest PTY (host→guest).
	ShellStdin = "stdin"
	// ShellStdout carries guest terminal output to the host (guest→host).
	ShellStdout = "stdout"
	// ShellResize carries a new window size to the guest PTY (host→guest).
	ShellResize = "resize"
	// ShellExit carries the shell's exit code and ends the session (guest→host).
	ShellExit = "exit"
)

// shellChunkBytes bounds one stdin/stdout pump read — small enough that a frame
// never approaches MaxFrameBytes, large enough that bulk terminal output is not
// shredded into thousands of frames.
const shellChunkBytes = 32 * 1024

// ShellFrame is one interactive-session frame, carried by the same
// length-prefix codec (WriteFrame/ReadFrame) as the batch Request/Response after
// a VerbShell handshake. Kind selects the meaning of the remaining fields.
//
//	WriteFrame(conn, ShellFrame{Kind: ShellResize, Cols: 120, Rows: 40})
type ShellFrame struct {
	// Kind is the frame type: stdin, stdout, resize or exit.
	Kind string `json:"kind"`
	// Data is the stdin/stdout byte payload (base64-encoded in JSON); empty for
	// resize and exit frames.
	Data []byte `json:"data,omitempty"`
	// Cols is the resize window width in columns; zero outside a resize frame.
	Cols int `json:"cols,omitempty"`
	// Rows is the resize window height in rows; zero outside a resize frame.
	Rows int `json:"rows,omitempty"`
	// Exit is the shell exit code on a ShellExit frame.
	Exit int `json:"exit,omitempty"`
}

// WinSize is a terminal window size for the ShellClient handshake and resize.
type WinSize struct {
	Cols int
	Rows int
}

// ShellSession is a running shell-on-PTY the guest opened for a VerbShell
// request. ServeShell treats it as the terminal: Read yields PTY output to
// forward to the host, Write delivers host keystrokes, Resize applies a window
// change, Wait blocks for the shell's exit code, and Close tears it down.
type ShellSession interface {
	io.ReadWriteCloser
	// Resize applies a new window size to the PTY.
	Resize(cols, rows int) error
	// Wait blocks until the shell exits and returns its exit code.
	Wait() int
}

// ShellHandler is the optional guest capability for interactive sessions. A
// Handler that also implements ShellHandler serves VerbShell; one that does not
// refuses it as unsupported. The batch path is unaffected either way.
type ShellHandler interface {
	// OpenShell starts a shell on a PTY sized to the request, running
	// req.Command (or the guest default when empty). A returned error becomes
	// the handshake refusal (Response OK=false).
	OpenShell(req Request) (ShellSession, error)
}

// ShellClient runs the host side of an interactive session over rw. It sends the
// VerbShell handshake carrying the initial window size, reads the ack, then
// forwards stdin→guest and guest stdout→stdout, relays resize events, and
// returns the shell's exit code when the exit frame arrives. It does not touch
// the terminal: the caller sets raw mode and, on return, closes stdin or rw to
// release the input pump (process teardown does this for a real terminal).
//
//	code, err := ShellClient(conn, os.Stdin, os.Stdout, resizes, WinSize{80, 24})
func ShellClient(rw io.ReadWriter, stdin io.Reader, stdout io.Writer, resize <-chan WinSize, initial WinSize) (int, error) {
	if err := WriteRequest(rw, Request{Verb: VerbShell, Cols: initial.Cols, Rows: initial.Rows}); err != nil {
		return 0, err
	}
	ack, err := ReadResponse(rw)
	if err != nil {
		return 0, err
	}
	if !ack.OK {
		msg := ack.Error
		if msg == "" {
			msg = "shell rejected by guest"
		}
		return 0, errors.New("vzproto: " + msg)
	}

	done := make(chan struct{})
	defer close(done)
	go shellForwardStdin(rw, stdin, done)
	go shellForwardResize(rw, resize, done)

	for {
		var frame ShellFrame
		if err := ReadFrame(rw, &frame); err != nil {
			if err == io.EOF {
				// Guest closed without an explicit exit frame — treat as code 0.
				return 0, nil
			}
			return 0, err
		}
		switch frame.Kind {
		case ShellStdout:
			if len(frame.Data) > 0 {
				if _, err := stdout.Write(frame.Data); err != nil {
					return 0, err
				}
			}
		case ShellExit:
			return frame.Exit, nil
		}
	}
}

// shellForwardStdin pumps stdin → ShellStdin frames until stdin ends or a frame
// write fails. A blocking stdin Read can outlive done (the caller closes stdin
// or rw on return to release it); done only avoids an extra frame after the
// session has already ended.
func shellForwardStdin(w io.Writer, stdin io.Reader, done <-chan struct{}) {
	buf := make([]byte, shellChunkBytes)
	for {
		n, readErr := stdin.Read(buf)
		if n > 0 {
			select {
			case <-done:
				return
			default:
			}
			if err := WriteFrame(w, ShellFrame{Kind: ShellStdin, Data: append([]byte(nil), buf[:n]...)}); err != nil {
				return
			}
		}
		if readErr != nil {
			return
		}
	}
}

// shellForwardResize pumps window-size events → ShellResize frames until the
// session ends or the resize channel closes.
func shellForwardResize(w io.Writer, resize <-chan WinSize, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case ws, ok := <-resize:
			if !ok {
				return
			}
			if err := WriteFrame(w, ShellFrame{Kind: ShellResize, Cols: ws.Cols, Rows: ws.Rows}); err != nil {
				return
			}
		}
	}
}

// serveShellSession is the guest side of an interactive session — called by
// Serve when a VerbShell frame arrives. It opens a shell on a PTY (refusing the
// handshake on error), acks, then pumps host frames ↔ the PTY until the shell
// exits or the host disconnects, always emitting a ShellExit frame on a clean
// shell exit.
func serveShellSession(rw io.ReadWriter, req Request, h ShellHandler) error {
	session, err := h.OpenShell(req)
	if err != nil {
		// The batch loop has already returned, so the refusal is the whole
		// exchange — a clean OK=false ack, no exit frame.
		return WriteResponse(rw, Response{OK: false, Error: err.Error()})
	}
	if err := WriteResponse(rw, Response{OK: true}); err != nil {
		_ = session.Close()
		return err
	}

	exited := make(chan struct{})
	go shellForwardOutput(rw, session, exited)

	for {
		var frame ShellFrame
		if err := ReadFrame(rw, &frame); err != nil {
			// Host disconnected (or sent junk): tear the PTY down and let the
			// output pump drain before returning.
			_ = session.Close()
			<-exited
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch frame.Kind {
		case ShellStdin:
			if len(frame.Data) > 0 {
				if _, err := session.Write(frame.Data); err != nil {
					_ = session.Close()
				}
			}
		case ShellResize:
			_ = session.Resize(frame.Cols, frame.Rows)
		}
	}
}

// shellForwardOutput pumps PTY output → ShellStdout frames; when the PTY closes
// (the shell exited) it writes one ShellExit frame carrying Wait's code, then
// signals exited so serveShellSession can finish.
func shellForwardOutput(w io.Writer, session ShellSession, exited chan<- struct{}) {
	defer close(exited)
	buf := make([]byte, shellChunkBytes)
	for {
		n, readErr := session.Read(buf)
		if n > 0 {
			if err := WriteFrame(w, ShellFrame{Kind: ShellStdout, Data: append([]byte(nil), buf[:n]...)}); err != nil {
				return
			}
		}
		if readErr != nil {
			break
		}
	}
	_ = WriteFrame(w, ShellFrame{Kind: ShellExit, Exit: session.Wait()})
}
