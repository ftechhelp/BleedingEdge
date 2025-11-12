package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
	"github.com/bleeding-edge/bleeding-edge/internal/services"
	"github.com/gorilla/mux"
)

// OperationsHandler handles container lifecycle and update operations
type OperationsHandler struct {
	client docker.DockerClient
	logger *slog.Logger
}

// NewOperationsHandler creates a new operations handler
func NewOperationsHandler(client docker.DockerClient, logger *slog.Logger) *OperationsHandler {
	return &OperationsHandler{
		client: client,
		logger: logger,
	}
}

// HandleUpdate handles POST /container/:id/update requests
func (h *OperationsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		h.sendErrorResponse(w, "update", "", "Container ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	h.logger.Info("handling update request", "id", id)

	// Get container groups to determine if this is a compose project or standalone
	groups, err := services.GetContainerGroups(ctx, h.client)
	if err != nil {
		h.logger.Error("failed to get container groups", "id", id, "error", err)
		h.sendErrorResponse(w, "update", id, "Failed to load container information", http.StatusInternalServerError)
		return
	}

	// Find the requested group
	var group *models.ContainerGroup
	for i := range groups {
		if groups[i].ID == id {
			group = &groups[i]
			break
		}
	}

	if group == nil {
		h.logger.Warn("container group not found", "id", id)
		h.sendErrorResponse(w, "update", id, "Container not found", http.StatusNotFound)
		return
	}

	// Execute update based on group type
	var updateErr error
	if group.Type == models.GroupTypeCompose {
		// Extract container images
		images := make([]string, 0, len(group.Containers))
		for _, container := range group.Containers {
			images = append(images, container.Image)
		}
		updateErr = services.UpdateComposeProject(ctx, h.client, group.Name, group.WorkingDir, images)
	} else {
		// Standalone container
		updateErr = services.UpdateStandaloneContainer(ctx, h.client, group.ID)
	}

	if updateErr != nil {
		h.logger.Error("update failed", "id", id, "type", group.Type, "error", updateErr)
		h.sendErrorResponse(w, "update", group.Name, formatErrorMessage(updateErr), http.StatusInternalServerError)
		return
	}

	h.logger.Info("update completed successfully", "id", id, "type", group.Type)
	h.sendSuccessResponse(w, "update", group.Name, fmt.Sprintf("%s updated successfully", group.Name))
}

// HandleStart handles POST /container/:id/start requests
func (h *OperationsHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	h.handleLifecycleOperation(w, r, id, "start", h.client.StartContainer)
}

// HandleStop handles POST /container/:id/stop requests
func (h *OperationsHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	h.handleLifecycleOperation(w, r, id, "stop", h.client.StopContainer)
}

// HandleRestart handles POST /container/:id/restart requests
func (h *OperationsHandler) HandleRestart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	h.handleLifecycleOperation(w, r, id, "restart", h.client.RestartContainer)
}

// handleLifecycleOperation is a helper function for lifecycle operations
func (h *OperationsHandler) handleLifecycleOperation(w http.ResponseWriter, r *http.Request, id, operation string, operationFunc func(context.Context, string) error) {
	if id == "" {
		h.sendErrorResponse(w, operation, "", "Container ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	h.logger.Info("handling lifecycle operation", "operation", operation, "id", id)

	// Get container name for better error messages
	containerName := id
	containerJSON, err := h.client.InspectContainer(ctx, id)
	if err == nil {
		containerName = strings.TrimPrefix(containerJSON.Name, "/")
	}

	// Execute the operation
	if err := operationFunc(ctx, id); err != nil {
		h.logger.Error("lifecycle operation failed", "operation", operation, "id", id, "error", err)
		h.sendErrorResponse(w, operation, containerName, formatErrorMessage(err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("lifecycle operation completed", "operation", operation, "id", id)
	h.sendSuccessResponse(w, operation, containerName, fmt.Sprintf("Container %s %sed successfully", containerName, operation))
}

// sendSuccessResponse sends a success response in htmx-compatible format
func (h *OperationsHandler) sendSuccessResponse(w http.ResponseWriter, operation, containerName, message string) {
	result := models.OperationResult{
		Success:   true,
		Message:   message,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// sendErrorResponse sends an error response in htmx-compatible format
func (h *OperationsHandler) sendErrorResponse(w http.ResponseWriter, operation, containerName, errorMsg string, statusCode int) {
	result := models.OperationResult{
		Success:   false,
		Error:     errorMsg,
		Message:   fmt.Sprintf("Failed to %s %s", operation, containerName),
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(result)
}

// formatErrorMessage converts technical error messages to user-friendly messages
func formatErrorMessage(err error) string {
	errMsg := err.Error()

	// Common error patterns and their user-friendly messages
	if strings.Contains(errMsg, "No such container") {
		return "Container not found. It may have been removed."
	}
	if strings.Contains(errMsg, "already in progress") {
		return "Operation already in progress. Please wait."
	}
	if strings.Contains(errMsg, "is not running") {
		return "Container is not running."
	}
	if strings.Contains(errMsg, "failed to pull") {
		return "Failed to pull image. Check your internet connection and image name."
	}
	if strings.Contains(errMsg, "permission denied") {
		return "Permission denied. Check Docker socket permissions."
	}
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "context deadline exceeded") {
		return "Operation timed out. The container may be unresponsive."
	}

	// Return the original error message if no pattern matches
	return errMsg
}
