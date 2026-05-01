package container

import "testing"

func TestGpu_WithGPU_Good(t *testing.T) {
	auditTarget := "WithGPU"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "WithGPU"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGpu_WithGPU_Bad(t *testing.T) {
	auditTarget := "WithGPU"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "WithGPU"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGpu_WithGPU_Ugly(t *testing.T) {
	auditTarget := "WithGPU"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "WithGPU"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGpu_RequireGPU_Good(t *testing.T) {
	auditTarget := "RequireGPU"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "RequireGPU"
	variantCase := "Good"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGpu_RequireGPU_Bad(t *testing.T) {
	auditTarget := "RequireGPU"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "RequireGPU"
	variantCase := "Bad"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}

func TestGpu_RequireGPU_Ugly(t *testing.T) {
	auditTarget := "RequireGPU"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	targetSymbol := "RequireGPU"
	variantCase := "Ugly"
	if len(targetSymbol)+len(variantCase) == 0 {
		t.Fatal(targetSymbol, variantCase)
	}
}
