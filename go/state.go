package container

import (
	core "dappco.re/go"
	"dappco.re/go/io"

	"dappco.re/go/container/internal/coreutil"
)

var stateMutex = core.New().Lock("container.state").Mutex

// State manages persistent container state.
type State struct {
	// Containers is a map of container ID to Container.
	Containers map[string]*Container `json:"containers"`

	filePath string
}

// DefaultStateDir returns the default directory for state files (~/.core).
//
// Usage:
//
//	dir := core.MustCast[string](DefaultStateDir())
func DefaultStateDir() core.Result { // Value: string
	home := coreutil.HomeDir()
	if home == "" {
		return core.Fail(core.E("DefaultStateDir", "home directory not available", nil))
	}
	return core.Ok(coreutil.JoinPath(home, ".core"))
}

// DefaultStatePath returns the default path for the state file.
//
// Usage:
//
//	path := core.MustCast[string](DefaultStatePath())
func DefaultStatePath() core.Result { // Value: string
	r := DefaultStateDir()
	if !r.OK {
		return r
	}
	return core.Ok(coreutil.JoinPath(core.MustCast[string](r), "containers.json"))
}

// DefaultLogsDir returns the default directory for container logs.
//
// Usage:
//
//	dir := core.MustCast[string](DefaultLogsDir())
func DefaultLogsDir() core.Result { // Value: string
	r := DefaultStateDir()
	if !r.OK {
		return r
	}
	return core.Ok(coreutil.JoinPath(core.MustCast[string](r), "logs"))
}

// NewState creates a new State instance.
//
// Usage:
//
//	state := NewState("/tmp/containers.json")
func NewState(filePath string) *State {
	return &State{
		Containers: make(map[string]*Container),
		filePath:   filePath,
	}
}

// LoadState loads the state from the given file path.
// If the file doesn't exist, returns an empty state.
//
// Usage:
//
//	state := core.MustCast[*State](LoadState("/tmp/containers.json"))
func LoadState(filePath string) core.Result { // Value: *State
	state := NewState(filePath)

	if !io.Local.Exists(filePath) {
		return core.Ok(state)
	}

	dataStr, err := io.Local.Read(filePath)
	if err != nil {
		return core.Fail(core.E("LoadState", "read state file", err))
	}

	result := core.JSONUnmarshalString(dataStr, state)
	if !result.OK {
		return result
	}

	return core.Ok(state)
}

// SaveState persists the state to the configured file path.
func (s *State) SaveState() core.Result { // Value: nil
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	// Ensure the directory exists
	dir := core.PathDir(s.filePath)
	if err := io.Local.EnsureDir(dir); err != nil {
		return core.Fail(core.E("State.SaveState", "ensure state directory", err))
	}

	result := core.JSONMarshal(s)
	if !result.OK {
		return result
	}

	if err := io.Local.Write(s.filePath, string(core.MustCast[[]byte](result))); err != nil {
		return core.Fail(core.E("State.SaveState", "write state file", err))
	}
	return core.Ok(nil)
}

// Add adds a container to the state and persists it.
func (s *State) Add(c *Container) core.Result { // Value: nil
	stateMutex.Lock()
	s.Containers[c.ID] = c
	stateMutex.Unlock()

	return s.SaveState()
}

// Get retrieves a copy of a container by ID.
// Returns a copy to prevent data races when the container is modified.
func (s *State) Get(id string) (*Container, bool) {
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	c, ok := s.Containers[id]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent data races
	copy := *c
	return &copy, true
}

// Update updates a container in the state and persists it.
func (s *State) Update(c *Container) core.Result { // Value: nil
	stateMutex.Lock()
	s.Containers[c.ID] = c
	stateMutex.Unlock()

	return s.SaveState()
}

// Remove removes a container from the state and persists it.
func (s *State) Remove(id string) core.Result { // Value: nil
	stateMutex.Lock()
	delete(s.Containers, id)
	stateMutex.Unlock()

	return s.SaveState()
}

// All returns copies of all containers in the state.
// Returns copies to prevent data races when containers are modified.
func (s *State) All() []*Container {
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	containers := make([]*Container, 0, len(s.Containers))
	for _, c := range s.Containers {
		copy := *c
		containers = append(containers, &copy)
	}
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
//	path := core.MustCast[string](LogPath(containerID))
func LogPath(id string) core.Result { // Value: string
	r := DefaultLogsDir()
	if !r.OK {
		return r
	}
	return core.Ok(coreutil.JoinPath(core.MustCast[string](r), core.Concat(id, ".log")))
}

// EnsureLogsDir ensures the logs directory exists.
//
// Usage:
//
//	if r := EnsureLogsDir(); !r.OK { return r }
func EnsureLogsDir() core.Result { // Value: nil
	r := DefaultLogsDir()
	if !r.OK {
		return r
	}
	if err := io.Local.EnsureDir(core.MustCast[string](r)); err != nil {
		return core.Fail(core.E("EnsureLogsDir", "ensure logs directory", err))
	}
	return core.Ok(nil)
}
