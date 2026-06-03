package container

func ExampleIsAppleAvailable() {
	// IsAppleAvailable usage is covered by the corresponding triplet tests.
}

func ExampleNewAppleProvider() {
	// NewAppleProvider usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Available() {
	// AppleProvider_Available usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Build() {
	// AppleProvider_Build usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Run() {
	// AppleProvider_Run usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Tracked() {
	// AppleProvider_Tracked usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Wait() {
	// AppleProvider_Wait usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Encrypt() {
	// AppleProvider_Encrypt usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Decrypt() {
	// AppleProvider_Decrypt usage is covered by the corresponding triplet tests.
}

func ExampleAppleProvider_Stop() {
	p := NewAppleProvider()
	// Stop halts a running container by id; the Result reports success.
	_ = p.Stop("my-container")
}

func ExampleAppleProvider_Kill() {
	p := NewAppleProvider()
	// Kill sends SIGKILL to a running container by id.
	_ = p.Kill("my-container")
}

func ExampleAppleProvider_Remove() {
	p := NewAppleProvider()
	// Remove deletes a container by id and drops it from the tracked map.
	_ = p.Remove("my-container")
}

func ExampleAppleProvider_Logs() {
	p := NewAppleProvider()
	// Logs returns the tail of a container's combined output.
	_ = p.Logs("my-container", 100)
}

func ExampleAppleProvider_Exec() {
	p := NewAppleProvider()
	// Exec runs a command inside a container and returns its output.
	_ = p.Exec("my-container", "echo", "hello")
}

func ExampleAppleProvider_List() {
	p := NewAppleProvider()
	// List returns every container known to the Apple container CLI.
	_ = p.List()
}

func ExampleAppleProvider_Inspect() {
	p := NewAppleProvider()
	// Inspect returns detailed information about a single container.
	_ = p.Inspect("my-container")
}

func ExampleAppleProvider_Pull() {
	p := NewAppleProvider()
	// Pull fetches an image from a registry by reference.
	_ = p.Pull("docker.io/library/alpine:latest")
}

func ExampleAppleProvider_Push() {
	p := NewAppleProvider()
	// Push uploads a locally-tagged image to a registry.
	_ = p.Push(&Image{Path: "alpine:latest"}, "ghcr.io/acme/app:v1")
}

func ExampleAppleProvider_RemoveImage() {
	p := NewAppleProvider()
	// RemoveImage deletes a local image by id or reference.
	_ = p.RemoveImage("alpine:latest")
}

func ExampleAppleProvider_ListImages() {
	p := NewAppleProvider()
	// ListImages returns every image known to the Apple container CLI.
	_ = p.ListImages()
}

func ExampleAppleProvider_SystemStart() {
	p := NewAppleProvider()
	// SystemStart brings up the apiserver, installing the default kernel.
	_ = p.SystemStart(true)
}

func ExampleAppleProvider_SystemStop() {
	p := NewAppleProvider()
	// SystemStop stops all container services.
	_ = p.SystemStop()
}

func ExampleAppleProvider_SystemStatus() {
	p := NewAppleProvider()
	// SystemStatus returns the raw `container system status` output.
	_ = p.SystemStatus()
}
