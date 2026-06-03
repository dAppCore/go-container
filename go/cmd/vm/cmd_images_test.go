package vm

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/container"
)

func TestCmdImages_requireApple_Good(t *testing.T) {
	r := requireApple()
	if r.OK {
		// Apple runtime present (dev mac): must yield a usable provider.
		if core.MustCast[*container.AppleProvider](r) == nil {
			t.Fatal("OK requireApple returned a nil provider")
		}
		return
	}
	// Absent (CI): the failure must carry an actionable message.
	if r.Error() == "" {
		t.Fatal("failed requireApple must carry a message")
	}
}

func TestCmdImages_pullImage_Bad(t *testing.T) {
	if pullImage("").OK {
		t.Fatal("expected error for empty ref")
	}
}

func TestCmdImages_pushImage_Bad(t *testing.T) {
	if pushImage("").OK {
		t.Fatal("expected error for empty ref")
	}
}

func TestCmdImages_removeImage_Bad(t *testing.T) {
	if removeImage("").OK {
		t.Fatal("expected error for empty ref")
	}
}

func TestCmdImages_formatImages_Good(t *testing.T) {
	out := formatImages([]*container.Image{
		{Name: "docker.io/library/alpine:latest", Digest: "sha256:deadbeefcafef00d0102"},
	})
	if !core.Contains(out, "alpine:latest") || !core.Contains(out, "sha256:deadbeef") {
		t.Fatalf("formatImages missing fields:\n%s", out)
	}
}
