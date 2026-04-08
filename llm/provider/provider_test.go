package provider

import (
	"context"
	"testing"

	"github.com/opencode-ai/opencode/agent/llm/models"
	"github.com/opencode-ai/opencode/agent/message"
)

func TestMockProviderSendMessages(t *testing.T) {
	p, err := NewProvider(models.ProviderMock, WithModel(models.Model{
		ID:       "mock-model",
		Name:     "Mock",
		Provider: models.ProviderMock,
	}))
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	resp, err := p.SendMessages(context.Background(), []message.Message{
		{
			Role:  message.User,
			Parts: []message.ContentPart{message.TextContent{Text: "answer:hello"}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("expected hello response, got %q", resp.Content)
	}
	if resp.FinishReason != message.FinishReasonEndTurn {
		t.Fatalf("expected end turn finish reason, got %q", resp.FinishReason)
	}
}

func TestMockProviderStreamToolCall(t *testing.T) {
	p, err := NewProvider(models.ProviderMock, WithModel(models.Model{
		ID:       "mock-model",
		Name:     "Mock",
		Provider: models.ProviderMock,
	}))
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	events := p.StreamResponse(context.Background(), []message.Message{
		{
			Role:  message.User,
			Parts: []message.ContentPart{message.TextContent{Text: `tool:echo:{"text":"hi"}`}},
		},
	}, nil)

	var sawToolStart bool
	var sawComplete bool
	for event := range events {
		switch event.Type {
		case EventToolUseStart:
			sawToolStart = true
		case EventComplete:
			sawComplete = true
			if event.Response == nil || len(event.Response.ToolCalls) != 1 {
				t.Fatalf("expected one tool call in complete event, got %+v", event.Response)
			}
			if event.Response.ToolCalls[0].Name != "echo" {
				t.Fatalf("expected echo tool call, got %q", event.Response.ToolCalls[0].Name)
			}
		}
	}

	if !sawToolStart || !sawComplete {
		t.Fatalf("expected tool start and complete events, got start=%v complete=%v", sawToolStart, sawComplete)
	}
}
