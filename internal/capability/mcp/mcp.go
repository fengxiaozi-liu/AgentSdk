package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"ferryman-agent/internal/security/permission"
	toolcore "ferryman-agent/internal/tools"

	"github.com/mark3labs/mcp-go/mcp"
)

type MCPType string

const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"
)

type MCPServer struct {
	Command string            `json:"command,omitempty"`
	Env     []string          `json:"env,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Type    MCPType           `json:"type,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type MCPClient interface {
	Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Close() error
}

type McpTool struct {
	mcpName     string
	tool        mcp.Tool
	server      MCPServer
	permissions permission.Service
	workingDir  string
}

func NewMcpTool(name string, tool mcp.Tool, permissions permission.Service, workingDir string, server MCPServer) toolcore.BaseTool {
	return &McpTool{
		mcpName:     name,
		tool:        tool,
		server:      server,
		permissions: permissions,
		workingDir:  workingDir,
	}
}

func (b *McpTool) Info() toolcore.ToolInfo {
	required := b.tool.InputSchema.Required
	if required == nil {
		required = make([]string, 0)
	}
	return toolcore.ToolInfo{
		Name:        fmt.Sprintf("%s_%s", b.mcpName, b.tool.Name),
		Description: b.tool.Description,
		Parameters:  b.tool.InputSchema.Properties,
		Required:    required,
	}
}

func (b *McpTool) Run(ctx context.Context, params toolcore.ToolCall) (toolcore.ToolResponse, error) {
	sessionID, messageID := toolcore.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return toolcore.ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	permissionDescription := fmt.Sprintf("execute %s with the following parameters: %s", b.Info().Name, params.Input)
	p := b.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        b.workingDir,
			ToolName:    b.Info().Name,
			Action:      "execute",
			Description: permissionDescription,
			Params:      params.Input,
		},
	)
	if !p {
		return toolcore.NewTextErrorResponse("permission denied"), nil
	}

	c, err := newClient(b.server)
	if err != nil {
		return toolcore.NewTextErrorResponse(err.Error()), nil
	}
	return runTool(ctx, c, b.tool.Name, params.Input)
}

func runTool(ctx context.Context, c MCPClient, toolName string, input string) (toolcore.ToolResponse, error) {
	defer c.Close()
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "ferry-agent",
		Version: "1.0.0",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		return toolcore.NewTextErrorResponse(err.Error()), nil
	}

	toolRequest := mcp.CallToolRequest{}
	toolRequest.Params.Name = toolName
	var args map[string]any
	if err = json.Unmarshal([]byte(input), &args); err != nil {
		return toolcore.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	toolRequest.Params.Arguments = args
	result, err := c.CallTool(ctx, toolRequest)
	if err != nil {
		return toolcore.NewTextErrorResponse(err.Error()), nil
	}

	output := ""
	for _, v := range result.Content {
		if v, ok := v.(mcp.TextContent); ok {
			output = v.Text
		} else {
			output = fmt.Sprintf("%v", v)
		}
	}

	return toolcore.NewTextResponse(output), nil
}
