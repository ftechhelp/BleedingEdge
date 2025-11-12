package handlers

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/services"
)

// HomeHandler handles the main grid view
type HomeHandler struct {
	client   docker.DockerClient
	template *template.Template
	logger   *slog.Logger
}

// NewHomeHandler creates a new home handler
func NewHomeHandler(client docker.DockerClient, tmpl *template.Template, logger *slog.Logger) *HomeHandler {
	return &HomeHandler{
		client:   client,
		template: tmpl,
		logger:   logger,
	}
}

// ServeHTTP handles GET / requests
func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	h.logger.Info("handling home page request")

	// Get container groups
	groups, err := services.GetContainerGroups(ctx, h.client)
	if err != nil {
		h.logger.Error("failed to get container groups",
			"error", err,
			"operation", "list_containers",
		)
		http.Error(w, "Failed to load containers. Please check Docker daemon connection.", http.StatusInternalServerError)
		return
	}

	// Check for updates
	if err := services.CheckUpdates(ctx, h.client, groups); err != nil {
		h.logger.Warn("failed to check updates",
			"error", err,
			"operation", "check_updates",
		)
		// Continue rendering even if update check fails
	}

	// Prepare template data
	data := map[string]interface{}{
		"Groups": groups,
		"Title":  "BleedingEdge - Container Manager",
	}

	// Render template
	if err := h.template.ExecuteTemplate(w, "grid.html", data); err != nil {
		h.logger.Error("failed to render template",
			"error", err,
			"template", "grid.html",
		)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	h.logger.Info("home page rendered successfully", "group_count", len(groups))
}
