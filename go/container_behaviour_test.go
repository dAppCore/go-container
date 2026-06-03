package container

import (
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/io"
)

// TestContainerBehaviour_ParseDigestFromOutput_Good extracts the sha256 digest
// from a build output line.
//
//	digest := parseDigestFromOutput([]byte("built sha256:abc123"))
func TestContainerBehaviour_ParseDigestFromOutput_Good(t *testing.T) {
	out := []byte("Successfully built image sha256:deadbeefcafef00d\nextra trailing line")
	if got := parseDigestFromOutput(out); got != "sha256:deadbeefcafef00d" {
		t.Fatalf("parseDigestFromOutput = %q, want the sha256 token", got)
	}
}

// TestContainerBehaviour_ParseDigestFromOutput_Bad falls back to the raw output
// when no digest token is present.
func TestContainerBehaviour_ParseDigestFromOutput_Bad(t *testing.T) {
	out := []byte("no digest here")
	if got := parseDigestFromOutput(out); got != "no digest here" {
		t.Fatalf("parseDigestFromOutput = %q, want the raw output fallback", got)
	}
}

// TestContainerBehaviour_FirstLine_Good returns the first line of multi-line output.
func TestContainerBehaviour_FirstLine_Good(t *testing.T) {
	if got := firstLine([]byte("first\nsecond\nthird")); got != "first" {
		t.Fatalf("firstLine = %q, want %q", got, "first")
	}
}

// TestContainerBehaviour_FirstLine_Ugly returns the whole string when there is no
// newline.
func TestContainerBehaviour_FirstLine_Ugly(t *testing.T) {
	if got := firstLine([]byte("single")); got != "single" {
		t.Fatalf("firstLine = %q, want %q", got, "single")
	}
}

// TestContainerBehaviour_ParseContainerList_Good decodes the Apple CLI container
// list JSON into Container structs, including port-map parsing.
func TestContainerBehaviour_ParseContainerList_Good(t *testing.T) {
	data := []byte(`[
		{"id":"abc","name":"web","image":"nginx","status":"running","created_at":"2026-01-02T15:04:05Z","ports":{"8080":"80"}},
		{"id":"def","name":"db","image":"pg","status":"stopped","ports":{}}
	]`)
	r := parseContainerList(data)
	if !r.OK {
		t.Fatalf("parseContainerList error: %v", r.Error())
	}
	list := core.MustCast[[]*Container](r)
	if len(list) != 2 {
		t.Fatalf("parseContainerList returned %d containers, want 2", len(list))
	}
	if list[0].ID != "abc" || list[0].Name != "web" {
		t.Fatalf("first container = %+v, want id=abc name=web", list[0])
	}
	if list[0].Ports[8080] != 80 {
		t.Fatalf("first container port map = %v, want 8080->80", list[0].Ports)
	}
	if list[0].StartedAt.IsZero() {
		t.Fatal("first container StartedAt not parsed from created_at")
	}
}

// TestContainerBehaviour_ParseContainerList_Bad errors on malformed JSON.
func TestContainerBehaviour_ParseContainerList_Bad(t *testing.T) {
	if r := parseContainerList([]byte("{not json")); r.OK {
		t.Fatal("parseContainerList of malformed JSON returned an OK result")
	}
}

// TestContainerBehaviour_ParseSingleContainer_Good decodes a single inspect doc
// and skips non-numeric port keys.
func TestContainerBehaviour_ParseSingleContainer_Good(t *testing.T) {
	data := []byte(`{"id":"xyz","name":"svc","image":"img","status":"running","ports":{"notnum":"80","2222":"22"}}`)
	r := parseSingleContainer(data)
	if !r.OK {
		t.Fatalf("parseSingleContainer error: %v", r.Error())
	}
	c := core.MustCast[*Container](r)
	if c.ID != "xyz" {
		t.Fatalf("container ID = %q, want xyz", c.ID)
	}
	if _, present := c.Ports[2222]; !present {
		t.Fatalf("port map %v missing the numeric 2222 entry", c.Ports)
	}
	if len(c.Ports) != 1 {
		t.Fatalf("port map %v should have skipped the non-numeric key", c.Ports)
	}
}

// TestContainerBehaviour_ParseSingleContainer_Bad errors on malformed JSON.
func TestContainerBehaviour_ParseSingleContainer_Bad(t *testing.T) {
	if r := parseSingleContainer([]byte("nope")); r.OK {
		t.Fatal("parseSingleContainer of malformed JSON returned an OK result")
	}
}

// TestContainerBehaviour_ParseImageList_Good decodes the Apple CLI image list.
func TestContainerBehaviour_ParseImageList_Good(t *testing.T) {
	data := []byte(`[{"id":"i1","name":"ghcr.io/foo/bar:latest","digest":"sha256:abc"}]`)
	r := parseImageList(data)
	if !r.OK {
		t.Fatalf("parseImageList error: %v", r.Error())
	}
	imgs := core.MustCast[[]*Image](r)
	if len(imgs) != 1 {
		t.Fatalf("parseImageList returned %d images, want 1", len(imgs))
	}
	if imgs[0].Name != "ghcr.io/foo/bar:latest" || imgs[0].Digest != "sha256:abc" {
		t.Fatalf("image = %+v, want name+digest populated", imgs[0])
	}
	if imgs[0].Format != FormatOCI {
		t.Fatalf("image Format = %q, want %q", imgs[0].Format, FormatOCI)
	}
	if imgs[0].Provider != string(RuntimeApple) {
		t.Fatalf("image Provider = %q, want %q", imgs[0].Provider, RuntimeApple)
	}
}

// TestContainerBehaviour_ParseImageList_Bad errors on malformed JSON.
func TestContainerBehaviour_ParseImageList_Bad(t *testing.T) {
	if r := parseImageList([]byte("garbage")); r.OK {
		t.Fatal("parseImageList of malformed JSON returned an OK result")
	}
}

// TestContainerBehaviour_DataCubeDelegation_Good drives the pass-through Medium
// methods of a DataCube and confirms they mirror the underlying medium.
func TestContainerBehaviour_DataCubeDelegation_Good(t *testing.T) {
	mem := io.NewMemoryMedium()
	cubeRes := NewDataCube(mem, []byte("workspace-key"), "worker-01")
	if !cubeRes.OK {
		t.Fatalf("NewDataCube error: %v", cubeRes.Error())
	}
	cube := core.MustCast[*DataCube](cubeRes)

	if err := cube.EnsureDir("app/state"); err != nil {
		t.Fatalf("EnsureDir error: %v", err)
	}
	if err := cube.Write("app/config.yml", "port: 8080"); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	if !cube.Exists("app/config.yml") {
		t.Fatal("Exists reported false for a written path")
	}
	if !cube.IsFile("app/config.yml") {
		t.Fatal("IsFile reported false for a written file")
	}
	if !cube.IsDir("app") {
		t.Fatal("IsDir reported false for a created directory")
	}

	if _, err := cube.Stat("app/config.yml"); err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if _, err := cube.List("app"); err != nil {
		t.Fatalf("List error: %v", err)
	}

	if got, err := cube.Read("app/config.yml"); err != nil || got != "port: 8080" {
		t.Fatalf("Read = (%q, %v), want plaintext round-trip", got, err)
	}
}

// TestContainerBehaviour_DataCubeRename_Good re-seals a Cube file under a new path.
func TestContainerBehaviour_DataCubeRename_Good(t *testing.T) {
	mem := io.NewMemoryMedium()
	cubeRes := NewDataCube(mem, []byte("workspace-key"), "worker-01")
	if !cubeRes.OK {
		t.Fatalf("NewDataCube error: %v", cubeRes.Error())
	}
	cube := core.MustCast[*DataCube](cubeRes)
	if err := cube.Write("drafts/todo.txt", "buy milk"); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if err := cube.Rename("drafts/todo.txt", "archive/todo.txt"); err != nil {
		t.Fatalf("Rename error: %v", err)
	}
	if cube.Exists("drafts/todo.txt") {
		t.Fatal("source path still exists after Rename")
	}
	if got, err := cube.Read("archive/todo.txt"); err != nil || got != "buy milk" {
		t.Fatalf("renamed Read = (%q, %v), want re-sealed plaintext", got, err)
	}
}

// TestContainerBehaviour_DataCubeDelete_Good removes a Cube file and tree.
func TestContainerBehaviour_DataCubeDelete_Good(t *testing.T) {
	mem := io.NewMemoryMedium()
	cubeRes := NewDataCube(mem, []byte("workspace-key"), "worker-01")
	if !cubeRes.OK {
		t.Fatalf("NewDataCube error: %v", cubeRes.Error())
	}
	cube := core.MustCast[*DataCube](cubeRes)
	if err := cube.Write("logs/app.log", "line"); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if err := cube.Delete("logs/app.log"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if cube.Exists("logs/app.log") {
		t.Fatal("file still exists after Delete")
	}
	if err := cube.DeleteAll("logs"); err != nil {
		t.Fatalf("DeleteAll error: %v", err)
	}
}

// TestContainerBehaviour_DataCubeStreaming_Good drives the raw streaming methods
// (Create/WriteMode/Open/ReadStream/Append/WriteStream) which deliberately bypass
// Cube encryption and pass straight through to the underlying medium.
func TestContainerBehaviour_DataCubeStreaming_Good(t *testing.T) {
	mem := io.NewMemoryMedium()
	cubeRes := NewDataCube(mem, []byte("workspace-key"), "worker-01")
	if !cubeRes.OK {
		t.Fatalf("NewDataCube error: %v", cubeRes.Error())
	}
	cube := core.MustCast[*DataCube](cubeRes)

	if err := cube.WriteMode("keys/private", "secret", 0o600); err != nil {
		t.Fatalf("WriteMode error: %v", err)
	}

	w, err := cube.Create("logs/app.log")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if _, err := w.Write([]byte("streamed")); err != nil {
		t.Fatalf("stream Write error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("stream Close error: %v", err)
	}

	f, err := cube.Open("logs/app.log")
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	_ = f.Close()

	r, err := cube.ReadStream("logs/app.log")
	if err != nil {
		t.Fatalf("ReadStream error: %v", err)
	}
	_ = r.Close()

	a, err := cube.Append("logs/app.log")
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	_ = a.Close()

	ws, err := cube.WriteStream("logs/app2.log")
	if err != nil {
		t.Fatalf("WriteStream error: %v", err)
	}
	_ = ws.Close()
}

// TestContainerBehaviour_DataNodeUptime_Good reports a positive uptime once the
// node's container has a start time, and zero before it starts.
func TestContainerBehaviour_DataNodeUptime_Good(t *testing.T) {
	node := &DataNode{Container: &Container{StartedAt: time.Now().Add(-2 * time.Second)}}
	if up := node.Uptime(); up <= 0 {
		t.Fatalf("Uptime = %s, want a positive duration", up)
	}
}

// TestContainerBehaviour_DataNodeUptime_Bad reports zero when no container or
// start time is set.
func TestContainerBehaviour_DataNodeUptime_Bad(t *testing.T) {
	if up := (&DataNode{}).Uptime(); up != 0 {
		t.Fatalf("Uptime with no container = %s, want 0", up)
	}
	if up := (&DataNode{Container: &Container{}}).Uptime(); up != 0 {
		t.Fatalf("Uptime with zero StartedAt = %s, want 0", up)
	}
}
