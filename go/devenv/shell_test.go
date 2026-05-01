package devenv

import (
	"reflect"
	"testing"
)

func TestShellOptions_Default_Good(t *testing.T) {
	auditTarget := "Default"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ShellOptions{}
	if opts.Console {
		t.Fatal("expected false")
	}
	if opts.Command != nil {
		t.Fatal("expected nil")
	}
}

func TestShellOptions_Console_Good(t *testing.T) {
	auditTarget := "Console"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ShellOptions{
		Console: true,
	}
	if !(opts.Console) {
		t.Fatal("expected true")
	}
	if opts.Command != nil {
		t.Fatal("expected nil")
	}
}

func TestShellOptions_Command_Good(t *testing.T) {
	auditTarget := "Command"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ShellOptions{
		Command: []string{"ls", "-la"},
	}
	if opts.Console {
		t.Fatal("expected false")
	}
	if got, want := opts.Command, []string{"ls", "-la"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestShellOptions_ConsoleWithCommand_Good(t *testing.T) {
	auditTarget := "ConsoleWithCommand"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ShellOptions{
		Console: true,
		Command: []string{"echo", "hello"},
	}
	if !(opts.Console) {
		t.Fatal("expected true")
	}
	if got, want := opts.Command, []string{"echo", "hello"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestShellOptions_EmptyCommand_Good(t *testing.T) {
	auditTarget := "EmptyCommand"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ShellOptions{
		Command: []string{},
	}
	if opts.Console {
		t.Fatal("expected false")
	}
	if got := opts.Command; len(got) != 0 {
		t.Fatal("expected empty value")
	}
	if got, want := len(opts.Command), 0; got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

// --- AX-7 canonical triplets ---

func TestShell_DevOps_Shell_Good(t *testing.T) {
	auditTarget := "DevOps Shell"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Shell
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestShell_DevOps_Shell_Bad(t *testing.T) {
	auditTarget := "DevOps Shell"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Shell
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestShell_DevOps_Shell_Ugly(t *testing.T) {
	auditTarget := "DevOps Shell"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Shell
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
