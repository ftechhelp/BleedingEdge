package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
)

// GetContainerGroups lists all containers and groups them by compose project
func GetContainerGroups(ctx context.Context, client docker.DockerClient) ([]models.ContainerGroup, error) {
	start := time.Now()
	logger := slog.Default()
	logger.Debug("getting container groups")
	
	// List all containers
	containers, err := client.ListContainers(ctx)
	if err != nil {
		logger.Error("failed to list containers in GetContainerGroups",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
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

	duration := time.Since(start)
	logger.Debug("got container groups successfully",
		"group_count", len(groups),
		"compose_projects", len(composeProjects),
		"standalone_containers", len(standaloneGroups),
		"duration_ms", duration.Milliseconds(),
	)

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
	start := time.Now()
	logger := slog.Default()
	
	// Count total containers
	totalContainers := 0
	for _, group := range groups {
		totalContainers += len(group.Containers)
	}
	
	logger.Debug("checking for updates",
		"group_count", len(groups),
		"container_count", totalContainers,
	)
	
	// Track unique images to avoid duplicate pulls
	imageDigests := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process all containers across all groups
	for i := range groups {
		group := &groups[i]
		
		for j := range group.Containers {
			container := &group.Containers[j]
			
			wg.Add(1)
			go func(c *models.ContainerInfo) {
				defer wg.Done()

				imageName := c.Image
				
				// Skip update check for locally built images (no registry prefix)
				if isLocalImage(imageName) {
					logger.Debug("skipping update check for local image",
						"container", c.Name,
						"image", imageName,
					)
					mu.Lock()
					c.HasUpdate = false
					mu.Unlock()
					return
				}
				
				// Check if we already have the latest digest for this image
				mu.Lock()
				latestDigest, exists := imageDigests[imageName]
				mu.Unlock()

				if !exists {
					// Pull the latest image
					if err := client.PullImage(ctx, imageName); err != nil {
						logger.Warn("failed to pull image, skipping update check",
							"container", c.Name,
							"image", imageName,
							"error", err,
						)
						mu.Lock()
						c.HasUpdate = false
						mu.Unlock()
						return
					}

					// Get the latest digest
					digest, err := client.GetImageDigest(ctx, imageName)
					if err != nil {
						logger.Warn("failed to get digest for image, skipping update check",
							"container", c.Name,
							"image", imageName,
							"error", err,
						)
						mu.Lock()
						c.HasUpdate = false
						mu.Unlock()
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
					logger.Warn("failed to get current digest for container, skipping update check",
						"container", c.Name,
						"image", imageName,
						"error", err,
					)
					mu.Lock()
					c.HasUpdate = false
					mu.Unlock()
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

	// Update group-level HasUpdates flag
	groupsWithUpdates := 0
	containersWithUpdates := 0
	for i := range groups {
		group := &groups[i]
		group.HasUpdates = false
		for _, container := range group.Containers {
			if container.HasUpdate {
				group.HasUpdates = true
				containersWithUpdates++
			}
		}
		if group.HasUpdates {
			groupsWithUpdates++
		}
		group.AllRunning = areAllContainersRunning(group.Containers)
	}

	duration := time.Since(start)
	logger.Debug("checked for updates successfully",
		"groups_with_updates", groupsWithUpdates,
		"containers_with_updates", containersWithUpdates,
		"unique_images", len(imageDigests),
		"duration_ms", duration.Milliseconds(),
	)

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

// isLocalImage checks if an image is locally built (not from a registry)
// Local images typically don't have a registry prefix (e.g., "myapp:latest")
// Registry images have formats like "docker.io/library/nginx:latest" or "nginx:latest"
func isLocalImage(imageName string) bool {
	// Images referenced by raw digest (sha256:...) cannot be pulled
	if strings.HasPrefix(imageName, "sha256:") {
		return true
	}
	
	// Images with no tag or with localhost are local
	if strings.HasPrefix(imageName, "localhost/") || strings.HasPrefix(imageName, "localhost:") {
		return true
	}
	
	// Check if image has a registry domain (contains a dot before the first slash)
	// Examples: docker.io/nginx, gcr.io/project/image, quay.io/repo/image
	parts := strings.SplitN(imageName, "/", 2)
	if len(parts) > 1 {
		// If the first part contains a dot or colon, it's likely a registry
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			return false
		}
	}
	
	// Check for official Docker Hub images (single name like "nginx", "postgres")
	// These are pullable from Docker Hub
	imageParts := strings.Split(imageName, ":")
	baseName := imageParts[0]
	
	// If it contains a hyphen or underscore and no slash, it's likely a local compose image
	// Examples: "myproject-web", "subset-tvdb-api", "bleedingedge-bleeding-edge"
	if !strings.Contains(baseName, "/") && (strings.Contains(baseName, "-") || strings.Contains(baseName, "_")) {
		return true
	}
	
	return false
}
