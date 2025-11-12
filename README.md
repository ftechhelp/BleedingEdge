# BleedingEdge 🚀

A modern, lightweight web UI for managing and updating Docker containers. Keep your containers on the bleeding edge with automatic update detection and one-click updates.

![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## Features

- 📊 **Visual Dashboard** - Clean grid view of all your containers and compose projects
- 🔄 **Update Detection** - Automatically checks for newer image versions
- ⚡ **One-Click Updates** - Update standalone containers or entire compose projects
- 🎮 **Container Controls** - Start, stop, and restart containers from the UI
- 🐳 **Compose Support** - Manages Docker Compose projects as grouped units
- 🎨 **Modern UI** - Built with Tailwind CSS, htmx, and Alpine.js
- 🔍 **Smart Detection** - Skips update checks for locally-built images

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/bleeding-edge.git
cd bleeding-edge

# Start the application
docker-compose up -d

# Access the UI
open http://localhost:8080
```

### Using Docker

```bash
docker run -d \
  --name bleeding-edge \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  bleeding-edge
```

### Building from Source

```bash
# Install dependencies
go mod download

# Build the binary
go build -o bleeding-edge ./cmd/server

# Run the application
./bleeding-edge
```

## Configuration

Configure BleedingEdge using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon socket |
| `UPDATE_CHECK_TIMEOUT` | `5m` | Timeout for update checks |

### Example with Custom Configuration

```yaml
version: '3.8'

services:
  bleeding-edge:
    build: .
    container_name: bleeding-edge
    ports:
      - "3000:3000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - PORT=3000
      - LOG_LEVEL=debug
      - UPDATE_CHECK_TIMEOUT=10m
    restart: unless-stopped
```

## How It Works

### Update Detection

BleedingEdge detects updates by:

1. **Pulling Latest Images** - Fetches the latest version of each container's image
2. **Comparing Digests** - Compares the running container's image digest with the latest
3. **Visual Indicators** - Shows orange badges and borders for containers with updates
4. **Smart Filtering** - Skips locally-built images (e.g., compose project images)

### Container Management

- **Standalone Containers** - Individual containers managed independently
- **Compose Projects** - Grouped containers managed as a unit using `docker compose`
- **Lifecycle Operations** - Start, stop, restart containers with a single click
- **Update Operations** - Recreate containers with the latest image while preserving configuration

## UI Overview

### Grid View

The main dashboard shows all containers in a card-based grid:

- **Green border** - Container is up to date
- **Orange border** - Update available
- **Blue badge** - Compose project with container count
- **Gray badge** - Standalone container

### Detail View

Click any container to see:

- Container status and metadata
- Individual container controls (start/stop/restart)
- Update button (when updates are available)
- All containers in a compose project

### Visual Indicators

- 🟢 **Green dot** - Container is running
- ⚫ **Gray dot** - Container is stopped
- 🟠 **Pulsing orange dot** - Update available

## Architecture

```
bleeding-edge/
├── cmd/server/          # Application entry point
├── internal/
│   ├── docker/          # Docker client wrapper
│   ├── handlers/        # HTTP request handlers
│   ├── models/          # Data structures
│   └── services/        # Business logic
│       ├── container.go # Container grouping and update detection
│       └── update.go    # Update operations
├── web/
│   ├── static/          # CSS and static assets
│   └── templates/       # HTML templates
└── docker-compose.yml   # Deployment configuration
```

## Development

### Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- Access to Docker socket

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/services/...

# Run integration tests (requires Docker)
go test ./internal/integration/...
```

### Project Structure

- **Handlers** - HTTP request handling and routing
- **Services** - Core business logic for container management
- **Docker Client** - Abstraction layer over Docker API
- **Models** - Data structures for containers and groups
- **Templates** - Server-side rendered HTML with Go templates

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Main dashboard (grid view) |
| `GET` | `/container/:id` | Container detail page |
| `POST` | `/container/:id/update` | Update container/project |
| `POST` | `/container/:id/start` | Start container |
| `POST` | `/container/:id/stop` | Stop container |
| `POST` | `/container/:id/restart` | Restart container |
| `GET` | `/static/*` | Static assets (CSS, etc.) |

## Security Considerations

⚠️ **Important**: BleedingEdge requires access to the Docker socket, which provides root-level access to the host system.

- Only run BleedingEdge in trusted environments
- Consider using Docker socket proxy for production deployments
- Implement authentication/authorization for production use
- Review container permissions and network access

## Troubleshooting

### Container not showing updates

- Ensure the container uses a mutable tag (e.g., `nginx:latest` not `nginx@sha256:...`)
- Check that the image is pullable from a registry (not locally built)
- Verify network connectivity to Docker registries
- Check logs: `docker logs bleeding-edge`

### Cannot connect to Docker daemon

```bash
# Verify Docker socket permissions
ls -la /var/run/docker.sock

# Check Docker is running
docker ps

# Verify socket mount in container
docker inspect bleeding-edge | grep docker.sock
```

### Update operation fails

- Check container logs for detailed error messages
- Verify sufficient disk space for new images
- Ensure no conflicting container names
- For compose projects, verify `docker-compose.yml` is accessible

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Go](https://golang.org/)
- UI powered by [Tailwind CSS](https://tailwindcss.com/), [htmx](https://htmx.org/), and [Alpine.js](https://alpinejs.dev/)
- Docker integration via [Docker Engine API](https://docs.docker.com/engine/api/)

## Roadmap

- [ ] Multi-host Docker support (Docker Swarm, remote hosts)
- [ ] Authentication and user management
- [ ] Scheduled automatic updates
- [ ] Webhook notifications
- [ ] Container resource monitoring
- [ ] Image vulnerability scanning
- [ ] Backup/restore container configurations

---

**Made with ❤️ for the Docker community**
