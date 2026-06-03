package vm

import (
	"testing"

	borgtim "forge.lthn.ai/Snider/Borg/pkg/tim"
)

func TestCmdTim_borgImport_Good(t *testing.T) {
	// Proves forge.lthn.ai/Snider/Borg/pkg/tim resolves + links in this module.
	m, err := borgtim.New()
	if err != nil {
		t.Fatalf("borgtim.New: %v", err)
	}
	if m == nil || m.RootFS == nil {
		t.Fatal("expected a TIM with a non-nil RootFS")
	}
}
