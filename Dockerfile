# Multi-stage build for ocp-doc-checker
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o ocp-doc-checker .

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -u 1000 checker

# Set working directory
WORKDIR /workspace

# Copy binary from builder
COPY --from=builder /build/ocp-doc-checker /usr/local/bin/ocp-doc-checker

# Make sure binary is executable
RUN chmod +x /usr/local/bin/ocp-doc-checker

# Switch to non-root user
USER checker

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/ocp-doc-checker"]

# Default command (shows help)
CMD ["--help"]

