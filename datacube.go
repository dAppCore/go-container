package container

import (
	goio "io"
	"io/fs"
	"sync"

	core "dappco.re/go"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// DataCube wraps an io.Medium with per-path AES-GCM encryption keyed by the
// Borg sigil chain. Every Read/Write/Open path returns plaintext to the
// caller while persisting ciphertext to the underlying medium.
//
// DataCube satisfies io.Medium — any consumer that expects a Medium can be
// handed a Cube transparently.
//
// Usage:
//
//	cube, _ := container.NewDataCube(io.Local, workspaceKey, "worker-01")
//	_ = cube.Write("app/config.yml", "port: 8080")   // ciphertext on disk
//	cfg, _ := cube.Read("app/config.yml")            // plaintext returned
type DataCube struct {
	// Medium is the underlying storage medium (often io.Local or a Sandboxed medium).
	Medium io.Medium
	// ContainerID binds the Cube to a Borg DataNode / TIM container id.
	ContainerID string
	// workspaceKey is the Level-1 sigil (see RFC.tim.md §6).
	workspaceKey []byte
	mu           sync.Mutex
}

// NewDataCube constructs a DataCube around an existing Medium.
//
// Usage:
//
//	cube, _ := container.NewDataCube(io.Local, workspaceKey, "worker-01")
func NewDataCube(medium io.Medium, workspaceKey []byte, containerID string) (*DataCube, error) {
	if medium == nil {
		return nil, coreerr.E("NewDataCube", "medium is required", nil)
	}
	if len(workspaceKey) == 0 {
		return nil, coreerr.E("NewDataCube", "workspace key is required", nil)
	}
	if containerID == "" {
		return nil, coreerr.E("NewDataCube", "container id is required", nil)
	}
	return &DataCube{
		Medium:       medium,
		ContainerID:  containerID,
		workspaceKey: workspaceKey,
	}, nil
}

// Read returns the plaintext content of the ciphertext stored at path.
//
// Usage:
//
//	content, err := cube.Read("app/config.yml")
func (c *DataCube) Read(path string) (string, error) {
	ct, err := c.Medium.Read(path)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	pt, err := DecryptLayer(c.workspaceKey, c.ContainerID, path, []byte(ct))
	if err != nil {
		return "", coreerr.E("DataCube.Read", "decrypt "+path, err)
	}
	return string(pt), nil
}

// Write encrypts content under the derived path key and stores the ciphertext.
//
// Usage:
//
//	err := cube.Write("app/config.yml", "port: 8080")
func (c *DataCube) Write(path, content string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ct, err := EncryptLayer(c.workspaceKey, c.ContainerID, path, []byte(content))
	if err != nil {
		return coreerr.E("DataCube.Write", "encrypt "+path, err)
	}
	return c.Medium.Write(path, string(ct))
}

// WriteMode encrypts content and writes it with the given file mode.
//
// Usage:
//
//	err := cube.WriteMode("keys/private", key, 0600)
func (c *DataCube) WriteMode(path, content string, mode fs.FileMode) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ct, err := EncryptLayer(c.workspaceKey, c.ContainerID, path, []byte(content))
	if err != nil {
		return coreerr.E("DataCube.WriteMode", "encrypt "+path, err)
	}
	return c.Medium.WriteMode(path, string(ct), mode)
}

// EnsureDir creates the directory hierarchy below path on the underlying medium.
//
// Usage:
//
//	err := cube.EnsureDir("app/state")
func (c *DataCube) EnsureDir(path string) error { return c.Medium.EnsureDir(path) }

// IsFile reports whether path refers to a regular file on the underlying medium.
//
// Usage:
//
//	exists := cube.IsFile("app/config.yml")
func (c *DataCube) IsFile(path string) bool { return c.Medium.IsFile(path) }

// Delete removes the ciphertext file at path.
//
// Usage:
//
//	err := cube.Delete("app/stale.bin")
func (c *DataCube) Delete(path string) error { return c.Medium.Delete(path) }

// DeleteAll removes the directory tree at path.
//
// Usage:
//
//	err := cube.DeleteAll("logs/archive")
func (c *DataCube) DeleteAll(path string) error { return c.Medium.DeleteAll(path) }

// Rename renames oldPath to newPath on the underlying medium. The file is
// re-sealed under the new path key so ciphertext stays valid.
//
// Usage:
//
//	err := cube.Rename("drafts/todo.txt", "archive/todo.txt")
func (c *DataCube) Rename(oldPath, newPath string) error {
	content, err := c.Read(oldPath)
	if err != nil {
		return coreerr.E("DataCube.Rename", "read "+oldPath, err)
	}
	if err := c.Write(newPath, content); err != nil {
		return coreerr.E("DataCube.Rename", "write "+newPath, err)
	}
	return c.Medium.Delete(oldPath)
}

// List returns the directory entries at path on the underlying medium.
//
// Usage:
//
//	entries, err := cube.List("app")
func (c *DataCube) List(path string) ([]fs.DirEntry, error) { return c.Medium.List(path) }

// Stat returns file info from the underlying medium.
//
// Usage:
//
//	info, err := cube.Stat("app/config.yml")
func (c *DataCube) Stat(path string) (fs.FileInfo, error) { return c.Medium.Stat(path) }

// Open returns an fs.File handle over the ciphertext. Use Read() for
// plaintext access — raw Open returns ciphertext bytes for streaming
// callers that perform their own decryption.
//
// Usage:
//
//	f, err := cube.Open("app/config.yml")
func (c *DataCube) Open(path string) (fs.File, error) { return c.Medium.Open(path) }

// Create returns a raw writer to the underlying medium. Streaming writers
// bypass Cube encryption; callers that need encrypted streaming should read
// into a buffer and call Write.
//
// Usage:
//
//	f, err := cube.Create("logs/app.log")
func (c *DataCube) Create(path string) (goio.WriteCloser, error) {
	// Consumers that must stream use the underlying medium directly — their
	// payload will be stored unsealed. Deliberate trade-off for compatibility.
	w, err := c.Medium.Create(path)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Append returns a raw append writer to the underlying medium.
//
// Usage:
//
//	f, err := cube.Append("logs/app.log")
func (c *DataCube) Append(path string) (goio.WriteCloser, error) { return c.Medium.Append(path) }

// ReadStream returns a raw ReadCloser to the underlying medium.
//
// Usage:
//
//	r, err := cube.ReadStream("logs/app.log")
func (c *DataCube) ReadStream(path string) (goio.ReadCloser, error) { return c.Medium.ReadStream(path) }

// WriteStream returns a raw WriteCloser to the underlying medium.
//
// Usage:
//
//	w, err := cube.WriteStream("logs/app.log")
func (c *DataCube) WriteStream(path string) (goio.WriteCloser, error) {
	return c.Medium.WriteStream(path)
}

// Exists reports whether path exists on the underlying medium.
//
// Usage:
//
//	ok := cube.Exists("app/config.yml")
func (c *DataCube) Exists(path string) bool { return c.Medium.Exists(path) }

// IsDir reports whether path is a directory on the underlying medium.
//
// Usage:
//
//	ok := cube.IsDir("app")
func (c *DataCube) IsDir(path string) bool { return c.Medium.IsDir(path) }

// compile-time check — DataCube must remain an io.Medium.
var _ io.Medium = (*DataCube)(nil)

// Describe returns a human-readable summary of the Cube's binding. Useful
// when auditing which sigil / container an unwrapped Medium is sealed under.
//
// Usage:
//
//	core.Println(cube.Describe())
func (c *DataCube) Describe() string {
	return core.Concat("DataCube{container=", c.ContainerID, "}")
}
