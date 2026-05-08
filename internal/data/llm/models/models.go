package models

import (
	"embed"
	"encoding/json"
	"sync"
)

//go:embed models.json
var defaultModelsFS embed.FS

type ModelID string

type ModelProvider string

const (
	ProviderAnthropic  ModelProvider = "anthropic"
	ProviderAzure      ModelProvider = "azure"
	ProviderBedrock    ModelProvider = "bedrock"
	ProviderCopilot    ModelProvider = "copilot"
	ProviderGemini     ModelProvider = "gemini"
	ProviderGROQ       ModelProvider = "groq"
	ProviderLocal      ModelProvider = "local"
	ProviderOpenAI     ModelProvider = "openai"
	ProviderOpenRouter ModelProvider = "openrouter"
	ProviderVertexAI   ModelProvider = "vertexai"
	ProviderXAI        ModelProvider = "xai"
	ProviderMock       ModelProvider = "__mock"
)

type ModelConfig struct {
	ModelId         ModelID `json:"model_id"`
	APIModel        string  `json:"api_model"`
	MaxTokens       int64   `json:"maxTokens,omitempty"`
	ReasoningEffort string  `json:"reasoning_effort,omitempty"`
}

type Model struct {
	ID                  ModelID `json:"id"`
	Name                string  `json:"name"`
	APIModel            string  `json:"api_model"`
	CostPer1MIn         float64 `json:"cost_per_1m_in"`
	CostPer1MOut        float64 `json:"cost_per_1m_out"`
	CostPer1MInCached   float64 `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached  float64 `json:"cost_per_1m_out_cached"`
	ContextWindow       int64   `json:"context_window"`
	DefaultMaxTokens    int64   `json:"default_max_tokens"`
	CanReason           bool    `json:"can_reason"`
	SupportsAttachments bool    `json:"supports_attachments"`
	ReasoningEffort     string  `json:"reasoning_effort,omitempty"`
}

type ProviderModels struct {
	Models map[ModelID]Model `json:"models"`
}

type Catalog struct {
	Providers map[ModelProvider]ProviderModels `json:"providers"`
}

var (
	onceCatalog sync.Once
	catalog     Catalog
	catalogErr  error
)

func LoadCatalog() (Catalog, error) {
	onceCatalog.Do(func() {
		content, err := defaultModelsFS.ReadFile("models.json")
		if err != nil {
			catalogErr = err
			return
		}
		catalogErr = json.Unmarshal(content, &catalog)
	})
	return catalog, catalogErr
}

func ResolveModel(provider ModelProvider, modelID ModelID) Model {
	if loaded, err := LoadCatalog(); err == nil {
		if providerModels, ok := loaded.Providers[provider]; ok {
			if model, ok := providerModels.Models[modelID]; ok {
				return normalizeModel(modelID, model)
			}
		}
	}
	return normalizeModel(modelID, Model{ID: modelID, Name: string(modelID), APIModel: string(modelID)})
}

func normalizeModel(modelID ModelID, model Model) Model {
	if model.ID == "" {
		model.ID = modelID
	}
	if model.Name == "" {
		model.Name = string(model.ID)
	}
	if model.APIModel == "" {
		model.APIModel = string(model.ID)
	}
	if model.DefaultMaxTokens <= 0 {
		model.DefaultMaxTokens = 4096
	}
	model.SupportsAttachments = true
	return model
}
