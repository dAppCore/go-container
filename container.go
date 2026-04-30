// Package container provides a runtime for managing LinuxKit containers.
// It supports running LinuxKit images (ISO, qcow2, vmdk, raw) using
// available hypervisors (QEMU on Linux, Hyperkit on macOS).
package container

import (
	"context"
	"crypto/rand" // Note: crypto primitive - no core equivalent yet.
	// Note: AX-6 — encoding/hex is structural here because container IDs are exposed as stable hex strings and no core primitive exists for this conversion yet.
	"encoding/hex"
	"time"
)

// Container represents a running LinuxKit container/VM instance.
type Container struct {
	// ID is a unique identifier for the container (8 character hex string).
	ID string `json:"id"`
	// Name is the optional human-readable name for the container.
	Name string `json:"name,omitempty"`
	// Image is the path to the LinuxKit image being run.
	Image string `json:"image"`
	// Status represents the current state of the container.
	Status Status `json:"status"`
	// PID is the process ID of the hypervisor running this container.
	PID int `json:"pid"`
	// StartedAt is when the container was started.
	StartedAt time.Time `json:"started_at"`
	// Ports maps host ports to container ports.
	Ports map[int]int `json:"ports,omitempty"`
	// Memory is the amount of memory allocated in MB.
	Memory int `json:"memory,omitempty"`
	// CPUs is the number of CPUs allocated.
	CPUs int `json:"cpus,omitempty"`
	// SSHPort is the host port used for SSH access into the VM.
	SSHPort int `json:"ssh_port,omitempty"`
	// SSHKey is the optional private key path used by Exec.
	SSHKey string `json:"ssh_key,omitempty"`
}

// Status represents the state of a container.
type Status string

const (
	// StatusRunning indicates the container is running.
	StatusRunning Status = "running"
	// StatusStopped indicates the container has stopped.
	StatusStopped Status = "stopped"
	// StatusError indicates the container encountered an error.
	StatusError Status = "error"
)

// RunOptions configures how a container should be run.
type RunOptions struct {
	// Name is an optional human-readable name for the container.
	Name string
	// Detach runs the container in the background.
	Detach bool
	// Memory is the amount of memory to allocate in MB (default: 1024).
	Memory int
	// CPUs is the number of CPUs to allocate (default: 1).
	CPUs int
	// Ports maps host ports to container ports.
	Ports map[int]int
	// Volumes maps host paths to container paths.
	Volumes map[string]string
	// SSHPort is the port to use for SSH access (default: 2222).
	SSHPort int
	// SSHKey is the path to the SSH private key for exec commands.
	SSHKey string
	// GPU requests GPU passthrough into the container. Providers that do not
	// support GPU passthrough return an error when this is set.
	GPU bool
}

// ReadCloser is the stream contract returned by Manager.Logs.
type ReadCloser interface {
	Read(p []byte) (n int, err error)
	Close() error
}

// Manager defines the interface for container lifecycle management.
type Manager interface {
	// Run starts a new container from the given image.
	Run(ctx context.Context, image string, opts RunOptions) (*Container, error)
	// Stop stops a running container by ID.
	Stop(ctx context.Context, id string) error
	// List returns all known containers.
	List(ctx context.Context) ([]*Container, error)
	// Logs returns a reader for the container's log output.
	// If follow is true, the reader will continue to stream new log entries.
	Logs(ctx context.Context, id string, follow bool) (ReadCloser, error)
	// Exec executes a command inside the container via SSH.
	Exec(ctx context.Context, id string, cmd []string) error
}

// GenerateID creates a new unique container ID (8 hex characters).
//
// Usage:
//
//	id, err := GenerateID()
func GenerateID() (string, error) {
	bytes, err := randomBytes(4)
	if err != nil {
		return "", err
	}
	return hexID(bytes), nil
}

func randomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	// Note: crypto primitive - no core equivalent yet.
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func hexID(bytes []byte) string {
	// Note: encoding primitive - no core equivalent yet.
	return hex.EncodeToString(bytes)
}

// ImageFormat represents the format of a LinuxKit image.
type ImageFormat string

const (
	// FormatISO is an ISO image format.
	FormatISO ImageFormat = "iso"
	// FormatQCOW2 is a QEMU Copy-On-Write image format.
	FormatQCOW2 ImageFormat = "qcow2"
	// FormatVMDK is a VMware disk image format.
	FormatVMDK ImageFormat = "vmdk"
	// FormatRaw is a raw disk image format.
	FormatRaw ImageFormat = "raw"
	// FormatUnknown indicates an unknown image format.
	FormatUnknown ImageFormat = "unknown"
)
