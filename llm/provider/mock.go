package provider

import (
	"context"
	"strings"

	"github.com/opencode-ai/opencode/agent/llm/tools"
	"github.com/opencode-ai/opencode/agent/message"
)

type MockClient struct {
	options providerClientOptions
}

func newMockClient(options providerClientOptions) MockClient {
	return MockClient{options: options}
}

func (m MockClient) send(ctx context.Context, messages []message.Message, _ []tools.BaseTool) (*ProviderResponse, error) {
	response, _, err := buildMockResponse(ctx, messages)
	return response, err
}

func (m MockClient) stream(ctx context.Context, messages []message.Message, _ []tools.BaseTool) <-chan ProviderEvent {
	ch := make(chan ProviderEvent, 4)
	go func() {
		defer close(ch)

		response, events, err := buildMockResponse(ctx, messages)
		if err != nil {
			ch <- ProviderEvent{Type: EventError, Error: err}
			return
		}

		for _, event := range events {
			select {
			case <-ctx.Done():
				ch <- ProviderEvent{Type: EventError, Error: ctx.Err()}
				return
			case ch <- event:
			}
		}

		ch <- ProviderEvent{
			Type:     EventComplete,
			Response: response,
		}
	}()
	return ch
}

func buildMockResponse(ctx context.Context, messages []message.Message) (*ProviderResponse, []ProviderEvent, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if len(messages) == 0 {
		resp := &ProviderResponse{Content: "mock: empty conversation", FinishReason: message.FinishReasonEndTurn}
		return resp, []ProviderEvent{{Type: EventContentDelta, Content: resp.Content}}, nil
	}

	last := messages[len(messages)-1]
	if last.Role == message.Tool {
		content := "mock tool handled: " + last.Content().String()
		resp := &ProviderResponse{Content: content, FinishReason: message.FinishReasonEndTurn}
		return resp, []ProviderEvent{{Type: EventContentDelta, Content: content}}, nil
	}

	content := last.Content().String()
	if strings.HasPrefix(content, "tool:") {
		nameAndInput := strings.TrimPrefix(content, "tool:")
		name := nameAndInput
		input := "{}"
		if idx := strings.Index(nameAndInput, ":"); idx >= 0 {
			name = nameAndInput[:idx]
			input = nameAndInput[idx+1:]
		}
		call := message.ToolCall{
			ID:    "mock-tool-call",
			Name:  name,
			Input: input,
		}
		resp := &ProviderResponse{
			ToolCalls:    []message.ToolCall{call},
			FinishReason: message.FinishReasonToolUse,
		}
		return resp, []ProviderEvent{
			{Type: EventToolUseStart, ToolCall: &call},
			{Type: EventToolUseStop, ToolCall: &call},
		}, nil
	}

	content = strings.TrimPrefix(content, "answer:")
	resp := &ProviderResponse{Content: content, FinishReason: message.FinishReasonEndTurn}
	return resp, []ProviderEvent{{Type: EventContentDelta, Content: content}}, nil
}
