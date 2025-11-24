# Single-stage Dockerfile for anomaly-detector backend
FROM golang:1.24-alpine

# Install dependencies
RUN apk add --no-cache git build-base ca-certificates

# Create a non-root user
RUN addgroup -S anomaly && adduser -S anomaly -G anomaly

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Build the API binary
ENV CGO_ENABLED=0
RUN go mod tidy && \
    go build -o /usr/local/bin/anomaly-api ./cmd/api && \
    chown anomaly:anomaly /usr/local/bin/anomaly-api

# Switch to non-root user
USER anomaly

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/usr/local/bin/anomaly-api"]
