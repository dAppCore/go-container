package main

import (
	"strings"
	"testing"

	"dappco.re/go/container/internal/vzproto"
)

func TestMain_Handle_Good(t *testing.T) {
	auditTarget := "agentHandler Handle"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	h := &agentHandler{}
	// exec captures stdout and reports exit 0.
	resp := h.Handle(vzproto.Request{Verb: vzproto.VerbExec, Command: "echo", Args: []string{"hello"}})
	if !resp.OK {
		t.Fatalf("exec refused: %+v", resp)
	}
	if !strings.Contains(resp.Stdout, "hello") || resp.Exit != 0 {
		t.Fatalf("unexpected exec response %+v", resp)
	}
	// status is a liveness probe — always OK.
	if resp := h.Handle(vzproto.Request{Verb: vzproto.VerbStatus}); !resp.OK {
		t.Fatalf("status refused: %+v", resp)
	}
	// stop acks and flips the power-off flag for the serve-loop owner.
	if resp := h.Handle(vzproto.Request{Verb: vzproto.VerbStop}); !resp.OK {
		t.Fatalf("stop refused: %+v", resp)
	}
	if !h.stopRequested {
		t.Fatal("expected stopRequested after acked stop")
	}
}

func TestMain_Handle_Bad(t *testing.T) {
	auditTarget := "agentHandler Handle"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	h := &agentHandler{}
	// Unknown verbs are refused with the verb named.
	resp := h.Handle(vzproto.Request{Verb: "reboot"})
	if resp.OK || !strings.Contains(resp.Error, "reboot") {
		t.Fatalf("expected named refusal, got %+v", resp)
	}
	// An empty exec command is refused before any process spawns.
	resp = h.Handle(vzproto.Request{Verb: vzproto.VerbExec})
	if resp.OK || resp.Error == "" {
		t.Fatalf("expected exec refusal, got %+v", resp)
	}
	// A command that cannot start is OK=false with the cause.
	resp = h.Handle(vzproto.Request{Verb: vzproto.VerbExec, Command: "/nonexistent/binary"})
	if resp.OK || resp.Error == "" {
		t.Fatalf("expected unstartable-command refusal, got %+v", resp)
	}
}

func TestMain_Handle_Ugly(t *testing.T) {
	auditTarget := "agentHandler Handle"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	h := &agentHandler{}
	// A command that runs and exits non-zero is OK=true with Exit set —
	// the §5 contract distinguishes "ran and failed" from "could not run".
	resp := h.Handle(vzproto.Request{Verb: vzproto.VerbExec, Command: "sh", Args: []string{"-c", "echo oops >&2; exit 3"}})
	if !resp.OK {
		t.Fatalf("expected OK for a command that ran, got %+v", resp)
	}
	if resp.Exit != 3 {
		t.Fatalf("expected exit 3, got %d", resp.Exit)
	}
	if !strings.Contains(resp.Stderr, "oops") {
		t.Fatalf("expected stderr capture, got %q", resp.Stderr)
	}
}

func TestMain_CapWriter_Good(t *testing.T) {
	auditTarget := "capWriter"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	w := &capWriter{max: 8}
	n, err := w.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("write: n=%d err=%v", n, err)
	}
	if w.String() != "hello" {
		t.Fatalf("unexpected capture %q", w.String())
	}
}

func TestMain_CapWriter_Bad(t *testing.T) {
	auditTarget := "capWriter"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Overflow is reported written (the command must not see EPIPE-style
	// failures) but capped in the capture with an honest marker.
	w := &capWriter{max: 4}
	n, err := w.Write([]byte("0123456789"))
	if err != nil || n != 10 {
		t.Fatalf("write: n=%d err=%v", n, err)
	}
	got := w.String()
	if !strings.HasPrefix(got, "0123") || !strings.Contains(got, "truncated 6 bytes") {
		t.Fatalf("unexpected truncation rendering %q", got)
	}
}

func TestMain_CapWriter_Ugly(t *testing.T) {
	auditTarget := "capWriter"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Writes after the cap keep counting without growing the buffer.
	w := &capWriter{max: 2}
	for i := 0; i < 100; i++ {
		if _, err := w.Write([]byte("xx")); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	if w.buf.Len() != 2 {
		t.Fatalf("buffer grew past cap: %d", w.buf.Len())
	}
	if w.truncated != 198 {
		t.Fatalf("expected 198 truncated, got %d", w.truncated)
	}
}

func TestMain_ReadUptimeSeconds_Good(t *testing.T) {
	auditTarget := "readUptimeSeconds"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Best-effort everywhere: a real /proc/uptime yields >0, its absence
	// (darwin dev hosts) yields exactly 0 — never a panic or error.
	got := readUptimeSeconds()
	if got < 0 {
		t.Fatalf("uptime must never be negative, got %d", got)
	}
}
