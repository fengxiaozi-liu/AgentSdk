package provider

import (
	"context"
	"errors"
	"testing"

	llmclient "ferryman-agent/internal/data/llm/client"
	"ferryman-agent/internal/data/llm/models"
)

type fakeClient struct {
	content string
	err     error
}

func (f fakeClient) Send(ctx context.Context, request llmclient.Request) (*llmclient.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &llmclient.Response{Content: f.content}, nil
}

func (f fakeClient) Stream(ctx context.Context, request llmclient.Request) <-chan llmclient.Event {
	ch := make(chan llmclient.Event, 1)
	ch <- llmclient.Event{Type: llmclient.EventComplete, Response: &llmclient.Response{Content: f.content}}
	close(ch)
	return ch
}

func TestSendMessagesUsesExactProviderTarget(t *testing.T) {
	service := &ProviderService{
		clients: ProviderRegistry{
			"model-a": {
				models.ProviderOpenAI: {Provider: models.ProviderOpenAI, Model: models.Model{ID: "model-a", APIModel: "openai-model"}, Client: fakeClient{content: "openai"}},
				models.ProviderAzure:  {Provider: models.ProviderAzure, Model: models.Model{ID: "model-a", APIModel: "azure-model"}, Client: fakeClient{content: "azure"}},
			},
		},
	}

	response, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a", Provider: models.ProviderAzure})
	if err != nil {
		t.Fatalf("send messages: %v", err)
	}
	if response.Content != "azure" {
		t.Fatalf("expected azure response, got %q", response.Content)
	}
}

func TestSendMessagesRequiresConfiguredModel(t *testing.T) {
	service := &ProviderService{clients: ProviderRegistry{}}

	_, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "missing", Provider: models.ProviderOpenAI})
	if !errors.Is(err, ErrModelNotConfigured) {
		t.Fatalf("expected ErrModelNotConfigured, got %v", err)
	}
}

func TestSendMessagesRequiresExactProvider(t *testing.T) {
	service := &ProviderService{
		clients: ProviderRegistry{
			"model-a": {
				models.ProviderOpenAI: {Provider: models.ProviderOpenAI, Model: models.Model{ID: "model-a"}, Client: fakeClient{content: "ok"}},
			},
		},
	}

	_, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a", Provider: models.ProviderAzure})
	if !errors.Is(err, ErrProviderTargetNotFound) {
		t.Fatalf("expected ErrProviderTargetNotFound, got %v", err)
	}
}

func TestSendMessagesDoesNotFallback(t *testing.T) {
	expectedErr := errors.New("boom")
	service := &ProviderService{
		clients: ProviderRegistry{
			"model-a": {
				models.ProviderOpenAI: {Provider: models.ProviderOpenAI, Model: models.Model{ID: "model-a"}, Client: fakeClient{err: expectedErr}},
				models.ProviderAzure:  {Provider: models.ProviderAzure, Model: models.Model{ID: "model-a"}, Client: fakeClient{content: "fallback"}},
			},
			"model-b": {
				models.ProviderAzure: {Provider: models.ProviderAzure, Model: models.Model{ID: "model-b"}, Client: fakeClient{content: "model-b"}},
			},
		},
	}

	_, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a", Provider: models.ProviderOpenAI})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected original target error, got %v", err)
	}
}

func TestRegisterRejectsDuplicateProviderTarget(t *testing.T) {
	service, err := NewService(ProviderRegister{
		Provider: models.ProviderMock,
		Models: []ModelRegister{
			{ModelId: "model-a"},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	err = service.Register(ProviderRegister{
		Provider: models.ProviderMock,
		Models: []ModelRegister{
			{ModelId: "model-a"},
		},
	})
	if !errors.Is(err, ErrProviderTargetExists) {
		t.Fatalf("expected ErrProviderTargetExists, got %v", err)
	}
}

func TestStreamResponseReturnsErrorEventForMissingProvider(t *testing.T) {
	service := &ProviderService{
		clients: ProviderRegistry{
			"model-a": {},
		},
	}

	events := service.StreamResponse(context.Background(), llmclient.Request{ModelID: "model-a", Provider: models.ProviderOpenAI})
	event := <-events
	if event.Type != llmclient.EventError {
		t.Fatalf("expected error event, got %s", event.Type)
	}
	if !errors.Is(event.Error, ErrProviderTargetNotFound) {
		t.Fatalf("expected ErrProviderTargetNotFound, got %v", event.Error)
	}
}
