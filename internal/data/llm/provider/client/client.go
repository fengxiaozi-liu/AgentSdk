package client

import (
	"context"

	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type EventType string

const MaxRetries = 8

const (
	EventContentStart  EventType = "content_start"
	EventToolUseStart  EventType = "tool_use_start"
	EventToolUseDelta  EventType = "tool_use_delta"
	EventToolUseStop   EventType = "tool_use_stop"
	EventContentDelta  EventType = "content_delta"
	EventThinkingDelta EventType = "thinking_delta"
	EventContentStop   EventType = "content_stop"
	EventComplete      EventType = "complete"
	EventError         EventType = "error"
	EventWarning       EventType = "warning"
)

type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type Response struct {
	Content      string
	ToolCalls    []message.ToolCall
	Usage        TokenUsage
	FinishReason message.FinishReason
}

type Event struct {
	Type     EventType
	Content  string
	Thinking string
	Response *Response
	ToolCall *message.ToolCall
	Error    error
}

type Options struct {
	APIKey           string
	Model            models.Model
	MaxTokens        int64
	SystemMessage    string
	Debug            bool
	AnthropicOptions []AnthropicOption
	OpenAIOptions    []OpenAIOption
	GeminiOptions    []GeminiOption
	BedrockOptions   []BedrockOption
	CopilotOptions   []CopilotOption
}

type Client interface {
	Send(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) (*Response, error)
	Stream(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) <-chan Event
}
