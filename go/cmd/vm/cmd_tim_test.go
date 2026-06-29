package vm

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/io"

	borgtim "forge.lthn.ai/Snider/Borg/pkg/tim"
)

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
	emptyKey := core.PathJoin(t.TempDir(), "empty.key")
	if err := io.Local.Write(emptyKey, " \n\t "); err != nil {
		t.Fatalf("seed empty key: %v", err)
	}
	if timKeyphrase(core.NewOptions(core.Option{Key: "key-file", Value: emptyKey})).OK {
		t.Fatal("expected error for empty key file")
	}
	if timKeyphrase(core.NewOptions(core.Option{Key: "key-file", Value: core.PathJoin(t.TempDir(), "missing.key")})).OK {
		t.Fatal("expected error for missing key file")
	}
}

func TestCmdTim_timKeyphrase_Good(t *testing.T) {
	keyFile := core.PathJoin(t.TempDir(), "k.key")
	if err := io.Local.Write(keyFile, "  file-secret\n"); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	fileRes := timKeyphrase(core.NewOptions(core.Option{Key: "key-file", Value: keyFile}))
	if !fileRes.OK {
		t.Fatalf("timKeyphrase key-file: %v", fileRes.Error())
	}
	if got := core.MustCast[string](fileRes); got != "file-secret" {
		t.Fatalf("key-file secret = %q, want trimmed secret", got)
	}

	t.Setenv("CORE_TIM_KEY", "env-secret")
	envRes := timKeyphrase(core.NewOptions())
	if !envRes.OK {
		t.Fatalf("timKeyphrase env: %v", envRes.Error())
	}
	if got := core.MustCast[string](envRes); got != "env-secret" {
		t.Fatalf("env secret = %q, want env-secret", got)
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

func TestCmdTim_timEncrypt_Good(t *testing.T) {
	// pack -> timEncrypt -> timDecrypt round-trips the payload; the .stim hides it.
	src := t.TempDir()
	if err := io.Local.Write(core.PathJoin(src, "hello.txt"), "world"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	dir := t.TempDir()
	timPath := core.PathJoin(dir, "app.tim")
	stimPath := core.PathJoin(dir, "app.stim")
	backPath := core.PathJoin(dir, "back.tim")
	keyPath := core.PathJoin(dir, "k.key")
	if err := io.Local.Write(keyPath, "correct horse battery staple"); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	keyOpts := core.NewOptions(core.Option{Key: "key-file", Value: keyPath})

	if r := timPack(src, timPath); !r.OK {
		t.Fatalf("pack: %v", r.Error())
	}
	if r := timEncrypt(timPath, stimPath, keyOpts); !r.OK {
		t.Fatalf("encrypt: %v", r.Error())
	}
	stim, err := io.Local.Read(stimPath)
	if err != nil {
		t.Fatalf("read stim: %v", err)
	}
	if !timIsSTIM([]byte(stim)) {
		t.Fatal("encrypted output should carry the STIM magic")
	}
	if core.Contains(stim, "world") {
		t.Fatal("encrypted .stim must not contain the cleartext payload")
	}
	if r := timDecrypt(stimPath, backPath, keyOpts); !r.OK {
		t.Fatalf("decrypt: %v", r.Error())
	}
	back, err := io.Local.Read(backPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !core.Contains(back, "world") {
		t.Fatal("decrypted .tim should restore the cleartext payload")
	}
}

func TestCmdTim_timEncrypt_Bad(t *testing.T) {
	keyPath := core.PathJoin(t.TempDir(), "k.key")
	if err := io.Local.Write(keyPath, "secret"); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	keyOpts := core.NewOptions(core.Option{Key: "key-file", Value: keyPath})
	if timEncrypt(core.PathJoin(t.TempDir(), "missing.tim"), core.PathJoin(t.TempDir(), "out.stim"), keyOpts).OK {
		t.Fatal("expected encrypt of a missing .tim to fail")
	}
	bogus := core.PathJoin(t.TempDir(), "bogus.tim")
	if err := io.Local.Write(bogus, "not a tar"); err != nil {
		t.Fatalf("seed bogus: %v", err)
	}
	if timEncrypt(bogus, core.PathJoin(t.TempDir(), "out.stim"), keyOpts).OK {
		t.Fatal("expected encrypt of an invalid .tim to fail")
	}
}

func TestCmdTim_timDecrypt_Bad(t *testing.T) {
	// Wrong key -> FromSigil fails.
	src := t.TempDir()
	if err := io.Local.Write(core.PathJoin(src, "f"), "secret"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	dir := t.TempDir()
	timPath := core.PathJoin(dir, "a.tim")
	stimPath := core.PathJoin(dir, "a.stim")
	goodKey := core.PathJoin(dir, "good.key")
	badKey := core.PathJoin(dir, "bad.key")
	_ = io.Local.Write(goodKey, "right-key")
	_ = io.Local.Write(badKey, "wrong-key")
	if r := timPack(src, timPath); !r.OK {
		t.Fatalf("pack: %v", r.Error())
	}
	if r := timEncrypt(timPath, stimPath, core.NewOptions(core.Option{Key: "key-file", Value: goodKey})); !r.OK {
		t.Fatalf("encrypt: %v", r.Error())
	}
	if timDecrypt(stimPath, core.PathJoin(dir, "out.tim"), core.NewOptions(core.Option{Key: "key-file", Value: badKey})).OK {
		t.Fatal("expected decrypt with the wrong key to fail")
	}
	if timDecrypt(core.PathJoin(t.TempDir(), "missing.stim"), core.PathJoin(t.TempDir(), "out.tim"), core.NewOptions(core.Option{Key: "key-file", Value: goodKey})).OK {
		t.Fatal("expected decrypt of missing .stim to fail")
	}
}

func TestCmdTim_timInspect_Good(t *testing.T) {
	// Inspecting a plain .tim succeeds (reads config without a key).
	src := t.TempDir()
	if err := io.Local.Write(core.PathJoin(src, "f"), "x"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	timPath := core.PathJoin(t.TempDir(), "a.tim")
	if r := timPack(src, timPath); !r.OK {
		t.Fatalf("pack: %v", r.Error())
	}
	if r := timInspect(timPath); !r.OK {
		t.Fatalf("inspect tim: %v", r.Error())
	}

	stimPath := core.PathJoin(t.TempDir(), "a.stim")
	keyPath := core.PathJoin(t.TempDir(), "k.key")
	if err := io.Local.Write(keyPath, "secret"); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	if r := timEncrypt(timPath, stimPath, core.NewOptions(core.Option{Key: "key-file", Value: keyPath})); !r.OK {
		t.Fatalf("encrypt stim: %v", r.Error())
	}
	if r := timInspect(stimPath); !r.OK {
		t.Fatalf("inspect stim: %v", r.Error())
	}
}

func TestCmdTim_timInspect_Bad(t *testing.T) {
	// A file that is neither a valid tar nor a STIM container -> Fail.
	bogus := core.PathJoin(t.TempDir(), "bogus.bin")
	if err := io.Local.Write(bogus, "not a tim or stim"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if timInspect(bogus).OK {
		t.Fatal("expected inspect of a bogus file to fail")
	}
	if timInspect(core.PathJoin(t.TempDir(), "missing.tim")).OK {
		t.Fatal("expected inspect of a missing file to fail")
	}
}
