# Blinko MCP Server (Go)

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for [Blinko](https://github.com/blinko-space/blinko) — the self-hosted note service.

Built with Go using [mcp-go](https://github.com/mark3labs/mcp-go). Docker image ~24MB, runtime memory ~1.5MB. Built-in Bearer token authentication — no external proxy needed.

## Features

| Tool | Description |
|------|-------------|
| `upsert_blinko_flash_note` | Create a flash note (type 0) |
| `upsert_blinko_note` | Create a normal note (type 1) |
| `upsert_blinko_todo` | Create a todo (type 2) |
| `share_blinko_note` | Share a note or cancel sharing |
| `search_blinko_notes` | Search notes with filters |
| `review_blinko_daily_notes` | Get daily review notes |
| `clear_blinko_recycle_bin` | Clear the recycle bin |

## Quick Start

### Docker Compose

```yaml
services:
  blinko-mcp:
    image: blinko-mcp-go:latest
    container_name: blinko-mcp-go
    restart: unless-stopped
    ports:
      - "8795:8080"
    environment:
      - BLINKO_DOMAIN=https://your-blinko.com
      - BLINKO_API_KEY=your-api-key
      - MCP_TRANSPORT=sse
      - MCP_AUTH_TOKEN=your-secret-token
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BLINKO_DOMAIN` | Blinko instance URL | `http://localhost:1111` |
| `BLINKO_API_KEY` | Blinko API key | — |
| `MCP_TRANSPORT` | Transport: `stdio`, `sse`, or `http` | `stdio` |
| `MCP_PORT` | Port for SSE/HTTP | `8080` |
| `MCP_AUTH_TOKEN` | Bearer token for MCP auth (optional) | — |

## Build

```bash
docker build -t blinko-mcp-go:latest .
```

## MCP Client Configuration

### SSE (for RikkaHub, etc.)

```json
{
  "type": "streamable_http",
  "url": "http://your-server:8795/sse",
  "headers": {
    "Authorization": "Bearer your-auth-token"
  }
}
```

### Stdio (for Cursor, etc.)

```json
{
  "mcpServers": {
    "blinko": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-e", "BLINKO_DOMAIN=https://your-blinko.com", "-e", "BLINKO_API_KEY=your-key", "blinko-mcp-go:latest"]
    }
  }
}
```

## Compared to mcp-server-blinko (Node.js)

| | Node.js (original) | Go (this) |
|---|---|---|
| Runtime | Node.js + npx | Single binary |
| Image size | ~200MB | ~24MB |
| Memory | ~50-100MB | ~1.5MB |
| External proxy | Needs mcp-proxy + auth-proxy | Built-in SSE + auth |
| Processes | 3 | 1 |

## License

MIT
