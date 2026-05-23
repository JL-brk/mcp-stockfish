# Build stage
FROM golang:1.23-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN go get github.com/mark3labs/mcp-go@481f05674f583f20ce114d9e7efdcc6348d792e7

COPY *.go ./

ARG VERSION=dev
ARG COMMIT_HASH
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o mcp-stockfish .

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        stockfish \
        ca-certificates \
        wget \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 mcpuser \
    && useradd -m -u 1000 -g mcpuser -s /usr/sbin/nologin mcpuser

# Copy binary from builder stage
COPY --from=builder /app/mcp-stockfish /usr/local/bin/mcp-stockfish
RUN chmod +x /usr/local/bin/mcp-stockfish

# Set environment
ENV MCP_STOCKFISH_PATH=/usr/games/stockfish
ENV MCP_STOCKFISH_HTTP_PORT=8080
ENV PATH="/usr/local/bin:/usr/games:${PATH}"

# Switch to non-root user
USER mcpuser
WORKDIR /home/mcpuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD if [ "$MCP_STOCKFISH_SERVER_MODE" = "http" ]; then wget -qO- "http://127.0.0.1:${PORT:-${MCP_STOCKFISH_HTTP_PORT}}/healthz" >/dev/null; else echo '{"jsonrpc": "2.0", "method": "ping", "id": 1}' | timeout 2 mcp-stockfish >/dev/null; fi || exit 1

ENTRYPOINT ["mcp-stockfish"]
