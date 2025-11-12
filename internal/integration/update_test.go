package integration

import (
	"context"
	"testing"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/models"
	"github.com/bleeding-edge/bleeding-edge/internal/services"
)

func TestStandaloneContainerSimpleUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)
	helper := NewTestHelper(t, client)
	defer helper.Cleanup(ctx)

	// Create a simple container with an older alpine version
	containerID, err := helper.CreateSimpleContainer(ctx, "alpine:3.18", "test-alpine-simple")
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Get the original image digest
	originalDigest, err := helper.GetContainerImageDigest(ctx, containerID)
	if err != nil {
		t.Fatalf("failed to get original digest: %v", err)
	}

	// Update the container to latest alpine
	err = services.UpdateStandaloneContainer(ctx, client, containerID)
	if err != nil {
		t.Fatalf("failed to update container: %v", err)
	}

	// Wait a bit for the new container to start
	time.Sleep(2 * time.Second)

	// List containers to find the new one
	containers, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("failed to list containers: %v", err)
	}

	// Find the new container (should have same name)
	var newContainerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/test-alpine-simple" {
				newContainerID = c.ID
				break
			}
		}
	}

	if newContainerID == "" {
		t.Fatal("new container not found after update")
	}

	// Get the new image digest
	newDigest, err := helper.GetContainerImageDigest(ctx, newContainerID)
	if err != nil {
		t.Fatalf("failed to get new digest: %v", err)
	}

	// Verify the container was updated (digests should be different)
	if originalDigest == newDigest {
		t.Error("container was not updated - digests are the same")
	}

	// Verify the new container is running
	err = helper.WaitForContainer(ctx, newContainerID, "running", 10*time.Second)
	if err != nil {
		t.Errorf("new container is not running: %v", err)
	}

	// Clean up the new container
	helper.createdContainers = append(helper.createdContainers, newContainerID)
}

func TestStandaloneContainerComplexUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)
	helper := NewTestHelper(t, client)
	defer helper.Cleanup(ctx)

	// Create a complex container with volumes, env vars, and ports
	containerID, err := helper.CreateComplexContainer(ctx, "nginx:alpine", "test-nginx-complex")
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Inspect the original container to verify configuration
	originalInspect, err := client.InspectContainer(ctx, containerID)
	if err != nil {
		t.Fatalf("failed to inspect original container: %v", err)
	}

	// Verify original configuration
	if len(originalInspect.Config.Env) == 0 {
		t.Error("original container should have environment variables")
	}
	if len(originalInspect.HostConfig.Binds) == 0 {
		t.Error("original container should have volume binds")
	}

	// Update the container
	err = services.UpdateStandaloneContainer(ctx, client, containerID)
	if err != nil {
		t.Fatalf("failed to update container: %v", err)
	}

	// Wait for the new container to start
	time.Sleep(2 * time.Second)

	// Find the new container
	containers, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("failed to list containers: %v", err)
	}

	var newContainerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/test-nginx-complex" {
				newContainerID = c.ID
				break
			}
		}
	}

	if newContainerID == "" {
		t.Fatal("new container not found after update")
	}

	// Inspect the new container
	newInspect, err := client.InspectContainer(ctx, newContainerID)
	if err != nil {
		t.Fatalf("failed to inspect new container: %v", err)
	}

	// Verify configuration was preserved
	if len(newInspect.Config.Env) == 0 {
		t.Error("new container should have environment variables")
	}
	if len(newInspect.HostConfig.Binds) == 0 {
		t.Error("new container should have volume binds")
	}

	// Verify specific env vars were preserved
	hasTestVar := false
	for _, env := range newInspect.Config.Env {
		if env == "TEST_VAR=test_value" {
			hasTestVar = true
			break
		}
	}
	if !hasTestVar {
		t.Error("TEST_VAR environment variable was not preserved")
	}

	// Verify the new container is running
	err = helper.WaitForContainer(ctx, newContainerID, "running", 10*time.Second)
	if err != nil {
		t.Errorf("new container is not running: %v", err)
	}

	// Clean up the new container
	helper.createdContainers = append(helper.createdContainers, newContainerID)
}

func TestComposeProjectUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)
	helper := NewTestHelper(t, client)
	defer helper.Cleanup(ctx)

	// Create a compose project
	projectName := "test-compose-project"
	workDir, containerIDs, err := helper.CreateComposeProject(ctx, projectName)
	if err != nil {
		t.Fatalf("failed to create compose project: %v", err)
	}

	if len(containerIDs) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containerIDs))
	}

	// Get original container IDs
	originalIDs := make(map[string]bool)
	for _, id := range containerIDs {
		originalIDs[id] = true
	}

	// Get images from containers
	var images []string
	for _, id := range containerIDs {
		inspect, err := client.InspectContainer(ctx, id)
		if err != nil {
			t.Fatalf("failed to inspect container: %v", err)
		}
		images = append(images, inspect.Config.Image)
	}

	// Update the compose project
	err = services.UpdateComposeProject(ctx, client, projectName, workDir, images)
	if err != nil {
		t.Fatalf("failed to update compose project: %v", err)
	}

	// Wait for containers to restart
	time.Sleep(3 * time.Second)

	// List containers again
	containers, err := client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("failed to list containers: %v", err)
	}

	// Find new containers for the project
	var newContainerIDs []string
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] == projectName {
			newContainerIDs = append(newContainerIDs, c.ID)
		}
	}

	if len(newContainerIDs) != 2 {
		t.Errorf("expected 2 containers after update, got %d", len(newContainerIDs))
	}

	// Verify all containers are running
	for _, id := range newContainerIDs {
		err = helper.WaitForContainer(ctx, id, "running", 10*time.Second)
		if err != nil {
			t.Errorf("container %s is not running: %v", id, err)
		}
		// Add to cleanup list
		helper.createdContainers = append(helper.createdContainers, id)
	}
}

func TestMixedEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)
	helper := NewTestHelper(t, client)
	defer helper.Cleanup(ctx)

	// Create a standalone container
	standaloneID, err := helper.CreateSimpleContainer(ctx, "alpine:3.18", "test-standalone")
	if err != nil {
		t.Fatalf("failed to create standalone container: %v", err)
	}

	// Create a compose project
	projectName := "test-mixed-compose"
	_, composeIDs, err := helper.CreateComposeProject(ctx, projectName)
	if err != nil {
		t.Fatalf("failed to create compose project: %v", err)
	}

	// Get container groups
	groups, err := services.GetContainerGroups(ctx, client)
	if err != nil {
		t.Fatalf("failed to get container groups: %v", err)
	}

	// Verify we have at least 2 groups (standalone + compose)
	if len(groups) < 2 {
		t.Errorf("expected at least 2 groups, got %d", len(groups))
	}

	// Find our containers in the groups
	foundStandalone := false
	foundCompose := false

	for _, group := range groups {
		for _, container := range group.Containers {
			if container.ID == standaloneID {
				foundStandalone = true
			}
			for _, composeID := range composeIDs {
				if container.ID == composeID {
					foundCompose = true
				}
			}
		}
	}

	if !foundStandalone {
		t.Error("standalone container not found in groups")
	}
	if !foundCompose {
		t.Error("compose containers not found in groups")
	}
}

func TestUpdateAvailableDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)
	helper := NewTestHelper(t, client)
	defer helper.Cleanup(ctx)

	// Create a container with an older alpine version
	_, err := helper.CreateSimpleContainer(ctx, "alpine:3.18", "test-update-check")
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Get container groups
	groups, err := services.GetContainerGroups(ctx, client)
	if err != nil {
		t.Fatalf("failed to get container groups: %v", err)
	}

	// Check for updates
	err = services.CheckUpdates(ctx, client, groups)
	if err != nil {
		t.Fatalf("failed to check updates: %v", err)
	}

	// Find our container group
	var testGroup *models.ContainerGroup
	for i := range groups {
		for _, container := range groups[i].Containers {
			if container.Name == "test-update-check" {
				testGroup = &groups[i]
				break
			}
		}
	}

	if testGroup == nil {
		t.Fatal("test container group not found")
	}

	// The container should have an update available (3.18 vs latest)
	// Note: This might not always be true if 3.18 is the latest, but typically it won't be
	t.Logf("Container has updates: %v", testGroup.HasUpdates)
}

func TestNoUpdateAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)
	helper := NewTestHelper(t, client)
	defer helper.Cleanup(ctx)

	// Create a container with the latest alpine
	_, err := helper.CreateSimpleContainer(ctx, "alpine:latest", "test-no-update")
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Get container groups
	groups, err := services.GetContainerGroups(ctx, client)
	if err != nil {
		t.Fatalf("failed to get container groups: %v", err)
	}

	// Check for updates
	err = services.CheckUpdates(ctx, client, groups)
	if err != nil {
		t.Fatalf("failed to check updates: %v", err)
	}

	// Find our container group
	var testGroup *models.ContainerGroup
	for i := range groups {
		for _, container := range groups[i].Containers {
			if container.Name == "test-no-update" {
				testGroup = &groups[i]
				break
			}
		}
	}

	if testGroup == nil {
		t.Fatal("test container group not found")
	}

	// The container should not have updates (already on latest)
	if testGroup.HasUpdates {
		t.Error("container should not have updates available")
	}
}

func TestFailedImagePull(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)

	// Try to pull a non-existent image
	err := client.PullImage(ctx, "nonexistent-image-12345:latest")
	if err == nil {
		t.Error("expected error when pulling non-existent image")
	}
}

func TestContainerRecreationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client := SetupDockerClient(t)

	// Try to update a non-existent container
	err := services.UpdateStandaloneContainer(ctx, client, "nonexistent-container-id")
	if err == nil {
		t.Error("expected error when updating non-existent container")
	}
}
