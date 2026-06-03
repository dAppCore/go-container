package vm

import (
	core "dappco.re/go"
	"dappco.re/go/io"

	"forge.lthn.ai/Snider/Borg/pkg/datanode"
	borgtim "forge.lthn.ai/Snider/Borg/pkg/tim"
	"forge.lthn.ai/Snider/Enchantrix/pkg/trix"
)

// timKeyphrase resolves the STIM passphrase from --key-file (read + trimmed) or
// the CORE_TIM_KEY env var. The value is fed to Borg's ToSigil/FromSigil, which
// derives the AEAD key via sha256. Fails if neither yields a non-empty secret.
func timKeyphrase(opts core.Options) core.Result { // Value: string
	if kf := opts.String("key-file"); kf != "" {
		raw, err := io.Local.Read(kf)
		if err != nil {
			return core.Fail(core.E("vm tim", "read key file: "+kf, err))
		}
		pass := core.Trim(raw)
		if pass == "" {
			return core.Fail(core.E("vm tim", "key file is empty: "+kf, nil))
		}
		return core.Ok(pass)
	}
	if env := core.Env("CORE_TIM_KEY"); env != "" {
		return core.Ok(env)
	}
	return core.Fail(core.E("vm tim", "no key: pass --key-file or set CORE_TIM_KEY", nil))
}

// timIsSTIM reports whether data carries the Borg STIM magic prefix.
func timIsSTIM(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == "STIM"
}

// timPack packs the source directory into a Borg-default .tim bundle at outPath.
func timPack(srcDir, outPath string) core.Result { // Value: nil
	m, err := borgtim.New()
	if err != nil {
		return core.Fail(core.E("vm tim pack", "new tim", err))
	}
	if err := m.RootFS.AddPath(srcDir, datanode.AddPathOptions{}); err != nil {
		return core.Fail(core.E("vm tim pack", "add path: "+srcDir, err))
	}
	tarBytes, err := m.ToTar()
	if err != nil {
		return core.Fail(core.E("vm tim pack", "to tar", err))
	}
	if err := io.Local.Write(outPath, string(tarBytes)); err != nil {
		return core.Fail(core.E("vm tim pack", "write: "+outPath, err))
	}
	core.Print(nil, "%s %s", successStyle.Render("packed"), outPath)
	core.Println()
	return core.Ok(nil)
}

// timEncrypt reads a plain .tim and writes an encrypted .stim under the
// passphrase resolved from opts.
func timEncrypt(inPath, outPath string, opts core.Options) core.Result { // Value: nil
	passRes := timKeyphrase(opts)
	if !passRes.OK {
		return passRes
	}
	raw, err := io.Local.Read(inPath)
	if err != nil {
		return core.Fail(core.E("vm tim encrypt", "read: "+inPath, err))
	}
	m, err := borgtim.FromTar([]byte(raw))
	if err != nil {
		return core.Fail(core.E("vm tim encrypt", "parse tim: "+inPath, err))
	}
	sealed, err := m.ToSigil(core.MustCast[string](passRes))
	if err != nil {
		return core.Fail(core.E("vm tim encrypt", "to sigil", err))
	}
	if err := io.Local.Write(outPath, string(sealed)); err != nil {
		return core.Fail(core.E("vm tim encrypt", "write: "+outPath, err))
	}
	core.Print(nil, "%s %s", successStyle.Render("encrypted"), outPath)
	core.Println()
	return core.Ok(nil)
}

// timDecrypt reads an encrypted .stim and writes a plain .tim under the
// passphrase resolved from opts.
func timDecrypt(inPath, outPath string, opts core.Options) core.Result { // Value: nil
	passRes := timKeyphrase(opts)
	if !passRes.OK {
		return passRes
	}
	raw, err := io.Local.Read(inPath)
	if err != nil {
		return core.Fail(core.E("vm tim decrypt", "read: "+inPath, err))
	}
	m, err := borgtim.FromSigil([]byte(raw), core.MustCast[string](passRes))
	if err != nil {
		return core.Fail(core.E("vm tim decrypt", "from sigil (wrong key?)", err))
	}
	tarBytes, err := m.ToTar()
	if err != nil {
		return core.Fail(core.E("vm tim decrypt", "to tar", err))
	}
	if err := io.Local.Write(outPath, string(tarBytes)); err != nil {
		return core.Fail(core.E("vm tim decrypt", "write: "+outPath, err))
	}
	core.Print(nil, "%s %s", successStyle.Render("decrypted"), outPath)
	core.Println()
	return core.Ok(nil)
}

// timInspect prints metadata for a .tim (decoded config) or .stim (Trix header,
// no key required). The file kind is sniffed from the STIM magic prefix.
func timInspect(path string) core.Result { // Value: nil
	raw, err := io.Local.Read(path)
	if err != nil {
		return core.Fail(core.E("vm tim inspect", "read: "+path, err))
	}
	data := []byte(raw)
	if timIsSTIM(data) {
		tx, derr := trix.Decode(data, "STIM", nil)
		if derr != nil {
			return core.Fail(core.E("vm tim inspect", "decode stim header", derr))
		}
		hdrRes := core.JSONMarshalIndent(tx.Header, "", "  ")
		if !hdrRes.OK {
			return hdrRes
		}
		core.Print(nil, "%s %s", dimStyle.Render("format"), "stim")
		core.Println()
		core.Println(string(core.MustCast[[]byte](hdrRes)))
		return core.Ok(nil)
	}
	m, ferr := borgtim.FromTar(data)
	if ferr != nil {
		return core.Fail(core.E("vm tim inspect", "parse tim: "+path, ferr))
	}
	core.Print(nil, "%s %s", dimStyle.Render("format"), "tim")
	core.Println()
	core.Println(string(m.Config))
	return core.Ok(nil)
}

// addVMTimCommand registers the `vm tim` subgroup (pack/encrypt/decrypt/inspect).
func addVMTimCommand(c *core.Core) {
	registerVMCommand(c, "vm/tim", core.Command{
		Description: "Manage Borg TIM/STIM container bundles",
	})
	registerVMCommand(c, "vm/tim/pack", core.Command{
		Description: "Pack a directory into a Borg .tim bundle",
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) < 2 {
				return core.Fail(core.E("vm tim pack", "usage: vm tim pack <src-dir> <out.tim>", nil))
			}
			return timPack(args[0], args[1])
		},
	})
	registerVMCommand(c, "vm/tim/encrypt", core.Command{
		Description: "Encrypt a .tim into a .stim (--key-file)",
		Flags:       core.NewOptions(core.Option{Key: "key-file", Value: ""}),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) < 2 {
				return core.Fail(core.E("vm tim encrypt", "usage: vm tim encrypt <in.tim> <out.stim> --key-file <p>", nil))
			}
			return timEncrypt(args[0], args[1], opts)
		},
	})
	registerVMCommand(c, "vm/tim/decrypt", core.Command{
		Description: "Decrypt a .stim back into a .tim (--key-file)",
		Flags:       core.NewOptions(core.Option{Key: "key-file", Value: ""}),
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) < 2 {
				return core.Fail(core.E("vm tim decrypt", "usage: vm tim decrypt <in.stim> <out.tim> --key-file <p>", nil))
			}
			return timDecrypt(args[0], args[1], opts)
		},
	})
	registerVMCommand(c, "vm/tim/inspect", core.Command{
		Description: "Show metadata for a .tim (config) or .stim (header)",
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm tim inspect", "usage: vm tim inspect <file>", nil))
			}
			return timInspect(args[0])
		},
	})
}
