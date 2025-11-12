package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// DockerClient defines the interface for Docker operations
type DockerClient interface {
	ListContainers(ctx context.Context) ([]types.Container, error)
	InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error)
	PullImage(ctx context.Context, imageName string) error
	GetImageDigest(ctx context.Context, imageName string) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	RestartContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string) error
	CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error)
	ExecuteCommand(ctx context.Context, workDir string, command string, args []string) error
}

// Client is a concrete implementation of DockerClient
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// ListContainers lists all containers (running and stopped)
func (c *Client) ListContainers(ctx context.Context) ([]types.Container, error) {
	return c.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
}

// InspectContainer returns detailed information about a container
func (c *Client) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(ctx, id)
}

// PullImage pulls an image from the registry
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	out, err := c.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Consume the output to ensure the pull completes
	_, err = io.Copy(io.Discard, out)
	return err
}

// GetImageDigest returns the digest of an image
func (c *Client) GetImageDigest(ctx context.Context, imageName string) (string, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return "", err
	}
	
	// Return the RepoDigests if available, otherwise return the ID
	if len(inspect.RepoDigests) > 0 {
		return inspect.RepoDigests[0], nil
	}
	return inspect.ID, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStop(ctx, id, nil)
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerRestart(ctx, id, nil)
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	return c.cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true})
}

// CreateContainer creates a new container
func (c *Client) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error) {
	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ExecuteCommand executes a command in a specific working directory
// This is used for running docker compose commands
func (c *Client) ExecuteCommand(ctx context.Context, workDir string, command string, args []string) error {
	// For compose commands, we'll use the exec package
	// This is a placeholder implementation that will be enhanced when needed
	return nil
}
