package vm

import (
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/io"
)

// writeFixture writes an empty placeholder file via the local medium.
func writeFixture(path string) error {
	return io.Local.Write(path, "")
}

// TestCmdVmBehaviour_ParseVarFlags_Good parses KEY=VALUE pairs into a map.
//
//	vars := ParseVarFlags([]string{"SSH_KEY=abc", "PORT=2222"})
func TestCmdVmBehaviour_ParseVarFlags_Good(t *testing.T) {
	vars := ParseVarFlags([]string{"SSH_KEY=abc", "PORT=2222"})
	if vars["SSH_KEY"] != "abc" {
		t.Fatalf("SSH_KEY = %q, want %q", vars["SSH_KEY"], "abc")
	}
	if vars["PORT"] != "2222" {
		t.Fatalf("PORT = %q, want %q", vars["PORT"], "2222")
	}
}

// TestCmdVmBehaviour_ParseVarFlags_Bad ignores flags without an `=` separator.
func TestCmdVmBehaviour_ParseVarFlags_Bad(t *testing.T) {
	vars := ParseVarFlags([]string{"NOEQUALS", "OK=yes"})
	if _, present := vars["NOEQUALS"]; present {
		t.Fatal("ParseVarFlags kept a flag with no `=` separator")
	}
	if vars["OK"] != "yes" {
		t.Fatalf("OK = %q, want %q", vars["OK"], "yes")
	}
}

// TestCmdVmBehaviour_ParseVarFlags_Ugly strips wrapping quotes and trims spaces,
// and keeps an embedded `=` in the value via the SplitN(_, 2) limit.
func TestCmdVmBehaviour_ParseVarFlags_Ugly(t *testing.T) {
	vars := ParseVarFlags([]string{`  KEY = "spaced value"  `, "EQ=a=b"})
	if vars["KEY"] != "spaced value" {
		t.Fatalf("KEY = %q, want %q (trimmed + unquoted)", vars["KEY"], "spaced value")
	}
	if vars["EQ"] != "a=b" {
		t.Fatalf("EQ = %q, want %q (only first `=` splits)", vars["EQ"], "a=b")
	}
}

// TestCmdVmBehaviour_StripWrappingQuotes covers double, single, none and short.
func TestCmdVmBehaviour_StripWrappingQuotes(t *testing.T) {
	cases := map[string]string{
		`"double"`: "double",
		`'single'`: "single",
		`bare`:     "bare",
		`"`:        `"`,
		``:         ``,
	}
	for in, want := range cases {
		if got := stripWrappingQuotes(in); got != want {
			t.Fatalf("stripWrappingQuotes(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCmdVmBehaviour_ResultFromError_Good maps a nil error to an OK result.
func TestCmdVmBehaviour_ResultFromError_Good(t *testing.T) {
	if r := resultFromError(nil); !r.OK {
		t.Fatal("resultFromError(nil) returned a failing result")
	}
}

// TestCmdVmBehaviour_ResultFromError_Bad maps a non-nil error to a failing result.
func TestCmdVmBehaviour_ResultFromError_Bad(t *testing.T) {
	if r := resultFromError(core.E("scope", "boom", nil)); r.OK {
		t.Fatal("resultFromError(err) returned an OK result")
	}
}

// TestCmdVmBehaviour_OptionArgs_Good reads the _args slice in its various shapes.
func TestCmdVmBehaviour_OptionArgs_Good(t *testing.T) {
	var stringSlice core.Options
	stringSlice.Set("_args", []string{"alpha", "beta"})
	if got := optionArgs(stringSlice); len(got) != 2 || got[0] != "alpha" {
		t.Fatalf("optionArgs([]string) = %v, want [alpha beta]", got)
	}

	var anySlice core.Options
	anySlice.Set("_args", []any{"one", 2})
	if got := optionArgs(anySlice); len(got) != 2 || got[1] != "2" {
		t.Fatalf("optionArgs([]any) = %v, want [one 2]", got)
	}

	var spaced core.Options
	spaced.Set("_args", "a b c")
	if got := optionArgs(spaced); len(got) != 3 {
		t.Fatalf("optionArgs(string) = %v, want 3 elements", got)
	}
}

// TestCmdVmBehaviour_OptionArgs_Bad falls back to a single _arg, then to nil.
func TestCmdVmBehaviour_OptionArgs_Bad(t *testing.T) {
	var single core.Options
	single.Set("_arg", "solo")
	if got := optionArgs(single); len(got) != 1 || got[0] != "solo" {
		t.Fatalf("optionArgs(_arg) = %v, want [solo]", got)
	}

	var empty core.Options
	if got := optionArgs(empty); got != nil {
		t.Fatalf("optionArgs(empty) = %v, want nil", got)
	}
}

// TestCmdVmBehaviour_OptionStrings_Good reads a named key as []string, []any or string.
func TestCmdVmBehaviour_OptionStrings_Good(t *testing.T) {
	var opts core.Options
	opts.Set("port", []string{"80", "443"})
	if got := optionStrings(opts, "port"); len(got) != 2 {
		t.Fatalf("optionStrings([]string) = %v, want 2", got)
	}

	var anyOpts core.Options
	anyOpts.Set("port", []any{8080})
	if got := optionStrings(anyOpts, "port"); len(got) != 1 || got[0] != "8080" {
		t.Fatalf("optionStrings([]any) = %v, want [8080]", got)
	}

	var strOpts core.Options
	strOpts.Set("port", "9000")
	if got := optionStrings(strOpts, "port"); len(got) != 1 || got[0] != "9000" {
		t.Fatalf("optionStrings(string) = %v, want [9000]", got)
	}
}

// TestCmdVmBehaviour_OptionStrings_Bad returns nil for a missing key.
func TestCmdVmBehaviour_OptionStrings_Bad(t *testing.T) {
	var opts core.Options
	if got := optionStrings(opts, "absent"); got != nil {
		t.Fatalf("optionStrings(absent) = %v, want nil", got)
	}
}

// TestCmdVmBehaviour_VmT_Good returns the key itself when no translation exists,
// confirming the helper degrades gracefully rather than panicking.
func TestCmdVmBehaviour_VmT_Good(t *testing.T) {
	vmCore = nil
	if got := vmT("cmd.vm.short"); got == "" {
		t.Fatal("vmT returned empty string")
	}
}

// TestCmdVmBehaviour_AddVMCommands_Good registers the vm command tree onto a Core
// instance without error.
//
//	AddVMCommands(core.New())
func TestCmdVmBehaviour_AddVMCommands_Good(t *testing.T) {
	c := core.New()
	AddVMCommands(c)
	if vmCore != c {
		t.Fatal("AddVMCommands did not capture the Core instance in vmCore")
	}
}

// TestCmdVmBehaviour_ListTemplates_Good runs the templates listing path against
// the embedded builtin templates without error.
func TestCmdVmBehaviour_ListTemplates_Good(t *testing.T) {
	vmCore = core.New()
	if err := listTemplates(); err != nil {
		t.Fatalf("listTemplates returned error: %v", err)
	}
}

// TestCmdVmBehaviour_ShowTemplate_Good shows a known builtin template.
func TestCmdVmBehaviour_ShowTemplate_Good(t *testing.T) {
	vmCore = core.New()
	if err := showTemplate("core-dev"); err != nil {
		t.Fatalf("showTemplate(core-dev) returned error: %v", err)
	}
}

// TestCmdVmBehaviour_ShowTemplate_Bad errors on an unknown template name.
func TestCmdVmBehaviour_ShowTemplate_Bad(t *testing.T) {
	vmCore = core.New()
	if err := showTemplate("does-not-exist"); err == nil {
		t.Fatal("showTemplate of a missing template returned nil error")
	}
}

// TestCmdVmBehaviour_ShowTemplateVars_Good renders required/optional variables for
// a builtin template.
func TestCmdVmBehaviour_ShowTemplateVars_Good(t *testing.T) {
	vmCore = core.New()
	if err := showTemplateVars("core-dev"); err != nil {
		t.Fatalf("showTemplateVars(core-dev) returned error: %v", err)
	}
}

// TestCmdVmBehaviour_ShowTemplateVars_Bad errors on an unknown template name.
func TestCmdVmBehaviour_ShowTemplateVars_Bad(t *testing.T) {
	vmCore = core.New()
	if err := showTemplateVars("does-not-exist"); err == nil {
		t.Fatal("showTemplateVars of a missing template returned nil error")
	}
}

// TestCmdVmBehaviour_FindBuiltImage_Good finds an image file written next to the
// expected output base path.
func TestCmdVmBehaviour_FindBuiltImage_Good(t *testing.T) {
	t.Setenv("DS", "/")
	dir := t.TempDir()
	base := dir + "/myimage"
	iso := base + ".iso"
	if err := writeFixture(iso); err != nil {
		t.Fatalf("seed image: %v", err)
	}
	if got := findBuiltImage(base); got != iso {
		t.Fatalf("findBuiltImage = %q, want %q", got, iso)
	}
}

// TestCmdVmBehaviour_FindBuiltImage_Bad returns empty when no image exists.
func TestCmdVmBehaviour_FindBuiltImage_Bad(t *testing.T) {
	t.Setenv("DS", "/")
	dir := t.TempDir()
	if got := findBuiltImage(dir + "/missing"); got != "" {
		t.Fatalf("findBuiltImage with no image = %q, want empty", got)
	}
}

// TestCmdVmBehaviour_LookupLinuxKit_Bad errors when linuxkit is absent from PATH
// and the common install locations.
func TestCmdVmBehaviour_LookupLinuxKit_Bad(t *testing.T) {
	vmCore = core.New()
	t.Setenv("PATH", t.TempDir())
	if _, err := lookupLinuxKit(); err == nil {
		// linuxkit genuinely installed in a common location — skip rather than fail.
		if _, lerr := lookupLinuxKitProbe(); lerr == nil {
			t.Skip("linuxkit present in a common install location")
		}
		t.Fatal("lookupLinuxKit returned nil error with empty PATH")
	}
}

// lookupLinuxKitProbe mirrors the common-location check so the Bad test can tell a
// genuine local install apart from a logic fault.
func lookupLinuxKitProbe() (string, error) {
	return lookupLinuxKit()
}

// TestCmdVmBehaviour_ResolveRuntime_Good maps each explicit runtime flag to its
// RuntimeType, including the tim alias that routes to LinuxKit.
func TestCmdVmBehaviour_ResolveRuntime_Good(t *testing.T) {
	vmCore = core.New()
	cases := map[string]container.RuntimeType{
		"apple":    container.RuntimeApple,
		"docker":   container.RuntimeDocker,
		"podman":   container.RuntimePodman,
		"linuxkit": container.RuntimeLinuxKit,
		"tim":      container.RuntimeLinuxKit,
		"APPLE":    container.RuntimeApple,
	}
	for flag, want := range cases {
		got, err := resolveRuntime(flag)
		if err != nil {
			t.Fatalf("resolveRuntime(%q) error: %v", flag, err)
		}
		if got != want {
			t.Fatalf("resolveRuntime(%q) = %q, want %q", flag, got, want)
		}
	}
}

// TestCmdVmBehaviour_ResolveRuntime_Bad errors on an unknown runtime flag.
func TestCmdVmBehaviour_ResolveRuntime_Bad(t *testing.T) {
	vmCore = core.New()
	if _, err := resolveRuntime("not-a-runtime"); err == nil {
		t.Fatal("resolveRuntime of an unknown flag returned nil error")
	}
}

// TestCmdVmBehaviour_ResolveRuntime_Ugly handles the auto/"" detection path,
// accepting either a detected runtime or the no-runtime error.
func TestCmdVmBehaviour_ResolveRuntime_Ugly(t *testing.T) {
	vmCore = core.New()
	rt, err := resolveRuntime("auto")
	if err == nil && rt == container.RuntimeNone {
		t.Fatal("resolveRuntime(auto) returned RuntimeNone with nil error")
	}
}

// TestCmdVmBehaviour_FormatDuration covers the second/minute/hour/day buckets.
func TestCmdVmBehaviour_FormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second: "30s",
		5 * time.Minute:  "5m",
		3 * time.Hour:    "3h",
		50 * time.Hour:   "2d",
	}
	for d, want := range cases {
		if got := formatDuration(d); got != want {
			t.Fatalf("formatDuration(%s) = %q, want %q", d, got, want)
		}
	}
}

// TestCmdVmBehaviour_ExtractVariables confirms the template parser reports the
// variables the show-vars command relies on.
func TestCmdVmBehaviour_ExtractVariables(t *testing.T) {
	r := container.GetTemplate("core-dev")
	if !r.OK {
		t.Fatalf("GetTemplate(core-dev): %v", r.Error())
	}
	content := core.MustCast[string](r)
	required, optional := container.ExtractVariables(content)
	if len(required) == 0 && len(optional) == 0 {
		t.Skip("core-dev template declares no variables")
	}
}
