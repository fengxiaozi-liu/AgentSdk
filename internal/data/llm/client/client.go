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

type Request struct {
	ModelID       models.ModelID
	Provider      models.ModelProvider
	Model         models.Model
	SystemMessage string
	Debug         bool
	Messages      []message.Message
	Tools         []toolcore.BaseTool
}

type Client interface {
	Send(ctx context.Context, request Request) (*Response, error)
	Stream(ctx context.Context, request Request) <-chan Event
}
