package models

type (
	ModelID       string
	ModelProvider string
)

type Model struct {
	ID                  ModelID       `json:"id"`
	Name                string        `json:"name"`
	Provider            ModelProvider `json:"provider"`
	APIModel            string        `json:"api_model"`
	CostPer1MIn         float64       `json:"cost_per_1m_in"`
	CostPer1MOut        float64       `json:"cost_per_1m_out"`
	CostPer1MInCached   float64       `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached  float64       `json:"cost_per_1m_out_cached"`
	ContextWindow       int64         `json:"context_window"`
	DefaultMaxTokens    int64         `json:"default_max_tokens"`
	CanReason           bool          `json:"can_reason"`
	SupportsAttachments bool          `json:"supports_attachments"`
}

// Model IDs
const ( // GEMINI
	// Bedrock
	BedrockClaude37Sonnet ModelID = "bedrock.claude-3.7-sonnet"
)

const (
	ProviderBedrock ModelProvider = "bedrock"
	// ForTests
	ProviderMock ModelProvider = "__mock"
)

var knownModels = map[ModelID]Model{
	BedrockClaude37Sonnet: {
		ID:                 BedrockClaude37Sonnet,
		Name:               "Bedrock: Claude 3.7 Sonnet",
		Provider:           ProviderBedrock,
		APIModel:           "anthropic.claude-3-7-sonnet-20250219-v1:0",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  3.75,
		CostPer1MOutCached: 0.30,
		CostPer1MOut:       15.0,
	},
}

func init() {
	registerKnownModels(AnthropicModels)
	registerKnownModels(OpenAIModels)
	registerKnownModels(GeminiModels)
	registerKnownModels(GroqModels)
	registerKnownModels(AzureModels)
	registerKnownModels(OpenRouterModels)
	registerKnownModels(XAIModels)
	registerKnownModels(VertexAIGeminiModels)
	registerKnownModels(CopilotModels)
}

func registerKnownModels(models map[ModelID]Model) {
	for id, model := range models {
		knownModels[id] = model
	}
}

func ProviderForModel(modelID ModelID) ModelProvider {
	if model, ok := knownModels[modelID]; ok {
		return model.Provider
	}
	return ""
}

func ResolveModel(provider ModelProvider, modelID ModelID) Model {
	if model, ok := knownModels[modelID]; ok {
		if provider == "" || provider == model.Provider {
			return model
		}
	}
	return Model{
		ID:                  modelID,
		Name:                string(modelID),
		Provider:            provider,
		APIModel:            string(modelID),
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	}
}
