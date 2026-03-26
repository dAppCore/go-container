package coreutil

import (
	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// DirSep returns the active directory separator.
func DirSep() string {
	if ds := core.Env("DS"); ds != "" {
		return ds
	}
	return "/"
}

// JoinPath joins path segments using the active directory separator.
func JoinPath(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return core.CleanPath(core.Join(DirSep(), parts...), DirSep())
}

// HomeDir returns the current home directory, honouring test-time env overrides.
func HomeDir() string {
	if home := core.Env("CORE_HOME"); home != "" {
		return home
	}
	if home := core.Env("HOME"); home != "" {
		return home
	}
	if home := core.Env("USERPROFILE"); home != "" {
		return home
	}
	return core.Env("DIR_HOME")
}

// CurrentDir returns the current working directory, honouring shell PWD.
func CurrentDir() string {
	if pwd := core.Env("PWD"); pwd != "" {
		return pwd
	}
	return core.Env("DIR_CWD")
}

// TempDir returns the process temp directory, honouring TMPDIR.
func TempDir() string {
	if dir := core.Env("TMPDIR"); dir != "" {
		return dir
	}
	return core.Env("DIR_TMP")
}

// AbsPath resolves a path against the current working directory.
func AbsPath(path string) string {
	if path == "" {
		return CurrentDir()
	}
	if core.PathIsAbs(path) {
		return core.CleanPath(path, DirSep())
	}
	return JoinPath(CurrentDir(), path)
}

// MkdirTemp creates a temporary directory with a deterministic Core-generated name.
func MkdirTemp(prefix string) (string, error) {
	name := prefix
	if name == "" {
		name = "tmp-"
	}
	path := JoinPath(TempDir(), core.Concat(name, core.ID()))
	if err := coreio.Local.EnsureDir(path); err != nil {
		return "", err
	}
	return path, nil
}
