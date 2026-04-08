package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ferryman-agent/config"
	"ferryman-agent/infra/logging"
	"ferryman-agent/llm/models"
)

func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	basePrompt := ""
	switch agentName {
	case config.AgentCoder:
		basePrompt = CoderPrompt(provider)
	case config.AgentTitle:
		basePrompt = TitlePrompt(provider)
	case config.AgentTask:
		basePrompt = TaskPrompt(provider)
	case config.AgentSummarizer:
		basePrompt = SummarizerPrompt(provider)
	default:
		basePrompt = "You are a helpful assistant"
	}

	if agentName == config.AgentCoder || agentName == config.AgentTask {
		// Add context from project-specific instruction files if they exist
		contextContent := getContextFromPaths()
		logging.Debug("Context content", "Context", contextContent)
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

var (
	onceContext    sync.Once
	contextContent string
)

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
	// Track processed files to avoid duplicates
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
