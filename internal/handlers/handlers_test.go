package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/mux"
)

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		containers     []types.Container
		expectedStatus int
	}{
		{
			name: "successful render with containers",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/nginx"},
					Image: "nginx:latest",
					State: "running",
					Labels: map[string]string{},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "successful render with no containers",
			containers:     []types.Container{},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{
				ListContainersFunc: func(ctx context.Context) ([]types.Container, error) {
					return tt.containers, nil
				},
				PullImageFunc: func(ctx context.Context, imageName string) error {
					return nil
				},
				GetImageDigestFunc: func(ctx context.Context, imageName string) (string, error) {
					return "sha256:digest", nil
				},
			}

			tmpl := template.Must(template.New("grid.html").Parse(`{{.Title}}`))
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			handler := NewHomeHandler(mockClient, tmpl, logger)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDetailHandler(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		containers     []types.Container
		expectedStatus int
	}{
		{
			name:        "container found",
			containerID: "container1",
			containers: []types.Container{
				{
					ID:    "container1",
					Names: []string{"/nginx"},
					Image: "nginx:latest",
					State: "running",
					Labels: map[string]string{},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "container not found",
			containerID:    "nonexistent",
			containers:     []types.Container{},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{
				ListContainersFunc: func(ctx context.Context) ([]types.Container, error) {
					return tt.containers, nil
				},
				PullImageFunc: func(ctx context.Context, imageName string) error {
					return nil
				},
				GetImageDigestFunc: func(ctx context.Context, imageName string) (string, error) {
					return "sha256:digest", nil
				},
			}

			tmpl := template.Must(template.New("detail.html").Parse(`{{.Title}}`))
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			handler := NewDetailHandler(mockClient, tmpl, logger)

			req := httptest.NewRequest(http.MethodGet, "/container/"+tt.containerID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.containerID})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestOperationsHandlerStart(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		setupMock      func(*docker.MockClient)
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:        "successful start",
			containerID: "container1",
			setupMock: func(m *docker.MockClient) {
				m.InspectContainerFunc = func(ctx context.Context, id string) (types.ContainerJSON, error) {
					return types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							Name: "/test-container",
						},
					}, nil
				}
				m.StartContainerFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{}
			tt.setupMock(mockClient)

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			handler := NewOperationsHandler(mockClient, logger)

			req := httptest.NewRequest(http.MethodPost, "/container/"+tt.containerID+"/start", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.containerID})
			w := httptest.NewRecorder()

			handler.HandleStart(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var result models.OperationResult
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("expected success=%v, got %v", tt.expectSuccess, result.Success)
			}
		})
	}
}

func TestOperationsHandlerUpdate(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		setupMock      func(*docker.MockClient)
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:        "successful standalone update",
			containerID: "container1",
			setupMock: func(m *docker.MockClient) {
				m.ListContainersFunc = func(ctx context.Context) ([]types.Container, error) {
					return []types.Container{
						{
							ID:    "container1",
							Names: []string{"/nginx"},
							Image: "nginx:latest",
							State: "running",
							Labels: map[string]string{},
						},
					}, nil
				}
				m.InspectContainerFunc = func(ctx context.Context, id string) (types.ContainerJSON, error) {
					return types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							Name: "/nginx",
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
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &docker.MockClient{}
			tt.setupMock(mockClient)

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			handler := NewOperationsHandler(mockClient, logger)

			req := httptest.NewRequest(http.MethodPost, "/container/"+tt.containerID+"/update", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.containerID})
			w := httptest.NewRecorder()

			handler.HandleUpdate(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var result models.OperationResult
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("expected success=%v, got %v", tt.expectSuccess, result.Success)
			}
		})
	}
}

func TestFormatErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "container not found",
			err:      &testError{msg: "No such container: abc123"},
			expected: "Container not found. It may have been removed.",
		},
		{
			name:     "operation in progress",
			err:      &testError{msg: "operation already in progress"},
			expected: "Operation already in progress. Please wait.",
		},
		{
			name:     "timeout error",
			err:      &testError{msg: "context deadline exceeded"},
			expected: "Operation timed out. The container may be unresponsive.",
		},
		{
			name:     "unknown error",
			err:      &testError{msg: "some unknown error"},
			expected: "some unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorMessage(tt.err)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
