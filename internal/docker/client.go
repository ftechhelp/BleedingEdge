package docker

import (
	"context"
	"io"
	"log/slog"
	"time"

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
	cli    *client.Client
	logger *slog.Logger
}

// NewClient creates a new Docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	
	// Use default logger if not provided
	logger := slog.Default()
	
	return &Client{
		cli:    cli,
		logger: logger,
	}, nil
}

// NewClientWithLogger creates a new Docker client with a custom logger
func NewClientWithLogger(logger *slog.Logger) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{
		cli:    cli,
		logger: logger,
	}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}

// ListContainers lists all containers (running and stopped)
func (c *Client) ListContainers(ctx context.Context) ([]types.Container, error) {
	start := time.Now()
	c.logger.Debug("listing containers")
	
	containers, err := c.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to list containers",
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}
	
	c.logger.Debug("listed containers successfully",
		"count", len(containers),
		"duration_ms", duration.Milliseconds(),
	)
	return containers, nil
}

// InspectContainer returns detailed information about a container
func (c *Client) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	start := time.Now()
	c.logger.Debug("inspecting container", "container_id", id)
	
	containerJSON, err := c.cli.ContainerInspect(ctx, id)
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to inspect container",
			"container_id", id,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return types.ContainerJSON{}, err
	}
	
	c.logger.Debug("inspected container successfully",
		"container_id", id,
		"container_name", containerJSON.Name,
		"duration_ms", duration.Milliseconds(),
	)
	return containerJSON, nil
}

// PullImage pulls an image from the registry
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	start := time.Now()
	c.logger.Debug("pulling image", "image", imageName)
	
	out, err := c.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		duration := time.Since(start)
		c.logger.Error("failed to pull image",
			"image", imageName,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}
	defer out.Close()
	
	// Consume the output to ensure the pull completes
	_, err = io.Copy(io.Discard, out)
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to complete image pull",
			"image", imageName,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}
	
	c.logger.Debug("pulled image successfully",
		"image", imageName,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// GetImageDigest returns the digest of an image
func (c *Client) GetImageDigest(ctx context.Context, imageName string) (string, error) {
	start := time.Now()
	c.logger.Debug("getting image digest", "image", imageName)
	
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to get image digest",
			"image", imageName,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return "", err
	}
	
	// Return the RepoDigests if available, otherwise return the ID
	var digest string
	if len(inspect.RepoDigests) > 0 {
		digest = inspect.RepoDigests[0]
	} else {
		digest = inspect.ID
	}
	
	c.logger.Debug("got image digest successfully",
		"image", imageName,
		"digest", digest,
		"duration_ms", duration.Milliseconds(),
	)
	return digest, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, id string) error {
	start := time.Now()
	c.logger.Debug("starting container", "container_id", id)
	
	err := c.cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to start container",
			"container_id", id,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}
	
	c.logger.Debug("started container successfully",
		"container_id", id,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// StopContainer stops a container
func (c *Client) StopContainer(ctx context.Context, id string) error {
	start := time.Now()
	c.logger.Debug("stopping container", "container_id", id)
	
	err := c.cli.ContainerStop(ctx, id, nil)
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to stop container",
			"container_id", id,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}
	
	c.logger.Debug("stopped container successfully",
		"container_id", id,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// RestartContainer restarts a container
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	start := time.Now()
	c.logger.Debug("restarting container", "container_id", id)
	
	err := c.cli.ContainerRestart(ctx, id, nil)
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to restart container",
			"container_id", id,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}
	
	c.logger.Debug("restarted container successfully",
		"container_id", id,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	start := time.Now()
	c.logger.Debug("removing container", "container_id", id)
	
	err := c.cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true})
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to remove container",
			"container_id", id,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}
	
	c.logger.Debug("removed container successfully",
		"container_id", id,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// CreateContainer creates a new container
func (c *Client) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) (string, error) {
	start := time.Now()
	c.logger.Debug("creating container",
		"name", name,
		"image", config.Image,
	)
	
	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
	
	duration := time.Since(start)
	if err != nil {
		c.logger.Error("failed to create container",
			"name", name,
			"image", config.Image,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return "", err
	}
	
	c.logger.Debug("created container successfully",
		"name", name,
		"container_id", resp.ID,
		"image", config.Image,
		"duration_ms", duration.Milliseconds(),
	)
	return resp.ID, nil
}

// ExecuteCommand executes a command in a specific working directory
// This is used for running docker compose commands
func (c *Client) ExecuteCommand(ctx context.Context, workDir string, command string, args []string) error {
	// For compose commands, we'll use the exec package
	// This is a placeholder implementation that will be enhanced when needed
	return nil
}
