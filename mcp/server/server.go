package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/mcp/client"
	"github.com/RedHatInsights/sources-api-go/mcp/config"
	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

type MCPServer struct {
	mcpServer         *server.MCPServer
	streamableServer  *server.StreamableHTTPServer
	sourcesClient     *client.SourcesClient
	log               *logrus.Logger
}

// NewMCPServer creates and configures a new MCP server instance
func NewMCPServer(cfg *config.Config) (*MCPServer, error) {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(level)
	}

	sourcesClient := client.NewSourcesClient(cfg.SourcesAPIURL, log)

	mcpServer := server.NewMCPServer(
		"sources-mcp",
		"1.0.0",
		server.WithLogging(),
	)

	s := &MCPServer{
		mcpServer:     mcpServer,
		sourcesClient: sourcesClient,
		log:           log,
	}

	s.registerTools()

	s.streamableServer = server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			if xrhIdentity := r.Header.Get("x-rh-identity"); xrhIdentity != "" {
				return context.WithValue(ctx, "x-rh-identity", xrhIdentity)
			}
			return ctx
		}),
	)

	return s, nil
}

// HandleHTTP processes incoming MCP requests over HTTP using Echo
func (s *MCPServer) HandleHTTP(c echo.Context) error {
	xrhIdentity := c.Request().Header.Get("x-rh-identity")
	if xrhIdentity == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "x-rh-identity header required",
		})
	}

	s.streamableServer.ServeHTTP(c.Response().Writer, c.Request())
	return nil
}

// registerTools registers all available MCP tools
func (s *MCPServer) registerTools() {
	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "sources_list",
			Description: "List all sources for the authenticated tenant with optional filtering",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"filter": map[string]interface{}{
						"type":        "object",
						"description": "Filter criteria (name, source_type_id, etc.)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results (default: 100, max: 1000)",
					},
				},
			},
		},
		s.handleSourcesList,
	)

	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "sources_get",
			Description: "Get a specific source by ID",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Source ID",
					},
				},
				Required: []string{"id"},
			},
		},
		s.handleSourcesGet,
	)

	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "applications_list",
			Description: "List all applications for the authenticated tenant",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
					},
				},
			},
		},
		s.handleApplicationsList,
	)

	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "applications_get",
			Description: "Get a specific application by ID",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Application ID",
					},
				},
				Required: []string{"id"},
			},
		},
		s.handleApplicationsGet,
	)

	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "endpoints_list",
			Description: "List all endpoints for the authenticated tenant",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
					},
				},
			},
		},
		s.handleEndpointsList,
	)

	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "application_types_list",
			Description: "List all available application types (metadata, not tenant-specific)",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
			},
		},
		s.handleApplicationTypesList,
	)

	s.mcpServer.AddTool(
		mcp.Tool{
			Name:        "source_types_list",
			Description: "List all available source types (metadata, not tenant-specific)",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
			},
		},
		s.handleSourceTypesList,
	)
}

// Tool handler implementations
func (s *MCPServer) handleSourcesList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	limit := request.GetInt("limit", 100)
	if limit > 1000 {
		limit = 1000
	}

	var filter map[string]interface{}
	if filterVal := request.GetArguments()["filter"]; filterVal != nil {
		if f, ok := filterVal.(map[string]interface{}); ok {
			filter = f
		}
	}

	sources, err := s.sourcesClient.ListSources(ctx, xrhIdentity, filter, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list sources: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", sources)), nil
}

func (s *MCPServer) handleSourcesGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id parameter is required"), nil
	}

	source, err := s.sourcesClient.GetSource(ctx, xrhIdentity, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get source: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", source)), nil
}

func (s *MCPServer) handleApplicationsList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	limit := request.GetInt("limit", 100)

	applications, err := s.sourcesClient.ListApplications(ctx, xrhIdentity, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list applications: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", applications)), nil
}

func (s *MCPServer) handleApplicationsGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("id parameter is required"), nil
	}

	application, err := s.sourcesClient.GetApplication(ctx, xrhIdentity, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get application: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", application)), nil
}

func (s *MCPServer) handleEndpointsList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	limit := request.GetInt("limit", 100)

	endpoints, err := s.sourcesClient.ListEndpoints(ctx, xrhIdentity, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list endpoints: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", endpoints)), nil
}

func (s *MCPServer) handleApplicationTypesList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	applicationTypes, err := s.sourcesClient.ListApplicationTypes(ctx, xrhIdentity)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list application types: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", applicationTypes)), nil
}

func (s *MCPServer) handleSourceTypesList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	xrhIdentity, ok := ctx.Value("x-rh-identity").(string)
	if !ok || xrhIdentity == "" {
		return mcp.NewToolResultError("x-rh-identity header required"), nil
	}

	sourceTypes, err := s.sourcesClient.ListSourceTypes(ctx, xrhIdentity)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list source types: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", sourceTypes)), nil
}
