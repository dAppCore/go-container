package container

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GenerateID ---

func TestContainer_GenerateID_Good(t *testing.T) {
	id, err := GenerateID()

	require.NoError(t, err)
	assert.Len(t, id, 8, "container IDs must be 8 hex characters")

	_, err = hex.DecodeString(id)
	assert.NoError(t, err, "container ID must be valid hex")
}

func TestContainer_GenerateID_Uniqueness_Bad(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		require.NoError(t, err)
		assert.False(t, seen[id], "GenerateID produced duplicate id %q", id)
		seen[id] = true
	}
}

func TestContainer_GenerateID_Repeatability_Ugly(t *testing.T) {
	// The contract is non-determinism — two consecutive calls must differ.
	a, err := GenerateID()
	require.NoError(t, err)
	b, err := GenerateID()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

// --- Status constants ---

func TestContainer_StatusValues_Good(t *testing.T) {
	assert.Equal(t, Status("running"), StatusRunning)
	assert.Equal(t, Status("stopped"), StatusStopped)
	assert.Equal(t, Status("error"), StatusError)
}

func TestContainer_StatusValues_Unknown_Bad(t *testing.T) {
	var s Status
	assert.Equal(t, Status(""), s, "zero value is empty string, not one of the known states")
}

func TestContainer_StatusValues_Switchable_Ugly(t *testing.T) {
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

	assert.Equal(t, "running", check(StatusRunning))
	assert.Equal(t, "stopped", check(StatusStopped))
	assert.Equal(t, "error", check(StatusError))
	assert.Equal(t, "unknown", check(Status("weird")))
}

// --- ImageFormat constants ---

func TestContainer_ImageFormatConstants_Good(t *testing.T) {
	assert.Equal(t, ImageFormat("iso"), FormatISO)
	assert.Equal(t, ImageFormat("qcow2"), FormatQCOW2)
	assert.Equal(t, ImageFormat("vmdk"), FormatVMDK)
	assert.Equal(t, ImageFormat("ami"), FormatAMI)
	assert.Equal(t, ImageFormat("raw"), FormatRaw)
	assert.Equal(t, ImageFormat("unknown"), FormatUnknown)
}

func TestContainer_ImageFormatConstants_Unique_Bad(t *testing.T) {
	seen := map[ImageFormat]bool{}
	for _, f := range []ImageFormat{FormatISO, FormatQCOW2, FormatVMDK, FormatAMI, FormatRaw, FormatUnknown} {
		assert.False(t, seen[f], "duplicate ImageFormat constant %q", f)
		seen[f] = true
	}
}

func TestContainer_ImageFormatConstants_String_Ugly(t *testing.T) {
	// ImageFormat is a string alias — it must roundtrip via string conversion.
	for _, f := range []ImageFormat{FormatISO, FormatQCOW2, FormatVMDK, FormatAMI, FormatRaw, FormatUnknown} {
		assert.Equal(t, string(f), string(ImageFormat(string(f))))
	}
}

// --- RunOptions / Container struct smoke ---

func TestContainer_RunOptions_Zero_Good(t *testing.T) {
	var o RunOptions
	assert.Equal(t, "", o.Name)
	assert.False(t, o.Detach)
	assert.Equal(t, 0, o.Memory)
}

func TestContainer_RunOptions_WithValues_Bad(t *testing.T) {
	o := RunOptions{Memory: -1, CPUs: -1, SSHPort: -1}
	// The struct is a plain DTO, callers are responsible for validation.
	assert.Equal(t, -1, o.Memory)
	assert.Equal(t, -1, o.CPUs)
	assert.Equal(t, -1, o.SSHPort)
}

func TestContainer_Struct_AllFields_Ugly(t *testing.T) {
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

	assert.Equal(t, StatusRunning, c.Status)
	assert.Equal(t, 80, c.Ports[8080])
	assert.NotZero(t, c.PID)
}
