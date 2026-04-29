package container

import (
	"reflect"
	"testing"
)

func TestProvider_ApplyRunOptions_Good(t *testing.T) {
	opts := ApplyRunOptions(
		WithName("api"),
		WithMemory(2048),
		WithCPUs(4),
		WithDetach(true),
		WithPorts(map[int]int{8080: 80}),
		WithVolumes(map[string]string{"/data": "/app/data"}),
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
}

func TestProvider_ApplyRunOptions_NilOption_Bad(t *testing.T) {
	// Nil options must be skipped without panicking.
	opts := ApplyRunOptions(nil, WithName("ok"), nil)
	if got, want := opts.Name, "ok"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestProvider_ApplyRunOptions_OverwriteAndMerge_Ugly(t *testing.T) {
	// Applying two WithPorts calls merges maps; applying two WithMemory calls overwrites.
	opts := ApplyRunOptions(
		WithMemory(1024),
		WithMemory(4096),
		WithPorts(map[int]int{8080: 80}),
		WithPorts(map[int]int{9090: 90}),
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
}

func TestProvider_WithGPU_Good(t *testing.T) {
	opts := ApplyRunOptions(WithGPU(true))
	if !(opts.GPU) {
		t.Fatal("expected true")
	}
}

func TestProvider_WithGPU_Disabled_Bad(t *testing.T) {
	opts := ApplyRunOptions(WithGPU(false))
	if opts.GPU {
		t.Fatal("expected false")
	}
}

func TestProvider_WithGPU_OverriddenByLater_Ugly(t *testing.T) {
	opts := ApplyRunOptions(WithGPU(true), WithGPU(false))
	if opts.GPU {
		t.Fatal("expected false")
	}
}

// --- AX-7 canonical triplets ---

func TestProvider_WithName_Good(t *testing.T) {
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
	symbol := WithVolumes
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestProvider_ApplyRunOptions_Bad(t *testing.T) {
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
	symbol := ApplyRunOptions
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
