// Package vzproto is the host↔guest control protocol for the VZProvider
// (RFC.vz.md §5): length-prefixed JSON frames carrying exec/status/stop
// over vsock port 1024.
//
// Wire format per frame: a 4-byte big-endian payload length followed by one
// JSON document. The package is transport-agnostic — any io.ReadWriter
// carries it (a vsock connection on a real VM, net.Pipe in tests), so the
// protocol unit-tests fully without a VM (RFC.vz.md §8).
//
// Stdlib note (AX-6): this package deliberately uses encoding/json,
// encoding/binary and io instead of core helpers because it is shared with
// the cmd/vzagent static GOOS=linux guest binary — host and agent must agree
// on these bytes, and pulling dappco.re/go into the guest agent would bloat
// the tiny static binary the §5 contract ships inside LinuxKit images.
package vzproto

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

const (
	// ControlPort is the guest vsock port the agent listens on (§5).
	ControlPort = 1024
	// MaxFrameBytes bounds a single frame payload so a corrupt or hostile
	// length prefix cannot make either side allocate unbounded memory.
	MaxFrameBytes = 16 << 20
	// frameHeaderBytes is the big-endian length prefix size.
	frameHeaderBytes = 4
)

// Control verbs the guest agent answers (§5).
const (
	// VerbExec runs Request.Command with Request.Args inside the guest.
	VerbExec = "exec"
	// VerbStatus reports agent liveness and guest uptime.
	VerbStatus = "status"
	// VerbStop asks the guest to power off gracefully.
	VerbStop = "stop"
)

// Request is one host→guest control frame.
//
// Usage:
//
//	req := vzproto.Request{Verb: vzproto.VerbExec, Command: "uname", Args: []string{"-a"}}
type Request struct {
	// Verb selects the agent action: exec, status or stop.
	Verb string `json:"verb"`
	// Command is the exec argv[0]; ignored by status and stop. For VerbShell it
	// names the shell to launch (empty uses the guest default).
	Command string `json:"command,omitempty"`
	// Args is the exec argv[1:]; ignored by status and stop.
	Args []string `json:"args,omitempty"`
	// Cols is the initial PTY width for VerbShell; zero (omitted) for every
	// batch verb, so their wire bytes are unchanged.
	Cols int `json:"cols,omitempty"`
	// Rows is the initial PTY height for VerbShell; zero (omitted) for every
	// batch verb, so their wire bytes are unchanged.
	Rows int `json:"rows,omitempty"`
}

// Response is one guest→host control frame answering a Request.
//
// OK reports whether the verb itself was carried out; a command that ran and
// exited non-zero is OK=true with Exit set — only a verb the agent could not
// perform (unknown verb, unstartable command) is OK=false with Error set.
type Response struct {
	// OK reports whether the agent performed the verb.
	OK bool `json:"ok"`
	// Error names why the verb failed when OK is false.
	Error string `json:"error,omitempty"`
	// Stdout is the captured standard output of an exec.
	Stdout string `json:"stdout,omitempty"`
	// Stderr is the captured standard error of an exec.
	Stderr string `json:"stderr,omitempty"`
	// Exit is the exec command's exit code (0 when the command succeeded).
	Exit int `json:"exit"`
	// UptimeSeconds is the guest uptime reported by status.
	UptimeSeconds int64 `json:"uptime_seconds,omitempty"`
}

// WriteFrame marshals payload as JSON and writes it as one length-prefixed
// frame. The header and payload go out in a single Write so concurrent
// writers on a shared transport cannot interleave half-frames.
//
// Usage:
//
//	if err := vzproto.WriteFrame(conn, req); err != nil { return err }
func WriteFrame(w io.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if len(body) > MaxFrameBytes {
		return errors.New("vzproto: frame payload " + strconv.Itoa(len(body)) + " exceeds " + strconv.Itoa(MaxFrameBytes) + " bytes")
	}
	frame := make([]byte, frameHeaderBytes+len(body))
	binary.BigEndian.PutUint32(frame[:frameHeaderBytes], uint32(len(body)))
	copy(frame[frameHeaderBytes:], body)
	_, err = w.Write(frame)
	return err
}

// ReadFrame reads one length-prefixed frame and unmarshals it into into.
// A clean EOF before any header byte returns io.EOF; a connection that dies
// mid-frame returns io.ErrUnexpectedEOF.
//
// Usage:
//
//	var resp vzproto.Response
//	if err := vzproto.ReadFrame(conn, &resp); err != nil { return err }
func ReadFrame(r io.Reader, into any) error {
	var header [frameHeaderBytes]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return errors.New("vzproto: zero-length frame")
	}
	if size > MaxFrameBytes {
		return errors.New("vzproto: frame length " + strconv.FormatUint(uint64(size), 10) + " exceeds " + strconv.Itoa(MaxFrameBytes) + " bytes")
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return json.Unmarshal(body, into)
}

// WriteRequest writes one Request frame.
func WriteRequest(w io.Writer, req Request) error { return WriteFrame(w, req) }

// ReadRequest reads one Request frame.
func ReadRequest(r io.Reader) (Request, error) {
	var req Request
	err := ReadFrame(r, &req)
	return req, err
}

// WriteResponse writes one Response frame.
func WriteResponse(w io.Writer, resp Response) error { return WriteFrame(w, resp) }

// ReadResponse reads one Response frame.
func ReadResponse(r io.Reader) (Response, error) {
	var resp Response
	err := ReadFrame(r, &resp)
	return resp, err
}

// RoundTrip sends one request and reads its response — the host side of a
// control exchange.
//
// Usage:
//
//	resp, err := vzproto.RoundTrip(conn, vzproto.Request{Verb: vzproto.VerbStatus})
func RoundTrip(rw io.ReadWriter, req Request) (Response, error) {
	if err := WriteRequest(rw, req); err != nil {
		return Response{}, err
	}
	return ReadResponse(rw)
}

// Handler answers one decoded control request — the guest side of a control
// exchange. Implementations must be safe for sequential reuse across frames.
type Handler interface {
	// Handle answers req; it must always return a Response, never panic.
	Handle(req Request) Response
}

// Serve answers control frames from rw until the peer disconnects (clean
// io.EOF returns nil) or a successful stop is acknowledged (returns nil so
// the agent can power the guest off). Any transport or codec error returns
// non-nil.
//
// Usage:
//
//	go func() { _ = vzproto.Serve(conn, agent) }()
func Serve(rw io.ReadWriter, h Handler) error {
	for {
		req, err := ReadRequest(rw)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		// VerbShell switches the connection to interactive ShellFrame streaming
		// and returns; the batch verbs below are reached only for non-shell
		// frames, so their handling is byte-for-byte unchanged.
		if req.Verb == VerbShell {
			if shellHandler, ok := h.(ShellHandler); ok {
				return serveShellSession(rw, req, shellHandler)
			}
			if err := WriteResponse(rw, Response{OK: false, Error: "shell not supported by this agent"}); err != nil {
				return err
			}
			continue
		}
		resp := h.Handle(req)
		if err := WriteResponse(rw, resp); err != nil {
			return err
		}
		if req.Verb == VerbStop && resp.OK {
			return nil
		}
	}
}
