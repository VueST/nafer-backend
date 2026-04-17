# ============================================================
# streaming-service Dockerfile
# !! Uses Debian (not Alpine) because FFmpeg requires libc !!
# Builds BOTH the API server and the worker in one image.
# The entrypoint is overridden per service in docker-compose.yml.
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.26-bookworm AS builder

ENV GOTOOLCHAIN=local
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries
RUN go build -ldflags="-s -w" -o /out/api    ./cmd/api
RUN go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# ── Stage 2: Runtime ─────────────────────────────────────────
# Use Debian slim + FFmpeg — this is why streaming uses a different base
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    wget \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/api    ./api
COPY --from=builder /out/worker ./worker

# Create working directory for FFmpeg temp files
RUN mkdir -p /tmp/nafer-transcode

# Non-root user
RUN groupadd -r nafer && useradd -r -g nafer nafer
RUN chown -R nafer:nafer /app /tmp/nafer-transcode
USER nafer

EXPOSE 8080

# Default entrypoint is the HTTP API.
# Override to "./worker" in docker-compose.yml for the worker replica.
ENTRYPOINT ["./api"]
