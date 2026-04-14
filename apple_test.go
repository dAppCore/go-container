package container

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApple_IsAppleAvailable_Good(t *testing.T) {
	got := IsAppleAvailable()

	// Function must not panic and must return a bool regardless of platform.
	assert.IsType(t, true, got)
	if runtime.GOOS != "darwin" {
		assert.False(t, got)
	}
}

func TestApple_NewAppleProvider_Good(t *testing.T) {
	p := NewAppleProvider()

	assert.NotNil(t, p)
	assert.Equal(t, "container", p.Binary)
}

func TestApple_Available_Bad(t *testing.T) {
	// A bogus binary name must fail Available().
	p := &AppleProvider{Binary: "nonexistent-apple-container-binary-xyz"}

	assert.False(t, p.Available())
}

func TestApple_Build_MissingSource_Bad(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}

	_, err := p.Build(ContainerConfig{})

	assert.Error(t, err)
}

func TestApple_Run_NilImage_Bad(t *testing.T) {
	p := NewAppleProvider()
	if !p.Available() {
		t.Skip("apple container runtime not available")
	}

	_, err := p.Run(nil)

	assert.Error(t, err)
}

func TestApple_Encrypt_Decrypt_Ugly(t *testing.T) {
	// Encrypt+Decrypt round-trip must preserve the path sans .stim suffix.
	p := NewAppleProvider()
	img := &Image{ID: "test", Path: "/tmp/example.qcow2", Size: 1024}
	key := []byte("workspace-key")

	enc, err := p.Encrypt(img, key)
	assert.NoError(t, err)
	assert.NotNil(t, enc)
	assert.Equal(t, "stim", enc.Scheme)
	assert.Equal(t, "/tmp/example.qcow2.stim", enc.Path)

	out, err := p.Decrypt(enc, key)
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "/tmp/example.qcow2", out.Path)
}

func TestApple_Encrypt_MissingKey_Bad(t *testing.T) {
	p := NewAppleProvider()
	img := &Image{Path: "/tmp/foo"}

	_, err := p.Encrypt(img, nil)

	assert.Error(t, err)
}
