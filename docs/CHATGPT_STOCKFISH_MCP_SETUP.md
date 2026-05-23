# ChatGPT Stockfish MCP setup

## Status

This repository already contains a working MCP server wrapper around Stockfish in Go.

Current confirmed structure:

- `mcp-stockfish` exposes a `chess_engine` MCP tool.
- The tool accepts raw UCI commands such as `uci`, `isready`, `position fen ...`, `go depth ...`, and `go movetime ...`.
- The default transport is `stdio`.
- The repository contains a Dockerfile that installs Stockfish inside the runtime image.

## Important limitation

The current `http` mode in `main.go` is not production-ready yet. At the moment, `runHTTPServer(...)` returns `nil` immediately instead of starting a real HTTP MCP server.

That means the repo can already work as a local stdio MCP server, for example in Claude Desktop, Cursor, or another local MCP client, but it is not yet ready as a hosted remote MCP server for ChatGPT Apps.

## Working local MCP setup

Build and install:

```bash
git clone https://github.com/JL-brk/mcp-stockfish.git
cd mcp-stockfish
make install
```

Make sure Stockfish is available:

```bash
stockfish
```

If Stockfish is installed somewhere else, set:

```bash
export MCP_STOCKFISH_PATH=/path/to/stockfish
```

Example local MCP client config:

```json
{
  "mcpServers": {
    "stockfish": {
      "command": "mcp-stockfish",
      "env": {
        "MCP_STOCKFISH_LOG_LEVEL": "info"
      }
    }
  }
}
```

## How to use the tool

First initialize the engine:

```text
uci
isready
```

Then set a position:

```text
position fen r4k1r/8/3q4/1N4p1/pP2Rp2/P2p1P1P/3Q2P1/6K1 b - - 0 1
```

Then ask for analysis:

```text
go depth 18
```

For faster interactive use:

```text
go movetime 3000
```

## Recommended next step for ChatGPT

To use this from ChatGPT as an app/MCP server, the repo needs one extra technical layer:

1. Implement a real HTTP or Streamable HTTP MCP transport.
2. Deploy it on a stable public HTTPS endpoint.
3. Add any required app metadata/security headers/authentication.
4. Connect that public MCP endpoint through the ChatGPT Apps/developer flow.

## Practical architecture

Recommended setup:

```text
ChatGPT
  -> public HTTPS MCP endpoint
  -> mcp-stockfish server
  -> Stockfish binary
```

For hosting, prefer a small always-on Docker host rather than a serverless runtime, because Stockfish is a long-running native process and analysis can take multiple seconds.

Good deployment targets:

- Fly.io
- Render Docker service
- Railway Docker service
- small VPS with Docker

Less ideal:

- Vercel serverless, because this repo runs a native engine process and currently has no real HTTP transport.

## Work needed in the repo

### 1. Fix HTTP mode

`main.go` currently has placeholder/commented HTTP server logic. Replace it with a real supported transport from the current `mark3labs/mcp-go` version or upgrade `mcp-go` and implement the supported HTTP transport.

### 2. Add CI

Add GitHub Actions to check:

```bash
go test ./...
go build ./...
docker build .
```

### 3. Add a hosted deployment profile

Add one deployment target, for example:

- `fly.toml`, or
- `render.yaml`, or
- Docker Compose for VPS use.

### 4. Add a safer analysis abstraction

The current MCP tool exposes raw UCI commands. That works, but it is easy for the model to use clumsily.

Recommended additional tool:

```text
analyze_position
```

Input:

```json
{
  "fen": "...",
  "depth": 18,
  "multipv": 3
}
```

Output:

```json
{
  "bestmove": "...",
  "evaluation": "...",
  "principal_variations": []
}
```

This is better for ChatGPT because the model can ask one structured question instead of managing UCI session state manually.

## Immediate conclusion

The repo is a good starting point, but it is not yet enough for a hosted ChatGPT MCP connection. It currently works best as a local stdio MCP server. The next concrete engineering task is to implement real HTTP MCP transport and deploy it as an always-on Docker service.