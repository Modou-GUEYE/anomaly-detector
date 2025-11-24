# Multi-stage Dockerfile for anomaly-detector (builds the Go binary and produces a small runtime image)

# Build stage
FROM golang:1.24-alpine AS build
RUN apk add --no-cache git build-base
WORKDIR /src
# Copy the entire repo so the build can reference modules and subfolders
COPY . .
# Build the API binary (disable CGO for static binary)
WORKDIR /src
ENV CGO_ENABLED=0
RUN go mod tidy
RUN go build -o /out/anomaly-api ./cmd/api

# Final stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
# Create a non-root user
RUN addgroup -S anomaly && adduser -S anomaly -G anomaly
# Copy binary from build stage
COPY --from=build /out/anomaly-api /usr/local/bin/anomaly-api
RUN chown anomaly:anomaly /usr/local/bin/anomaly-api
USER anomaly
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/anomaly-api"]
