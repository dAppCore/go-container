package vm

import "testing"

func TestCmdSystem_systemStart_Bad(t *testing.T) {
	t.Setenv("GOOS", "linux")
	if r := systemStart(true); r.OK {
		t.Fatal("expected error when apple runtime is unavailable")
	}
}

func TestCmdSystem_systemStatus_Good(t *testing.T) {
	// Returns a Result without panicking regardless of host: OK when the Apple
	// runtime is up, else a Fail carrying requireApple's actionable message.
	r := systemStatus()
	if !r.OK && r.Error() == "" {
		t.Fatal("failed systemStatus must carry a message")
	}
}

func TestCmdSystem_systemStop_Bad(t *testing.T) {
	t.Setenv("GOOS", "linux")
	if r := systemStop(); r.OK {
		t.Fatal("expected error when apple runtime is unavailable")
	}
}
