# Build stage
FROM golang:1.22.2-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    libusb-1.0-0-dev \
    pkg-config \
    gcc \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN go build -o inspect ./cmd/inspect
RUN if [ -d "./cmd/audio_stream" ]; then go build -o audio_stream ./cmd/audio_stream; fi

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libusb-1.0-0 \
    usbutils \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN useradd -m -u 1000 appuser

WORKDIR /app

# Copy built binaries from builder
COPY --from=builder /app/inspect /app/inspect
COPY --from=builder /app/audio_stream* /app/ || true

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Default command (can be overridden)
CMD ["/app/inspect", "-help"]