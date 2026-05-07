package agent

import (
	toolcore "ferryman-agent/internal/tools"
)

type AgentOption func(*agentOptions)

type agentOptions struct {
	tools               []toolcore.BaseTool
	enableAgentTool     bool
	enableMcpTool       bool
	enableWorkSpaceTool bool
	promptKey           string
}

func WithTools(tools ...toolcore.BaseTool) AgentOption {
	return func(opts *agentOptions) {
		opts.tools = append(opts.tools, tools...)
	}
}

func WithAgentTool() AgentOption {
	return func(opts *agentOptions) {
		opts.enableAgentTool = true
	}
}

func WithWorkSpaceTool() AgentOption {
	return func(opts *agentOptions) {
		opts.enableWorkSpaceTool = true
	}
}

func WithMcpTool() AgentOption {
	return func(opts *agentOptions) {
		opts.enableMcpTool = true
	}
}

func WithMCPTool() AgentOption {
	return WithMcpTool()
}

func WithPromptKey(key string) AgentOption {
	return func(opts *agentOptions) {
		opts.promptKey = key
	}
}

func applyAgentOptions(opts ...AgentOption) agentOptions {
	agentOpts := agentOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&agentOpts)
		}
	}
	return agentOpts
}
