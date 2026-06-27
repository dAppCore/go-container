package vzproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
)

// pipeExchange runs server against one end of a net.Pipe and returns the
// client end — the §8 no-VM transport for protocol tests.
func pipeExchange(t *testing.T, server func(conn net.Conn)) net.Conn {
	t.Helper()
	client, srv := net.Pipe()
	go func() {
		defer srv.Close()
		server(srv)
	}()
	return client
}

// echoHandler answers every verb with a canned per-verb response.
type echoHandler struct{}

func (echoHandler) Handle(req Request) Response {
	switch req.Verb {
	case VerbExec:
		return Response{OK: true, Stdout: "ran " + req.Command, Exit: 0}
	case VerbStatus:
		return Response{OK: true, UptimeSeconds: 42}
	case VerbStop:
		return Response{OK: true}
	default:
		return Response{OK: false, Error: "unknown verb: " + req.Verb}
	}
}

type failingReadWriter struct{}

func (failingReadWriter) Read([]byte) (int, error)  { return 0, io.EOF }
func (failingReadWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestVzproto_WriteFrame_Good(t *testing.T) {
	auditTarget := "WriteFrame"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	var buf bytes.Buffer
	req := Request{Verb: VerbExec, Command: "uname", Args: []string{"-a"}}
	if err := WriteFrame(&buf, req); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	// The header carries the exact JSON payload length.
	if got, want := binary.BigEndian.Uint32(buf.Bytes()[:4]), uint32(buf.Len()-4); got != want {
		t.Fatalf("header length %d, payload length %d", got, want)
	}
	var back Request
	if err := ReadFrame(&buf, &back); err != nil {
		t.Fatalf("read frame back: %v", err)
	}
	if back.Verb != VerbExec || back.Command != "uname" || len(back.Args) != 1 || back.Args[0] != "-a" {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

func TestVzproto_WriteFrame_Bad(t *testing.T) {
	auditTarget := "WriteFrame"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// An unmarshalable payload fails before any byte hits the wire.
	var buf bytes.Buffer
	if err := WriteFrame(&buf, func() {}); err == nil {
		t.Fatal("expected marshal error")
	}
	if buf.Len() != 0 {
		t.Fatalf("expected nothing written, got %d bytes", buf.Len())
	}
	// An oversize payload is rejected by the frame bound.
	huge := Response{Stdout: string(make([]byte, MaxFrameBytes+1))}
	if err := WriteFrame(&buf, huge); err == nil {
		t.Fatal("expected oversize error")
	}
}

func TestVzproto_WriteFrame_Ugly(t *testing.T) {
	auditTarget := "WriteFrame"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Back-to-back frames on one buffer decode in order — no drift between
	// header and payload boundaries.
	var buf bytes.Buffer
	for _, verb := range []string{VerbExec, VerbStatus, VerbStop} {
		if err := WriteFrame(&buf, Request{Verb: verb}); err != nil {
			t.Fatalf("write %s: %v", verb, err)
		}
	}
	for _, want := range []string{VerbExec, VerbStatus, VerbStop} {
		req, err := ReadRequest(&buf)
		if err != nil {
			t.Fatalf("read %s: %v", want, err)
		}
		if req.Verb != want {
			t.Fatalf("want %q, got %q", want, req.Verb)
		}
	}
}

func TestVzproto_ReadFrame_Good(t *testing.T) {
	auditTarget := "ReadFrame"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	var buf bytes.Buffer
	if err := WriteResponse(&buf, Response{OK: true, Stdout: "hello", Exit: 0}); err != nil {
		t.Fatalf("write: %v", err)
	}
	resp, err := ReadResponse(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !resp.OK || resp.Stdout != "hello" {
		t.Fatalf("unexpected response %+v", resp)
	}
}

func TestVzproto_ReadFrame_Bad(t *testing.T) {
	auditTarget := "ReadFrame"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A hostile length prefix is rejected without allocating the claimed size.
	var huge [4]byte
	binary.BigEndian.PutUint32(huge[:], MaxFrameBytes+1)
	var into Request
	if err := ReadFrame(bytes.NewReader(huge[:]), &into); err == nil {
		t.Fatal("expected oversize error")
	}
	// A zero-length frame is malformed, not an empty document.
	var zero [4]byte
	if err := ReadFrame(bytes.NewReader(zero[:]), &into); err == nil {
		t.Fatal("expected zero-length error")
	}
	// A header that promises more payload than arrives is an unexpected EOF.
	var short bytes.Buffer
	binary.Write(&short, binary.BigEndian, uint32(100))
	short.WriteString("{}")
	if err := ReadFrame(&short, &into); err != io.ErrUnexpectedEOF {
		t.Fatalf("expected unexpected-EOF, got %v", err)
	}
}

func TestVzproto_ReadFrame_Ugly(t *testing.T) {
	auditTarget := "ReadFrame"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A clean disconnect before any byte is io.EOF — Serve's clean-exit signal.
	var empty bytes.Buffer
	var into Request
	if err := ReadFrame(&empty, &into); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
	// A frame whose payload is not JSON fails decode, not panic.
	var junk bytes.Buffer
	binary.Write(&junk, binary.BigEndian, uint32(3))
	junk.WriteString("@@@")
	if err := ReadFrame(&junk, &into); err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestVzproto_RoundTrip_Good(t *testing.T) {
	auditTarget := "RoundTrip"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	conn := pipeExchange(t, func(srv net.Conn) { _ = Serve(srv, echoHandler{}) })
	defer conn.Close()
	resp, err := RoundTrip(conn, Request{Verb: VerbExec, Command: "uname"})
	if err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !resp.OK || resp.Stdout != "ran uname" {
		t.Fatalf("unexpected response %+v", resp)
	}
}

func TestVzproto_RoundTrip_Bad(t *testing.T) {
	auditTarget := "RoundTrip"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// An agent-side refusal travels back as a well-formed OK=false response.
	conn := pipeExchange(t, func(srv net.Conn) { _ = Serve(srv, echoHandler{}) })
	defer conn.Close()
	resp, err := RoundTrip(conn, Request{Verb: "reboot"})
	if err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if resp.OK || resp.Error == "" {
		t.Fatalf("expected refusal, got %+v", resp)
	}
}

func TestVzproto_RoundTrip_Ugly(t *testing.T) {
	auditTarget := "RoundTrip"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A server that dies before answering surfaces a transport error, not a hang.
	conn := pipeExchange(t, func(srv net.Conn) {
		_, _ = ReadRequest(srv) // swallow the request, close without reply
	})
	defer conn.Close()
	if _, err := RoundTrip(conn, Request{Verb: VerbStatus}); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestVzproto_RoundTrip_WriteError_Bad(t *testing.T) {
	if _, err := RoundTrip(failingReadWriter{}, Request{Verb: VerbStatus}); err == nil {
		t.Fatal("expected write error")
	}
}

func TestVzproto_Serve_Good(t *testing.T) {
	auditTarget := "Serve"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Serve answers a sequence of verbs and exits nil after an acked stop.
	client, srv := net.Pipe()
	defer client.Close()
	served := make(chan error, 1)
	go func() { served <- Serve(srv, echoHandler{}) }()

	for _, verb := range []string{VerbStatus, VerbExec, VerbStop} {
		resp, err := RoundTrip(client, Request{Verb: verb, Command: "x"})
		if err != nil {
			t.Fatalf("%s: %v", verb, err)
		}
		if !resp.OK {
			t.Fatalf("%s refused: %+v", verb, resp)
		}
	}
	if err := <-served; err != nil {
		t.Fatalf("serve exited with error after stop: %v", err)
	}
}

func TestVzproto_Serve_Bad(t *testing.T) {
	auditTarget := "Serve"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A garbage frame from the peer terminates Serve with the codec error.
	client, srv := net.Pipe()
	defer client.Close()
	served := make(chan error, 1)
	go func() { served <- Serve(srv, echoHandler{}) }()

	var junk bytes.Buffer
	binary.Write(&junk, binary.BigEndian, uint32(4))
	junk.WriteString("!!!!")
	if _, err := client.Write(junk.Bytes()); err != nil {
		t.Fatalf("write junk: %v", err)
	}
	if err := <-served; err == nil {
		t.Fatal("expected serve to fail on junk frame")
	}
}

func TestVzproto_Serve_WriteError_Bad(t *testing.T) {
	var req bytes.Buffer
	if err := WriteRequest(&req, Request{Verb: VerbStatus}); err != nil {
		t.Fatalf("write request: %v", err)
	}
	rw := struct {
		io.Reader
		io.Writer
	}{
		Reader: &req,
		Writer: failingReadWriter{},
	}
	if err := Serve(rw, echoHandler{}); err == nil {
		t.Fatal("expected serve write error")
	}
}

func TestVzproto_Serve_Ugly(t *testing.T) {
	auditTarget := "Serve"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// A peer that connects and leaves without a word is a clean nil exit —
	// an unanswered stop refusal keeps the loop alive until disconnect.
	client, srv := net.Pipe()
	served := make(chan error, 1)
	go func() { served <- Serve(srv, refuseStopHandler{}) }()

	resp, err := RoundTrip(client, Request{Verb: VerbStop})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if resp.OK {
		t.Fatal("expected stop refusal")
	}
	// Refused stop must NOT terminate the loop; a follow-up verb still answers.
	resp, err = RoundTrip(client, Request{Verb: VerbStatus})
	if err != nil {
		t.Fatalf("status after refused stop: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected status ok, got %+v", resp)
	}
	client.Close()
	if err := <-served; err != nil {
		t.Fatalf("expected clean exit on disconnect, got %v", err)
	}
}

// refuseStopHandler answers status but refuses stop — exercises the
// loop-continues-after-refused-stop branch.
type refuseStopHandler struct{}

func (refuseStopHandler) Handle(req Request) Response {
	if req.Verb == VerbStatus {
		return Response{OK: true}
	}
	return Response{OK: false, Error: "refused"}
}
