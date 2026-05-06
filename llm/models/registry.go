package models

import (
	"embed"
	"encoding/json"
	"sync"
)

//go:embed models.json
var defaultModelsFS embed.FS

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
