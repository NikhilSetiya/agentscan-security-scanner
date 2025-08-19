# Build stage
FROM golang:1.23.9-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agentscan-security-scanner ./cmd/api

# Final stage
FROM alpine:3.20

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/agentscan-security-scanner .

# Expose port (DigitalOcean will set PORT env var)
EXPOSE 8080

# Run the binary
CMD ["./agentscan-security-scanner"]