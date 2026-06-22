package vzproto

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
)

// fakePTY is an in-memory ShellSession for the no-VM shell-protocol tests
// (RFC.vz.md §8): host stdin is captured in received, the test drives PTY output
// by writing to outW, Resize records window changes, and Wait reports a preset
// code once the session closes.
type fakePTY struct {
	out  *io.PipeReader // serveShellSession reads PTY output here
	outW *io.PipeWriter // the test writes PTY output here

	mu       sync.Mutex
	received []byte
	resizes  []WinSize
	code     int
}

func newFakePTY(code int) *fakePTY {
	r, w := io.Pipe()
	return &fakePTY{out: r, outW: w, code: code}
}

func (p *fakePTY) Read(b []byte) (int, error) { return p.out.Read(b) }

func (p *fakePTY) Write(b []byte) (int, error) {
	p.mu.Lock()
	p.received = append(p.received, b...)
	p.mu.Unlock()
	return len(b), nil
}

func (p *fakePTY) Resize(cols, rows int) error {
	p.mu.Lock()
	p.resizes = append(p.resizes, WinSize{Cols: cols, Rows: rows})
	p.mu.Unlock()
	return nil
}

func (p *fakePTY) Wait() int    { return p.code }
func (p *fakePTY) Close() error { return p.out.Close() }

func (p *fakePTY) capturedStdin() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return string(p.received)
}

func (p *fakePTY) windowSizes() []WinSize {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]WinSize(nil), p.resizes...)
}

// shellAgent hands back the test's fake PTY for VerbShell (or refuses with
// openErr), and satisfies the batch Handler so it can drive Serve.
type shellAgent struct {
	pty     *fakePTY
	openErr error
	opened  Request
}

func (a *shellAgent) Handle(Request) Response { return Response{OK: false, Error: "batch unused"} }

func (a *shellAgent) OpenShell(req Request) (ShellSession, error) {
	a.opened = req
	if a.openErr != nil {
		return nil, a.openErr
	}
	return a.pty, nil
}

func TestVzproto_Shell_Good(t *testing.T) {
	// End-to-end host path: ShellClient relays guest terminal output to stdout
	// and returns the shell's exit code from the exit frame.
	client, srv := net.Pipe()
	pty := newFakePTY(7)
	agent := &shellAgent{pty: pty}
	served := make(chan error, 1)
	go func() { served <- Serve(srv, agent) }()

	type outcome struct {
		code int
		err  error
	}
	result := make(chan outcome, 1)
	var stdout bytes.Buffer
	go func() {
		code, err := ShellClient(client, strings.NewReader(""), &stdout, nil, WinSize{Cols: 80, Rows: 24}, "/bin/sh")
		result <- outcome{code, err}
	}()

	if _, err := pty.outW.Write([]byte("total 0\n")); err != nil {
		t.Fatalf("write pty out: %v", err)
	}
	_ = pty.outW.Close() // shell exits → guest emits the exit frame

	got := <-result
	if got.err != nil {
		t.Fatalf("client: %v", got.err)
	}
	if got.code != 7 {
		t.Fatalf("exit code = %d, want 7", got.code)
	}
	if !strings.Contains(stdout.String(), "total 0") {
		t.Fatalf("stdout = %q, want guest output", stdout.String())
	}

	client.Close()
	if err := <-served; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if agent.opened.Command != "/bin/sh" {
		t.Fatalf("OpenShell command = %q, want /bin/sh forwarded by ShellClient", agent.opened.Command)
	}
}

func TestVzproto_ShellForwardStdin_Good(t *testing.T) {
	// The host input pump turns stdin bytes into one ShellStdin frame and stops
	// cleanly at EOF.
	var buf bytes.Buffer
	shellForwardStdin(&buf, strings.NewReader("whoami\n"), make(chan struct{}))

	var frame ShellFrame
	if err := ReadFrame(&buf, &frame); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if frame.Kind != ShellStdin || string(frame.Data) != "whoami\n" {
		t.Fatalf("frame = %+v, want stdin %q", frame, "whoami\n")
	}
}

func TestVzproto_ShellForwardResize_Good(t *testing.T) {
	// The host resize pump turns a window-size event into one ShellResize frame
	// and stops when the channel closes.
	var buf bytes.Buffer
	resize := make(chan WinSize, 1)
	resize <- WinSize{Cols: 100, Rows: 30}
	close(resize)
	shellForwardResize(&buf, resize, make(chan struct{}))

	var frame ShellFrame
	if err := ReadFrame(&buf, &frame); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if frame.Kind != ShellResize || frame.Cols != 100 || frame.Rows != 30 {
		t.Fatalf("frame = %+v, want resize 100x30", frame)
	}
}

func TestVzproto_ServeShell_InputAndResize_Good(t *testing.T) {
	// Guest path: stdin and resize frames reach the PTY, and OpenShell sees the
	// handshake's shell + initial size. Frames are written synchronously, so the
	// guest has drained them before the disconnect.
	client, srv := net.Pipe()
	pty := newFakePTY(0)
	agent := &shellAgent{pty: pty}
	served := make(chan error, 1)
	go func() { served <- Serve(srv, agent) }()

	if err := WriteRequest(client, Request{Verb: VerbShell, Command: "/bin/sh", Cols: 80, Rows: 24}); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	ack, err := ReadResponse(client)
	if err != nil || !ack.OK {
		t.Fatalf("ack: %+v err=%v", ack, err)
	}
	if err := WriteFrame(client, ShellFrame{Kind: ShellStdin, Data: []byte("whoami\n")}); err != nil {
		t.Fatalf("stdin: %v", err)
	}
	if err := WriteFrame(client, ShellFrame{Kind: ShellResize, Cols: 132, Rows: 50}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	client.Close() // host disconnects → guest drains the sent frames, then returns

	if err := <-served; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if got := pty.capturedStdin(); got != "whoami\n" {
		t.Fatalf("guest stdin = %q, want %q", got, "whoami\n")
	}
	if sizes := pty.windowSizes(); len(sizes) != 1 || sizes[0] != (WinSize{Cols: 132, Rows: 50}) {
		t.Fatalf("resizes = %+v, want one {132,50}", sizes)
	}
	if agent.opened.Cols != 80 || agent.opened.Rows != 24 || agent.opened.Command != "/bin/sh" {
		t.Fatalf("OpenShell req = %+v, want cols80 rows24 /bin/sh", agent.opened)
	}
}

func TestVzproto_Shell_Bad(t *testing.T) {
	// A guest that cannot open a PTY refuses the handshake; the client surfaces
	// the reason, not a hang.
	client, srv := net.Pipe()
	go func() { _ = Serve(srv, &shellAgent{openErr: errors.New("no pty available")}) }()
	if _, err := ShellClient(client, strings.NewReader(""), io.Discard, nil, WinSize{Cols: 80, Rows: 24}, ""); err == nil {
		t.Fatal("expected refusal error")
	} else if !strings.Contains(err.Error(), "no pty available") {
		t.Fatalf("error = %v, want guest refusal reason", err)
	}
}

func TestVzproto_Shell_Unsupported_Bad(t *testing.T) {
	// echoHandler implements only the batch Handler — VerbShell is refused and
	// the batch path is untouched.
	client, srv := net.Pipe()
	go func() { _ = Serve(srv, echoHandler{}) }()
	if _, err := ShellClient(client, strings.NewReader(""), io.Discard, nil, WinSize{Cols: 80, Rows: 24}, ""); err == nil {
		t.Fatal("expected unsupported-shell error")
	}
}

func TestVzproto_Shell_Ugly(t *testing.T) {
	// The guest acks then dies without an exit frame — the client returns
	// cleanly (code 0), never hangs.
	client, srv := net.Pipe()
	go func() {
		_, _ = ReadRequest(srv)                    // VerbShell handshake
		_ = WriteResponse(srv, Response{OK: true}) // ack
		srv.Close()                                // abrupt mid-session death
	}()
	code, err := ShellClient(client, strings.NewReader(""), io.Discard, nil, WinSize{Cols: 80, Rows: 24}, "")
	if err != nil {
		t.Fatalf("expected clean EOF handling, got %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0 on abrupt guest close", code)
	}
}

func TestVzproto_BatchByteIdentical_Good(t *testing.T) {
	// Adding the VerbShell Cols/Rows fields (omitempty) must not change the wire
	// bytes of any batch verb (R3): host and agent ship together, but the batch
	// path the SP2 dispatch fork relies on stays stable.
	for _, req := range []Request{
		{Verb: VerbExec, Command: "uname", Args: []string{"-a"}},
		{Verb: VerbStatus},
		{Verb: VerbStop},
	} {
		var buf bytes.Buffer
		if err := WriteFrame(&buf, req); err != nil {
			t.Fatalf("write %s: %v", req.Verb, err)
		}
		if bytes.Contains(buf.Bytes(), []byte("cols")) || bytes.Contains(buf.Bytes(), []byte("rows")) {
			t.Fatalf("%s frame leaked cols/rows: %s", req.Verb, buf.Bytes())
		}
	}
}
