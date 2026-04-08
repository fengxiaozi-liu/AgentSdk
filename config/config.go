package config

import (
	"ferryman-agent/llm/models"
)

type AgentRuntimeConfig struct {
	Name            AgentName
	Model           models.ModelID
	MaxTokens       int64
	ReasoningEffort string
}

func Current() *Config {
	return Get()
}

func RuntimeFor(agentName AgentName) (AgentRuntimeConfig, bool) {
	cfg := Get()
	if cfg == nil {
		return AgentRuntimeConfig{}, false
	}
	agentCfg, ok := cfg.Agents[agentName]
	if !ok {
		return AgentRuntimeConfig{}, false
	}
	return AgentRuntimeConfig{
		Name:            agentName,
		Model:           agentCfg.Model,
		MaxTokens:       agentCfg.MaxTokens,
		ReasoningEffort: agentCfg.ReasoningEffort,
	}, true
}
