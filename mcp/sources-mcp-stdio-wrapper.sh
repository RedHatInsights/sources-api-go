#!/bin/bash
# Wrapper script to bridge stdio to HTTP for Claude Desktop integration
# This script forwards MCP protocol messages from stdin to the running MCP server

# Configuration - can be overridden via environment variables
MCP_HOST="${MCP_HOST:-localhost}"
MCP_PORT="${MCP_PORT:-8080}"
MCP_ENDPOINT="${MCP_ENDPOINT:-/_private/mcp}"

# Test identity for local development - can be customized
ACCOUNT_NUMBER="${ACCOUNT_NUMBER:-12345}"
ORG_ID="${ORG_ID:-67890}"
USERNAME="${USERNAME:-testuser}"

IDENTITY="{\"identity\":{\"account_number\":\"${ACCOUNT_NUMBER}\",\"org_id\":\"${ORG_ID}\",\"type\":\"User\",\"user\":{\"username\":\"${USERNAME}\"}}}"
XRH_IDENTITY=$(echo -n "$IDENTITY" | base64 -w 0)

# Read from stdin and forward to HTTP endpoint, return response to stdout
while IFS= read -r line; do
    curl -s -X POST "http://${MCP_HOST}:${MCP_PORT}${MCP_ENDPOINT}" \
      -H "Content-Type: application/json" \
      -H "x-rh-identity: $XRH_IDENTITY" \
      -d "$line"
done
