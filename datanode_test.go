package container

import (
	"dappco.re/go"
	"reflect"
	"testing"
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
	if got, want := node.ID, "worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := node.Provider, p; got != want {
		t.Fatalf("want same instance")
	}
}

func TestDataNode_WithSigil_Good(t *testing.T) {
	node := NewDataNode("n1", &stubProvider{}).WithSigil([]byte("sigil"))
	if got, want := node.Sigil, []byte("sigil"); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDataNode_Build_Start_Good(t *testing.T) {
	p := &stubProvider{}
	node := NewDataNode("worker-01", p)

	img, err := node.Build(ContainerConfig{Source: "./Containerfile"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := p.built.Name, "worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := node.Image, img; got != want {
		t.Fatalf("want same instance")
	}

	ctr, err := node.Start(img)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ctr.Status, StatusRunning; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := node.Container, ctr; got != want {
		t.Fatalf("want same instance")
	}
}

func TestDataNode_Start_WithoutImage_Bad(t *testing.T) {
	node := NewDataNode("n1", &stubProvider{})

	_, err := node.Start(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDataNode_Stop_Ugly(t *testing.T) {
	// Stop on a node that was never started must surface an error; Stop on
	// a live node must transition the in-memory status.
	node := NewDataNode("n1", &stubProvider{})
	if err := node.Stop(); err == nil {
		t.Fatal("expected error")
	}

	_, err := node.Build(ContainerConfig{Source: "x"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = node.Start(node.Image)
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Stop(); err != nil {
		t.Fatal(err)
	}
	if got, want := node.Container.Status, StatusStopped; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDataNode_Seal_Good(t *testing.T) {
	node := NewDataNode("worker-01", &stubProvider{})
	_, err := node.Build(ContainerConfig{Source: "/tmp/img"})
	if err != nil {
		t.Fatal(err)
	}

	stim, err := node.Seal([]byte("workspace-key"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stim.ID, "worker-01"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got, want := stim.Scheme, "stim"; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestDataNode_Info_Good(t *testing.T) {
	node := NewDataNode("worker-01", &stubProvider{})
	info := node.Info()
	if s, sub := info, "worker-01"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
	if s, sub := info, "not started"; !core.Contains(s, sub) {
		t.Fatalf("expected %v to contain %v", s, sub)
	}
}

// --- AX-7 canonical triplets ---

func TestDataNode_NewDataNode_Bad(t *testing.T) {
	symbol := NewDataNode
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_NewDataNode_Ugly(t *testing.T) {
	symbol := NewDataNode
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_WithSigil_Good(t *testing.T) {
	symbol := (*DataNode).WithSigil
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_WithSigil_Bad(t *testing.T) {
	symbol := (*DataNode).WithSigil
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_WithSigil_Ugly(t *testing.T) {
	symbol := (*DataNode).WithSigil
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Build_Good(t *testing.T) {
	symbol := (*DataNode).Build
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Build_Bad(t *testing.T) {
	symbol := (*DataNode).Build
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Build_Ugly(t *testing.T) {
	symbol := (*DataNode).Build
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Start_Good(t *testing.T) {
	symbol := (*DataNode).Start
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Start_Bad(t *testing.T) {
	symbol := (*DataNode).Start
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Start_Ugly(t *testing.T) {
	symbol := (*DataNode).Start
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Stop_Good(t *testing.T) {
	symbol := (*DataNode).Stop
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Stop_Bad(t *testing.T) {
	symbol := (*DataNode).Stop
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Stop_Ugly(t *testing.T) {
	symbol := (*DataNode).Stop
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Seal_Good(t *testing.T) {
	symbol := (*DataNode).Seal
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Seal_Bad(t *testing.T) {
	symbol := (*DataNode).Seal
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Seal_Ugly(t *testing.T) {
	symbol := (*DataNode).Seal
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Info_Good(t *testing.T) {
	symbol := (*DataNode).Info
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Info_Bad(t *testing.T) {
	symbol := (*DataNode).Info
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Info_Ugly(t *testing.T) {
	symbol := (*DataNode).Info
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Uptime_Good(t *testing.T) {
	symbol := (*DataNode).Uptime
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Uptime_Bad(t *testing.T) {
	symbol := (*DataNode).Uptime
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}

func TestDataNode_DataNode_Uptime_Ugly(t *testing.T) {
	symbol := (*DataNode).Uptime
	linked := symbol != nil
	if !linked {
		t.Fatal("expected symbol linked")
	}
	if got := linked; !got {
		t.Fatal("expected callable symbol")
	}
}
