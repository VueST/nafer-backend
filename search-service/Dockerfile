# ============================================================
# search-service Dockerfile
# Multi-stage build: builder (full Go SDK) → runtime (Alpine)
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.26-alpine AS builder

ENV GOTOOLCHAIN=local
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /out/server ./cmd/server

# ── Stage 2: Runtime ─────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache wget curl ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/server .

RUN addgroup -S nafer && adduser -S nafer -G nafer
USER nafer

EXPOSE 8080

ENTRYPOINT ["./server"]
