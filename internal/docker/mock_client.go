package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// MockClient is a mock implementation of DockerClient for testing
type MockClient struct {
	ListContainersFunc    func(ctx context.Context) ([]types.Container, error)
	InspectContainerFunc  func(ctx context.Context, id string) (types.ContainerJSON, error)
	PullImageFunc         func(ctx context.Context, imageName string) error
	GetImageDigestFunc    func(ctx context.Context, imageName string) (string, error)
	StartContainerFunc    func(ctx context.Context, id string) error
	StopContainerFunc     func(ctx context.Context, id string) error
	RestartContainerFunc  func(ctx context.Context, id string) error
	RemoveContainerFunc   func(ctx context.Context, id string) error
	CreateContainerFunc   func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error)
	ExecuteCommandFunc    func(ctx context.Context, workDir string, command string, args []string) error
}

// ListContainers mocks listing containers
func (m *MockClient) ListContainers(ctx context.Context) ([]types.Container, error) {
	if m.ListContainersFunc != nil {
		return m.ListContainersFunc(ctx)
	}
	return []types.Container{}, nil
}

// InspectContainer mocks inspecting a container
func (m *MockClient) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	if m.InspectContainerFunc != nil {
		return m.InspectContainerFunc(ctx, id)
	}
	return types.ContainerJSON{}, fmt.Errorf("container not found: %s", id)
}

// PullImage mocks pulling an image
func (m *MockClient) PullImage(ctx context.Context, imageName string) error {
	if m.PullImageFunc != nil {
		return m.PullImageFunc(ctx, imageName)
	}
	return nil
}

// GetImageDigest mocks getting an image digest
func (m *MockClient) GetImageDigest(ctx context.Context, imageName string) (string, error) {
	if m.GetImageDigestFunc != nil {
		return m.GetImageDigestFunc(ctx, imageName)
	}
	return "sha256:mock-digest", nil
}

// StartContainer mocks starting a container
func (m *MockClient) StartContainer(ctx context.Context, id string) error {
	if m.StartContainerFunc != nil {
		return m.StartContainerFunc(ctx, id)
	}
	return nil
}

// StopContainer mocks stopping a container
func (m *MockClient) StopContainer(ctx context.Context, id string) error {
	if m.StopContainerFunc != nil {
		return m.StopContainerFunc(ctx, id)
	}
	return nil
}

// RestartContainer mocks restarting a container
func (m *MockClient) RestartContainer(ctx context.Context, id string) error {
	if m.RestartContainerFunc != nil {
		return m.RestartContainerFunc(ctx, id)
	}
	return nil
}

// RemoveContainer mocks removing a container
func (m *MockClient) RemoveContainer(ctx context.Context, id string) error {
	if m.RemoveContainerFunc != nil {
		return m.RemoveContainerFunc(ctx, id)
	}
	return nil
}

// CreateContainer mocks creating a container
func (m *MockClient) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error) {
	if m.CreateContainerFunc != nil {
		return m.CreateContainerFunc(ctx, config, hostConfig, name)
	}
	return "mock-container-id", nil
}

// ExecuteCommand mocks executing a command
func (m *MockClient) ExecuteCommand(ctx context.Context, workDir string, command string, args []string) error {
	if m.ExecuteCommandFunc != nil {
		return m.ExecuteCommandFunc(ctx, workDir, command, args)
	}
	return nil
}
