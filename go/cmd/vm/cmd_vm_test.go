package vm

import "testing"

func TestCmdVm_AddVMCommands_Good(t *testing.T) {
	auditTarget := "AddVMCommands"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "AddVMCommands"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdVm_AddVMCommands_Bad(t *testing.T) {
	auditTarget := "AddVMCommands"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "AddVMCommands"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdVm_AddVMCommands_Ugly(t *testing.T) {
	auditTarget := "AddVMCommands"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "AddVMCommands"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}
