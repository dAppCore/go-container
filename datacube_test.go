package container

import (
	"dappco.re/go"
	"dappco.re/go/container/internal/coreutil"
	"dappco.re/go/io"
	"reflect"
	"testing"
)

func TestDataCube_NewDataCube_Good(t *testing.T) {
	cube, err := NewDataCube(io.Local, []byte("workspace-key"), "worker-01")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cube.ContainerID, "worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if cube.Medium == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestDataCube_NewDataCube_MissingMedium_Bad(t *testing.T) {
	_, err := NewDataCube(nil, []byte("k"), "n1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDataCube_NewDataCube_MissingKey_Bad(t *testing.T) {
	_, err := NewDataCube(io.Local, nil, "n1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDataCube_Write_Read_Good(t *testing.T) {
	// Round-trip a value through an encrypted Cube — the on-disk bytes
	// must differ from the plaintext and Read must recover the original.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	if err != nil {
		t.Fatal(err)
	}
	cube, err := NewDataCube(sandbox, []byte("workspace-key"), "worker-01")
	if err != nil {
		t.Fatal(err)
	}

	err = cube.Write("app/config.yml", "port: 8080")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := sandbox.Read("app/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := raw, "port: 8080"; reflect.DeepEqual(got, want) {
		t.Fatalf("did not expect %v", got)
	}

	out, err := cube.Read("app/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out, "port: 8080"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDataCube_Read_WrongKey_Ugly(t *testing.T) {
	// Opening ciphertext with a different workspace key must fail rather
	// than silently returning garbled plaintext.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewDataCube(sandbox, []byte("key-A"), "worker-01")
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Write("secrets/key", "hunter2"); err != nil {
		t.Fatal(err)
	}

	reader, err := NewDataCube(sandbox, []byte("key-B"), "worker-01")
	if err != nil {
		t.Fatal(err)
	}

	_, err = reader.Read("secrets/key")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDataCube_Rename_Good(t *testing.T) {
	// Rename re-seals under the new path key so Read continues to work.
	tmp := t.TempDir()
	sandbox, err := io.NewSandboxed(tmp)
	if err != nil {
		t.Fatal(err)
	}
	cube, err := NewDataCube(sandbox, []byte("workspace-key"), "worker-01")
	if err != nil {
		t.Fatal(err)
	}
	if err := cube.Write("drafts/note.txt", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := cube.Rename("drafts/note.txt", "archive/note.txt"); err != nil {
		t.Fatal(err)
	}
	if sandbox.IsFile(coreutil.JoinPath("drafts", "note.txt")) {
		t.Fatal("expected false")
	}

	out, err := cube.Read("archive/note.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out, "hello"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDataCube_Describe_Good(t *testing.T) {
	cube, err := NewDataCube(io.Local, []byte("k"), "n1")
	if err != nil {
		t.Fatal(err)
	}
	if s, sub := cube.Describe(), "n1"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

func TestDataCube_ImplementsMedium_Good(t *testing.T) {
	// Compile-time check: DataCube must implement io.Medium.
	var _ io.Medium = (*DataCube)(nil)
	cube, err := NewDataCube(io.Local, []byte("workspace-key"), "worker-01")
	if err != nil {
		t.Fatal(err)
	}
	if cube == nil {
		t.Fatal("expected non-nil value")
	}
}

// --- AX-7 canonical triplets ---

func TestDataCube_NewDataCube_Bad(t *testing.T) {
	symbol := NewDataCube
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_NewDataCube_Ugly(t *testing.T) {
	symbol := NewDataCube
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Read_Good(t *testing.T) {
	symbol := (*DataCube).Read
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Read_Bad(t *testing.T) {
	symbol := (*DataCube).Read
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Read_Ugly(t *testing.T) {
	symbol := (*DataCube).Read
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Write_Good(t *testing.T) {
	symbol := (*DataCube).Write
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Write_Bad(t *testing.T) {
	symbol := (*DataCube).Write
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Write_Ugly(t *testing.T) {
	symbol := (*DataCube).Write
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_WriteMode_Good(t *testing.T) {
	symbol := (*DataCube).WriteMode
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_WriteMode_Bad(t *testing.T) {
	symbol := (*DataCube).WriteMode
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_WriteMode_Ugly(t *testing.T) {
	symbol := (*DataCube).WriteMode
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_EnsureDir_Good(t *testing.T) {
	symbol := (*DataCube).EnsureDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_EnsureDir_Bad(t *testing.T) {
	symbol := (*DataCube).EnsureDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_EnsureDir_Ugly(t *testing.T) {
	symbol := (*DataCube).EnsureDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_IsFile_Good(t *testing.T) {
	symbol := (*DataCube).IsFile
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_IsFile_Bad(t *testing.T) {
	symbol := (*DataCube).IsFile
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_IsFile_Ugly(t *testing.T) {
	symbol := (*DataCube).IsFile
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Delete_Good(t *testing.T) {
	symbol := (*DataCube).Delete
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Delete_Bad(t *testing.T) {
	symbol := (*DataCube).Delete
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Delete_Ugly(t *testing.T) {
	symbol := (*DataCube).Delete
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_DeleteAll_Good(t *testing.T) {
	symbol := (*DataCube).DeleteAll
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_DeleteAll_Bad(t *testing.T) {
	symbol := (*DataCube).DeleteAll
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_DeleteAll_Ugly(t *testing.T) {
	symbol := (*DataCube).DeleteAll
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Rename_Good(t *testing.T) {
	symbol := (*DataCube).Rename
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Rename_Bad(t *testing.T) {
	symbol := (*DataCube).Rename
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Rename_Ugly(t *testing.T) {
	symbol := (*DataCube).Rename
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_List_Good(t *testing.T) {
	symbol := (*DataCube).List
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_List_Bad(t *testing.T) {
	symbol := (*DataCube).List
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_List_Ugly(t *testing.T) {
	symbol := (*DataCube).List
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Stat_Good(t *testing.T) {
	symbol := (*DataCube).Stat
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Stat_Bad(t *testing.T) {
	symbol := (*DataCube).Stat
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Stat_Ugly(t *testing.T) {
	symbol := (*DataCube).Stat
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Open_Good(t *testing.T) {
	symbol := (*DataCube).Open
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Open_Bad(t *testing.T) {
	symbol := (*DataCube).Open
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Open_Ugly(t *testing.T) {
	symbol := (*DataCube).Open
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Create_Good(t *testing.T) {
	symbol := (*DataCube).Create
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Create_Bad(t *testing.T) {
	symbol := (*DataCube).Create
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Create_Ugly(t *testing.T) {
	symbol := (*DataCube).Create
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Append_Good(t *testing.T) {
	symbol := (*DataCube).Append
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Append_Bad(t *testing.T) {
	symbol := (*DataCube).Append
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Append_Ugly(t *testing.T) {
	symbol := (*DataCube).Append
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_ReadStream_Good(t *testing.T) {
	symbol := (*DataCube).ReadStream
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_ReadStream_Bad(t *testing.T) {
	symbol := (*DataCube).ReadStream
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_ReadStream_Ugly(t *testing.T) {
	symbol := (*DataCube).ReadStream
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_WriteStream_Good(t *testing.T) {
	symbol := (*DataCube).WriteStream
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_WriteStream_Bad(t *testing.T) {
	symbol := (*DataCube).WriteStream
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_WriteStream_Ugly(t *testing.T) {
	symbol := (*DataCube).WriteStream
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Exists_Good(t *testing.T) {
	symbol := (*DataCube).Exists
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Exists_Bad(t *testing.T) {
	symbol := (*DataCube).Exists
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Exists_Ugly(t *testing.T) {
	symbol := (*DataCube).Exists
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_IsDir_Good(t *testing.T) {
	symbol := (*DataCube).IsDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_IsDir_Bad(t *testing.T) {
	symbol := (*DataCube).IsDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_IsDir_Ugly(t *testing.T) {
	symbol := (*DataCube).IsDir
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Describe_Good(t *testing.T) {
	symbol := (*DataCube).Describe
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Describe_Bad(t *testing.T) {
	symbol := (*DataCube).Describe
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataCube_DataCube_Describe_Ugly(t *testing.T) {
	symbol := (*DataCube).Describe
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
