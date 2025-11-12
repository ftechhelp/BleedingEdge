package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

// TestHelper provides utilities for integration tests
type TestHelper struct {
	t              *testing.T
	client         docker.DockerClient
	createdContainers []string
	createdNetworks   []string
	tempDirs          []string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T, client docker.DockerClient) *TestHelper {
	return &TestHelper{
		t:              t,
		client:         client,
		createdContainers: []string{},
		createdNetworks:   []string{},
		tempDirs:          []string{},
	}
}

// CreateSimpleContainer creates a basic container for testing
func (h *TestHelper) CreateSimpleContainer(ctx context.Context, image, name string) (string, error) {
	// Pull image first
	if err := h.client.PullImage(ctx, image); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container
	containerID, err := h.client.CreateContainer(ctx,
		&container.Config{
			Image: image,
			Cmd:   []string{"sleep", "3600"},
		},
		&container.HostConfig{
			AutoRemove: false,
		},
		name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	h.createdContainers = append(h.createdContainers, containerID)

	// Start container
	if err := h.client.StartContainer(ctx, containerID); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return containerID, nil
}

// CreateComplexContainer creates a container with volumes, env vars, and ports
func (h *TestHelper) CreateComplexContainer(ctx context.Context, image, name string) (string, error) {
	// Pull image first
	if err := h.client.PullImage(ctx, image); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Create a temp directory for volume mount
	tempDir, err := os.MkdirTemp("", "bleeding-edge-test-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	h.tempDirs = append(h.tempDirs, tempDir)

	// Create container with complex configuration
	containerID, err := h.client.CreateContainer(ctx,
		&container.Config{
			Image: image,
			Cmd:   []string{"sleep", "3600"},
			Env: []string{
				"TEST_VAR=test_value",
				"ANOTHER_VAR=another_value",
			},
			ExposedPorts: nat.PortSet{
				"80/tcp": struct{}{},
			},
			Labels: map[string]string{
				"test.label": "test_value",
			},
		},
		&container.HostConfig{
			AutoRemove: false,
			Binds: []string{
				fmt.Sprintf("%s:/data", tempDir),
			},
			PortBindings: nat.PortMap{
				"80/tcp": []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: "0"}, // Random port
				},
			},
			RestartPolicy: container.RestartPolicy{
				Name: "unless-stopped",
			},
		},
		name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	h.createdContainers = append(h.createdContainers, containerID)

	// Start container
	if err := h.client.StartContainer(ctx, containerID); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return containerID, nil
}

// CreateComposeProject creates a docker compose project for testing
func (h *TestHelper) CreateComposeProject(ctx context.Context, projectName string) (string, []string, error) {
	// Create temp directory for compose file
	tempDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	h.tempDirs = append(h.tempDirs, tempDir)

	// Create docker-compose.yml
	composeContent := `version: '3.8'
services:
  web:
    image: nginx:alpine
    command: sleep 3600
    labels:
      test.service: web
  
  app:
    image: alpine:latest
    command: sleep 3600
    labels:
      test.service: app
`
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write compose file: %w", err)
	}

	// Pull images
	if err := h.client.PullImage(ctx, "nginx:alpine"); err != nil {
		return "", nil, fmt.Errorf("failed to pull nginx:alpine: %w", err)
	}
	if err := h.client.PullImage(ctx, "alpine:latest"); err != nil {
		return "", nil, fmt.Errorf("failed to pull alpine:latest: %w", err)
	}

	// Execute docker compose up
	if err := h.client.ExecuteCommand(ctx, tempDir, "docker", []string{"compose", "-p", projectName, "up", "-d"}); err != nil {
		return "", nil, fmt.Errorf("failed to start compose project: %w", err)
	}

	// Wait a bit for containers to start
	time.Sleep(2 * time.Second)

	// Get container IDs
	containers, err := h.client.ListContainers(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containerIDs []string
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] == projectName {
			containerIDs = append(containerIDs, c.ID)
			h.createdContainers = append(h.createdContainers, c.ID)
		}
	}

	return tempDir, containerIDs, nil
}

// CreateNetwork creates a Docker network for testing
func (h *TestHelper) CreateNetwork(ctx context.Context, name string) (string, error) {
	// This is a simplified version - in real tests we'd use the Docker SDK directly
	// For now, we'll skip network creation in integration tests
	h.t.Log("Network creation skipped in integration tests")
	return "", nil
}

// WaitForContainer waits for a container to reach a specific state
func (h *TestHelper) WaitForContainer(ctx context.Context, containerID string, state string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		inspect, err := h.client.InspectContainer(ctx, containerID)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}

		if inspect.State.Status == state {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for container to reach state %s", state)
}

// GetContainerImageDigest gets the image digest of a running container
func (h *TestHelper) GetContainerImageDigest(ctx context.Context, containerID string) (string, error) {
	inspect, err := h.client.InspectContainer(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	return inspect.Image, nil
}

// Cleanup removes all created resources
func (h *TestHelper) Cleanup(ctx context.Context) {
	// Stop and remove containers
	for _, containerID := range h.createdContainers {
		_ = h.client.StopContainer(ctx, containerID)
		_ = h.client.RemoveContainer(ctx, containerID)
	}

	// Remove temp directories
	for _, dir := range h.tempDirs {
		_ = os.RemoveAll(dir)
	}
}

// SetupDockerClient creates a Docker client for integration tests
func SetupDockerClient(t *testing.T) docker.DockerClient {
	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}

	// Verify Docker is accessible
	ctx := context.Background()
	_, err = client.ListContainers(ctx)
	if err != nil {
		t.Fatalf("Docker daemon not accessible: %v", err)
	}

	return client
}

// SkipIfDockerNotAvailable skips the test if Docker is not available
func SkipIfDockerNotAvailable(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "alpine:latest",
			Cmd:   []string{"echo", "test"},
		},
		Started: false,
	}

	_, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		t.Skip("Docker not available, skipping integration test")
	}
}

// PullTestImages pre-pulls common test images to speed up tests
func PullTestImages(ctx context.Context, client docker.DockerClient) error {
	images := []string{
		"alpine:latest",
		"alpine:3.18",
		"nginx:alpine",
		"busybox:latest",
	}

	for _, image := range images {
		if err := client.PullImage(ctx, image); err != nil {
			return fmt.Errorf("failed to pull %s: %w", image, err)
		}
	}

	return nil
}

// SuppressOutput suppresses stdout/stderr during test execution
func SuppressOutput() func() {
	null, _ := os.Open(os.DevNull)
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = null
	os.Stderr = null

	return func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		null.Close()
	}
}

// CaptureOutput captures stdout/stderr during test execution
func CaptureOutput() (func() string, func()) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w

	getOutput := func() string {
		w.Close()
		out, _ := io.ReadAll(r)
		return string(out)
	}

	restore := func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}

	return getOutput, restore
}
