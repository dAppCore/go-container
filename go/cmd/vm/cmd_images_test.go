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

// TestCmdImages_E2E_ImageRoundTrip_Smoke drives the image handlers end-to-end
// against the LIVE container binary: pull → images → rmi all return OK,
// certifying the requireApple → handler → AppleProvider chain. Opt-in.
func TestCmdImages_E2E_ImageRoundTrip_Smoke(t *testing.T) {
	if core.Env("CORE_APPLE_E2E") == "" {
		t.Skip("set CORE_APPLE_E2E=1 to run the live container CLI smoke")
	}
	if !container.NewAppleProvider().Available() {
		t.Skip("apple container runtime not available")
	}
	const ref = "docker.io/library/alpine:latest"
	if r := pullImage(ref); !r.OK {
		t.Fatalf("pullImage: %v", r.Error())
	}
	if r := listImages(); !r.OK {
		t.Fatalf("listImages: %v", r.Error())
	}
	if r := removeImage(ref); !r.OK {
		t.Fatalf("removeImage: %v", r.Error())
	}
}
