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

func TestSelectTargetSupportsProviderFilter(t *testing.T) {
	service := &ProviderService{
		clients: map[models.ModelID][]ProviderClient{
			"model-a": {
				{Provider: models.ProviderOpenAI, Model: models.Model{ID: "model-a"}, Client: fakeClient{}},
				{Provider: models.ProviderAzure, Model: models.Model{ID: "model-a"}, Client: fakeClient{}},
			},
		},
		activeTargets: map[models.ModelID]ProviderClient{},
	}

	target, err := service.selectTarget("model-a", models.ProviderAzure)
	if err != nil {
		t.Fatalf("select target: %v", err)
	}
	if target.Provider != models.ProviderAzure {
		t.Fatalf("expected azure target, got %s", target.Provider)
	}
}

func TestSendMessagesWritesActiveTarget(t *testing.T) {
	service := &ProviderService{
		clients: map[models.ModelID][]ProviderClient{
			"model-a": {
				{
					Provider: models.ProviderOpenAI,
					Model:    models.Model{ID: "model-a", APIModel: "model-a"},
					Client:   fakeClient{content: "ok"},
				},
			},
		},
		activeTargets: map[models.ModelID]ProviderClient{},
	}

	response, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a"})
	if err != nil {
		t.Fatalf("send messages: %v", err)
	}
	if response.Content != "ok" {
		t.Fatalf("expected response content ok, got %q", response.Content)
	}

	active, ok := service.ActiveTarget("model-a")
	if !ok {
		t.Fatal("expected active target")
	}
	if active.Provider != models.ProviderOpenAI {
		t.Fatalf("expected active provider openai, got %s", active.Provider)
	}
}

func TestSendMessagesFallsBackToSameModelProvider(t *testing.T) {
	service := &ProviderService{
		clients: map[models.ModelID][]ProviderClient{
			"model-a": {
				{
					Provider: models.ProviderOpenAI,
					Model:    models.Model{ID: "model-a", APIModel: "model-a"},
					Client:   fakeClient{err: errors.New("boom")},
				},
				{
					Provider: models.ProviderAzure,
					Model:    models.Model{ID: "model-a", APIModel: "model-a"},
					Client:   fakeClient{content: "fallback"},
				},
			},
		},
		activeTargets: map[models.ModelID]ProviderClient{},
	}

	response, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a"})
	if err != nil {
		t.Fatalf("send messages: %v", err)
	}
	if response.Content != "fallback" {
		t.Fatalf("expected fallback response, got %q", response.Content)
	}
}

func TestSendMessagesUsesRandomFallbackModelCandidate(t *testing.T) {
	service := &ProviderService{
		clients: map[models.ModelID][]ProviderClient{
			"model-a": {
				{
					Provider: models.ProviderOpenAI,
					Model:    models.Model{ID: "model-a", APIModel: "model-a"},
					Client:   fakeClient{err: errors.New("boom")},
				},
			},
			"model-b": {
				{
					Provider: models.ProviderAzure,
					Model:    models.Model{ID: "model-b", APIModel: "model-b"},
					Client:   fakeClient{content: "model-b"},
				},
			},
		},
		activeTargets: map[models.ModelID]ProviderClient{},
	}

	response, err := service.SendMessages(context.Background(), llmclient.Request{ModelID: "model-a"})
	if err != nil {
		t.Fatalf("send messages: %v", err)
	}
	if response.Content != "model-b" {
		t.Fatalf("expected fallback model response, got %q", response.Content)
	}
}
