package container

import (
	"testing"

	"dappco.re/go/core/io"

	"dappco.re/go/container/internal/coreutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataCube_NewDataCube_Good(t *testing.T) {
	cube, err := NewDataCube(io.Local, []byte("workspace-key"), "worker-01")

	require.NoError(t, err)
	assert.Equal(t, "worker-01", cube.ContainerID)
	assert.NotNil(t, cube.Medium)
}

func TestDataCube_NewDataCube_MissingMedium_Bad(t *testing.T) {
	_, err := NewDataCube(nil, []byte("k"), "n1")

	assert.Error(t, err)
}

func TestDataCube_NewDataCube_MissingKey_Bad(t *testing.T) {
	_, err := NewDataCube(io.Local, nil, "n1")

	assert.Error(t, err)
}

func TestDataCube_Write_Read_Good(t *testing.T) {
	// Round-trip a value through an encrypted Cube — the on-disk bytes
	// must differ from the plaintext and Read must recover the original.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	require.NoError(t, err)
	cube, err := NewDataCube(sandbox, []byte("workspace-key"), "worker-01")
	require.NoError(t, err)

	err = cube.Write("app/config.yml", "port: 8080")
	require.NoError(t, err)

	raw, err := sandbox.Read("app/config.yml")
	require.NoError(t, err)
	assert.NotEqual(t, "port: 8080", raw, "on-disk content should be ciphertext")

	out, err := cube.Read("app/config.yml")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080", out)
}

func TestDataCube_Read_WrongKey_Ugly(t *testing.T) {
	// Opening ciphertext with a different workspace key must fail rather
	// than silently returning garbled plaintext.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	require.NoError(t, err)
	writer, err := NewDataCube(sandbox, []byte("key-A"), "worker-01")
	require.NoError(t, err)
	require.NoError(t, writer.Write("secrets/key", "hunter2"))

	reader, err := NewDataCube(sandbox, []byte("key-B"), "worker-01")
	require.NoError(t, err)

	_, err = reader.Read("secrets/key")
	assert.Error(t, err)
}

func TestDataCube_Rename_Good(t *testing.T) {
	// Rename re-seals under the new path key so Read continues to work.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	require.NoError(t, err)
	cube, err := NewDataCube(sandbox, []byte("workspace-key"), "worker-01")
	require.NoError(t, err)

	require.NoError(t, cube.Write("drafts/note.txt", "hello"))
	require.NoError(t, cube.Rename("drafts/note.txt", "archive/note.txt"))
	assert.False(t, sandbox.IsFile(coreutil.JoinPath("drafts", "note.txt")))

	out, err := cube.Read("archive/note.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", out)
}

func TestDataCube_Describe_Good(t *testing.T) {
	cube, err := NewDataCube(io.Local, []byte("k"), "n1")
	require.NoError(t, err)

	assert.Contains(t, cube.Describe(), "n1")
}

func TestDataCube_ImplementsMedium_Good(t *testing.T) {
	// Compile-time check: DataCube must implement io.Medium.
	var m io.Medium = (*DataCube)(nil)

	assert.Nil(t, m)
}
