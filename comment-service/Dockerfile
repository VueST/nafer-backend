# ============================================================
# comment-service Dockerfile
# Multi-stage build: builder (full Go SDK) → runtime (Alpine)
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Prevent automatic Go toolchain upgrades
ENV GOTOOLCHAIN=local
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app

# Copy dependency manifests first — Docker will cache this layer
# as long as go.mod/go.sum don't change
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN go build -ldflags="-s -w" -o /out/server ./cmd/server

# ── Stage 2: Runtime ─────────────────────────────────────────
FROM alpine:3.20

# wget is needed for the Docker Compose healthcheck
RUN apk add --no-cache wget ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/server .

# Non-root user for security
RUN addgroup -S nafer && adduser -S nafer -G nafer
USER nafer

EXPOSE 8080

ENTRYPOINT ["./server"]
