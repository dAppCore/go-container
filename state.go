package container

import (
	"io/fs"

	core "dappco.re/go/core"
	"dappco.re/go/core/io"

	"dappco.re/go/core/container/internal/coreutil"
)

// State manages persistent container state.
//
// Uses core.Registry for thread-safe container storage.
// The Containers field is kept for JSON serialisation only.
type State struct {
	// Containers is used for JSON serialisation (LoadState/SaveState).
	Containers map[string]*Container `json:"containers"`

	registry *core.Registry[*Container]
	filePath string
}

// DefaultStateDir returns the default directory for state files (~/.core).
//
// Usage:
//
//	dir, err := DefaultStateDir()
func DefaultStateDir() (string, error) {
	home := coreutil.HomeDir()
	if home == "" {
		return "", core.E("DefaultStateDir", "home directory not available", nil)
	}
	return coreutil.JoinPath(home, ".core"), nil
}

// DefaultStatePath returns the default path for the state file.
//
// Usage:
//
//	path, err := DefaultStatePath()
func DefaultStatePath() (string, error) {
	dir, err := DefaultStateDir()
	if err != nil {
		return "", err
	}
	return coreutil.JoinPath(dir, "containers.json"), nil
}

// DefaultLogsDir returns the default directory for container logs.
//
// Usage:
//
//	dir, err := DefaultLogsDir()
func DefaultLogsDir() (string, error) {
	dir, err := DefaultStateDir()
	if err != nil {
		return "", err
	}
	return coreutil.JoinPath(dir, "logs"), nil
}

// NewState creates a new State instance.
//
// Usage:
//
//	state := NewState("/tmp/containers.json")
func NewState(filePath string) *State {
	return &State{
		Containers: make(map[string]*Container),
		registry:   core.NewRegistry[*Container](),
		filePath:   filePath,
	}
}

// LoadState loads the state from the given file path.
// If the file doesn't exist, returns an empty state.
//
// Usage:
//
//	state, err := LoadState("/tmp/containers.json")
func LoadState(filePath string) (*State, error) {
	state := NewState(filePath)

	dataStr, err := io.Local.Read(filePath)
	if err != nil {
		if core.Is(err, fs.ErrNotExist) {
			return state, nil
		}
		return nil, err
	}

	result := core.JSONUnmarshalString(dataStr, state)
	if !result.OK {
		return nil, result.Value.(error)
	}

	// Populate registry from deserialised map
	for id, c := range state.Containers {
		state.registry.Set(id, c)
	}

	return state, nil
}

// SaveState persists the state to the configured file path.
func (s *State) SaveState() error {
	// Sync registry contents to map for JSON serialisation
	s.Containers = make(map[string]*Container)
	s.registry.Each(func(id string, c *Container) {
		s.Containers[id] = c
	})

	// Ensure the directory exists
	dir := core.PathDir(s.filePath)
	if err := io.Local.EnsureDir(dir); err != nil {
		return err
	}

	result := core.JSONMarshal(s)
	if !result.OK {
		return result.Value.(error)
	}

	return io.Local.Write(s.filePath, string(result.Value.([]byte)))
}

// Add adds a container to the state and persists it.
func (s *State) Add(c *Container) error {
	s.registry.Set(c.ID, c)
	return s.SaveState()
}

// Get retrieves a copy of a container by ID.
// Returns a copy to prevent data races when the container is modified.
func (s *State) Get(id string) (*Container, bool) {
	r := s.registry.Get(id)
	if !r.OK {
		return nil, false
	}
	// Return a copy to prevent data races
	c := *r.Value.(*Container)
	return &c, true
}

// Update updates a container in the state and persists it.
func (s *State) Update(c *Container) error {
	s.registry.Set(c.ID, c)
	return s.SaveState()
}

// Remove removes a container from the state and persists it.
func (s *State) Remove(id string) error {
	s.registry.Delete(id)
	return s.SaveState()
}

// All returns copies of all containers in the state.
// Returns copies to prevent data races when containers are modified.
func (s *State) All() []*Container {
	var containers []*Container
	s.registry.Each(func(_ string, c *Container) {
		cp := *c
		containers = append(containers, &cp)
	})
	return containers
}

// FilePath returns the path to the state file.
func (s *State) FilePath() string {
	return s.filePath
}

// LogPath returns the log file path for a given container ID.
//
// Usage:
//
//	path, err := LogPath(containerID)
func LogPath(id string) (string, error) {
	logsDir, err := DefaultLogsDir()
	if err != nil {
		return "", err
	}
	return coreutil.JoinPath(logsDir, core.Concat(id, ".log")), nil
}

// EnsureLogsDir ensures the logs directory exists.
//
// Usage:
//
//	err := EnsureLogsDir()
func EnsureLogsDir() error {
	logsDir, err := DefaultLogsDir()
	if err != nil {
		return err
	}
	return io.Local.EnsureDir(logsDir)
}
