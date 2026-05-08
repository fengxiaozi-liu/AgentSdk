package provider

import (
	"encoding/json"

	"ferryman-agent/internal/data/llm/models"
)

type ProviderRegister struct {
	Provider             models.ModelProvider `json:"provider"`
	APIKey               string               `json:"apiKey"`
	BaseURL              string               `json:"baseURL"`
	Models               []ModelRegister      `json:"models"`
	Disabled             bool                 `json:"disabled"`
	hasLegacyModelConfig bool
}

func (p *ProviderRegister) UnmarshalJSON(data []byte) error {
	type providerRegisterAlias ProviderRegister
	var raw struct {
		providerRegisterAlias
		ModelConfig json.RawMessage `json:"modelConfig"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = ProviderRegister(raw.providerRegisterAlias)
	p.hasLegacyModelConfig = len(raw.ModelConfig) > 0 && string(raw.ModelConfig) != "null"
	return nil
}

func (p ProviderRegister) Configured() bool {
	return p.Provider != "" || p.APIKey != "" || p.BaseURL != "" || len(p.Models) > 0 || p.hasLegacyModelConfig
}

func (p ProviderRegister) HasLegacyModelConfig() bool {
	return p.hasLegacyModelConfig
}

func (p ProviderRegister) PrimaryModelConfig() (ModelRegister, bool) {
	if len(p.Models) == 0 {
		return ModelRegister{}, false
	}
	return p.Models[0], true
}

type ModelRegister struct {
	ModelId         models.ModelID `json:"model_id"`
	APIModel        string         `json:"api_model"`
	MaxTokens       int64          `json:"maxTokens,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	Weight          int            `json:"weight,omitempty"`
	Priority        int            `json:"priority,omitempty"`
}

func ApplyModelConfig(model models.Model, modelCfg ModelRegister) models.Model {
	if modelCfg.APIModel != "" {
		model.APIModel = modelCfg.APIModel
	}
	if modelCfg.MaxTokens > 0 {
		model.MaxTokens = modelCfg.MaxTokens
	}
	if modelCfg.ReasoningEffort != "" {
		model.ReasoningEffort = modelCfg.ReasoningEffort
	}
	return model
}
