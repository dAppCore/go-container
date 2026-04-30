package container

import (
	"time"

	core "dappco.re/go"
	coreerr "dappco.re/go/log"
)

// DataNode wraps a TIM container with a Borg identity and lifecycle. Each
// Borg.DataNode IS a container — identity comes from the sigil, storage is
// STIM-sealed, and lifecycle tracks the underlying Provider.Container.
//
// See RFC.tim.md §7 for the wrapping contract.
//
// Usage:
//
//	node := container.NewDataNode("worker-01", container.NewAppleProvider())
//	img, _ := node.Build(container.ContainerConfig{Source: "./Containerfile"})
//	_, err := node.Start(img)
type DataNode struct {
	// ID is the node identifier (typically matches the TIM bundle ID).
	ID string
	// Provider is the container backend used to build and run the node.
	Provider Provider
	// Sigil holds opaque Borg identity material. The container package does
	// not inspect it — downstream packages (borg) read/write this field.
	Sigil []byte
	// Container is the running Container record once Start has been called.
	Container *Container
	// Image is the built Image record associated with the node.
	Image *Image
}

// NewDataNode constructs a DataNode bound to a Provider. Sigil material is
// optional at construction and may be attached later.
//
// Usage:
//
//	node := container.NewDataNode("worker-01", container.NewAppleProvider())
func NewDataNode(id string, provider Provider) *DataNode {
	return &DataNode{
		ID:       id,
		Provider: provider,
	}
}

// WithSigil attaches Borg sigil identity material to the node. Returns the
// node for chainable construction.
//
// Usage:
//
//	node := container.NewDataNode("n1", p).WithSigil(sigilBytes)
func (n *DataNode) WithSigil(sigil []byte) *DataNode {
	n.Sigil = sigil
	return n
}

// Build compiles an Image via the node's Provider.
//
// Usage:
//
//	img, err := node.Build(container.ContainerConfig{Source: "./Containerfile"})
func (n *DataNode) Build(config ContainerConfig) (*Image, error) {
	if n.Provider == nil {
		return nil, coreerr.E("DataNode.Build", "provider is required", nil)
	}
	if config.Name == "" {
		config.Name = n.ID
	}
	img, err := n.Provider.Build(config)
	if err != nil {
		return nil, err
	}
	n.Image = img
	return img, nil
}

// Start boots the image via the Provider and records the Container handle.
// The image argument is optional — if nil, the last Build result is used.
//
// Usage:
//
//	_, err := node.Start(img, container.WithMemory(4096))
func (n *DataNode) Start(img *Image, opts ...RunOption) (*Container, error) {
	if n.Provider == nil {
		return nil, coreerr.E("DataNode.Start", "provider is required", nil)
	}
	if img == nil {
		img = n.Image
	}
	if img == nil {
		return nil, coreerr.E("DataNode.Start", "image is required — call Build first", nil)
	}
	ctr, err := n.Provider.Run(img, opts...)
	if err != nil {
		return nil, err
	}
	n.Container = ctr
	return ctr, nil
}

// Stop marks the node as stopped. The Provider's own Stop surface (when
// present) is invoked when available; otherwise the in-memory record is
// updated so callers can observe the transition.
//
// Usage:
//
//	err := node.Stop()
func (n *DataNode) Stop() error {
	if n.Container == nil {
		return coreerr.E("DataNode.Stop", "container has not been started", nil)
	}
	n.Container.Status = StatusStopped
	return nil
}

// Seal produces a STIM record for the node's bundle using the supplied
// workspace key. Layer file payloads are left untouched here — use
// EncryptTIMOnMedium with a concrete io.Medium for full-fidelity sealing.
//
// Usage:
//
//	stim, err := node.Seal(workspaceKey)
func (n *DataNode) Seal(workspaceKey []byte) (*STIMBundle, error) {
	if n.Image == nil {
		return nil, coreerr.E("DataNode.Seal", "image is required", nil)
	}
	bundle := NewTIMBundle(n.ID, n.Image.Path)
	return EncryptTIM(bundle, workspaceKey)
}

// Info returns a short description of the node state. Suitable for log lines.
//
// Usage:
//
//	core.Println(node.Info())
func (n *DataNode) Info() string {
	status := "not started"
	if n.Container != nil {
		status = string(n.Container.Status)
	}
	return core.Concat("DataNode{id=", n.ID, ", status=", status, "}")
}

// Uptime reports how long the node has been running since Start.
//
// Usage:
//
//	d := node.Uptime()
func (n *DataNode) Uptime() time.Duration {
	if n.Container == nil || n.Container.StartedAt.IsZero() {
		return 0
	}
	return time.Since(n.Container.StartedAt)
}
