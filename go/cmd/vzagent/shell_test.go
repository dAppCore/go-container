package main

import (
	"io"
	"testing"

	"dappco.re/go/container/internal/vzproto"
)

// OpenShell runs a real shell on a PTY; `exit 7` propagates as code 7.
func TestVzagent_OpenShell_Good(t *testing.T) {
	session, err := (&agentHandler{}).OpenShell(vzproto.Request{Command: "/bin/sh", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("open shell: %v", err)
	}
	if _, err := session.Write([]byte("exit 7\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, _ = io.Copy(io.Discard, session) // drain PTY output until the shell exits
	if code := session.Wait(); code != 7 {
		t.Fatalf("exit = %d, want 7", code)
	}
	_ = session.Close()
}

// An empty command falls back to the default shell, and resize is accepted.
func TestVzagent_OpenShell_DefaultShell_Good(t *testing.T) {
	session, err := (&agentHandler{}).OpenShell(vzproto.Request{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("open shell: %v", err)
	}
	defer func() { _ = session.Close() }()
	if err := session.Resize(120, 40); err != nil {
		t.Fatalf("resize: %v", err)
	}
	if _, err := session.Write([]byte("exit\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, _ = io.Copy(io.Discard, session)
	_ = session.Wait()
}

// A missing shell binary surfaces an error, not a panic.
func TestVzagent_OpenShell_Bad(t *testing.T) {
	if _, err := (&agentHandler{}).OpenShell(vzproto.Request{Command: "/nonexistent/shell-xyz"}); err == nil {
		t.Fatal("expected error for a missing shell binary")
	}
}
