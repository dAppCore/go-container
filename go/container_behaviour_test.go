package container

import (
	"testing"

	core "dappco.re/go"
)

// TestContainerBehaviour_ParseDigestFromOutput_Good extracts the sha256 digest
// from a build output line.
//
//	digest := parseDigestFromOutput([]byte("built sha256:abc123"))
func TestContainerBehaviour_parseDigestFromOutput_Good(t *testing.T) {
	out := []byte("Successfully built image sha256:deadbeefcafef00d\nextra trailing line")
	if got := parseDigestFromOutput(out); got != "sha256:deadbeefcafef00d" {
		t.Fatalf("parseDigestFromOutput = %q, want the sha256 token", got)
	}
}

// TestContainerBehaviour_ParseDigestFromOutput_Bad falls back to the raw output
// when no digest token is present.
func TestContainerBehaviour_parseDigestFromOutput_Bad(t *testing.T) {
	out := []byte("no digest here")
	if got := parseDigestFromOutput(out); got != "no digest here" {
		t.Fatalf("parseDigestFromOutput = %q, want the raw output fallback", got)
	}
}

// TestContainerBehaviour_FirstLine_Good returns the first line of multi-line output.
func TestContainerBehaviour_firstLine_Good(t *testing.T) {
	if got := firstLine([]byte("first\nsecond\nthird")); got != "first" {
		t.Fatalf("firstLine = %q, want %q", got, "first")
	}
}

// TestContainerBehaviour_FirstLine_Ugly returns the whole string when there is no
// newline.
func TestContainerBehaviour_firstLine_Ugly(t *testing.T) {
	if got := firstLine([]byte("single")); got != "single" {
		t.Fatalf("firstLine = %q, want %q", got, "single")
	}
}

// TestContainerBehaviour_ParseContainerList_Good decodes the Apple CLI container
// list JSON into Container structs, including port-map parsing.
func TestContainerBehaviour_parseContainerList_Good(t *testing.T) {
	data := []byte(`[
		{"status":"running","startedDate":802181959.4,"configuration":{"id":"abc","image":{"reference":"docker.io/library/nginx:latest"},"resources":{"cpus":2,"memoryInBytes":536870912},"publishedPorts":[{"hostPort":8080,"containerPort":80}]}},
		{"status":"stopped","configuration":{"id":"def","image":{"reference":"docker.io/library/postgres:16"},"resources":{"cpus":1,"memoryInBytes":268435456},"publishedPorts":[]}}
	]`)
	r := parseContainerList(data)
	if !r.OK {
		t.Fatalf("parseContainerList error: %v", r.Error())
	}
	list := core.MustCast[[]*Container](r)
	if len(list) != 2 {
		t.Fatalf("parseContainerList returned %d containers, want 2", len(list))
	}
	if list[0].ID != "abc" || list[0].Name != "abc" {
		t.Fatalf("first container = %+v, want id=abc name=abc", list[0])
	}
	if list[0].Image != "docker.io/library/nginx:latest" {
		t.Fatalf("first container image = %q, want docker.io/library/nginx:latest", list[0].Image)
	}
	if list[0].Ports[8080] != 80 {
		t.Fatalf("first container port map = %v, want 8080->80", list[0].Ports)
	}
	if list[0].StartedAt.IsZero() {
		t.Fatal("first container StartedAt not parsed from startedDate")
	}
	if list[1].Status != StatusStopped {
		t.Fatalf("second container status = %q, want stopped", list[1].Status)
	}
}

// TestContainerBehaviour_ParseContainerList_Bad errors on malformed JSON.
func TestContainerBehaviour_parseContainerList_Bad(t *testing.T) {
	if r := parseContainerList([]byte("{not json")); r.OK {
		t.Fatal("parseContainerList of malformed JSON returned an OK result")
	}
}

// TestContainerBehaviour_ParseSingleContainer_Good decodes an inspect doc
// (a JSON array of one element) into a Container.
func TestContainerBehaviour_ParseSingleContainer_Good(t *testing.T) {
	data := []byte(`[{"status":"running","configuration":{"id":"xyz","image":{"reference":"img:latest"},"publishedPorts":[{"hostPort":2222,"containerPort":22}]}}]`)
	r := parseSingleContainer(data)
	if !r.OK {
		t.Fatalf("parseSingleContainer error: %v", r.Error())
	}
	c := core.MustCast[*Container](r)
	if c.ID != "xyz" {
		t.Fatalf("container ID = %q, want xyz", c.ID)
	}
	if c.Ports[2222] != 22 {
		t.Fatalf("port map %v missing the 2222->22 entry", c.Ports)
	}
	if len(c.Ports) != 1 {
		t.Fatalf("port map %v should have exactly one entry", c.Ports)
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
	data := []byte(`[{"reference":"ghcr.io/foo/bar:latest","fullSize":"12 MB","descriptor":{"digest":"sha256:abc"}}]`)
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
