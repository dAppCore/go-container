package vm

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/io"

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

func TestCmdTim_timIsSTIM_Good(t *testing.T) {
	if !timIsSTIM([]byte("STIM\x02rest")) {
		t.Fatal("STIM-prefixed data should be detected")
	}
}

func TestCmdTim_timIsSTIM_Bad(t *testing.T) {
	if timIsSTIM([]byte("not-a-stim")) {
		t.Fatal("non-STIM data should not be detected")
	}
	if timIsSTIM([]byte("ST")) {
		t.Fatal("too-short data should not be detected")
	}
}

func TestCmdTim_timKeyphrase_Bad(t *testing.T) {
	// No --key-file and (assuming) no CORE_TIM_KEY env -> Fail.
	if core.Env("CORE_TIM_KEY") != "" {
		t.Skip("CORE_TIM_KEY set in environment")
	}
	if timKeyphrase(core.NewOptions()).OK {
		t.Fatal("expected error when no key is provided")
	}
}

func TestCmdTim_timPack_Good(t *testing.T) {
	src := t.TempDir()
	if err := io.Local.Write(core.PathJoin(src, "hello.txt"), "world"); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	out := core.PathJoin(t.TempDir(), "app.tim")
	if r := timPack(src, out); !r.OK {
		t.Fatalf("timPack: %v", r.Error())
	}
	if !io.Local.IsFile(out) {
		t.Fatal("expected packed .tim to exist")
	}
	raw, err := io.Local.Read(out)
	if err != nil {
		t.Fatalf("read packed: %v", err)
	}
	// A plain .tim is a tar: the filename + content appear in cleartext.
	if !core.Contains(raw, "hello.txt") || !core.Contains(raw, "world") {
		t.Fatal("packed tim should contain the source file name and content")
	}
	if _, ferr := borgtim.FromTar([]byte(raw)); ferr != nil {
		t.Fatalf("packed tim should parse via FromTar: %v", ferr)
	}
}

func TestCmdTim_timPack_Bad(t *testing.T) {
	// Non-existent source directory -> AddPath errors -> Fail.
	if timPack(core.PathJoin(t.TempDir(), "does-not-exist"), core.PathJoin(t.TempDir(), "o.tim")).OK {
		t.Fatal("expected failure packing a missing directory")
	}
}
