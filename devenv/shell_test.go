package devenv

import (
	"reflect"
	"testing"
)

func TestShellOptions_Default_Good(t *testing.T) {
	opts := ShellOptions{}
	if opts.Console {
		t.Fatal("expected false")
	}
	if opts.Command != nil {
		t.Fatal("expected nil")
	}
}

func TestShellOptions_Console_Good(t *testing.T) {
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
