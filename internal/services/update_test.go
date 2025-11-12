package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

func TestExtractContainerParams(t *testing.T) {
	tests := []struct {
		name          string
		containerJSON types.ContainerJSON
		expectError   bool
	}{
		{
			name: "basic container configuration",
			containerJSON: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					Name:       "/test-container",
					HostConfig: &container.HostConfig{
						Binds: []string{"/host:/container"},
					},
				},
				Config: &container.Config{
					Image: "nginx:latest",
					Env:   []string{"ENV_VAR=value"},
				},
			},
			expectError: false,
		},
		{
			name: "container with port bindings",
			containerJSON: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					Name: "/web-server",
					HostConfig: &container.HostConfig{
						PortBindings: nat.PortMap{
							"80/tcp": []nat.PortBinding{
								{HostIP: "0.0.0.0", HostPort: "8080"},
							},
						},
					},
				},
				Config: &container.Config{
					Image: "nginx:latest",
					ExposedPorts: nat.PortSet{
						"80/tcp": struct{}{},
					},
				},
			},
			expectError: false,
		},
		{
			name: "nil config",
			containerJSON: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &container.HostConfig{},
				},
				Config: nil,
			},
			expectError: true,
		},
		{
			name: "nil host config",
			containerJSON: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: nil,
				},
				Config: &container.Config{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ExtractContainerParams(tt.containerJSON)
			if (err != nil) != tt.expectError {
				t.Errorf("ExtractContainerParams() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if params == nil {
					t.Error("expected params to be non-nil")
					return
				}
				if params.Image != tt.containerJSON.Config.Image {
					t.Errorf("expected image %s, got %s", tt.containerJSON.Config.Image, params.Image)
				}
			}
		})
	}
}

func TestUpdateStandaloneContainer(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		setupMock   func(*docker.MockClient)
		expectError bool
	}{
		{
			name:        "successful update",
			containerID: "container123",
			setupMock: func(m *docker.MockClient) {
				m.InspectContainerFunc = func(ctx context.Context, id string) (types.ContainerJSON, error) {
					return types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							Name: "/test-container",
							HostConfig: &container.HostConfig{
								Binds: []string{},
							},
						},
						Config: &container.Config{
							Image: "nginx:latest",
							Env:   []string{"TEST=value"},
						},
					}, nil
				}
				m.PullImageFunc = func(ctx context.Context, imageName string) error {
					return nil
				}
				m.StopContainerFunc = func(ctx context.Context, id string) error {
					return nil
				}
				m.RemoveContainerFunc = func(ctx context.Context, id string) error {
					return nil
				}
				m.CreateContainerFunc = func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error) {
					return "new-container-id", nil
				}
				m.StartContainerFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name:        "inspect fails",
			containerID: "container123",
			setupMock: func(m *docker.MockClient) {
				m.InspectContainerFunc = func(ctx context.Context, id string) (types.ContainerJSON, error) {
					return types.ContainerJSON{}, fmt.Errorf("container not found")
				}
			},
			expectError: true,
		},
		{
			name:        "pull image fails",
			containerID: "container123",
			setupMock: func(m *docker.MockClient) {
				m.InspectContainerFunc = func(ctx context.Context, id string) (types.ContainerJSON, error) {
					return types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							Name: "/test-container",
							HostConfig: &container.HostConfig{},
						},
						Config: &container.Config{
							Image: "nginx:latest",
						},
					}, nil
				}
				m.PullImageFunc = func(ctx context.Context, imageName string) error {
					return fmt.Errorf("failed to pull image")
				}
			},
			expectError: true,
		},
		{
			name:        "stop container fails",
			containerID: "container123",
			setupMock: func(m *docker.MockClient) {
				m.InspectContainerFunc = func(ctx context.Context, id string) (types.ContainerJSON, error) {
					return types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							Name: "/test-container",
							HostConfig: &container.HostConfig{},
						},
						Config: &container.Config{
							Image: "nginx:latest",
						},
					}, nil
				}
				m.PullImageFunc = func(ctx context.Context, imageName string) error {
					return nil
				}
				m.StopContainerFunc = func(ctx context.Context, id string) error {
					return fmt.Errorf("failed to stop container")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{}
			tt.setupMock(mockClient)

			err := UpdateStandaloneContainer(context.Background(), mockClient, tt.containerID)
			if (err != nil) != tt.expectError {
				t.Errorf("UpdateStandaloneContainer() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestUpdateComposeProject(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		workDir     string
		images      []string
		setupMock   func(*docker.MockClient)
		expectError bool
	}{
		{
			name:        "missing working directory",
			projectName: "myapp",
			workDir:     "",
			images:      []string{"nginx:latest"},
			setupMock:   func(m *docker.MockClient) {},
			expectError: true,
		},
		{
			name:        "pull image fails",
			projectName: "myapp",
			workDir:     "/home/user/app",
			images:      []string{"nginx:latest"},
			setupMock: func(m *docker.MockClient) {
				m.PullImageFunc = func(ctx context.Context, imageName string) error {
					return fmt.Errorf("failed to pull image")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{}
			tt.setupMock(mockClient)

			err := UpdateComposeProject(context.Background(), mockClient, tt.projectName, tt.workDir, tt.images)
			if (err != nil) != tt.expectError {
				t.Errorf("UpdateComposeProject() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
