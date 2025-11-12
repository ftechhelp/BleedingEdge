# Integration Tests

This directory contains integration tests for the BleedingEdge application that test real Docker daemon interactions.

## Running Tests

### Run all tests (including integration tests)

```bash
go test -v ./internal/integration/...
```

### Run only unit tests (skip integration tests)

```bash
go test -v ./internal/integration/... -short
```

### Run all tests in the project

```bash
# Unit tests only
go test -v ./... -short

# All tests including integration
go test -v ./...
```

## Prerequisites

- Docker daemon must be running and accessible
- Docker socket must be available at `/var/run/docker.sock` (or `DOCKER_HOST` environment variable set)
- Sufficient permissions to create/remove containers
- Internet connection to pull test images

## Test Coverage

The integration tests cover:

1. **Standalone Container Updates**
   - Simple container with minimal configuration
   - Complex container with volumes, environment variables, and port bindings
   - Configuration preservation during updates

2. **Compose Project Updates**
   - Multi-container compose projects
   - Compose down/up workflow
   - Container recreation

3. **Mixed Environments**
   - Standalone containers and compose projects together
   - Container grouping logic

4. **Update Detection**
   - Detecting when updates are available
   - Detecting when containers are up to date
   - Image digest comparison

5. **Error Scenarios**
   - Failed image pulls
   - Container recreation failures
   - Non-existent containers

## Test Images

The tests use the following Docker images:
- `alpine:latest`
- `alpine:3.18`
- `nginx:alpine`
- `busybox:latest`

These images are pulled automatically during test execution.

## Cleanup

All tests use the `TestHelper` which automatically cleans up created resources (containers, volumes, temp directories) after each test completes.

## CI/CD

For CI/CD pipelines, use the `-short` flag to skip integration tests if Docker is not available:

```bash
go test -v ./... -short
```

To run integration tests in CI, ensure Docker-in-Docker or a Docker daemon is available.
