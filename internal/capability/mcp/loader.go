package mcp

import (
	"context"
	"ferryman-agent/internal/data/logging"
	"ferryman-agent/internal/security/permission"
	toolcore "ferryman-agent/internal/tools"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

type MCPToolLoader interface {
	Load(ctx context.Context) (map[string]MCPServer, error)
}

func LoadTools(ctx context.Context, servers map[string]MCPServer, permissions permission.Service, workingDir string) ([]toolcore.BaseTool, error) {
	tools := make([]toolcore.BaseTool, 0)
	for name, server := range servers {
		c, err := newClient(server)
		if err != nil {
			logging.Error("error creating mcp client", "name", name, "error", err)
			continue
		}
		serverTools := loadServerTools(ctx, name, server, permissions, workingDir, c)
		tools = append(tools, serverTools...)
	}
	return tools, nil
}

func loadServerTools(ctx context.Context, name string, server MCPServer, permissions permission.Service, workingDir string, c MCPClient) []toolcore.BaseTool {
	defer c.Close()

	initRequest := mcpsdk.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcpsdk.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcpsdk.Implementation{
		Name:    "ferry-agent",
		Version: "1.0.0",
	}

	if _, err := c.Initialize(ctx, initRequest); err != nil {
		logging.Error("error initializing mcp client", "name", name, "error", err)
		return nil
	}
	toolsRequest := mcpsdk.ListToolsRequest{}
	availableTools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		logging.Error("error listing tools", "name", name, "error", err)
		return nil
	}

	tools := make([]toolcore.BaseTool, 0, len(availableTools.Tools))
	for _, tool := range availableTools.Tools {
		tools = append(tools, NewMcpTool(name, tool, permissions, workingDir, server))
	}
	return tools
}
