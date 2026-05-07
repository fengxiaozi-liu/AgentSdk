package client

import (
	"context"
	"strings"

	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type MockClient struct {
	options Options
}

func NewMockClient(options Options) MockClient {
	return MockClient{options: options}
}

func (m MockClient) Send(ctx context.Context, messages []message.Message, _ []toolcore.BaseTool) (*Response, error) {
	response, _, err := buildMockResponse(ctx, messages)
	return response, err
}

func (m MockClient) Stream(ctx context.Context, messages []message.Message, _ []toolcore.BaseTool) <-chan Event {
	ch := make(chan Event, 4)
	go func() {
		defer close(ch)

		response, events, err := buildMockResponse(ctx, messages)
		if err != nil {
			ch <- Event{Type: EventError, Error: err}
			return
		}

		for _, event := range events {
			select {
			case <-ctx.Done():
				ch <- Event{Type: EventError, Error: ctx.Err()}
				return
			case ch <- event:
			}
		}

		ch <- Event{
			Type:     EventComplete,
			Response: response,
		}
	}()
	return ch
}

func buildMockResponse(ctx context.Context, messages []message.Message) (*Response, []Event, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if len(messages) == 0 {
		resp := &Response{Content: "mock: empty conversation", FinishReason: message.FinishReasonEndTurn}
		return resp, []Event{{Type: EventContentDelta, Content: resp.Content}}, nil
	}

	last := messages[len(messages)-1]
	if last.Role == message.Tool {
		content := "mock tool handled: " + last.Content().String()
		resp := &Response{Content: content, FinishReason: message.FinishReasonEndTurn}
		return resp, []Event{{Type: EventContentDelta, Content: content}}, nil
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
		resp := &Response{
			ToolCalls:    []message.ToolCall{call},
			FinishReason: message.FinishReasonToolUse,
		}
		return resp, []Event{
			{Type: EventToolUseStart, ToolCall: &call},
			{Type: EventToolUseStop, ToolCall: &call},
		}, nil
	}

	content = strings.TrimPrefix(content, "answer:")
	resp := &Response{Content: content, FinishReason: message.FinishReasonEndTurn}
	return resp, []Event{{Type: EventContentDelta, Content: content}}, nil
}
