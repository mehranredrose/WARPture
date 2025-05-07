FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o tunnel-agent ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /build/tunnel-agent .
EXPOSE 8080 8081
HEALTHCHECK --interval=10s --timeout=5s --retries=3 CMD curl -f http://localhost:8080/health || exit 1
ENV LOG_LEVEL=info HTTP_ADDR=:8080 WS_ADDR=:8081 WARP_CLI_PATH=warp-cli PROXY_HOST=127.0.0.1 PROXY_PORT=40000
ENTRYPOINT ["./tunnel-agent"]
