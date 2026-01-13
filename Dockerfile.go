# Go Development Dockerfile for Travel Calendar
#
# This image is used for Go services (backend, mcp-server).
# No hot-reload - Claude Code rebuilds containers as needed.
#
# Usage:
#   docker compose up backend      # Start backend service
#   docker compose up mcp          # Start MCP server
#   docker compose exec backend sh # Shell into container

FROM golang:1.23-alpine

# CGO is required for mattn/go-sqlite3
# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite \
    sqlite-dev \
    git \
    curl \
    bash

# Set working directory
WORKDIR /app

# Environment for CGO
ENV CGO_ENABLED=1

# Default command (overridden per-service in docker-compose.yml)
CMD ["go", "run", "./cmd/server"]
