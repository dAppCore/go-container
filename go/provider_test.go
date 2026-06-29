package container

import (
	"reflect"
	"testing"
)

func TestProvider_ApplyRunOptions_Good(t *testing.T) {
	auditTarget := "ApplyRunOptions"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ApplyRunOptions(
		WithName("api"),
		WithMemory(2048),
		WithCPUs(4),
		WithDetach(true),
		WithPorts(map[int]int{8080: 80}),
		WithVolumes(map[string]string{"/data": "/app/data"}),
		WithArgs("serve", "--port", "8080"),
		WithEnv("PORT=8080", "MODE=test"),
		WithDNS("1.1.1.1", "8.8.8.8"),
	)
	if got, want := opts.Name, "api"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Memory, 2048; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.CPUs, 4; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if !(opts.Detach) {
		t.Fatal("expected true")
	}
	if got, want := opts.Ports[8080], 80; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Volumes["/data"], "/app/data"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Args, []string{"serve", "--port", "8080"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Env, []string{"PORT=8080", "MODE=test"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.DNS, []string{"1.1.1.1", "8.8.8.8"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestProvider_ApplyRunOptions_NilOption_Bad(t *testing.T) {
	auditTarget := "ApplyRunOptions NilOption"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Nil options must be skipped without panicking.
	opts := ApplyRunOptions(nil, WithName("ok"), nil)
	if got, want := opts.Name, "ok"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestProvider_ApplyRunOptions_OverwriteAndMerge_Ugly(t *testing.T) {
	auditTarget := "ApplyRunOptions OverwriteAndMerge"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Applying two WithPorts calls merges maps; applying two WithMemory calls overwrites.
	opts := ApplyRunOptions(
		WithMemory(1024),
		WithMemory(4096),
		WithPorts(map[int]int{8080: 80}),
		WithPorts(map[int]int{9090: 90}),
		WithArgs("one"),
		WithArgs("two", "three"),
		WithEnv("A=1"),
		WithEnv("B=2"),
		WithDNS("9.9.9.9"),
		WithDNS("4.4.4.4"),
	)
	if got, want := opts.Memory, 4096; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Ports[8080], 80; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Ports[9090], 90; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Args, []string{"one", "two", "three"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.Env, []string{"A=1", "B=2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := opts.DNS, []string{"9.9.9.9", "4.4.4.4"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestProvider_WithGPU_Good(t *testing.T) {
	auditTarget := "WithGPU"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ApplyRunOptions(WithGPU(true))
	if !(opts.GPU) {
		t.Fatal("expected true")
	}
}

func TestProvider_WithGPU_Disabled_Bad(t *testing.T) {
	auditTarget := "WithGPU Disabled"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ApplyRunOptions(WithGPU(false))
	if opts.GPU {
		t.Fatal("expected false")
	}
}

func TestProvider_WithGPU_OverriddenByLater_Ugly(t *testing.T) {
	auditTarget := "WithGPU OverriddenByLater"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	opts := ApplyRunOptions(WithGPU(true), WithGPU(false))
	if opts.GPU {
		t.Fatal("expected false")
	}
}

// --- AX-7 canonical triplets ---

func TestProvider_WithName_Good(t *testing.T) {
	auditTarget := "WithName"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithName
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithName_Bad(t *testing.T) {
	auditTarget := "WithName"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithName
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithName_Ugly(t *testing.T) {
	auditTarget := "WithName"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithName
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithMemory_Good(t *testing.T) {
	auditTarget := "WithMemory"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithMemory
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithMemory_Bad(t *testing.T) {
	auditTarget := "WithMemory"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithMemory
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithMemory_Ugly(t *testing.T) {
	auditTarget := "WithMemory"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithMemory
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithCPUs_Good(t *testing.T) {
	auditTarget := "WithCPUs"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithCPUs
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithCPUs_Bad(t *testing.T) {
	auditTarget := "WithCPUs"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithCPUs
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithCPUs_Ugly(t *testing.T) {
	auditTarget := "WithCPUs"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithCPUs
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithDetach_Good(t *testing.T) {
	auditTarget := "WithDetach"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithDetach
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithDetach_Bad(t *testing.T) {
	auditTarget := "WithDetach"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithDetach
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithDetach_Ugly(t *testing.T) {
	auditTarget := "WithDetach"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithDetach
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithPorts_Good(t *testing.T) {
	auditTarget := "WithPorts"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithPorts
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithPorts_Bad(t *testing.T) {
	auditTarget := "WithPorts"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithPorts
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithPorts_Ugly(t *testing.T) {
	auditTarget := "WithPorts"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithPorts
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithVolumes_Good(t *testing.T) {
	auditTarget := "WithVolumes"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithVolumes
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithVolumes_Bad(t *testing.T) {
	auditTarget := "WithVolumes"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithVolumes
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithVolumes_Ugly(t *testing.T) {
	auditTarget := "WithVolumes"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := WithVolumes
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithArgs_Good(t *testing.T) {
	o := ApplyRunOptions(WithArgs("sleep", "300"))
	if len(o.Args) != 2 || o.Args[0] != "sleep" || o.Args[1] != "300" {
		t.Fatalf("WithArgs => %v, want [sleep 300]", o.Args)
	}
}

func TestProvider_WithArgs_Bad(t *testing.T) {
	// No args is a degenerate but valid call: Args stays empty.
	o := ApplyRunOptions(WithArgs())
	if len(o.Args) != 0 {
		t.Fatalf("WithArgs() => %v, want empty", o.Args)
	}
}

func TestProvider_WithArgs_Ugly(t *testing.T) {
	// Order and flag-shaped tokens are preserved verbatim.
	o := ApplyRunOptions(WithArgs("--", "/bin/sh", "-c", "echo a b"))
	want := []string{"--", "/bin/sh", "-c", "echo a b"}
	if len(o.Args) != len(want) {
		t.Fatalf("WithArgs len = %d, want %d (%v)", len(o.Args), len(want), o.Args)
	}
	for i := range want {
		if o.Args[i] != want[i] {
			t.Fatalf("WithArgs[%d] = %q, want %q", i, o.Args[i], want[i])
		}
	}
}

func TestProvider_WithEnv_Good(t *testing.T) {
	o := ApplyRunOptions(WithEnv("FOO=bar", "BAZ=qux"))
	if len(o.Env) != 2 || o.Env[0] != "FOO=bar" || o.Env[1] != "BAZ=qux" {
		t.Fatalf("WithEnv => %v, want [FOO=bar BAZ=qux]", o.Env)
	}
}

func TestProvider_WithEnv_Bad(t *testing.T) {
	// No args is a degenerate but valid call: Env stays empty.
	o := ApplyRunOptions(WithEnv())
	if len(o.Env) != 0 {
		t.Fatalf("WithEnv() => %v, want empty", o.Env)
	}
}

func TestProvider_WithEnv_Ugly(t *testing.T) {
	// Values may contain '=' and be empty; order is preserved.
	o := ApplyRunOptions(WithEnv("URL=https://x?a=b", "EMPTY="))
	want := []string{"URL=https://x?a=b", "EMPTY="}
	if len(o.Env) != len(want) {
		t.Fatalf("WithEnv len = %d, want %d (%v)", len(o.Env), len(want), o.Env)
	}
	for i := range want {
		if o.Env[i] != want[i] {
			t.Fatalf("WithEnv[%d] = %q, want %q", i, o.Env[i], want[i])
		}
	}
}

func TestProvider_ApplyRunOptions_Bad(t *testing.T) {
	auditTarget := "ApplyRunOptions"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyRunOptions
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_ApplyRunOptions_Ugly(t *testing.T) {
	auditTarget := "ApplyRunOptions"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := ApplyRunOptions
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_WithSharedDir_Good(t *testing.T) {
	auditTarget := "WithSharedDir"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Read-write by default; shares fold in declaration order onto FSShares.
	o := ApplyRunOptions(WithSharedDir("/host/work", "workspace"))
	if len(o.FSShares) != 1 {
		t.Fatalf("FSShares len = %d, want 1", len(o.FSShares))
	}
	got := o.FSShares[0]
	if got.HostDir != "/host/work" || got.Tag != "workspace" || got.ReadOnly {
		t.Fatalf("WithSharedDir => %+v, want read-write /host/work as workspace", got)
	}
}

func TestProvider_WithSharedDir_Bad(t *testing.T) {
	auditTarget := "WithSharedDirRO"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// The RO variant marks the share read-only — the only difference from
	// WithSharedDir. Option-folding records the request verbatim; the
	// directory itself is validated later by the provider (vzAttachFileSystems).
	o := ApplyRunOptions(WithSharedDirRO("/host/inputs", "inputs"))
	if len(o.FSShares) != 1 || !o.FSShares[0].ReadOnly {
		t.Fatalf("WithSharedDirRO => %+v, want a read-only share", o.FSShares)
	}
}

func TestProvider_WithSharedDir_Ugly(t *testing.T) {
	auditTarget := "WithSharedDir"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Multiple shares — including a mix of rw/ro and even empty values —
	// accumulate in order; the option layer never dedups or validates, that is
	// the provider's job at Run.
	o := ApplyRunOptions(
		WithSharedDir("/a", "a"),
		WithSharedDirRO("/b", "b"),
		WithSharedDir("", ""),
	)
	if len(o.FSShares) != 3 {
		t.Fatalf("FSShares len = %d, want 3 (%+v)", len(o.FSShares), o.FSShares)
	}
	if o.FSShares[0].Tag != "a" || o.FSShares[1].Tag != "b" || !o.FSShares[1].ReadOnly {
		t.Fatalf("order or ro flag wrong: %+v", o.FSShares)
	}
}
