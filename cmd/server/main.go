package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

func main() {
	// Get configuration from environment
	port := getEnv("PORT", "8080")
	dockerHost := getEnv("DOCKER_HOST", "unix:///var/run/docker.sock")

	// Initialize Docker client
	dockerClient, err := initDockerClient(dockerHost)
	if err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	defer dockerClient.Close()

	// Verify Docker connectivity
	if err := verifyDockerConnection(dockerClient); err != nil {
		log.Fatalf("Failed to connect to Docker daemon: %v", err)
	}

	log.Println("Successfully connected to Docker daemon")

	// Initialize HTTP router
	router := mux.NewRouter()

	// Placeholder route
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "BleedingEdge - Docker Container Manager")
	})

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initDockerClient creates a new Docker client
func initDockerClient(host string) (*client.Client, error) {
	opts := []client.Opt{
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return cli, nil
}

// verifyDockerConnection checks if the Docker daemon is accessible
func verifyDockerConnection(cli *client.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker daemon not accessible: %w", err)
	}

	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
