# Implementation Plan

- [ ] 1. Initialize Go project and core structure
  - Create Go module with required dependencies (Docker SDK, HTTP router)
  - Set up directory structure following the design (cmd/, internal/, web/)
  - Create basic main.go with application entry point and Docker connectivity check
  - _Requirements: 9.1, 9.2, 9.3, 9.4_

- [ ] 2. Implement Docker client wrapper and data models
  - [ ] 2.1 Create DockerClient interface in internal/docker/client.go
    - Define all methods: ListContainers, InspectContainer, PullImage, GetImageDigest, lifecycle operations, CreateContainer, ExecuteCommand
    - Implement concrete client wrapping Docker SDK with context support and error handling
    - _Requirements: 9.2_
  
  - [ ] 2.2 Create data models in internal/models/container.go
    - Implement ContainerGroup, ContainerInfo, ContainerParams, OperationResult structs
    - Define GroupType constants (compose/standalone)
    - _Requirements: 1.2, 1.3_

- [ ] 3. Implement container service for listing and grouping
  - [ ] 3.1 Create container service in internal/services/container.go
    - Implement GetContainerGroups function to list and group containers
    - Implement IsComposeProject function to detect compose projects via labels
    - Group containers by com.docker.compose.project label
    - Handle standalone containers as single-container groups
    - _Requirements: 1.1, 1.2, 1.3_
  
  - [ ] 3.2 Implement update checking functionality
    - Implement CheckUpdates function to pull latest images and compare digests
    - Compare running container image digest with latest pulled image digest
    - Mark containers and groups with update status
    - Handle concurrent image pulls with goroutines
    - _Requirements: 1.7, 1.8, 2.1, 2.2, 2.3, 2.4_

- [ ] 4. Implement update service for container updates
  - [ ] 4.1 Create update service in internal/services/update.go
    - Implement ExtractContainerParams function to extract all container configuration
    - Extract ports, volumes, env vars, networks, restart policy, labels, resources, command/entrypoint
    - _Requirements: 6.1_
  
  - [ ] 4.2 Implement standalone container update logic
    - Implement UpdateStandaloneContainer function
    - Pull latest image, stop/remove old container, create/start new container with preserved params
    - Handle errors at each step and provide detailed error messages
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
  
  - [ ] 4.3 Implement compose project update logic
    - Implement UpdateComposeProject function
    - Pull images, execute docker compose down, execute docker compose up -d --build
    - Handle errors at each step and provide detailed error messages
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [ ] 5. Implement HTTP handlers and routing
  - [ ] 5.1 Create home handler in internal/handlers/home.go
    - Implement GET / handler to display container grid
    - Call container service to get groups and check updates
    - Pass data to template for rendering
    - _Requirements: 1.1, 1.4, 1.5, 1.6, 1.8_
  
  - [ ] 5.2 Create detail handler in internal/handlers/detail.go
    - Implement GET /container/:id handler for detail page
    - Display compose project containers or standalone container details
    - Show update status and lifecycle controls
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_
  
  - [ ] 5.3 Create operations handler in internal/handlers/operations.go
    - Implement POST /container/:id/update for update operations
    - Implement POST /container/:id/start, stop, restart for lifecycle operations
    - Return htmx-compatible responses with success/error messages
    - Handle errors and display user-friendly messages
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 5.1-5.5, 6.1-6.5, 7.1, 7.2, 7.3, 7.5_
  
  - [ ] 5.4 Set up HTTP router and middleware in cmd/server/main.go
    - Configure routes for all handlers
    - Add logging middleware for request/response logging
    - Add error recovery middleware
    - _Requirements: 7.4_

- [ ] 6. Create HTML templates and UI
  - [ ] 6.1 Create base layout template in web/templates/layout.html
    - Include htmx and Alpine.js from CDN
    - Include Tailwind CSS from CDN for styling
    - Define base HTML structure with header and content area
    - _Requirements: 1.1_
  
  - [ ] 6.2 Create grid view template in web/templates/grid.html
    - Display container groups in responsive CSS grid
    - Show status indicators (running/stopped, update available)
    - Show compose project badge with container count
    - Make cards clickable to navigate to detail page
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_
  
  - [ ] 6.3 Create detail view template in web/templates/detail.html
    - Display container/project name and type
    - Show container list for compose projects with individual status
    - Display prominent update button when updates available
    - Show start/stop/restart buttons for each container
    - Display success/error messages with htmx target areas
    - Add htmx attributes for dynamic updates without page reload
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 4.1, 4.2, 7.2, 7.5_
  
  - [ ] 6.4 Create minimal CSS in web/static/styles.css
    - Style status indicators (green for running, gray for stopped, orange for updates)
    - Style action buttons and update button prominence
    - Style error and success message banners
    - _Requirements: 1.5, 1.6, 3.6_

- [ ] 7. Implement error handling and logging
  - [ ] 7.1 Add structured logging throughout application
    - Use log/slog for structured logging
    - Log Docker API calls at DEBUG level
    - Log errors at ERROR level with context
    - Include operation type, container name, and duration in logs
    - _Requirements: 7.4_
  
  - [ ] 7.2 Implement error response formatting
    - Create ErrorResponse struct with operation, container, message, details, timestamp
    - Convert Docker errors to user-friendly messages in handlers
    - Ensure errors display in UI within 2 seconds
    - _Requirements: 7.1, 7.2, 7.3_

- [ ] 8. Create Dockerfile and deployment configuration
  - [ ] 8.1 Create multi-stage Dockerfile
    - Build stage with Go 1.21+ to compile binary
    - Runtime stage with Alpine, docker-cli, and docker-cli-compose
    - Copy binary and web assets to runtime image
    - Expose port 8080
    - _Requirements: 9.1, 9.5_
  
  - [ ] 8.2 Create docker-compose.yml for deployment
    - Mount Docker socket at /var/run/docker.sock
    - Configure port mapping and environment variables
    - Set restart policy
    - _Requirements: 9.2, 9.5_
  
  - [ ] 8.3 Add environment variable configuration
    - Support PORT, LOG_LEVEL, DOCKER_HOST, UPDATE_CHECK_TIMEOUT
    - Load and validate environment variables on startup
    - _Requirements: 9.5_

- [ ] 9. Write unit tests for core functionality
  - [ ] 9.1 Create mock Docker client for testing
    - Implement mock DockerClient interface
    - Provide test data for various scenarios
    - _Requirements: 8.1_
  
  - [ ] 9.2 Write container service tests in internal/services/container_test.go
    - Test container grouping with various label combinations
    - Test compose project detection
    - Test update status detection with different digest scenarios
    - Test error handling
    - _Requirements: 8.1, 8.4_
  
  - [ ] 9.3 Write update service tests in internal/services/update_test.go
    - Test parameter extraction with various container configurations
    - Test standalone container update flow
    - Test compose project update flow
    - Test error handling for all failure scenarios
    - _Requirements: 8.1, 8.3_
  
  - [ ] 9.4 Write handler tests in internal/handlers/handlers_test.go
    - Test HTTP responses for all endpoints
    - Test template rendering with various data
    - Test error response formatting
    - _Requirements: 8.1, 8.5_

- [ ] 10. Write integration tests
  - [ ] 10.1 Set up integration test framework
    - Use testcontainers-go to spin up Docker daemon
    - Create test helper functions for container setup
    - _Requirements: 8.2_
  
  - [ ] 10.2 Write integration tests for update scenarios
    - Test standalone container with simple configuration
    - Test standalone container with complex configuration (volumes, networks, env vars)
    - Test compose project with multiple containers
    - Test mixed environment (standalone + compose)
    - Test update available and no update scenarios
    - Test failed image pull and container recreation
    - _Requirements: 8.2, 8.3, 8.4, 8.5_
