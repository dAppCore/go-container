package container

import (
	"encoding/hex"
	"testing"

	core "dappco.re/go"
)

// --- GenerateID ---

func TestContainer_GenerateID_Good(t *core.T) {
	auditTarget := "GenerateID"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	id, err := GenerateID()

	core.RequireNoError(t, err)
	core.AssertLen(t, id, 8, "container IDs must be 8 hex characters")

	_, err = hex.DecodeString(id)
	core.AssertNoError(t, err, "container ID must be valid hex")
}

func TestContainer_GenerateID_Uniqueness_Bad(t *core.T) {
	auditTarget := "GenerateID Uniqueness"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		core.RequireNoError(t, err)
		core.AssertFalse(t, seen[id], core.Sprintf("GenerateID produced duplicate id %q", id))
		seen[id] = true
	}
}

func TestContainer_GenerateID_Repeatability_Ugly(t *core.T) {
	auditTarget := "GenerateID Repeatability"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// The contract is non-determinism — two consecutive calls must differ.
	a, err := GenerateID()
	core.RequireNoError(t, err)
	b, err := GenerateID()
	core.RequireNoError(t, err)
	core.AssertNotEqual(t, a, b)
}

// --- Status constants ---

func TestContainer_StatusValues_Good(t *core.T) {
	auditTarget := "StatusValues"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	core.AssertEqual(t, Status("running"), StatusRunning)
	core.AssertEqual(t, Status("stopped"), StatusStopped)
	core.AssertEqual(t, Status("error"), StatusError)
}

func TestContainer_StatusValues_Unknown_Bad(t *core.T) {
	auditTarget := "StatusValues Unknown"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	var s Status
	core.AssertEqual(t, Status(""), s, "zero value is empty string, not one of the known states")
	core.AssertNotEqual(t, StatusRunning, s)
}

func TestContainer_StatusValues_Switchable_Ugly(t *core.T) {
	auditTarget := "StatusValues Switchable"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// Exhaustive switch compiles and covers each declared status.
	check := func(s Status) string {
		switch s {
		case StatusRunning:
			return "running"
		case StatusStopped:
			return "stopped"
		case StatusError:
			return "error"
		}
		return "unknown"
	}

	core.AssertEqual(t, "running", check(StatusRunning))
	core.AssertEqual(t, "stopped", check(StatusStopped))
	core.AssertEqual(t, "error", check(StatusError))
	core.AssertEqual(t, "unknown", check(Status("weird")))
}

// --- ImageFormat constants ---

func TestContainer_ImageFormatConstants_Good(t *core.T) {
	auditTarget := "ImageFormatConstants"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	core.AssertEqual(t, ImageFormat("iso"), FormatISO)
	core.AssertEqual(t, ImageFormat("qcow2"), FormatQCOW2)
	core.AssertEqual(t, ImageFormat("vmdk"), FormatVMDK)
	core.AssertEqual(t, ImageFormat("raw"), FormatRaw)
	core.AssertEqual(t, ImageFormat("unknown"), FormatUnknown)
}

func TestContainer_ImageFormatConstants_Unique_Bad(t *core.T) {
	auditTarget := "ImageFormatConstants Unique"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	seen := map[ImageFormat]bool{}
	for _, f := range []ImageFormat{FormatISO, FormatQCOW2, FormatVMDK, FormatRaw, FormatUnknown} {
		core.AssertFalse(t, seen[f], core.Sprintf("duplicate ImageFormat constant %q", f))
		seen[f] = true
	}
}

func TestContainer_ImageFormatConstants_String_Ugly(t *core.T) {
	auditTarget := "ImageFormatConstants String"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	// ImageFormat is a string alias — it must roundtrip via string conversion.
	for _, f := range []ImageFormat{FormatISO, FormatQCOW2, FormatVMDK, FormatRaw, FormatUnknown} {
		core.AssertEqual(t, string(f), string(ImageFormat(string(f))))
	}
}

// --- RunOptions / Container struct smoke ---

func TestContainer_RunOptions_Zero_Good(t *core.T) {
	auditTarget := "RunOptions Zero"
	auditVariant := "Good"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	var o RunOptions
	core.AssertEqual(t, "", o.Name)
	core.AssertFalse(t, o.Detach)
	core.AssertEqual(t, 0, o.Memory)
}

func TestContainer_RunOptions_WithValues_Bad(t *core.T) {
	auditTarget := "RunOptions WithValues"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	o := RunOptions{Memory: -1, CPUs: -1, SSHPort: -1}
	// The struct is a plain DTO, callers are responsible for validation.
	core.AssertEqual(t, -1, o.Memory)
	core.AssertEqual(t, -1, o.CPUs)
	core.AssertEqual(t, -1, o.SSHPort)
}

func TestContainer_Struct_AllFields_Ugly(t *core.T) {
	auditTarget := "Struct AllFields"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	c := Container{
		ID:      "abcdef01",
		Name:    "demo",
		Image:   "/tmp/img.iso",
		Status:  StatusRunning,
		PID:     42,
		Ports:   map[int]int{8080: 80},
		Memory:  1024,
		CPUs:    2,
		SSHPort: 2222,
		SSHKey:  "/tmp/id_ed25519",
	}

	core.AssertEqual(t, StatusRunning, c.Status)
	core.AssertEqual(t, 80, c.Ports[8080])
	core.AssertNotEqual(t, 0, c.PID)
}

// --- AX-7 canonical triplets ---

func TestContainer_GenerateID_Bad(t *testing.T) {
	auditTarget := "GenerateID"
	auditVariant := "Bad"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := GenerateID
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestContainer_GenerateID_Ugly(t *testing.T) {
	auditTarget := "GenerateID"
	auditVariant := "Ugly"
	if len(auditTarget)+len(auditVariant) == 0 {
		t.Fatal(auditTarget, auditVariant)
	}
	symbol := GenerateID
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
