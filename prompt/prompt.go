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
	"ferryman-agent/llm/models"
	"ferryman-agent/logging"
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

func GetAgentPrompt(agentName config.AgentName, _ models.ModelProvider) string {
	basePrompt, err := ResolveSystemPromptByKey(string(agentName))
	if err != nil {
		basePrompt = "You are a helpful assistant."
	}

	if agentName == config.AgentCoder || agentName == config.AgentTask {
		contextContent := getContextFromPaths()
		logging.Debug("Context content", "Context", contextContent)
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

var (
	oncePrompts sync.Once
	prompts     map[string]string
	promptsErr  error

	onceContext    sync.Once
	contextContent string
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

func getContextFromPaths() string {
	onceContext.Do(func() {
		var (
			cfg          = config.Get()
			workDir      = cfg.WorkingDir
			contextPaths = cfg.ContextPaths
		)

		contextContent = processContextPaths(workDir, contextPaths)
	})

	return contextContent
}

func processContextPaths(workDir string, paths []string) string {
	processedFiles := make(map[string]bool)
	results := make([]string, 0)

	for _, path := range paths {
		if strings.HasSuffix(path, "/") {
			_ = filepath.WalkDir(filepath.Join(workDir, path), func(currentPath string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				lowerPath := strings.ToLower(currentPath)
				if processedFiles[lowerPath] {
					return nil
				}
				processedFiles[lowerPath] = true

				if result := processFile(currentPath); result != "" {
					results = append(results, result)
				}
				return nil
			})
			continue
		}

		fullPath := filepath.Join(workDir, path)
		lowerPath := strings.ToLower(fullPath)
		if processedFiles[lowerPath] {
			continue
		}
		processedFiles[lowerPath] = true

		if result := processFile(fullPath); result != "" {
			results = append(results, result)
		}
	}

	return strings.Join(results, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return "# From:" + filepath.ToSlash(filePath) + "\n" + string(content)
}
