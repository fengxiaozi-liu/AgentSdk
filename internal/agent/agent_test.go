package agent

import (
	"context"
	"testing"

	"ferryman-agent/internal/capability/workspace"
	"ferryman-agent/internal/config"
	"ferryman-agent/internal/data/llm/client"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/provider"
)

type fakeProviderService struct {
	active    map[models.ModelID]provider.ProviderClient
	available map[models.ModelID][]provider.ProviderClient
}

func (s fakeProviderService) SendMessages(context.Context, client.Request) (*client.Response, error) {
	return &client.Response{}, nil
}

func (s fakeProviderService) StreamResponse(context.Context, client.Request) <-chan client.Event {
	ch := make(chan client.Event)
	close(ch)
	return ch
}

func (s fakeProviderService) ActiveTarget(modelID models.ModelID) (provider.ProviderClient, bool) {
	target, ok := s.active[modelID]
	return target, ok
}

func (s fakeProviderService) AvailableTargets(modelID models.ModelID) []provider.ProviderClient {
	return append([]provider.ProviderClient(nil), s.available[modelID]...)
}

func TestNewAgentUsesProviderServiceRuntimeConfig(t *testing.T) {
	const (
		mainModel      models.ModelID = "main-model"
		titleModel     models.ModelID = "title-model"
		summarizeModel models.ModelID = "summary-model"
	)
	cfg := config.Config{
		Debug: true,
		Provider: config.ProviderConfig{
			Provider: models.ProviderMock,
			Models:   []config.ModelConfig{{ModelId: mainModel}},
		},
		Agent: config.AgentModelConfig{
			ModelID:  mainModel,
			Provider: models.ProviderMock,
		},
		TitleAgent: config.AgentModelConfig{
			ModelID:  titleModel,
			Provider: models.ProviderMock,
		},
		SummarizeAgent: config.AgentModelConfig{
			ModelID:  summarizeModel,
			Provider: models.ProviderMock,
		},
	}
	providerSvc := fakeProviderService{
		active: map[models.ModelID]provider.ProviderClient{
			mainModel: {Provider: models.ProviderMock, Model: models.Model{ID: mainModel, APIModel: "runtime-main"}},
		},
	}

	service, err := NewAgent(cfg, nil, nil, nil, nil, nil, workspace.Workspace{}, providerSvc)
	if err != nil {
		t.Fatalf("NewAgent returned error: %v", err)
	}
	runner := service.(*agent)

	if got := runner.Model(); got.APIModel != "runtime-main" {
		t.Fatalf("Model() APIModel = %q, want active target model", got.APIModel)
	}
	if runner.titleAgent == nil || runner.titleAgent.modelID != titleModel {
		t.Fatalf("titleAgent was not configured from Config.TitleAgent")
	}
	if runner.summarizeAgent == nil || runner.summarizeAgent.modelID != summarizeModel {
		t.Fatalf("summarizeAgent was not configured from Config.SummarizeAgent")
	}

	req := runner.providerRequest(*runner.titleAgent, []message.Message{{Role: message.User}})
	if req.ModelID != titleModel || req.Provider != models.ProviderMock || !req.Debug || len(req.Messages) != 1 {
		t.Fatalf("title provider request not populated from runtime config: %+v", req)
	}
}

func TestModelForFallsBackToAvailableTarget(t *testing.T) {
	const modelID models.ModelID = "available-model"
	runner := &agent{
		providerService: fakeProviderService{
			active: map[models.ModelID]provider.ProviderClient{},
			available: map[models.ModelID][]provider.ProviderClient{
				modelID: {{Provider: models.ProviderMock, Model: models.Model{ID: modelID, APIModel: "available"}}},
			},
		},
	}

	if got := runner.modelFor(modelID); got.APIModel != "available" {
		t.Fatalf("modelFor() APIModel = %q, want available target model", got.APIModel)
	}
}
