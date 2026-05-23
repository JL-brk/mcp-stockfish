# Deployment guide

## Current deploy target

The repository is prepared for Fly.io deployment through `fly.toml`.

The service runs as a Docker app:

```text
public HTTPS URL
  -> Fly.io app
  -> Docker container
  -> mcp-stockfish HTTP server
  -> Stockfish binary
```

## Important

ChatGPT cannot provision the external Fly.io account from inside this repository alone. The Fly.io app must be created from an authenticated Fly.io CLI session or through a connected deployment provider.

## One-time Fly.io setup

Install and authenticate Fly CLI:

```bash
brew install flyctl
fly auth login
```

Clone and enter the repo:

```bash
git clone https://github.com/JL-brk/mcp-stockfish.git
cd mcp-stockfish
```

Create the Fly app from the existing config:

```bash
fly launch --copy-config --no-deploy
```

If the app name `mcp-stockfish` is already taken, choose a unique app name and let Fly update `fly.toml`.

Deploy:

```bash
fly deploy
```

Check health:

```bash
fly status
fly logs
curl https://YOUR-FLY-APP.fly.dev/healthz
```

The expected health response is:

```text
ok
```

## MCP endpoint

After deployment, the MCP endpoint is:

```text
https://YOUR-FLY-APP.fly.dev/mcp
```

## Runtime environment

The deployment config sets:

```text
MCP_STOCKFISH_SERVER_MODE=http
MCP_STOCKFISH_HTTP_HOST=0.0.0.0
MCP_STOCKFISH_HTTP_PORT=8080
MCP_STOCKFISH_LOG_LEVEL=info
MCP_STOCKFISH_LOG_OUTPUT=stderr
```

The Docker image sets:

```text
MCP_STOCKFISH_PATH=/usr/bin/stockfish
```

## Local Docker test

Build locally:

```bash
docker build -t mcp-stockfish .
```

Run locally in HTTP mode:

```bash
docker run --rm -p 8080:8080 \
  -e MCP_STOCKFISH_SERVER_MODE=http \
  -e MCP_STOCKFISH_HTTP_HOST=0.0.0.0 \
  -e MCP_STOCKFISH_HTTP_PORT=8080 \
  mcp-stockfish
```

Check health:

```bash
curl http://localhost:8080/healthz
```

Expected:

```text
ok
```

## Notes

- This is an always-on service. It should run on a Docker-capable host, not as a short-lived serverless function.
- Stockfish is a native process. A persistent Docker service is the safest first deployment model.
- The public MCP endpoint should be connected to ChatGPT only after `/healthz` works and the `/mcp` endpoint is reachable over HTTPS.
