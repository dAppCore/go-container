package devenv

import (
	"reflect"
	"testing"
)

func TestClaudeOptions_Default_Good(t *testing.T) {
	opts := ClaudeOptions{}
	if opts.NoAuth {
		t.Fatal("expected false")
	}
	if opts.Auth != nil {
		t.Fatal("expected nil")
	}
	if got := opts.Model; len(got) != 0 {
		t.Fatal("expected empty value")
	}
}

func TestClaudeOptions_Custom_Good(t *testing.T) {
	opts := ClaudeOptions{
		NoAuth: true,
		Auth:   []string{"gh", "anthropic"},
		Model:  "opus",
	}
	if !(opts.NoAuth) {
		t.Fatal("expected true")
	}
	if got, want := opts.Auth, []string{"gh", "anthropic"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Model, "opus"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_NoAuth_Good(t *testing.T) {
	opts := ClaudeOptions{NoAuth: true}
	result := formatAuthList(opts)
	if got, want := result, " (none)"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_Default_Good(t *testing.T) {
	opts := ClaudeOptions{}
	result := formatAuthList(opts)
	if got, want := result, ", gh, anthropic, git"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_CustomAuth_Good(t *testing.T) {
	opts := ClaudeOptions{
		Auth: []string{"gh"},
	}
	result := formatAuthList(opts)
	if got, want := result, ", gh"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_MultipleAuth_Good(t *testing.T) {
	opts := ClaudeOptions{
		Auth: []string{"gh", "ssh", "git"},
	}
	result := formatAuthList(opts)
	if got, want := result, ", gh, ssh, git"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_EmptyAuth_Good(t *testing.T) {
	opts := ClaudeOptions{
		Auth: []string{},
	}
	result := formatAuthList(opts)
	if got, want := result, ", gh, anthropic, git"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
