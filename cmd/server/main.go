package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/bleeding-edge/bleeding-edge/internal/docker"
	"github.com/bleeding-edge/bleeding-edge/internal/handlers"
	"github.com/gorilla/mux"
)

func main() {
	// Get configuration from environment
	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "info")

	// Initialize structured logger
	logger := initLogger(logLevel)

	// Initialize Docker client wrapper
	dockerClient, err := docker.NewClient()
	if err != nil {
		logger.Error("failed to initialize Docker client", "error", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	// Verify Docker connectivity
	if err := verifyDockerConnection(dockerClient); err != nil {
		logger.Error("failed to connect to Docker daemon", "error", err)
		os.Exit(1)
	}

	logger.Info("successfully connected to Docker daemon")

	// Load templates
	tmpl, err := loadTemplates()
	if err != nil {
		logger.Error("failed to load templates", "error", err)
		os.Exit(1)
	}

	// Initialize handlers
	homeHandler := handlers.NewHomeHandler(dockerClient, tmpl, logger)
	detailHandler := handlers.NewDetailHandler(dockerClient, tmpl, logger)
	opsHandler := handlers.NewOperationsHandler(dockerClient, logger)

	// Initialize HTTP router
	router := mux.NewRouter()

	// Add middleware
	router.Use(loggingMiddleware(logger))
	router.Use(recoveryMiddleware(logger))

	// Configure routes
	router.Handle("/", homeHandler).Methods("GET")
	router.HandleFunc("/container/{id}", detailHandler.ServeHTTP).Methods("GET")
	router.HandleFunc("/container/{id}/update", opsHandler.HandleUpdate).Methods("POST")
	router.HandleFunc("/container/{id}/start", opsHandler.HandleStart).Methods("POST")
	router.HandleFunc("/container/{id}/stop", opsHandler.HandleStop).Methods("POST")
	router.HandleFunc("/container/{id}/restart", opsHandler.HandleRestart).Methods("POST")

	// Serve static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	logger.Info("starting server", "address", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// initLogger initializes a structured logger with the specified level
func initLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}

// loadTemplates loads all HTML templates
func loadTemplates() (*template.Template, error) {
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return tmpl, nil
}

// verifyDockerConnection checks if the Docker daemon is accessible
func verifyDockerConnection(cli *docker.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to list containers as a connectivity check
	_, err := cli.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("Docker daemon not accessible: %w", err)
	}

	return nil
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

// recoveryMiddleware recovers from panics and logs them
func recoveryMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
