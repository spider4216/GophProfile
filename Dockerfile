# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o crt ./cmd/crt

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app


WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/worker .
COPY --from=builder /app/crt .
COPY --from=builder /app/web ./web/

RUN mkdir -p /app/certs \
    && ./crt \
    && chown -R 10001:10001 /app

USER 10001:10001