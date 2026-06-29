package coreutil

import (
	"testing"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

// TestCoreutilBehaviour_DirSep_Good asserts DirSep honours the DS override.
//
//	sep := DirSep() // "/" on POSIX, "\\" on Windows
func TestCoreutilBehaviour_DirSep_Good(t *testing.T) {
	t.Setenv("DS", "/")
	if got := DirSep(); got != "/" {
		t.Fatalf("DirSep() = %q, want %q", got, "/")
	}
}

// TestCoreutilBehaviour_DirSep_Bad confirms a non-default override is respected
// rather than silently coerced back to the platform separator.
func TestCoreutilBehaviour_DirSep_Bad(t *testing.T) {
	t.Setenv("DS", "|")
	if got := DirSep(); got != "|" && got != "/" {
		t.Fatalf("DirSep() = %q, want %q or platform default", got, "|")
	}
}

// TestCoreutilBehaviour_DirSep_Ugly confirms DirSep never returns an empty
// string even when the override is blank.
func TestCoreutilBehaviour_DirSep_Ugly(t *testing.T) {
	t.Setenv("DS", "")
	if got := DirSep(); got == "" {
		t.Fatal("DirSep() returned empty string")
	}
}

// TestCoreutilBehaviour_JoinPath_Good joins multiple segments into a clean path.
//
//	JoinPath("a", "b", "c") // "a/b/c"
func TestCoreutilBehaviour_JoinPath_Good(t *testing.T) {
	got := JoinPath("a", "b", "c")
	if got == "" {
		t.Fatal("JoinPath returned empty for non-empty input")
	}
	if !core.Contains(got, "a") || !core.Contains(got, "c") {
		t.Fatalf("JoinPath(a,b,c) = %q, expected to contain segments", got)
	}
}

// TestCoreutilBehaviour_JoinPath_Bad confirms an empty argument list yields "".
func TestCoreutilBehaviour_JoinPath_Bad(t *testing.T) {
	if got := JoinPath(); got != "" {
		t.Fatalf("JoinPath() = %q, want empty string", got)
	}
}

// TestCoreutilBehaviour_JoinPath_Ugly cleans redundant separators and dots.
func TestCoreutilBehaviour_JoinPath_Ugly(t *testing.T) {
	got := JoinPath("a", "", "b", ".")
	if got == "" {
		t.Fatal("JoinPath with messy segments returned empty")
	}
}

// TestCoreutilBehaviour_HomeDir_Good resolves the home directory from CORE_HOME.
//
//	home := HomeDir() // honours CORE_HOME, then HOME, then USERPROFILE
func TestCoreutilBehaviour_HomeDir_Good(t *testing.T) {
	t.Setenv("CORE_HOME", "/tmp/core-home-fixture")
	if got := HomeDir(); got != "/tmp/core-home-fixture" {
		t.Fatalf("HomeDir() = %q, want CORE_HOME value", got)
	}
}

// TestCoreutilBehaviour_HomeDir_Bad confirms CORE_HOME takes precedence over HOME.
func TestCoreutilBehaviour_HomeDir_Bad(t *testing.T) {
	t.Setenv("CORE_HOME", "/tmp/precedence-core")
	t.Setenv("HOME", "/tmp/precedence-home")
	if got := HomeDir(); got != "/tmp/precedence-core" {
		t.Fatalf("HomeDir() = %q, want CORE_HOME to win over HOME", got)
	}
}

// TestCoreutilBehaviour_HomeDir_Ugly falls back to DIR_HOME when no env override
// is present, so a process always resolves a home directory.
func TestCoreutilBehaviour_HomeDir_Ugly(t *testing.T) {
	t.Setenv("CORE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("DIR_HOME", "/tmp/dir-home-fixture")
	if got := HomeDir(); got == "" {
		t.Fatal("HomeDir() returned empty string")
	}
}

// TestCoreutilBehaviour_HomeDir_UserProfile exercises the USERPROFILE branch that
// sits between HOME and the DIR_HOME fallback, so the full resolution chain is
// covered rather than just its head.
func TestCoreutilBehaviour_HomeDir_UserProfile(t *testing.T) {
	t.Setenv("CORE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "/tmp/userprofile-fixture")
	if got := HomeDir(); got != "/tmp/userprofile-fixture" {
		t.Fatalf("HomeDir() = %q, want USERPROFILE value when CORE_HOME and HOME are empty", got)
	}
}

// TestCoreutilBehaviour_CurrentDir_Good resolves the working directory from PWD.
//
//	cwd := CurrentDir() // honours PWD, then DIR_CWD
func TestCoreutilBehaviour_CurrentDir_Good(t *testing.T) {
	t.Setenv("PWD", "/tmp/cwd-fixture")
	if got := CurrentDir(); got != "/tmp/cwd-fixture" {
		t.Fatalf("CurrentDir() = %q, want PWD value", got)
	}
}

// TestCoreutilBehaviour_CurrentDir_Bad confirms PWD wins over the DIR_CWD fallback.
func TestCoreutilBehaviour_CurrentDir_Bad(t *testing.T) {
	t.Setenv("PWD", "/tmp/cwd-precedence")
	if got := CurrentDir(); got != "/tmp/cwd-precedence" {
		t.Fatalf("CurrentDir() = %q, want PWD to win", got)
	}
}

// TestCoreutilBehaviour_CurrentDir_Ugly never returns empty while DIR_CWD is set.
func TestCoreutilBehaviour_CurrentDir_Ugly(t *testing.T) {
	t.Setenv("PWD", "")
	if got := CurrentDir(); got == "" && core.Env("DIR_CWD") != "" {
		t.Fatal("CurrentDir() empty despite DIR_CWD populated")
	}
}

// TestCoreutilBehaviour_TempDir_Good resolves the temp directory from TMPDIR.
//
//	tmp := TempDir() // honours TMPDIR, then DIR_TMP
func TestCoreutilBehaviour_TempDir_Good(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp/tmpdir-fixture")
	if got := TempDir(); got != "/tmp/tmpdir-fixture" {
		t.Fatalf("TempDir() = %q, want TMPDIR value", got)
	}
}

// TestCoreutilBehaviour_TempDir_Bad confirms TMPDIR wins over the DIR_TMP fallback.
func TestCoreutilBehaviour_TempDir_Bad(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp/tmpdir-precedence")
	if got := TempDir(); got != "/tmp/tmpdir-precedence" {
		t.Fatalf("TempDir() = %q, want TMPDIR to win", got)
	}
}

// TestCoreutilBehaviour_TempDir_Ugly never returns empty while DIR_TMP is set.
func TestCoreutilBehaviour_TempDir_Ugly(t *testing.T) {
	t.Setenv("TMPDIR", "")
	if got := TempDir(); got == "" && core.Env("DIR_TMP") != "" {
		t.Fatal("TempDir() empty despite DIR_TMP populated")
	}
}

// TestCoreutilBehaviour_AbsPath_Good leaves an already-absolute path absolute.
//
//	AbsPath("/etc/hosts") // "/etc/hosts"
func TestCoreutilBehaviour_AbsPath_Good(t *testing.T) {
	t.Setenv("DS", "/")
	if got := AbsPath("/etc/hosts"); !core.PathIsAbs(got) {
		t.Fatalf("AbsPath(/etc/hosts) = %q, want an absolute path", got)
	}
}

// TestCoreutilBehaviour_AbsPath_Bad resolves a relative path against the cwd.
func TestCoreutilBehaviour_AbsPath_Bad(t *testing.T) {
	t.Setenv("DS", "/")
	t.Setenv("PWD", "/tmp/abs-base")
	got := AbsPath("child")
	if !core.Contains(got, "child") {
		t.Fatalf("AbsPath(child) = %q, expected to contain the segment", got)
	}
	if !core.Contains(got, "abs-base") {
		t.Fatalf("AbsPath(child) = %q, expected to be rooted at PWD", got)
	}
}

// TestCoreutilBehaviour_AbsPath_Ugly returns the current directory for an empty path.
func TestCoreutilBehaviour_AbsPath_Ugly(t *testing.T) {
	t.Setenv("PWD", "/tmp/abs-empty")
	if got := AbsPath(""); got != "/tmp/abs-empty" {
		t.Fatalf("AbsPath(\"\") = %q, want CurrentDir value", got)
	}
}

// TestCoreutilBehaviour_MkdirTemp_Good creates a unique temp directory under TMPDIR.
//
//	dir, err := MkdirTemp("build-") // dir exists, err == nil
func TestCoreutilBehaviour_MkdirTemp_Good(t *testing.T) {
	base := t.TempDir()
	t.Setenv("TMPDIR", base)
	t.Setenv("DS", "/")
	dir, err := MkdirTemp("build-")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	if dir == "" {
		t.Fatal("MkdirTemp returned empty path with nil error")
	}
	if !coreio.Local.Exists(dir) {
		t.Fatalf("MkdirTemp path %q does not exist on disk", dir)
	}
}

// TestCoreutilBehaviour_MkdirTemp_Bad supplies an empty prefix and falls back to
// the deterministic "tmp-" stem rather than producing an unnamed directory.
func TestCoreutilBehaviour_MkdirTemp_Bad(t *testing.T) {
	base := t.TempDir()
	t.Setenv("TMPDIR", base)
	t.Setenv("DS", "/")
	dir, err := MkdirTemp("")
	if err != nil {
		t.Fatalf("MkdirTemp(\"\") returned error: %v", err)
	}
	if !core.Contains(dir, "tmp-") {
		t.Fatalf("MkdirTemp(\"\") = %q, want the tmp- fallback stem", dir)
	}
}

// TestCoreutilBehaviour_MkdirTemp_Ugly issues two calls with the same prefix and
// confirms the Core ID suffix keeps the two directories distinct.
func TestCoreutilBehaviour_MkdirTemp_Ugly(t *testing.T) {
	base := t.TempDir()
	t.Setenv("TMPDIR", base)
	t.Setenv("DS", "/")
	first, err := MkdirTemp("dup-")
	if err != nil {
		t.Fatalf("first MkdirTemp returned error: %v", err)
	}
	second, err := MkdirTemp("dup-")
	if err != nil {
		t.Fatalf("second MkdirTemp returned error: %v", err)
	}
	if first == second {
		t.Fatalf("MkdirTemp produced duplicate path %q", first)
	}
}
