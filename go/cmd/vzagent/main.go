// vzagent is the VZProvider guest control agent (RFC.vz.md §5): a static
// Linux binary listening on vsock port 1024 that answers exec, status and
// stop as length-prefixed JSON frames. It is baked into LinuxKit images as a
// service — see go/testdata/vz/linuxkit-vzagent.yml — and versioned with the
// host provider, so host and agent always ship together.
//
// Build (from the module root):
//
//	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o vzagent ./cmd/vzagent
//
// Stdlib note (AX-6): vzagent is a standalone GOOS=linux main that runs PID-1
// adjacent inside a minimal guest — it uses the stdlib (flag, os/exec,
// bytes) rather than dappco.re/go helpers so the binary stays small, static
// and free of host-side framework dependencies.
package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"dappco.re/go/container/internal/vzproto"
)

const (
	// maxCaptureBytes bounds each captured exec stream so a chatty guest
	// command cannot exceed the vzproto frame bound (16MiB) — two streams
	// plus JSON envelope stay comfortably inside one frame.
	maxCaptureBytes = 4 << 20
	// stopFlushDelay is how long the agent lets the acked stop response
	// drain to the host before initiating guest power-off.
	stopFlushDelay = 200 * time.Millisecond
	// procUptime is the Linux uptime source for the status verb.
	procUptime = "/proc/uptime"
)

// capWriter captures up to max bytes and counts the overflow — exec output
// is truncated honestly rather than bursting the frame bound.
type capWriter struct {
	buf       bytes.Buffer
	max       int
	truncated int64
}

// Write appends p up to the cap; overflow is counted, never buffered.
func (w *capWriter) Write(p []byte) (int, error) {
	room := w.max - w.buf.Len()
	if room > len(p) {
		room = len(p)
	}
	if room > 0 {
		w.buf.Write(p[:room])
	}
	w.truncated += int64(len(p) - room)
	return len(p), nil
}

// String returns the captured stream, with a truncation marker when overflow
// occurred.
func (w *capWriter) String() string {
	if w.truncated == 0 {
		return w.buf.String()
	}
	return w.buf.String() + "\n[vzagent: truncated " + strconv.FormatInt(w.truncated, 10) + " bytes]"
}

// agentHandler answers control verbs on the guest side. stopRequested flips
// when a stop verb is acknowledged so the serve loop's owner can power off.
type agentHandler struct {
	stopRequested bool
}

// Handle dispatches one control request (vzproto.Handler).
func (h *agentHandler) Handle(req vzproto.Request) vzproto.Response {
	switch req.Verb {
	case vzproto.VerbExec:
		return h.execCommand(req)
	case vzproto.VerbStatus:
		return vzproto.Response{OK: true, UptimeSeconds: readUptimeSeconds()}
	case vzproto.VerbStop:
		h.stopRequested = true
		return vzproto.Response{OK: true}
	default:
		return vzproto.Response{OK: false, Error: "unknown verb: " + req.Verb}
	}
}

// execCommand runs the requested command, capturing stdout/stderr separately
// per the §5 {stdout, stderr, exit} contract. A command that runs and exits
// non-zero is OK=true with Exit set; a command that cannot start is OK=false.
func (h *agentHandler) execCommand(req vzproto.Request) vzproto.Response {
	if req.Command == "" {
		return vzproto.Response{OK: false, Error: "exec: command is required"}
	}
	stdout := &capWriter{max: maxCaptureBytes}
	stderr := &capWriter{max: maxCaptureBytes}
	cmd := exec.Command(req.Command, req.Args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	resp := vzproto.Response{OK: true, Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			resp.Exit = exitErr.ExitCode()
			return resp
		}
		return vzproto.Response{OK: false, Error: "exec: " + err.Error(), Stderr: stderr.String()}
	}
	return resp
}

// readUptimeSeconds reads guest uptime from /proc/uptime. Best-effort: a
// host without procfs (or a parse failure) reports zero, never an error —
// status is a liveness probe first.
func readUptimeSeconds() int64 {
	raw, err := os.ReadFile(procUptime)
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return int64(seconds)
}

func main() {
	port := flag.Uint("port", vzproto.ControlPort, "vsock port to listen on")
	flag.Parse()
	if err := run(uint32(*port)); err != nil {
		os.Stderr.WriteString("vzagent: " + err.Error() + "\n")
		os.Exit(1)
	}
}
