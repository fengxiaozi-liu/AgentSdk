package client

import (
	"context"
	llmclient "ferryman-agent/internal/data/llm/client"
	"strings"

	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type MockClient struct {
	options llmclient.Options
}

type Client = MockClient

func NewClient(options llmclient.Options) MockClient {
	return MockClient{options: options}
}

func (m MockClient) Send(ctx context.Context, messages []message.Message, _ []toolcore.BaseTool) (*llmclient.Response, error) {
	response, _, err := buildMockResponse(ctx, messages)
	return response, err
}

func (m MockClient) Stream(ctx context.Context, messages []message.Message, _ []toolcore.BaseTool) <-chan llmclient.Event {
	ch := make(chan llmclient.Event, 4)
	go func() {
		defer close(ch)

		response, events, err := buildMockResponse(ctx, messages)
		if err != nil {
			ch <- llmclient.Event{Type: llmclient.EventError, Error: err}
			return
		}

		for _, event := range events {
			select {
			case <-ctx.Done():
				ch <- llmclient.Event{Type: llmclient.EventError, Error: ctx.Err()}
				return
			case ch <- event:
			}
		}

		ch <- llmclient.Event{
			Type:     llmclient.EventComplete,
			Response: response,
		}
	}()
	return ch
}

func buildMockResponse(ctx context.Context, messages []message.Message) (*llmclient.Response, []llmclient.Event, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if len(messages) == 0 {
		resp := &llmclient.Response{Content: "mock: empty conversation", FinishReason: message.FinishReasonEndTurn}
		return resp, []llmclient.Event{{Type: llmclient.EventContentDelta, Content: resp.Content}}, nil
	}

	last := messages[len(messages)-1]
	if last.Role == message.Tool {
		content := "mock tool handled: " + last.Content().String()
		resp := &llmclient.Response{Content: content, FinishReason: message.FinishReasonEndTurn}
		return resp, []llmclient.Event{{Type: llmclient.EventContentDelta, Content: content}}, nil
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
		resp := &llmclient.Response{
			ToolCalls:    []message.ToolCall{call},
			FinishReason: message.FinishReasonToolUse,
		}
		return resp, []llmclient.Event{
			{Type: llmclient.EventToolUseStart, ToolCall: &call},
			{Type: llmclient.EventToolUseStop, ToolCall: &call},
		}, nil
	}

	content = strings.TrimPrefix(content, "answer:")
	resp := &llmclient.Response{Content: content, FinishReason: message.FinishReasonEndTurn}
	return resp, []llmclient.Event{{Type: llmclient.EventContentDelta, Content: content}}, nil
}
