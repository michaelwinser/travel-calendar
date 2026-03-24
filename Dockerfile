# Stage 1: Build frontend
FROM node:20-alpine AS frontend
ENV PNPM_HOME="/root/.local/share/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/
RUN cd frontend && pnpm install --frozen-lockfile
COPY frontend/ frontend/
RUN cd frontend && pnpm build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist frontend/dist
RUN CGO_ENABLED=0 go build -o travel .

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/travel .
COPY app.yaml .
EXPOSE 8080
ENV PORT=8080
CMD ["./travel", "serve"]
