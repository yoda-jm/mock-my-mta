# Stage 1: build the Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: minimal runtime image
FROM alpine:3.19
RUN apk add --no-cache curl
WORKDIR /app

COPY --from=builder /app/server ./server
COPY cmd/server/config/ ./cmd/server/config/
COPY e2e/testdata/ ./e2e/testdata/

RUN mkdir -p new-data

EXPOSE 8025 1025

HEALTHCHECK --interval=2s --timeout=5s --retries=20 \
  CMD curl -sf http://localhost:8025/ > /dev/null || exit 1

CMD ["./server", "--init-with-test-data", "e2e/testdata/emails"]
