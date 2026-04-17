# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.26-alpine AS builder

ENV GOTOOLCHAIN=local

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /media-service cmd/server/main.go

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.20

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /media-service .

EXPOSE 8080

CMD ["./media-service"]
