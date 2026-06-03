package vm

import "testing"

func TestCmdSystem_systemStatus_Good(t *testing.T) {
	// Returns a Result without panicking regardless of host: OK when the Apple
	// runtime is up, else a Fail carrying requireApple's actionable message.
	r := systemStatus()
	if !r.OK && r.Error() == "" {
		t.Fatal("failed systemStatus must carry a message")
	}
}
