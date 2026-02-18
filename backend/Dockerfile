FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /out/anomaly-api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -S anomaly && adduser -S anomaly -G anomaly

WORKDIR /app

COPY --from=builder /out/anomaly-api /usr/local/bin/anomaly-api

USER anomaly

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/anomaly-api"]
