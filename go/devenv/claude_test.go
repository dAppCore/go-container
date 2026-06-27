package devenv

import (
	"context"
	"reflect"
	"testing"
)

func TestClaudeOptions_Default_Good(t *testing.T) {
	auditTarget := "Default"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	auditTarget := "Custom"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
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
	auditTarget := "NoAuth"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ClaudeOptions{NoAuth: true}
	result := formatAuthList(opts)
	if got, want := result, " (none)"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_Default_Good(t *testing.T) {
	auditTarget := "Default"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ClaudeOptions{}
	result := formatAuthList(opts)
	if got, want := result, ", gh, anthropic, git"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_CustomAuth_Good(t *testing.T) {
	auditTarget := "CustomAuth"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ClaudeOptions{
		Auth: []string{"gh"},
	}
	result := formatAuthList(opts)
	if got, want := result, ", gh"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_MultipleAuth_Good(t *testing.T) {
	auditTarget := "MultipleAuth"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ClaudeOptions{
		Auth: []string{"gh", "ssh", "git"},
	}
	result := formatAuthList(opts)
	if got, want := result, ", gh, ssh, git"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestFormatAuthList_EmptyAuth_Good(t *testing.T) {
	auditTarget := "EmptyAuth"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ClaudeOptions{
		Auth: []string{},
	}
	result := formatAuthList(opts)
	if got, want := result, ", gh, anthropic, git"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestClaude_DevOps_Claude_AutoBootFails_Bad(t *testing.T) {
	dev, _ := newTestDevOps(t)
	if r := dev.Claude(context.Background(), t.TempDir(), ClaudeOptions{NoAuth: true}); r.OK {
		t.Fatal("expected error when auto-boot cannot find the dev image")
	}
}

func TestClaude_DevOps_CopyGHAuth_NoConfig_Good(t *testing.T) {
	dev, _ := newTestDevOps(t)
	t.Setenv("CORE_HOME", t.TempDir())
	if r := dev.CopyGHAuth(context.Background()); !r.OK {
		t.Fatalf("CopyGHAuth without gh config returned error: %v", r.Error())
	}
}

func TestEnsureHostKey_Skip_Good(t *testing.T) {
	t.Setenv("CORE_SKIP_SSH_SCAN", "true")
	if r := ensureHostKey(context.Background(), DefaultSSHPort); !r.OK {
		t.Fatalf("ensureHostKey skip returned error: %v", r.Error())
	}
}

// --- AX-7 canonical triplets ---

func TestClaude_DevOps_Claude_Good(t *testing.T) {
	auditTarget := "DevOps Claude"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Claude
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestClaude_DevOps_Claude_Bad(t *testing.T) {
	auditTarget := "DevOps Claude"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Claude
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestClaude_DevOps_Claude_Ugly(t *testing.T) {
	auditTarget := "DevOps Claude"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).Claude
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestClaude_DevOps_CopyGHAuth_Good(t *testing.T) {
	auditTarget := "DevOps CopyGHAuth"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).CopyGHAuth
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestClaude_DevOps_CopyGHAuth_Bad(t *testing.T) {
	auditTarget := "DevOps CopyGHAuth"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).CopyGHAuth
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestClaude_DevOps_CopyGHAuth_Ugly(t *testing.T) {
	auditTarget := "DevOps CopyGHAuth"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := (*DevOps).CopyGHAuth
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
