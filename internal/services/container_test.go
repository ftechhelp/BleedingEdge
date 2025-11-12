package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
	"github.com/docker/docker/api/types"
)

func TestGetContainerGroups(t *testing.T) {
	tests := []struct {
		name           string
		containers     []types.Container
		expectedGroups int
		expectedCompose int
		expectedStandalone int
	}{
		{
			name: "compose project with multiple containers",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/app-web-1"},
					Image: "nginx:latest",
					State: "running",
					Labels: map[string]string{
						"com.docker.compose.project": "app",
						"com.docker.compose.project.working_dir": "/home/user/app",
					},
				},
				{
					ID:    "container2",
					Names: []string{"/app-db-1"},
					Image: "postgres:latest",
					State: "running",
					Labels: map[string]string{
						"com.docker.compose.project": "app",
						"com.docker.compose.project.working_dir": "/home/user/app",
					},
				},
			},
			expectedGroups: 1,
			expectedCompose: 1,
			expectedStandalone: 0,
		},
		{
			name: "standalone containers only",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/nginx"},
					Image: "nginx:latest",
					State: "running",
					Labels: map[string]string{},
				},
				{
					ID:    "container2",
					Names: []string{"/redis"},
					Image: "redis:latest",
					State: "stopped",
					Labels: map[string]string{},
				},
			},
			expectedGroups: 2,
			expectedCompose: 0,
			expectedStandalone: 2,
		},
		{
			name: "mixed compose and standalone",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/app-web-1"},
					Image: "nginx:latest",
					State: "running",
					Labels: map[string]string{
						"com.docker.compose.project": "app",
						"com.docker.compose.project.working_dir": "/home/user/app",
					},
				},
				{
					ID:    "container2",
					Names: []string{"/standalone"},
					Image: "redis:latest",
					State: "running",
					Labels: map[string]string{},
				},
			},
			expectedGroups: 2,
			expectedCompose: 1,
			expectedStandalone: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{
				ListContainersFunc: func(ctx context.Context) ([]types.Container, error) {
					return tt.containers, nil
				},
			}

			groups, err := GetContainerGroups(context.Background(), mockClient)
			if err != nil {
				t.Fatalf("GetContainerGroups() error = %v", err)
			}

			if len(groups) != tt.expectedGroups {
				t.Errorf("expected %d groups, got %d", tt.expectedGroups, len(groups))
			}

			composeCount := 0
			standaloneCount := 0
			for _, group := range groups {
				if group.Type == models.GroupTypeCompose {
					composeCount++
				} else {
					standaloneCount++
				}
			}

			if composeCount != tt.expectedCompose {
				t.Errorf("expected %d compose groups, got %d", tt.expectedCompose, composeCount)
			}
			if standaloneCount != tt.expectedStandalone {
				t.Errorf("expected %d standalone groups, got %d", tt.expectedStandalone, standaloneCount)
			}
		})
	}
}

func TestIsComposeProject(t *testing.T) {
	tests := []struct {
		name            string
		container       types.Container
		expectedCompose bool
		expectedProject string
	}{
		{
			name: "container with compose label",
			container: types.Container{
				Labels: map[string]string{
					"com.docker.compose.project": "myapp",
				},
			},
			expectedCompose: true,
			expectedProject: "myapp",
		},
		{
			name: "container without compose label",
			container: types.Container{
				Labels: map[string]string{
					"some.other.label": "value",
				},
			},
			expectedCompose: false,
			expectedProject: "",
		},
		{
			name: "container with empty compose label",
			container: types.Container{
				Labels: map[string]string{
					"com.docker.compose.project": "",
				},
			},
			expectedCompose: false,
			expectedProject: "",
		},
		{
			name: "container with nil labels",
			container: types.Container{
				Labels: nil,
			},
			expectedCompose: false,
			expectedProject: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCompose, projectName := IsComposeProject(tt.container)
			if isCompose != tt.expectedCompose {
				t.Errorf("expected isCompose=%v, got %v", tt.expectedCompose, isCompose)
			}
			if projectName != tt.expectedProject {
				t.Errorf("expected projectName=%s, got %s", tt.expectedProject, projectName)
			}
		})
	}
}

func TestCheckUpdates(t *testing.T) {
	tests := []struct {
		name                string
		groups              []models.ContainerGroup
		imageDigests        map[string]string
		expectedGroupUpdate bool
		expectedError       bool
	}{
		{
			name: "container with update available",
			groups: []models.ContainerGroup{
				{
					ID:   "container1",
					Name: "nginx",
					Type: models.GroupTypeStandalone,
					Containers: []models.ContainerInfo{
						{
							ID:    "container1",
							Name:  "nginx",
							Image: "nginx:latest",
							State: "running",
						},
					},
				},
			},
			imageDigests: map[string]string{
				"nginx:latest": "sha256:new-digest",
			},
			expectedGroupUpdate: true,
			expectedError:       false,
		},
		{
			name: "container up to date",
			groups: []models.ContainerGroup{
				{
					ID:   "container1",
					Name: "nginx",
					Type: models.GroupTypeStandalone,
					Containers: []models.ContainerInfo{
						{
							ID:    "container1",
							Name:  "nginx",
							Image: "nginx:latest",
							State: "running",
						},
					},
				},
			},
			imageDigests: map[string]string{
				"nginx:latest": "sha256:same-digest",
			},
			expectedGroupUpdate: false,
			expectedError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := make(map[string]int)
			mockClient := &docker.MockClient{
				PullImageFunc: func(ctx context.Context, imageName string) error {
					return nil
				},
				GetImageDigestFunc: func(ctx context.Context, imageName string) (string, error) {
					callCount[imageName]++
					// First call is for current digest, second call is for latest digest after pull
					if callCount[imageName] == 1 {
						// Return old digest for current image
						if tt.expectedGroupUpdate {
							return "sha256:old-digest", nil
						}
						return "sha256:same-digest", nil
					}
					// Second call returns the latest digest
					if digest, ok := tt.imageDigests[imageName]; ok {
						return digest, nil
					}
					return "sha256:same-digest", nil
				},
			}

			err := CheckUpdates(context.Background(), mockClient, tt.groups)
			if (err != nil) != tt.expectedError {
				t.Errorf("CheckUpdates() error = %v, expectedError %v", err, tt.expectedError)
			}

			if !tt.expectedError {
				hasUpdate := tt.groups[0].HasUpdates
				if hasUpdate != tt.expectedGroupUpdate {
					t.Errorf("expected group HasUpdates=%v, got %v", tt.expectedGroupUpdate, hasUpdate)
				}
			}
		})
	}
}

func TestCheckUpdatesError(t *testing.T) {
	groups := []models.ContainerGroup{
		{
			ID:   "container1",
			Name: "nginx",
			Type: models.GroupTypeStandalone,
			Containers: []models.ContainerInfo{
				{
					ID:    "container1",
					Name:  "nginx",
					Image: "nginx:latest",
					State: "running",
				},
			},
		},
	}

	mockClient := &docker.MockClient{
		PullImageFunc: func(ctx context.Context, imageName string) error {
			return fmt.Errorf("failed to pull image")
		},
	}

	err := CheckUpdates(context.Background(), mockClient, groups)
	if err == nil {
		t.Error("expected error when pull fails, got nil")
	}
}
