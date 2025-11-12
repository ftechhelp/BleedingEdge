package services

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

// ExtractContainerParams extracts all configuration parameters from a running container
// This is used to recreate the container with the same configuration but a new image
func ExtractContainerParams(containerJSON types.ContainerJSON) (*models.ContainerParams, error) {
	if containerJSON.Config == nil {
		return nil, fmt.Errorf("container config is nil")
	}
	if containerJSON.HostConfig == nil {
		return nil, fmt.Errorf("container host config is nil")
	}

	config := containerJSON.Config
	hostConfig := containerJSON.HostConfig

	// Extract container name (remove leading slash)
	name := strings.TrimPrefix(containerJSON.Name, "/")

	// Extract port bindings
	portBindings := make(nat.PortMap)
	if hostConfig.PortBindings != nil {
		for port, bindings := range hostConfig.PortBindings {
			portBindings[port] = bindings
		}
	}

	// Extract volume binds
	binds := []string{}
	if hostConfig.Binds != nil {
		binds = append(binds, hostConfig.Binds...)
	}

	// Extract networks
	networks := []string{}
	if containerJSON.NetworkSettings != nil && containerJSON.NetworkSettings.Networks != nil {
		for networkName := range containerJSON.NetworkSettings.Networks {
			networks = append(networks, networkName)
		}
	}

	// Extract exposed ports for container config
	exposedPorts := make(nat.PortSet)
	if config.ExposedPorts != nil {
		for port := range config.ExposedPorts {
			exposedPorts[port] = struct{}{}
		}
	}

	params := &models.ContainerParams{
		Image:         config.Image,
		Name:          name,
		Env:           config.Env,
		Cmd:           config.Cmd,
		Entrypoint:    config.Entrypoint,
		PortBindings:  portBindings,
		Binds:         binds,
		Networks:      networks,
		RestartPolicy: hostConfig.RestartPolicy,
		Labels:        config.Labels,
		Resources:     hostConfig.Resources,
	}

	return params, nil
}

// UpdateStandaloneContainer updates a standalone container by recreating it with the latest image
// This preserves all container configuration while updating to the latest image version
func UpdateStandaloneContainer(ctx context.Context, client docker.DockerClient, containerID string) error {
	start := time.Now()
	logger := slog.Default()
	logger.Info("starting standalone container update",
		"container_id", containerID,
		"operation", "update",
	)
	
	// Step 1: Inspect the container to get full configuration
	containerJSON, err := client.InspectContainer(ctx, containerID)
	if err != nil {
		logger.Error("failed to inspect container for update",
			"container_id", containerID,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	
	containerName := strings.TrimPrefix(containerJSON.Name, "/")

	// Step 2: Extract all container parameters
	params, err := ExtractContainerParams(containerJSON)
	if err != nil {
		logger.Error("failed to extract container parameters",
			"container_id", containerID,
			"container_name", containerName,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to extract container parameters for %s: %w", containerID, err)
	}
	
	logger.Debug("extracted container parameters",
		"container_name", containerName,
		"image", params.Image,
	)

	// Step 3: Pull the latest image
	logger.Debug("pulling latest image",
		"container_name", containerName,
		"image", params.Image,
	)
	if err := client.PullImage(ctx, params.Image); err != nil {
		logger.Error("failed to pull latest image",
			"container_name", containerName,
			"image", params.Image,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to pull latest image %s: %w", params.Image, err)
	}

	// Step 4: Stop the old container
	logger.Debug("stopping old container",
		"container_name", containerName,
	)
	if err := client.StopContainer(ctx, containerID); err != nil {
		logger.Error("failed to stop container",
			"container_name", containerName,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	// Step 5: Remove the old container
	logger.Debug("removing old container",
		"container_name", containerName,
	)
	if err := client.RemoveContainer(ctx, containerID); err != nil {
		logger.Error("failed to remove container",
			"container_name", containerName,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	// Step 6: Create container config
	exposedPorts := make(nat.PortSet)
	for port := range params.PortBindings {
		exposedPorts[port] = struct{}{}
	}

	containerConfig := &container.Config{
		Image:        params.Image,
		Env:          params.Env,
		Cmd:          params.Cmd,
		Entrypoint:   params.Entrypoint,
		Labels:       params.Labels,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings:  params.PortBindings,
		Binds:         params.Binds,
		RestartPolicy: params.RestartPolicy,
		Resources:     params.Resources,
	}

	// Step 7: Create new container with the same name and configuration
	logger.Debug("creating new container",
		"container_name", params.Name,
		"image", params.Image,
	)
	newContainerID, err := client.CreateContainer(ctx, containerConfig, hostConfig, params.Name)
	if err != nil {
		logger.Error("failed to create new container",
			"container_name", params.Name,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to create new container %s: %w", params.Name, err)
	}

	// Step 8: Start the new container
	logger.Debug("starting new container",
		"container_name", params.Name,
		"new_container_id", newContainerID,
	)
	if err := client.StartContainer(ctx, newContainerID); err != nil {
		logger.Error("failed to start new container",
			"container_name", params.Name,
			"new_container_id", newContainerID,
			"operation", "update",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to start new container %s: %w", newContainerID, err)
	}

	duration := time.Since(start)
	logger.Info("standalone container updated successfully",
		"container_name", params.Name,
		"new_container_id", newContainerID,
		"operation", "update",
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// UpdateComposeProject updates all containers in a Docker Compose project
// This uses docker compose commands to properly handle the project lifecycle
func UpdateComposeProject(ctx context.Context, client docker.DockerClient, projectName, workDir string, containerImages []string) error {
	start := time.Now()
	logger := slog.Default()
	logger.Info("starting compose project update",
		"project_name", projectName,
		"working_dir", workDir,
		"image_count", len(containerImages),
		"operation", "update",
	)
	
	if workDir == "" {
		logger.Error("working directory is required for compose project",
			"project_name", projectName,
			"operation", "update",
		)
		return fmt.Errorf("working directory is required for compose project %s", projectName)
	}

	// Step 1: Pull latest images for all containers in the project
	logger.Debug("pulling images for compose project",
		"project_name", projectName,
		"image_count", len(containerImages),
	)
	for _, image := range containerImages {
		if err := client.PullImage(ctx, image); err != nil {
			logger.Error("failed to pull image for compose project",
				"project_name", projectName,
				"image", image,
				"operation", "update",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return fmt.Errorf("failed to pull image %s for project %s: %w", image, projectName, err)
		}
	}

	// Step 2: Execute docker compose down
	logger.Debug("executing docker compose down",
		"project_name", projectName,
		"working_dir", workDir,
	)
	downCmd := exec.CommandContext(ctx, "docker", "compose", "down")
	downCmd.Dir = workDir
	if output, err := downCmd.CombinedOutput(); err != nil {
		logger.Error("failed to execute docker compose down",
			"project_name", projectName,
			"working_dir", workDir,
			"operation", "update",
			"error", err,
			"output", string(output),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to execute 'docker compose down' for project %s: %w\nOutput: %s", projectName, err, string(output))
	}

	// Step 3: Execute docker compose up -d --build
	logger.Debug("executing docker compose up",
		"project_name", projectName,
		"working_dir", workDir,
	)
	upCmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--build")
	upCmd.Dir = workDir
	if output, err := upCmd.CombinedOutput(); err != nil {
		logger.Error("failed to execute docker compose up",
			"project_name", projectName,
			"working_dir", workDir,
			"operation", "update",
			"error", err,
			"output", string(output),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return fmt.Errorf("failed to execute 'docker compose up -d --build' for project %s: %w\nOutput: %s", projectName, err, string(output))
	}

	duration := time.Since(start)
	logger.Info("compose project updated successfully",
		"project_name", projectName,
		"operation", "update",
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}
