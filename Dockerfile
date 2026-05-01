# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o modelfleet .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates openssh-client

# Create data directory
RUN mkdir -p /app/data

# Copy binary from builder
COPY --from=builder /app/modelfleet /app/modelfleet

# Copy frontend
COPY --from=builder /app/frontend /app/frontend

# Expose port
EXPOSE 3456

# Volume for persistent data
VOLUME ["/app/data"]

# Run the application
ENTRYPOINT ["./modelfleet"]
