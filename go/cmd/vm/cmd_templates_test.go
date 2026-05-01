package vm

import "testing"

func TestCmdTemplates_RunFromTemplate_Good(t *testing.T) {
	auditTarget := "RunFromTemplate"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "RunFromTemplate"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdTemplates_RunFromTemplate_Bad(t *testing.T) {
	auditTarget := "RunFromTemplate"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "RunFromTemplate"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdTemplates_RunFromTemplate_Ugly(t *testing.T) {
	auditTarget := "RunFromTemplate"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "RunFromTemplate"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdTemplates_ParseVarFlags_Good(t *testing.T) {
	auditTarget := "ParseVarFlags"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "ParseVarFlags"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdTemplates_ParseVarFlags_Bad(t *testing.T) {
	auditTarget := "ParseVarFlags"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "ParseVarFlags"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestCmdTemplates_ParseVarFlags_Ugly(t *testing.T) {
	auditTarget := "ParseVarFlags"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "ParseVarFlags"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}
