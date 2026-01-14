# Travel Calendar - Multi-stage Dockerfile
#
# This Dockerfile builds all project components in a single multi-stage build.
# Docker layer caching ensures rebuilds only affect changed components.
#
# Stages:
#   base-node      - Node.js base with pnpm
#   base-go        - Go base with CGO dependencies
#   deps           - Install pnpm dependencies
#   frontend-build - Build SvelteKit static site
#   backend-build  - Build Go backend binary (includes MCP server)
#   runtime        - Final minimal runtime image
#
# Usage:
#   docker compose build           # Build (uses cache)
#   docker compose build --no-cache  # Fresh build
#   docker compose up              # Start services
#
# Testing (via docker run):
#   docker run --rm travel-calendar go test ./packages/backend/...
#   docker run --rm travel-calendar pnpm --filter frontend test

# ===========================================
# Stage: base-node - Node.js with pnpm
# ===========================================
FROM node:20-alpine AS base-node

# Install pnpm via corepack
RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# ===========================================
# Stage: base-go - Go with CGO dependencies
# ===========================================
FROM golang:1.23-alpine AS base-go

# CGO is required for mattn/go-sqlite3
RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite \
    sqlite-dev \
    git

ENV CGO_ENABLED=1

WORKDIR /app

# ===========================================
# Stage: deps - Install pnpm dependencies
# ===========================================
FROM base-node AS deps

# Copy workspace configuration
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./

# Copy package.json files for each package
COPY packages/frontend/package.json ./packages/frontend/
COPY packages/shared/package.json ./packages/shared/

# Install dependencies
RUN pnpm install --frozen-lockfile || pnpm install

# ===========================================
# Stage: frontend-build - Build SvelteKit
# ===========================================
FROM deps AS frontend-build

# Copy shared package source (frontend depends on it)
COPY packages/shared ./packages/shared/

# Copy frontend source
COPY packages/frontend ./packages/frontend/

# Build the frontend
WORKDIR /app/packages/frontend
RUN pnpm build

# Output is in /app/packages/frontend/build/

# ===========================================
# Stage: backend-build - Build Go backend
# ===========================================
FROM base-go AS backend-build

# Copy go.mod and go.sum first for caching
COPY packages/backend/go.mod packages/backend/go.sum ./packages/backend/

# Download backend dependencies
WORKDIR /app/packages/backend
RUN go mod download

# Copy backend source
COPY packages/backend ./

# Copy frontend build output to embed location
COPY --from=frontend-build /app/packages/frontend/build ./cmd/server/dist/

# Build the backend binary
RUN go build -o /app/bin/server ./cmd/server

# ===========================================
# Stage: runtime - Final minimal image
# ===========================================
FROM alpine:3.19 AS runtime

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    sqlite-libs \
    curl

# Create non-root user
RUN addgroup -g 1000 app && adduser -u 1000 -G app -s /bin/sh -D app

WORKDIR /app

# Copy binary from build stage
COPY --from=backend-build /app/bin/server /app/bin/server

# Create data directory
RUN mkdir -p /app/data && chown -R app:app /app

USER app

# Default environment
ENV DATABASE_PATH=/app/data/travel.db
ENV PORT=3000

EXPOSE 3000

# Default to running the backend server
CMD ["/app/bin/server"]

# ===========================================
# Stage: dev - Development image with all tools
# ===========================================
FROM base-go AS dev

# Install Node.js and pnpm for frontend development
# Use npm to install pnpm since alpine's nodejs doesn't include corepack
# Also install Playwright browser dependencies
RUN apk add --no-cache \
    nodejs \
    npm \
    curl \
    bash \
    # Playwright Chromium dependencies
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont \
    # ffmpeg for Playwright video recording
    ffmpeg

RUN npm install -g pnpm

# Set Playwright to use system Chromium
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
ENV PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium-browser

# Copy workspace config and install dependencies
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./
COPY packages/frontend/package.json ./packages/frontend/
COPY packages/shared/package.json ./packages/shared/

RUN pnpm install --frozen-lockfile || pnpm install

# Install Playwright's ffmpeg for video recording (browsers use system Chromium)
RUN cd packages/frontend && pnpm exec playwright install ffmpeg 2>/dev/null || true

# Copy all source code
COPY packages ./packages/

# Download Go dependencies
WORKDIR /app/packages/backend
RUN go mod download

WORKDIR /app

# Default command for dev
CMD ["sh", "-c", "cd packages/backend && go run ./cmd/server"]
