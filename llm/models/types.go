package models

type (
	ModelID       string
	ModelProvider string
)

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

const (
	O4Mini ModelID = "o4-mini"
)

var CopilotAnthropicModels = []ModelID{
	"copilot.claude-3.5-sonnet",
	"copilot.claude-3.7-sonnet",
	"copilot.claude-3.7-sonnet-thought",
	"copilot.claude-sonnet-4",
}
