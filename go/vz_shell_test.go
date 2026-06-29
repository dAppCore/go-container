//go:build darwin

package container

import (
	"io"
	"strings"
	"testing"

	core "dappco.re/go"
)

func TestVz_Shell_Bad(t *testing.T) {
	if !IsVZAvailable() {
		t.Skip("virtualization framework not available")
	}
	p := NewVZProvider()
	if r := p.Shell("", strings.NewReader(""), io.Discard, nil, WinSize{}, ""); r.OK {
		t.Fatal("expected error for empty id")
	}
	r := p.Shell("never-ran", strings.NewReader(""), io.Discard, nil, WinSize{}, "")
	if r.OK {
		t.Fatal("expected error for untracked id")
	}
	if err, ok := r.Value.(error); !ok || !core.Contains(err.Error(), "not tracked") {
		t.Fatalf("expected not-tracked error, got %v", r.Value)
	}
}

func TestVz_VzShellResize_Good(t *testing.T) {
	// A nil resize channel bridges to nil (no resize stream).
	if vzShellResize(nil) != nil {
		t.Fatal("nil input should bridge to a nil channel")
	}
	// A window-size event bridges across to the internal protocol channel, and
	// closing the input closes the bridge.
	in := make(chan WinSize, 1)
	out := vzShellResize(in)
	in <- WinSize{Cols: 120, Rows: 40}
	if got := <-out; got.Cols != 120 || got.Rows != 40 {
		t.Fatalf("bridged = %+v, want 120x40", got)
	}
	close(in)
	if _, ok := <-out; ok {
		t.Fatal("bridge channel should close when the input closes")
	}
}
