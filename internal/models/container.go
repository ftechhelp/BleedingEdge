package models

import (
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

// GroupType represents the type of container group
type GroupType string

const (
	// GroupTypeCompose represents a Docker Compose project
	GroupTypeCompose GroupType = "compose"
	// GroupTypeStandalone represents a standalone container
	GroupTypeStandalone GroupType = "standalone"
)

// ContainerGroup represents a group of containers (compose project or standalone)
type ContainerGroup struct {
	ID         string          // Unique identifier (container ID or project name)
	Name       string          // Display name
	Type       GroupType       // "compose" or "standalone"
	Containers []ContainerInfo // List of containers in group
	WorkingDir string          // For compose projects
	HasUpdates bool            // True if any container has updates
	AllRunning bool            // True if all containers running
}

// ContainerInfo represents information about a single container
type ContainerInfo struct {
	ID           string            // Container ID
	Name         string            // Container name
	Image        string            // Image name
	ImageDigest  string            // Current image digest
	LatestDigest string            // Latest available image digest
	State        string            // "running", "stopped", "exited"
	HasUpdate    bool              // True if update is available
	Labels       map[string]string // Container labels
}

// ContainerParams represents the parameters needed to recreate a container
type ContainerParams struct {
	Image         string                      // Image name
	Name          string                      // Container name
	Env           []string                    // Environment variables
	Cmd           []string                    // Command
	Entrypoint    []string                    // Entrypoint
	PortBindings  nat.PortMap                 // Port bindings
	Binds         []string                    // Volume binds
	Networks      []string                    // Network names
	RestartPolicy container.RestartPolicy     // Restart policy
	Labels        map[string]string           // Labels
	Resources     container.Resources         // Resource limits
}

// OperationResult represents the result of a container operation
type OperationResult struct {
	Success   bool      // True if operation succeeded
	Message   string    // User-friendly message
	Error     string    // Error message if failed
	Timestamp time.Time // When the operation completed
}
