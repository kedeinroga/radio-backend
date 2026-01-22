# ============================================
# Stage 1: Dependencies (cached layer)
# ============================================
FROM golang:1.24-alpine AS dependencies

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (this layer is cached if go.mod/go.sum don't change)
RUN go mod download && go mod verify

# ============================================
# Stage 2: Build
# ============================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

WORKDIR /app

# Copy dependencies from previous stage
COPY --from=dependencies /go/pkg /go/pkg

# Copy go mod files
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Generate Swagger documentation
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Build the application with optimizations for Cloud Run
# - CGO_ENABLED=0: Static binary (no C dependencies)
# - -ldflags="-s -w": Strip debug info and symbol table (reduce size)
# - -trimpath: Remove file system paths from binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -trimpath \
    -o radio-backend \
    ./cmd/server

# ============================================
# Stage 3: Runtime (minimal production image)
# ============================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /app/radio-backend .

# Copy necessary runtime files
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/locales ./locales

# Switch to non-root user (distroless default)
USER nonroot:nonroot

# Cloud Run expects applications to listen on $PORT (defaults to 8080)
ENV PORT=8080

# Expose port (documentation only, Cloud Run ignores this)
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/radio-backend"]
