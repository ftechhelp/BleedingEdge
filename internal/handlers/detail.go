package handlers

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/models"
	"github.com/bleeding-edge/bleeding-edge/internal/services"
	"github.com/gorilla/mux"
)

// DetailHandler handles the container detail view
type DetailHandler struct {
	client   docker.DockerClient
	template *template.Template
	logger   *slog.Logger
}

// NewDetailHandler creates a new detail handler
func NewDetailHandler(client docker.DockerClient, tmpl *template.Template, logger *slog.Logger) *DetailHandler {
	return &DetailHandler{
		client:   client,
		template: tmpl,
		logger:   logger,
	}
}

// ServeHTTP handles GET /container/:id requests
func (h *DetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract container/group ID from URL path using gorilla/mux
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "Container ID required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	h.logger.Info("handling detail page request", "id", id)

	// Get all container groups to find the requested one
	groups, err := services.GetContainerGroups(ctx, h.client)
	if err != nil {
		h.logger.Error("failed to get container groups",
			"error", err,
			"operation", "list_containers",
			"container_id", id,
		)
		http.Error(w, "Failed to load container details. Please check Docker daemon connection.", http.StatusInternalServerError)
		return
	}

	// Check for updates only if requested
	checkUpdates := r.URL.Query().Get("check_updates")
	if checkUpdates == "true" {
		// Use a longer timeout for update checks since pulling images can take time
		updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer updateCancel()
		
		if err := services.CheckUpdates(updateCtx, h.client, groups); err != nil {
			h.logger.Warn("failed to check updates",
				"error", err,
				"operation", "check_updates",
				"container_id", id,
			)
			// Continue rendering even if update check fails
		}
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
		h.logger.Warn("container group not found",
			"id", id,
			"operation", "get_container_details",
		)
		http.Error(w, "Container not found. It may have been removed.", http.StatusNotFound)
		return
	}

	// Prepare template data
	data := map[string]interface{}{
		"Group":        group,
		"Title":        "BleedingEdge - " + group.Name,
		"CheckUpdates": checkUpdates != "true",
	}

	// Render template
	if err := h.template.ExecuteTemplate(w, "detail.html", data); err != nil {
		h.logger.Error("failed to render template",
			"error", err,
			"template", "detail.html",
			"container_id", id,
		)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	h.logger.Info("detail page rendered successfully", "id", id, "type", group.Type)
}
