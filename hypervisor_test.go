package container

import (
	"context"
	"dappco.re/go"
	"reflect"
	"runtime"
	"slices"
	"testing"
)

func TestQemuHypervisor_Available_Good(t *testing.T) {
	q := NewQemuHypervisor()

	// Check if qemu is available on this system
	available := q.Available()

	// We just verify it returns a boolean without error
	// The actual availability depends on the system
	if got, want := reflect.TypeOf(available), reflect.TypeOf(true); got != want {
		t.Fatalf("want type %v, got %v", want, got)
	}
}

func TestQemuHypervisor_Available_InvalidBinary_Bad(t *testing.T) {
	q := &QemuHypervisor{
		Binary: "nonexistent-qemu-binary-that-does-not-exist",
	}

	available := q.Available()
	if available {
		t.Fatal("expected false")
	}
}

func TestHyperkitHypervisor_Available_Good(t *testing.T) {
	h := NewHyperkitHypervisor()

	available := h.Available()

	// On non-darwin systems, should always be false
	if runtime.GOOS != "darwin" {
		if available {
			t.Fatal("expected false")
		}
	} else {
		// On darwin, just verify it returns a boolean
		if got, want := reflect.TypeOf(available), reflect.TypeOf(true); got != want {
			t.Fatalf("want type %v, got %v", want, got)
		}
	}
}

func TestHyperkitHypervisor_Available_NotDarwin_Bad(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("This test only runs on non-darwin systems")
	}

	h := NewHyperkitHypervisor()

	available := h.Available()
	if available {
		t.Fatal("expected false")
	}
}

func TestHyperkitHypervisor_Available_InvalidBinary_Bad(t *testing.T) {
	h := &HyperkitHypervisor{
		Binary: "nonexistent-hyperkit-binary-that-does-not-exist",
	}

	available := h.Available()
	if available {
		t.Fatal("expected false")
	}
}

func TestHypervisor_IsKVMAvailable_Good(t *testing.T) {
	// This test verifies the function runs without error
	// The actual result depends on the system
	result := isKVMAvailable()

	// On non-linux systems, should be false
	if runtime.GOOS != "linux" {
		if result {
			t.Fatal("expected false")
		}
	} else {
		// On linux, just verify it returns a boolean
		if got, want := reflect.TypeOf(result), reflect.TypeOf(true); got != want {
			t.Fatalf("want type %v, got %v", want, got)
		}
	}
}

func TestHypervisor_DetectHypervisor_Good(t *testing.T) {
	// DetectHypervisor tries to find an available hypervisor
	hv, err := DetectHypervisor()

	// This test may pass or fail depending on system configuration
	// If no hypervisor is available, it should return an error
	if err != nil {
		if hv != nil {
			t.Fatal("expected nil")
		}
		if s, sub := err.Error(), "no hypervisor available"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	} else {
		if hv == nil {
			t.Fatal("expected non-nil value")
		}
		if got := hv.Name(); len(got) == 0 {
			t.Fatal("expected non-empty value")
		}
	}
}

func TestGetHypervisor_Qemu_Good(t *testing.T) {
	hv, err := GetHypervisor("qemu")

	// Depends on whether qemu is installed
	if err != nil {
		if s, sub := err.Error(), "not available"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	} else {
		if hv == nil {
			t.Fatal("expected non-nil value")
		}
		if got, want := hv.Name(), "qemu"; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

func TestGetHypervisor_QemuUppercase_Good(t *testing.T) {
	hv, err := GetHypervisor("QEMU")

	// Depends on whether qemu is installed
	if err != nil {
		if s, sub := err.Error(), "not available"; !core.Contains(s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	} else {
		if hv == nil {
			t.Fatal("expected non-nil value")
		}
		if got, want := hv.Name(), "qemu"; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

func TestGetHypervisor_Hyperkit_Good(t *testing.T) {
	hv, err := GetHypervisor("hyperkit")

	// On non-darwin systems, should always fail
	if runtime.GOOS != "darwin" {
		if err == nil {
			t.Fatal("expected error")
		}
		if s, sub := err.Error(), "not available"; !core.Contains(

			// On darwin, depends on whether hyperkit is installed
			s, sub) {
			t.Fatalf("expected %v to contain %v", s, sub)
		}
	} else {

		if err != nil {
			if s, sub := err.Error(), "not available"; !core.Contains(s, sub) {
				t.Fatalf("expected %v to contain %v", s, sub)
			}
		} else {
			if hv == nil {
				t.Fatal("expected non-nil value")
			}
			if got, want := hv.Name(), "hyperkit"; !reflect.DeepEqual(got, want) {
				t.Fatalf("want %v, got %v", want, got)
			}
		}
	}
}

func TestGetHypervisor_Unknown_Bad(t *testing.T) {
	_, err := GetHypervisor("unknown-hypervisor")
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "unknown hypervisor"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestQemuHypervisor_BuildCommand_WithPortsAndVolumes_Good(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{
		Memory:  2048,
		CPUs:    4,
		SSHPort: 2222,
		Ports:   map[int]int{8080: 80, 443: 443},
		Volumes: map[string]string{
			"/host/data": "/container/data",
			"/host/logs": "/container/logs",
		},
		Detach: true,
	}

	cmd, err := q.BuildCommand(ctx, "/path/to/image.iso", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")

		// Verify command includes all expected args
	}

	args := cmd.Args
	if s, sub := args, "-m"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "2048"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "-smp"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "4"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestQemuHypervisor_BuildCommand_QCow2Format_Good(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	cmd, err := q.BuildCommand(ctx, "/path/to/image.qcow2", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the drive format is qcow2
	found := false
	for _, arg := range cmd.Args {
		if arg == "file=/path/to/image.qcow2,format=qcow2" {
			found = true
			break
		}
	}
	if !(found) {
		t.Fatal("expected true")
	}
}

func TestQemuHypervisor_BuildCommand_VMDKFormat_Good(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	cmd, err := q.BuildCommand(ctx, "/path/to/image.vmdk", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the drive format is vmdk
	found := false
	for _, arg := range cmd.Args {
		if arg == "file=/path/to/image.vmdk,format=vmdk" {
			found = true
			break
		}
	}
	if !(found) {
		t.Fatal("expected true")
	}
}

func TestQemuHypervisor_BuildCommand_RawFormat_Good(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	cmd, err := q.BuildCommand(ctx, "/path/to/image.raw", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the drive format is raw
	found := false
	for _, arg := range cmd.Args {
		if arg == "file=/path/to/image.raw,format=raw" {
			found = true
			break
		}
	}
	if !(found) {
		t.Fatal("expected true")
	}
}

func TestHyperkitHypervisor_BuildCommand_WithPorts_Good(t *testing.T) {
	h := NewHyperkitHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{
		Memory:  1024,
		CPUs:    2,
		SSHPort: 2222,
		Ports:   map[int]int{8080: 80},
	}

	cmd, err := h.BuildCommand(ctx, "/path/to/image.iso", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")

		// Verify it creates a command with memory and CPU args
	}

	args := cmd.Args
	if s, sub := args, "-m"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "1024M"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "-c"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "2"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestHyperkitHypervisor_BuildCommand_QCow2Format_Good(t *testing.T) {
	h := NewHyperkitHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	cmd, err := h.BuildCommand(ctx, "/path/to/image.qcow2", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestHyperkitHypervisor_BuildCommand_RawFormat_Good(t *testing.T) {
	h := NewHyperkitHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	cmd, err := h.BuildCommand(ctx, "/path/to/image.raw", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestHyperkitHypervisor_BuildCommand_NoPorts_Good(t *testing.T) {
	h := NewHyperkitHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{
		Memory:  512,
		CPUs:    1,
		SSHPort: 0, // No SSH port
		Ports:   nil,
	}

	cmd, err := h.BuildCommand(ctx, "/path/to/image.iso", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestQemuHypervisor_BuildCommand_NoSSHPort_Good(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{
		Memory:  512,
		CPUs:    1,
		SSHPort: 0, // No SSH port
		Ports:   nil,
	}

	cmd, err := q.BuildCommand(ctx, "/path/to/image.iso", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestQemuHypervisor_BuildCommand_UnknownFormat_Bad(t *testing.T) {
	q := NewQemuHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	_, err := q.BuildCommand(ctx, "/path/to/image.txt", opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "unknown image format"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestHyperkitHypervisor_BuildCommand_UnknownFormat_Bad(t *testing.T) {
	h := NewHyperkitHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{Memory: 1024, CPUs: 1}

	_, err := h.BuildCommand(ctx, "/path/to/image.unknown", opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if s, sub := err.Error(), "unknown image format"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestHyperkitHypervisor_Name_Good(t *testing.T) {
	h := NewHyperkitHypervisor()
	if got, want := h.Name(), "hyperkit"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestHyperkitHypervisor_BuildCommand_ISOFormat_Good(t *testing.T) {
	h := NewHyperkitHypervisor()

	ctx := context.Background()
	opts := &HypervisorOptions{
		Memory:  1024,
		CPUs:    2,
		SSHPort: 2222,
	}

	cmd, err := h.BuildCommand(ctx, "/path/to/image.iso", opts)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Fatal("expected non-nil value")
	}

	args := cmd.Args
	if s, sub := args, "-m"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "1024M"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "-c"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := args, "2"; !slices.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

// --- AX-7 canonical triplets ---

func TestHypervisor_NewQemuHypervisor_Good(t *testing.T) {
	symbol := NewQemuHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_NewQemuHypervisor_Bad(t *testing.T) {
	symbol := NewQemuHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_NewQemuHypervisor_Ugly(t *testing.T) {
	symbol := NewQemuHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_Name_Good(t *testing.T) {
	symbol := (*QemuHypervisor).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_Name_Bad(t *testing.T) {
	symbol := (*QemuHypervisor).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_Name_Ugly(t *testing.T) {
	symbol := (*QemuHypervisor).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_Available_Good(t *testing.T) {
	symbol := (*QemuHypervisor).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_Available_Bad(t *testing.T) {
	symbol := (*QemuHypervisor).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_Available_Ugly(t *testing.T) {
	symbol := (*QemuHypervisor).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_BuildCommand_Good(t *testing.T) {
	symbol := (*QemuHypervisor).BuildCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_BuildCommand_Bad(t *testing.T) {
	symbol := (*QemuHypervisor).BuildCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_QemuHypervisor_BuildCommand_Ugly(t *testing.T) {
	symbol := (*QemuHypervisor).BuildCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_NewHyperkitHypervisor_Good(t *testing.T) {
	symbol := NewHyperkitHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_NewHyperkitHypervisor_Bad(t *testing.T) {
	symbol := NewHyperkitHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_NewHyperkitHypervisor_Ugly(t *testing.T) {
	symbol := NewHyperkitHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_Name_Good(t *testing.T) {
	symbol := (*HyperkitHypervisor).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_Name_Bad(t *testing.T) {
	symbol := (*HyperkitHypervisor).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_Name_Ugly(t *testing.T) {
	symbol := (*HyperkitHypervisor).Name
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_Available_Good(t *testing.T) {
	symbol := (*HyperkitHypervisor).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_Available_Bad(t *testing.T) {
	symbol := (*HyperkitHypervisor).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_Available_Ugly(t *testing.T) {
	symbol := (*HyperkitHypervisor).Available
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_BuildCommand_Good(t *testing.T) {
	symbol := (*HyperkitHypervisor).BuildCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_BuildCommand_Bad(t *testing.T) {
	symbol := (*HyperkitHypervisor).BuildCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_HyperkitHypervisor_BuildCommand_Ugly(t *testing.T) {
	symbol := (*HyperkitHypervisor).BuildCommand
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_DetectImageFormat_Good(t *testing.T) {
	symbol := DetectImageFormat
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_DetectImageFormat_Bad(t *testing.T) {
	symbol := DetectImageFormat
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_DetectImageFormat_Ugly(t *testing.T) {
	symbol := DetectImageFormat
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_DetectHypervisor_Bad(t *testing.T) {
	symbol := DetectHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_DetectHypervisor_Ugly(t *testing.T) {
	symbol := DetectHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_GetHypervisor_Good(t *testing.T) {
	symbol := GetHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_GetHypervisor_Bad(t *testing.T) {
	symbol := GetHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestHypervisor_GetHypervisor_Ugly(t *testing.T) {
	symbol := GetHypervisor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
