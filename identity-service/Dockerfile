# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.26-alpine AS builder

# Prevent automatic toolchain upgrades that could pull Go 1.25
ENV GOTOOLCHAIN=local

WORKDIR /app

# Cache dependencies layer separately — only re-runs when go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /identity-service cmd/server/main.go

# =============================================================================
# Stage 2: Runtime (minimal image)
# =============================================================================
FROM alpine:3.20

# Install ca-certificates for HTTPS calls and wget for healthcheck
RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /identity-service .

EXPOSE 8080

CMD ["./identity-service"]
