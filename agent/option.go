package agent

import toolcore "ferryman-agent/tools/core"

type AgentOption func(*agentOptions)

type agentOptions struct {
	tools     []toolcore.BaseTool
	promptKey string
}

func WithTools(tools ...toolcore.BaseTool) AgentOption {
	return func(opts *agentOptions) {
		opts.tools = append(opts.tools, tools...)
	}
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
