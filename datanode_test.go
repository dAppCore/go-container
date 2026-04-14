package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProvider is a minimal Provider used to exercise DataNode without
// touching the operating system.
type stubProvider struct {
	buildErr error
	runErr   error
	built    *Image
	ran      *Container
}

func (s *stubProvider) Build(config ContainerConfig) (*Image, error) {
	if s.buildErr != nil {
		return nil, s.buildErr
	}
	img := &Image{ID: "img-1", Name: config.Name, Path: "/tmp/img"}
	s.built = img
	return img, nil
}

func (s *stubProvider) Run(image *Image, opts ...RunOption) (*Container, error) {
	if s.runErr != nil {
		return nil, s.runErr
	}
	ctr := &Container{ID: "ctr-1", Image: image.Path, Status: StatusRunning}
	s.ran = ctr
	return ctr, nil
}

func (s *stubProvider) Encrypt(image *Image, key []byte) (*EncryptedImage, error) {
	return &EncryptedImage{ID: "enc-1", Path: image.Path + ".stim", Scheme: "stim"}, nil
}

func (s *stubProvider) Decrypt(encrypted *EncryptedImage, key []byte) (*Image, error) {
	return &Image{ID: "img-out", Path: encrypted.Path}, nil
}

func TestDataNode_NewDataNode_Good(t *testing.T) {
	p := &stubProvider{}
	node := NewDataNode("worker-01", p)

	assert.Equal(t, "worker-01", node.ID)
	assert.Same(t, p, node.Provider)
}

func TestDataNode_WithSigil_Good(t *testing.T) {
	node := NewDataNode("n1", &stubProvider{}).WithSigil([]byte("sigil"))

	assert.Equal(t, []byte("sigil"), node.Sigil)
}

func TestDataNode_Build_Start_Good(t *testing.T) {
	p := &stubProvider{}
	node := NewDataNode("worker-01", p)

	img, err := node.Build(ContainerConfig{Source: "./Containerfile"})
	require.NoError(t, err)
	assert.Equal(t, "worker-01", p.built.Name, "node id becomes image name when name empty")
	assert.Same(t, img, node.Image)

	ctr, err := node.Start(img)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, ctr.Status)
	assert.Same(t, ctr, node.Container)
}

func TestDataNode_Start_WithoutImage_Bad(t *testing.T) {
	node := NewDataNode("n1", &stubProvider{})

	_, err := node.Start(nil)

	assert.Error(t, err)
}

func TestDataNode_Stop_Ugly(t *testing.T) {
	// Stop on a node that was never started must surface an error; Stop on
	// a live node must transition the in-memory status.
	node := NewDataNode("n1", &stubProvider{})
	assert.Error(t, node.Stop())

	_, err := node.Build(ContainerConfig{Source: "x"})
	require.NoError(t, err)
	_, err = node.Start(node.Image)
	require.NoError(t, err)

	require.NoError(t, node.Stop())
	assert.Equal(t, StatusStopped, node.Container.Status)
}

func TestDataNode_Seal_Good(t *testing.T) {
	node := NewDataNode("worker-01", &stubProvider{})
	_, err := node.Build(ContainerConfig{Source: "/tmp/img"})
	require.NoError(t, err)

	stim, err := node.Seal([]byte("workspace-key"))
	require.NoError(t, err)
	assert.Equal(t, "worker-01", stim.ID)
	assert.Equal(t, "stim", stim.Scheme)
}

func TestDataNode_Info_Good(t *testing.T) {
	node := NewDataNode("worker-01", &stubProvider{})
	info := node.Info()

	assert.Contains(t, info, "worker-01")
	assert.Contains(t, info, "not started")
}
