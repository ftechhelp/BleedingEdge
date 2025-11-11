# Requirements Document

## Introduction

BleedingEdge is a lightweight web application focused on keeping Docker containers up to date. The application runs inside a Docker container with access to the host's Docker daemon, automatically detecting when containers are running outdated images and providing a simple interface to update them. The system handles both manually-run containers and Docker Compose projects, preserving their original configuration during updates.

## Glossary

- **BleedingEdge**: The Docker container management web application system
- **Host Docker Daemon**: The Docker service running on the host machine that BleedingEdge connects to
- **Container Instance**: A running or stopped Docker container on the host system
- **Container Image**: The Docker image that a Container Instance is based on
- **Remote Registry**: The Docker registry (e.g., Docker Hub) where Container Images are stored
- **Docker Compose Project**: A multi-container application defined and managed by Docker Compose, identified by the "com.docker.compose.project" label
- **Compose Project Group**: A collection of Container Instances that belong to the same Docker Compose Project
- **Container Parameters**: The configuration options (ports, volumes, environment variables, etc.) used to launch a Container Instance
- **Update Status**: An indicator showing whether a Container Instance is running the latest available Container Image version
- **Container Grid**: The main dashboard view displaying all Container Instances
- **Container Detail Page**: The individual page showing detailed information and controls for a specific Container Instance

## Requirements

### Requirement 1

**User Story:** As a system administrator, I want to see all containers and their update status at a glance, so that I can quickly identify which containers need updating

#### Acceptance Criteria

1. WHEN the user navigates to the main page, THE BleedingEdge SHALL display a Container Grid showing all Container Instances and Compose Project Groups on the Host Docker Daemon
2. THE BleedingEdge SHALL group Container Instances that share the same "com.docker.compose.project" label into a single Compose Project Group entry in the Container Grid
3. THE BleedingEdge SHALL display standalone Container Instances as individual entries in the Container Grid
4. THE BleedingEdge SHALL display each entry with its name and Update Status prominently
5. THE BleedingEdge SHALL indicate visually when containers are running versus stopped with a simple indicator
6. THE BleedingEdge SHALL highlight entries that have available image updates
7. WHEN checking updates for a Compose Project Group, THE BleedingEdge SHALL mark the group as having updates if any Container Instance in the group has an available update
8. THE BleedingEdge SHALL automatically check for updates on page load

### Requirement 2

**User Story:** As a system administrator, I want to check if my containers are using the latest images, so that I can keep my applications up to date

#### Acceptance Criteria

1. WHEN the BleedingEdge checks for updates, THE BleedingEdge SHALL pull the latest Container Image from the Remote Registry for each Container Instance
2. WHEN comparing image versions, THE BleedingEdge SHALL compare the image digest of the running Container Instance with the newly pulled Container Image digest
3. IF the image digests differ, THEN THE BleedingEdge SHALL mark the Container Instance as having an available update
4. IF the image digests match, THEN THE BleedingEdge SHALL mark the Container Instance as up to date
5. THE BleedingEdge SHALL display the comparison results on the Container Grid within 5 seconds of completing the check

### Requirement 3

**User Story:** As a system administrator, I want to view essential information about a specific container or compose project, so that I can understand how it will be updated

#### Acceptance Criteria

1. WHEN the user clicks on an entry in the Container Grid, THE BleedingEdge SHALL navigate to the Container Detail Page for that entry
2. WHEN displaying a Compose Project Group, THE BleedingEdge SHALL list all Container Instances that belong to the Docker Compose Project
3. WHEN displaying a Compose Project Group, THE BleedingEdge SHALL show the Update Status for each Container Instance in the group
4. WHEN displaying a standalone Container Instance, THE BleedingEdge SHALL show its Update Status
5. THE BleedingEdge SHALL display the current running state for each Container Instance
6. THE BleedingEdge SHALL display the update button prominently when an update is available

### Requirement 4

**User Story:** As a system administrator, I want basic container controls on the detail page, so that I can perform simple operations if needed after an update

#### Acceptance Criteria

1. WHEN the user is on the Container Detail Page for a standalone Container Instance, THE BleedingEdge SHALL provide buttons to start, stop, and restart that Container Instance
2. WHEN the user is on the Container Detail Page for a Compose Project Group, THE BleedingEdge SHALL provide buttons to start, stop, and restart each individual Container Instance in the group
3. WHEN the user clicks a lifecycle button, THE BleedingEdge SHALL execute the operation and display the result within 10 seconds
4. IF any lifecycle operation fails, THEN THE BleedingEdge SHALL display the error message on the Container Detail Page

### Requirement 5

**User Story:** As a system administrator, I want to update all containers in a Docker Compose project together, so that I can keep multi-container applications in sync

#### Acceptance Criteria

1. WHEN the user clicks the update button for a Compose Project Group, THE BleedingEdge SHALL pull the latest Container Images from the Remote Registry for all Container Instances in the group
2. WHEN the image pulls complete successfully, THE BleedingEdge SHALL execute "docker compose down" in the Docker Compose Project working directory
3. WHEN the compose down completes successfully, THE BleedingEdge SHALL execute "docker compose up -d --build" in the Docker Compose Project working directory
4. WHEN the compose up completes successfully, THE BleedingEdge SHALL display a success message on the Container Detail Page
5. IF any step of the update process fails, THEN THE BleedingEdge SHALL display the error message on the Container Detail Page and halt the update process

### Requirement 6

**User Story:** As a system administrator, I want to update manually-run containers, so that I can keep standalone containers current while preserving their configuration

#### Acceptance Criteria

1. WHEN the user clicks the update button for a Container Instance that is not part of a Docker Compose Project, THE BleedingEdge SHALL retrieve all Container Parameters from the running Container Instance
2. WHEN the Container Parameters are retrieved, THE BleedingEdge SHALL pull the latest Container Image from the Remote Registry
3. WHEN the image pull completes successfully, THE BleedingEdge SHALL stop and remove the existing Container Instance
4. WHEN the container removal completes successfully, THE BleedingEdge SHALL create and start a new Container Instance using the retrieved Container Parameters and the updated Container Image
5. IF any step of the update process fails, THEN THE BleedingEdge SHALL display the error message on the Container Detail Page and halt the update process

### Requirement 7

**User Story:** As a system administrator, I want to see clear error messages when operations fail, so that I can troubleshoot issues effectively

#### Acceptance Criteria

1. WHEN any Docker operation fails, THE BleedingEdge SHALL capture the error message from the Host Docker Daemon
2. THE BleedingEdge SHALL display error messages on the user interface within 2 seconds of the error occurring
3. THE BleedingEdge SHALL include the operation type and Container Instance name in error messages
4. THE BleedingEdge SHALL log all errors with timestamps to the application logs
5. THE BleedingEdge SHALL clear error messages from the user interface when the user initiates a new operation

### Requirement 8

**User Story:** As a developer, I want the application to have automated tests, so that I can verify functionality and prevent regressions

#### Acceptance Criteria

1. THE BleedingEdge SHALL include unit tests for all core business logic functions
2. THE BleedingEdge SHALL include integration tests that verify Docker daemon interactions using test containers
3. THE BleedingEdge SHALL include tests that verify container parameter extraction and reconstruction
4. THE BleedingEdge SHALL include tests that verify Docker Compose project detection
5. WHEN all tests are executed, THE BleedingEdge SHALL report test results with pass/fail status for each test case

### Requirement 9

**User Story:** As a system administrator, I want the application to run in a Docker container with access to the host Docker daemon, so that I can deploy it easily alongside my other containers

#### Acceptance Criteria

1. THE BleedingEdge SHALL provide a Dockerfile that builds a container image with all required dependencies
2. THE BleedingEdge SHALL connect to the Host Docker Daemon via the Docker socket mounted at "/var/run/docker.sock"
3. WHEN the BleedingEdge container starts, THE BleedingEdge SHALL verify connectivity to the Host Docker Daemon within 5 seconds
4. IF the Host Docker Daemon is not accessible, THEN THE BleedingEdge SHALL log an error and exit with a non-zero status code
5. THE BleedingEdge SHALL expose a web interface on a configurable port (default 8080)
