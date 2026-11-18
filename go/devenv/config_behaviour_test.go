package devenv

import (
	"testing"

	core "github.com/dappcore/go"
	"github.com/dappcore/go-io/go"
)

// TestConfigBehaviour_New_Good constructs a DevOps instance backed by a clean
// temp HOME, so the default config and image manager wire up without error.
//
//	dev := core.MustCast[*DevOps](New(io.Local))
func TestConfigBehaviour_New_Good(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := New(io.Local)
	if !r.OK {
		t.Fatalf("New(io.Local) error: %v", r.Error())
	}
	if core.MustCast[*DevOps](r) == nil {
		t.Fatal("New returned a nil DevOps with nil error")
	}
}
