package agenttool

import (
	"context"
	"encoding/json"
	"ferryman-agent/agent"
	sdkconfig "ferryman-agent/config"
	"fmt"

	"ferryman-agent/message"
	"ferryman-agent/prompt"
	"ferryman-agent/session"
	toolcore "ferryman-agent/tools/core"
	basetools "ferryman-agent/tools/workspace"
)

type Params struct {
	Prompt string `json:"prompt"`
}

type AgentTool struct {
	sessions session.Service
	messages message.Service
	prompts  prompt.Service
}

func NewAgentTool(sessions session.Service, messages message.Service, prompts prompt.Service) *AgentTool {
	return &AgentTool{sessions: sessions, messages: messages, prompts: prompts}
}

func (b *AgentTool) Info() toolcore.ToolInfo {
	return toolcore.ToolInfo{
		Name:        "agent_tool",
		Description: "Launch a new agent that has access to the following tools: GlobTool, GrepTool, LS, View. When you are searching for a keyword or file and are not confident that you will find the right match on the first try, use the Agent tool to perform the search for you. For example:\n\n- If you are searching for a keyword like \"config\" or \"logger\", or for questions like \"which file does X?\", the Agent tool is strongly recommended\n- If you want to read a specific file path, use the View or GlobTool tool instead of the Agent tool, to find the match more quickly\n- If you are searching for a specific class definition like \"class Foo\", use the GlobTool tool instead, to find the match more quickly\n\nUsage notes:\n1. Launch multiple agents concurrently whenever possible, to maximize performance; to do that, use a single message with multiple tool uses\n2. When the agent is done, it will return a single message back to you. The result returned by the agent is not visible to the user. To show the user the result, you should send a text message back to the user with a concise summary of the result.\n3. Each agent invocation is stateless. You will not be able to send additional messages to the agent, nor will the agent be able to communicate with you outside of its final report. Therefore, your prompt should contain a highly detailed task description for the agent to perform autonomously and you should specify exactly what information the agent should return back to you in its final and only message to you.\n4. The agent's outputs should generally be trusted\n5. IMPORTANT: The agent can not use Bash, Replace, Edit, so can not modify files. If you want to use these tools, use them directly instead of going through the agent.",
		Parameters: map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task for the agent to perform",
			},
		},
		Required: []string{"prompt"},
	}
}

func (b *AgentTool) Run(ctx context.Context, call toolcore.ToolCall) (toolcore.ToolResponse, error) {
	var params Params
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return toolcore.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Prompt == "" {
		return toolcore.NewTextErrorResponse("prompt is required"), nil
	}

	sessionID, messageID := toolcore.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return toolcore.ToolResponse{}, fmt.Errorf("session_id and message_id are required")
	}
	ws := basetools.Workspace{Root: sdkconfig.Get().WorkingDir}

	runner, err := agent.NewAgent(
		*sdkconfig.Get(),
		b.sessions,
		b.messages,
		nil,
		b.prompts,
		agent.WithTools(
			basetools.NewGlobTool(ws),
			basetools.NewGrepTool(ws),
			basetools.NewLsTool(ws),
			basetools.NewSourcegraphTool(),
			basetools.NewViewTool(ws),
		),
		agent.WithPromptKey(prompt.KeyTask),
	)
	if err != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error creating agent: %s", err)
	}

	taskSession, err := b.sessions.CreateTaskSession(ctx, call.ID, sessionID, "New Agent Session")
	if err != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error creating session: %s", err)
	}

	done, err := runner.Run(ctx, taskSession.ID, params.Prompt)
	if err != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error generating agent: %s", err)
	}
	result := <-done
	if result.Error != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error generating agent: %s", result.Error)
	}

	response := result.Message
	if response.Role != message.Assistant {
		return toolcore.NewTextErrorResponse("no response"), nil
	}

	updatedSession, err := b.sessions.Get(ctx, taskSession.ID)
	if err != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error getting session: %s", err)
	}
	parentSession, err := b.sessions.Get(ctx, sessionID)
	if err != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error getting parent session: %s", err)
	}

	parentSession.Cost += updatedSession.Cost

	_, err = b.sessions.Save(ctx, parentSession)
	if err != nil {
		return toolcore.ToolResponse{}, fmt.Errorf("error saving parent session: %s", err)
	}
	return toolcore.NewTextResponse(response.Content().String()), nil
}

func New(
	sessions session.Service,
	messages message.Service,
	prompts prompt.Service,
) toolcore.BaseTool {
	return &AgentTool{
		sessions: sessions,
		messages: messages,
		prompts:  prompts,
	}
}
