package provider

import (
	"context"
	"errors"
	"testing"

	llmclient "ferryman-agent/internal/data/llm/client"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/message"
)

type fakeClient struct {
	content string
	err     error
	request llmclient.Request
}

func (f *fakeClient) Send(ctx context.Context, request llmclient.Request) (*llmclient.Response, error) {
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	return &llmclient.Response{Content: f.content}, nil
}

func (f *fakeClient) Stream(ctx context.Context, request llmclient.Request) <-chan llmclient.Event {
	f.request = request
	ch := make(chan llmclient.Event, 1)
	if f.err != nil {
		ch <- llmclient.Event{Type: llmclient.EventError, Error: f.err}
	} else {
		ch <- llmclient.Event{Type: llmclient.EventComplete, Response: &llmclient.Response{Content: f.content}}
	}
	close(ch)
	return ch
}

func TestNewDefaultRouterRejectsInvalidConfigs(t *testing.T) {
	_, err := NewDefaultRouter(ProviderConfig{})
	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("expected ErrProviderNotConfigured, got %v", err)
	}

	_, err = NewDefaultRouter(ProviderConfig{Provider: models.ProviderMock})
	if !errors.Is(err, ErrModelNotConfigured) {
		t.Fatalf("expected ErrModelNotConfigured, got %v", err)
	}

	_, err = NewDefaultRouter(ProviderConfig{
		Provider: models.ProviderMock,
		Models:   []ModelConfig{{}},
	})
	if !errors.Is(err, ErrModelNotConfigured) {
		t.Fatalf("expected ErrModelNotConfigured for empty model id, got %v", err)
	}
}

func TestNewDefaultRouterSkipsDisabledConfig(t *testing.T) {
	router, err := NewDefaultRouter(ProviderConfig{
		Provider: models.ProviderMock,
		Disabled: true,
	})
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	_, err = router.Route(context.Background(), RouteRequest{Provider: models.ProviderMock, ModelID: "model-a"})
	if !errors.Is(err, ErrProviderTargetNotFound) {
		t.Fatalf("expected ErrProviderTargetNotFound, got %v", err)
	}
}

func TestNewDefaultRouterRejectsDuplicateTarget(t *testing.T) {
	_, err := NewDefaultRouter(ProviderConfig{
		Provider: models.ProviderMock,
		Models: []ModelConfig{
			{ModelID: "model-a"},
			{ModelID: "model-a"},
		},
	})
	if !errors.Is(err, ErrProviderTargetExists) {
		t.Fatalf("expected ErrProviderTargetExists, got %v", err)
	}
}

func TestRouteUsesExactProviderTarget(t *testing.T) {
	router := &DefaultRouter{
		targets: ProviderTargets{
			models.ProviderOpenAI: {
				"model-a": {Provider: models.ProviderOpenAI, Model: models.Model{ID: "model-a", APIModel: "openai-model"}, Client: &fakeClient{content: "openai"}},
			},
			models.ProviderAzure: {
				"model-a": {Provider: models.ProviderAzure, Model: models.Model{ID: "model-a", APIModel: "azure-model"}, Client: &fakeClient{content: "azure"}},
			},
		},
	}

	target, err := router.Route(context.Background(), RouteRequest{ModelID: "model-a", Provider: models.ProviderAzure})
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	response, err := target.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a", Provider: models.ProviderAzure})
	if err != nil {
		t.Fatalf("send messages: %v", err)
	}
	if response.Content != "azure" {
		t.Fatalf("expected azure response, got %q", response.Content)
	}
	if target.Model.APIModel != "azure-model" {
		t.Fatalf("expected azure model, got %q", target.Model.APIModel)
	}
}

func TestRouteDoesNotFallback(t *testing.T) {
	router := &DefaultRouter{
		targets: ProviderTargets{
			models.ProviderOpenAI: {
				"model-a": {Provider: models.ProviderOpenAI, Model: models.Model{ID: "model-a"}, Client: &fakeClient{content: "openai"}},
			},
		},
	}

	_, err := router.Route(context.Background(), RouteRequest{ModelID: "model-a", Provider: models.ProviderAzure})
	if !errors.Is(err, ErrProviderTargetNotFound) {
		t.Fatalf("expected ErrProviderTargetNotFound, got %v", err)
	}
}

func TestProviderClientSendMessagesCleansMessagesAndSetsModel(t *testing.T) {
	client := &fakeClient{content: "ok"}
	target := ProviderClient{
		Provider: models.ProviderMock,
		Model:    models.Model{ID: "model-a", APIModel: "api-model"},
		Client:   client,
	}

	_, err := target.SendMessages(context.Background(), llmclient.Request{
		Messages: []message.Message{
			{},
			{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hello"}}},
		},
	})
	if err != nil {
		t.Fatalf("send messages: %v", err)
	}
	if len(client.request.Messages) != 1 {
		t.Fatalf("expected one cleaned message, got %d", len(client.request.Messages))
	}
	if client.request.Model.ID != "model-a" {
		t.Fatalf("expected request model to be set, got %q", client.request.Model.ID)
	}
}

func TestProviderClientStreamResponseReturnsClientEvents(t *testing.T) {
	expectedErr := errors.New("boom")
	target := ProviderClient{
		Provider: models.ProviderMock,
		Model:    models.Model{ID: "model-a"},
		Client:   &fakeClient{err: expectedErr},
	}

	events := target.StreamResponse(context.Background(), llmclient.Request{})
	event := <-events
	if event.Type != llmclient.EventError {
		t.Fatalf("expected error event, got %s", event.Type)
	}
	if !errors.Is(event.Error, expectedErr) {
		t.Fatalf("expected original error, got %v", event.Error)
	}
}
