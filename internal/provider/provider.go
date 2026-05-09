package provider

import (
	"context"

	llmclient "ferryman-agent/internal/data/llm/client"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/message"
)

type ProviderClient struct {
	Provider models.ModelProvider
	Model    models.Model
	Client   llmclient.Client
}

func (p ProviderClient) SendMessages(ctx context.Context, request llmclient.Request) (*llmclient.Response, error) {
	request.Messages = cleanMessages(request.Messages)
	request.Model = p.Model
	return p.Client.Send(ctx, request)
}

func (p ProviderClient) StreamResponse(ctx context.Context, request llmclient.Request) <-chan llmclient.Event {
	request.Messages = cleanMessages(request.Messages)
	request.Model = p.Model
	return p.Client.Stream(ctx, request)
}

func cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return cleaned
}
