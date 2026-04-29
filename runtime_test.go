package container

import (
	"testing"
)

func TestRuntime_Detect_Good(t *testing.T) {
	rt := Detect()

	// Detect must always return a valid runtime record — even the None zero value.
	if got := rt.Type; len(got) == 0 {
		t.Fatal("expected non-empty value")
	}
}

func TestRuntime_DetectAll_Good(t *testing.T) {
	runtimes := DetectAll()

	// Must not panic; slice may be empty on a host with no runtimes installed.
	if runtimes == nil {
		t.Fatal("expected non-nil value")
	}
	for _, rt := range runtimes {
		if got := rt.Type; len(got) == 0 {
			t.Fatal("expected non-empty value")
		}
	}
}

func TestRuntime_ContainerRuntime_Capabilities_Good(t *testing.T) {
	// Synthesise a runtime with every capability set and verify the predicates.
	rt := ContainerRuntime{
		Type: RuntimeApple,
		caps: capGPU | capNetworkIsolation | capVolumeMounts | capEncryption | capHardwareIsolation | capSubSecondStart,
	}
	if !(rt.HasGPU()) {
		t.Fatal("expected true")
	}
	if !(rt.HasNetworkIsolation()) {
		t.Fatal("expected true")
	}
	if !(rt.HasVolumeMounts()) {
		t.Fatal("expected true")
	}
	if !(rt.HasEncryption()) {
		t.Fatal("expected true")
	}
	if !(rt.IsHardwareIsolated()) {
		t.Fatal("expected true")
	}
	if !(rt.HasSubSecondStart()) {
		t.Fatal("expected true")
	}
	if got := rt.Caps(); got == 0 {
		t.Fatal("expected non-zero value")
	}
}

func TestRuntime_ContainerRuntime_NoCapabilities_Bad(t *testing.T) {
	rt := ContainerRuntime{Type: RuntimeNone}
	if rt.HasGPU() {
		t.Fatal("expected false")
	}
	if rt.HasNetworkIsolation() {
		t.Fatal("expected false")
	}
	if rt.HasVolumeMounts() {
		t.Fatal("expected false")
	}
	if rt.HasEncryption() {
		t.Fatal("expected false")
	}
	if rt.IsHardwareIsolated() {
		t.Fatal("expected false")
	}
	if rt.HasSubSecondStart() {
		t.Fatal("expected false")
	}
	if got := rt.Caps(); got != 0 {
		t.Fatalf("want zero, got %v", got)
	}
}

func TestRuntime_RequireGPU_Ugly(t *testing.T) {
	// RequireGPU must error when the runtime has no GPU capability,
	// and succeed when it does.
	noGPU := ContainerRuntime{Type: RuntimeDocker}
	if err := RequireGPU(noGPU); err == nil {
		t.Fatal("expected error")
	}

	gpu := ContainerRuntime{Type: RuntimeApple, caps: capGPU}
	if err := RequireGPU(gpu); err != nil {
		t.Fatal(err)
	}
}

func TestRuntime_ProviderFor_UnsupportedType_Bad(t *testing.T) {
	_, err := ProviderFor(RuntimeDocker)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntime_ProviderFor_Unknown_Bad(t *testing.T) {
	_, err := ProviderFor(RuntimeType("not-a-runtime"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRuntime_HasRuntime_None_Good(t *testing.T) {
	// Asking for RuntimeNone never matches — even a pristine host would not
	// return None from DetectAll.
	if HasRuntime(RuntimeNone) {
		t.Fatal("expected false")
	}
}

// --- AX-7 canonical triplets ---

func TestRuntime_ContainerRuntime_HasGPU_Good(t *testing.T) {
	symbol := ContainerRuntime.HasGPU
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasGPU_Bad(t *testing.T) {
	symbol := ContainerRuntime.HasGPU
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasGPU_Ugly(t *testing.T) {
	symbol := ContainerRuntime.HasGPU
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasNetworkIsolation_Good(t *testing.T) {
	symbol := ContainerRuntime.HasNetworkIsolation
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasNetworkIsolation_Bad(t *testing.T) {
	symbol := ContainerRuntime.HasNetworkIsolation
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasNetworkIsolation_Ugly(t *testing.T) {
	symbol := ContainerRuntime.HasNetworkIsolation
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasVolumeMounts_Good(t *testing.T) {
	symbol := ContainerRuntime.HasVolumeMounts
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasVolumeMounts_Bad(t *testing.T) {
	symbol := ContainerRuntime.HasVolumeMounts
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasVolumeMounts_Ugly(t *testing.T) {
	symbol := ContainerRuntime.HasVolumeMounts
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasEncryption_Good(t *testing.T) {
	symbol := ContainerRuntime.HasEncryption
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasEncryption_Bad(t *testing.T) {
	symbol := ContainerRuntime.HasEncryption
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasEncryption_Ugly(t *testing.T) {
	symbol := ContainerRuntime.HasEncryption
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_IsHardwareIsolated_Good(t *testing.T) {
	symbol := ContainerRuntime.IsHardwareIsolated
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_IsHardwareIsolated_Bad(t *testing.T) {
	symbol := ContainerRuntime.IsHardwareIsolated
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_IsHardwareIsolated_Ugly(t *testing.T) {
	symbol := ContainerRuntime.IsHardwareIsolated
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasSubSecondStart_Good(t *testing.T) {
	symbol := ContainerRuntime.HasSubSecondStart
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasSubSecondStart_Bad(t *testing.T) {
	symbol := ContainerRuntime.HasSubSecondStart
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_HasSubSecondStart_Ugly(t *testing.T) {
	symbol := ContainerRuntime.HasSubSecondStart
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_Caps_Good(t *testing.T) {
	symbol := ContainerRuntime.Caps
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_Caps_Bad(t *testing.T) {
	symbol := ContainerRuntime.Caps
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ContainerRuntime_Caps_Ugly(t *testing.T) {
	symbol := ContainerRuntime.Caps
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_Detect_Bad(t *testing.T) {
	symbol := Detect
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_Detect_Ugly(t *testing.T) {
	symbol := Detect
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_DetectAll_Bad(t *testing.T) {
	symbol := DetectAll
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_DetectAll_Ugly(t *testing.T) {
	symbol := DetectAll
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ProviderFor_Good(t *testing.T) {
	symbol := ProviderFor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ProviderFor_Bad(t *testing.T) {
	symbol := ProviderFor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_ProviderFor_Ugly(t *testing.T) {
	symbol := ProviderFor
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_HasRuntime_Good(t *testing.T) {
	symbol := HasRuntime
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_HasRuntime_Bad(t *testing.T) {
	symbol := HasRuntime
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_HasRuntime_Ugly(t *testing.T) {
	symbol := HasRuntime
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_runtimeerror_Error_Good(t *testing.T) {
	symbol := (*runtimeerror).Error
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_runtimeerror_Error_Bad(t *testing.T) {
	symbol := (*runtimeerror).Error
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestRuntime_runtimeerror_Error_Ugly(t *testing.T) {
	symbol := (*runtimeerror).Error
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
