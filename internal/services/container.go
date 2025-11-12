package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
)

// GetContainerGroups lists all containers and groups them by compose project
func GetContainerGroups(ctx context.Context, client docker.DockerClient) ([]models.ContainerGroup, error) {
	// List all containers
	containers, err := client.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Map to store compose projects
	composeProjects := make(map[string]*models.ContainerGroup)
	// Slice to store standalone containers
	var standaloneGroups []models.ContainerGroup

	// Process each container
	for _, container := range containers {
		isCompose, projectName := IsComposeProject(container)
		
		// Create ContainerInfo from container data
		containerInfo := models.ContainerInfo{
			ID:     container.ID,
			Name:   getContainerName(container.Names),
			Image:  container.Image,
			State:  container.State,
			Labels: container.Labels,
		}

		if isCompose {
			// Add to compose project group
			if group, exists := composeProjects[projectName]; exists {
				group.Containers = append(group.Containers, containerInfo)
			} else {
				// Create new compose project group
				workingDir := container.Labels["com.docker.compose.project.working_dir"]
				composeProjects[projectName] = &models.ContainerGroup{
					ID:         projectName,
					Name:       projectName,
					Type:       models.GroupTypeCompose,
					Containers: []models.ContainerInfo{containerInfo},
					WorkingDir: workingDir,
				}
			}
		} else {
			// Create standalone container group
			standaloneGroups = append(standaloneGroups, models.ContainerGroup{
				ID:         container.ID,
				Name:       getContainerName(container.Names),
				Type:       models.GroupTypeStandalone,
				Containers: []models.ContainerInfo{containerInfo},
			})
		}
	}

	// Combine compose projects and standalone containers
	var groups []models.ContainerGroup
	for _, group := range composeProjects {
		// Update AllRunning status
		group.AllRunning = areAllContainersRunning(group.Containers)
		groups = append(groups, *group)
	}
	groups = append(groups, standaloneGroups...)

	return groups, nil
}

// IsComposeProject checks if a container is part of a compose project
// Returns (isCompose, projectName)
func IsComposeProject(container types.Container) (bool, string) {
	if container.Labels == nil {
		return false, ""
	}

	projectName, exists := container.Labels["com.docker.compose.project"]
	if exists && projectName != "" {
		return true, projectName
	}

	return false, ""
}

// CheckUpdates pulls latest images and compares digests to mark update status
func CheckUpdates(ctx context.Context, client docker.DockerClient, groups []models.ContainerGroup) error {
	// Track unique images to avoid duplicate pulls
	imageDigests := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(groups)*10) // Buffer for potential errors

	// Process all containers across all groups
	for i := range groups {
		group := &groups[i]
		
		for j := range group.Containers {
			container := &group.Containers[j]
			
			wg.Add(1)
			go func(c *models.ContainerInfo) {
				defer wg.Done()

				imageName := c.Image
				
				// Check if we already have the latest digest for this image
				mu.Lock()
				latestDigest, exists := imageDigests[imageName]
				mu.Unlock()

				if !exists {
					// Pull the latest image
					if err := client.PullImage(ctx, imageName); err != nil {
						errChan <- fmt.Errorf("failed to pull image %s: %w", imageName, err)
						return
					}

					// Get the latest digest
					digest, err := client.GetImageDigest(ctx, imageName)
					if err != nil {
						errChan <- fmt.Errorf("failed to get digest for image %s: %w", imageName, err)
						return
					}

					mu.Lock()
					imageDigests[imageName] = digest
					latestDigest = digest
					mu.Unlock()
				}

				// Get the current container's image digest
				currentDigest, err := client.GetImageDigest(ctx, c.Image)
				if err != nil {
					errChan <- fmt.Errorf("failed to get current digest for container %s: %w", c.Name, err)
					return
				}

				// Update container info
				mu.Lock()
				c.ImageDigest = currentDigest
				c.LatestDigest = latestDigest
				c.HasUpdate = currentDigest != latestDigest
				mu.Unlock()
			}(container)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		// Return the first error (could be enhanced to return all errors)
		return errs[0]
	}

	// Update group-level HasUpdates flag
	for i := range groups {
		group := &groups[i]
		group.HasUpdates = false
		for _, container := range group.Containers {
			if container.HasUpdate {
				group.HasUpdates = true
				break
			}
		}
		group.AllRunning = areAllContainersRunning(group.Containers)
	}

	return nil
}

// getContainerName extracts the container name from the Names slice
// Docker returns names with a leading slash, so we strip it
func getContainerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	name := names[0]
	return strings.TrimPrefix(name, "/")
}

// areAllContainersRunning checks if all containers in a group are running
func areAllContainersRunning(containers []models.ContainerInfo) bool {
	if len(containers) == 0 {
		return false
	}
	for _, container := range containers {
		if container.State != "running" {
			return false
		}
	}
	return true
}
