package prompt

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ferryman-agent/config"
	"gopkg.in/yaml.v3"
)

//go:embed prompts.json
var defaultPromptFS embed.FS

var (
	ErrPromptKeyNotFound   = errors.New("prompt key not found")
	ErrPromptConfigInvalid = errors.New("prompt config invalid")
)

type ConfigFile struct {
	Prompts map[string]string `json:"prompts" yaml:"prompts"`
}

func ResolveSystemPromptByKey(key string) (string, error) {
	prompts, err := loadPrompts()
	if err != nil {
		return "", err
	}
	prompt, ok := prompts[key]
	if !ok || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("%w: %s", ErrPromptKeyNotFound, key)
	}
	return prompt, nil
}

var (
	oncePrompts sync.Once
	prompts     map[string]string
	promptsErr  error
)

func loadPrompts() (map[string]string, error) {
	oncePrompts.Do(func() {
		prompts, promptsErr = readPromptConfig()
	})
	return prompts, promptsErr
}

func readPromptConfig() (map[string]string, error) {
	if cfg := config.Get(); cfg != nil && cfg.PromptConfigPath != "" {
		content, err := os.ReadFile(cfg.PromptConfigPath)
		if err != nil {
			return nil, err
		}
		return parsePromptConfig(cfg.PromptConfigPath, content)
	}

	content, err := defaultPromptFS.ReadFile("prompts.json")
	if err != nil {
		return nil, err
	}
	return parsePromptConfig("prompts.json", content)
}

func parsePromptConfig(path string, content []byte) (map[string]string, error) {
	var cfg ConfigFile
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPromptConfigInvalid, err)
		}
	default:
		if err := json.Unmarshal(content, &cfg); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPromptConfigInvalid, err)
		}
	}
	if len(cfg.Prompts) == 0 {
		return nil, fmt.Errorf("%w: prompts is empty", ErrPromptConfigInvalid)
	}
	return cfg.Prompts, nil
}
