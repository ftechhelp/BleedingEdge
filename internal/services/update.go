package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

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
	// Step 1: Inspect the container to get full configuration
	containerJSON, err := client.InspectContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	// Step 2: Extract all container parameters
	params, err := ExtractContainerParams(containerJSON)
	if err != nil {
		return fmt.Errorf("failed to extract container parameters for %s: %w", containerID, err)
	}

	// Step 3: Pull the latest image
	if err := client.PullImage(ctx, params.Image); err != nil {
		return fmt.Errorf("failed to pull latest image %s: %w", params.Image, err)
	}

	// Step 4: Stop the old container
	if err := client.StopContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	// Step 5: Remove the old container
	if err := client.RemoveContainer(ctx, containerID); err != nil {
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
	newContainerID, err := client.CreateContainer(ctx, containerConfig, hostConfig, params.Name)
	if err != nil {
		return fmt.Errorf("failed to create new container %s: %w", params.Name, err)
	}

	// Step 8: Start the new container
	if err := client.StartContainer(ctx, newContainerID); err != nil {
		return fmt.Errorf("failed to start new container %s: %w", newContainerID, err)
	}

	return nil
}

// UpdateComposeProject updates all containers in a Docker Compose project
// This uses docker compose commands to properly handle the project lifecycle
func UpdateComposeProject(ctx context.Context, client docker.DockerClient, projectName, workDir string, containerImages []string) error {
	if workDir == "" {
		return fmt.Errorf("working directory is required for compose project %s", projectName)
	}

	// Step 1: Pull latest images for all containers in the project
	for _, image := range containerImages {
		if err := client.PullImage(ctx, image); err != nil {
			return fmt.Errorf("failed to pull image %s for project %s: %w", image, projectName, err)
		}
	}

	// Step 2: Execute docker compose down
	downCmd := exec.CommandContext(ctx, "docker", "compose", "down")
	downCmd.Dir = workDir
	if output, err := downCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to execute 'docker compose down' for project %s: %w\nOutput: %s", projectName, err, string(output))
	}

	// Step 3: Execute docker compose up -d --build
	upCmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--build")
	upCmd.Dir = workDir
	if output, err := upCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to execute 'docker compose up -d --build' for project %s: %w\nOutput: %s", projectName, err, string(output))
	}

	return nil
}
