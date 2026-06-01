package devenv

import (
	"testing"

	"dappco.re/go/io"
)

// TestConfigBehaviour_ConfigMedium_RoundTrip drives the configmedium adapter end
// to end: EnsureDir then Write then Read against an in-memory medium, exercising
// the Result-wrapping shims the config service relies on.
//
//	m := configmedium{Medium: io.NewMemoryMedium()}
//	m.EnsureDir("etc"); m.Write("etc/app.yaml", "k: v"); m.Read("etc/app.yaml")
func TestConfigBehaviour_ConfigMedium_RoundTrip(t *testing.T) {
	m := configmedium{Medium: io.NewMemoryMedium()}

	if r := m.EnsureDir("etc"); !r.OK {
		t.Fatalf("EnsureDir failed: %v", r.Value)
	}
	if r := m.Write("etc/app.yaml", "key: value"); !r.OK {
		t.Fatalf("Write failed: %v", r.Value)
	}
	r := m.Read("etc/app.yaml")
	if !r.OK {
		t.Fatalf("Read failed: %v", r.Value)
	}
	if content, ok := r.Value.(string); !ok || content != "key: value" {
		t.Fatalf("Read = %v, want %q", r.Value, "key: value")
	}
}

// TestConfigBehaviour_ConfigMedium_ReadMissing_Bad reports a failing Result when
// the underlying path does not exist.
func TestConfigBehaviour_ConfigMedium_ReadMissing_Bad(t *testing.T) {
	m := configmedium{Medium: io.NewMemoryMedium()}
	if r := m.Read("does/not/exist.yaml"); r.OK {
		t.Fatal("Read of a missing path returned an OK Result")
	}
}

// TestConfigBehaviour_New_Good constructs a DevOps instance backed by a clean
// temp HOME, so the default config and image manager wire up without error.
//
//	dev, err := New(io.Local)
func TestConfigBehaviour_New_Good(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dev, err := New(io.Local)
	if err != nil {
		t.Fatalf("New(io.Local) error: %v", err)
	}
	if dev == nil {
		t.Fatal("New returned a nil DevOps with nil error")
	}
}
