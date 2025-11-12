# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies (allow Go toolchain auto-download)
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o bleeding-edge ./cmd/server

# Runtime stage
FROM alpine:latest

# Install docker-cli and docker-compose
RUN apk --no-cache add ca-certificates docker-cli docker-cli-compose

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bleeding-edge .

# Copy web assets
COPY --from=builder /app/web ./web

# Expose port 8080
EXPOSE 8080

# Run the application
CMD ["./bleeding-edge"]
