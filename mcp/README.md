# Sources MCP Server

Model Context Protocol (MCP) server for Red Hat Hybrid Cloud Console Sources API integration.

## Overview

This MCP server exposes read-only tools for querying Sources/Integrations data through Claude Desktop and other MCP clients. It acts as a sidecar service that forwards authenticated requests to the Sources API.

## Available Tools

The following MCP tools are available:

- **sources_list** - List all sources for authenticated tenant with optional filtering
  - Parameters: `filter` (object, optional), `limit` (integer, default: 100, max: 1000)
  
- **sources_get** - Get a specific source by ID
  - Parameters: `id` (string, required)
  
- **applications_list** - List all applications for authenticated tenant
  - Parameters: `limit` (integer, default: 100)
  
- **applications_get** - Get a specific application by ID
  - Parameters: `id` (string, required)
  
- **endpoints_list** - List all endpoints for authenticated tenant
  - Parameters: `limit` (integer, default: 100)
  
- **application_types_list** - List all available application types (metadata)
  - No parameters required
  
- **source_types_list** - List all available source types (metadata)
  - No parameters required

## Development

### Prerequisites

- Go 1.24 or later
- Access to a running sources-api-go instance

### Running Locally

```bash
# Build the server
cd mcp
go build -o sources-mcp .

# Run with environment variables
export SOURCES_API_URL=http://localhost:8000
export PORT=8080
export METRICS_PORT=9000
export LOG_LEVEL=INFO

./sources-mcp
```

### Running Tests

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests (requires sources-api running)
go test -tags=integration ./...
```

### Building Docker Image

```bash
# From repository root
docker build -f Dockerfile.mcp -t sources-mcp:latest .
```

## Configuration

The server is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Main HTTP server port | 8080 |
| `METRICS_PORT` | Prometheus metrics port | 9000 |
| `SOURCES_API_URL` | Sources API base URL | http://sources-api-svc:8000 |
| `LOG_LEVEL` | Log level (DEBUG, INFO, WARN, ERROR) | INFO |

## Endpoints

- `GET /health` - Health check endpoint
- `POST /_private/mcp` - MCP protocol endpoint (requires `x-rh-identity` header)
- `GET /metrics` - Prometheus metrics (on metrics port)

## Authentication

All MCP tool calls require an `x-rh-identity` header containing a base64-encoded JSON object with tenant information. This header is forwarded to the Sources API for authentication and tenant filtering.

Example identity header:
```json
{
  "identity": {
    "account_number": "12345",
    "org_id": "67890",
    "type": "User",
    "user": {
      "username": "testuser"
    }
  }
}
```

## Claude Desktop Integration

For local development with Claude Desktop, use the included stdio wrapper script:

### Setup

1. Start the MCP server locally:
```bash
cd mcp
export SOURCES_API_URL=http://localhost:8000
./sources-mcp
```

2. Configure Claude Desktop by adding to `~/.config/claude/claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "sources-mcp": {
      "command": "/path/to/sources-api-go/mcp/sources-mcp-stdio-wrapper.sh",
      "args": []
    }
  }
}
```

3. Restart Claude Desktop

The wrapper script bridges Claude's stdio protocol to the MCP server's HTTP endpoint. You can customize the identity by setting environment variables:

```bash
export ACCOUNT_NUMBER=12345
export ORG_ID=67890
export USERNAME=testuser
```

Or by modifying the script directly.

## Deployment

See the main repository documentation for deployment instructions to stage and production environments.

### ClowdApp

The MCP server is deployed as a separate ClowdApp (`sources-mcp`) alongside the main Sources API. It has no database or Kafka dependencies and communicates with the Sources API via HTTP.

### Konflux

The server is built and deployed via Konflux CI/CD pipelines:
- Push to main: `sources-mcp-push.yaml`
- Pull requests: `sources-mcp-pull-request.yaml`

## Architecture

```
Claude Desktop / MCP Client
    |
    | (with x-rh-identity header)
    v
Sources MCP Server (:8080/_private/mcp)
    |
    | (forwards x-rh-identity)
    v
Sources API (:8000/api/sources/v3.1/*)
    |
    v
PostgreSQL
```

## Metrics

Prometheus metrics are exposed on the metrics port (default 9000) at `/metrics`. Key metrics include:

- HTTP request counts and durations
- Tool call counts
- Error rates

## Troubleshooting

### Server won't start

Check that the Sources API is accessible:
```bash
curl http://sources-api-svc:8000/health
```

### Authentication failures

Verify the `x-rh-identity` header is properly formatted:
```bash
echo "$XRH_IDENTITY" | base64 -d | jq .
```

### Tool calls failing

Check the MCP server logs for error details:
```bash
kubectl logs deployment/sources-mcp-svc -n sources-stage
```

## Contributing

1. Make changes in the `mcp/` directory
2. Run tests: `go test ./...`
3. Build locally: `go build .`
4. Create a pull request
5. Konflux will automatically build and test the changes

## License

See LICENSE file in the repository root.
