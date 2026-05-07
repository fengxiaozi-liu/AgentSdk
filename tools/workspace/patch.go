package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ferryman-agent/history"
	"ferryman-agent/logging"
	"ferryman-agent/permission"
	toolcore "ferryman-agent/tools/core"
	"ferryman-agent/utils/diff"
	"ferryman-agent/utils/fileutil"
)

type PatchParams struct {
	PatchText string `json:"patch_text"`
}

type PatchResponseMetadata struct {
	FilesChanged []string `json:"files_changed"`
	Additions    int      `json:"additions"`
	Removals     int      `json:"removals"`
}

type patchTool struct {
	workspace   Workspace
	permissions permission.Service
	files       history.Service
	hooks       *toolcore.FileHookDispatcher
}

const (
	PatchToolName    = "patch"
	patchDescription = `Applies a patch to multiple files in one operation. This tool is useful for making coordinated changes across multiple files.

The patch text must follow this format:
*** Begin Patch
*** Update File: /path/to/file
@@ Context line (unique within the file)
 Line to keep
-Line to remove
+Line to add
 Line to keep
*** Add File: /path/to/new/file
+Content of the new file
+More content
*** Delete File: /path/to/file/to/delete
*** End Patch

Before using this tool:
1. Use the FileRead tool to understand the files' contents and context
2. Verify all file paths are correct (use the LS tool)

CRITICAL REQUIREMENTS FOR USING THIS TOOL:

1. UNIQUENESS: Context lines MUST uniquely identify the specific sections you want to change
2. PRECISION: All whitespace, indentation, and surrounding code must match exactly
3. VALIDATION: Ensure edits result in idiomatic, correct code
4. PATHS: Always use absolute file paths (starting with /)

The tool will apply all changes in a single atomic operation.`
)

func NewPatchTool(workspace Workspace, permissions permission.Service, files history.Service, hooks ...toolcore.FileHook) toolcore.BaseTool {
	return &patchTool{
		workspace:   workspace,
		permissions: permissions,
		files:       files,
		hooks:       toolcore.NewFileHookDispatcher(hooks...),
	}
}

func (p *patchTool) Info() toolcore.ToolInfo {
	return toolcore.ToolInfo{
		Name:        PatchToolName,
		Description: patchDescription,
		Parameters: map[string]any{
			"patch_text": map[string]any{
				"type":        "string",
				"description": "The full patch text that describes all changes to be made",
			},
		},
		Required: []string{"patch_text"},
	}
}

func (p *patchTool) Run(ctx context.Context, call toolcore.ToolCall) (toolcore.ToolResponse, error) {
	var params PatchParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return toolcore.NewTextErrorResponse("invalid parameters"), nil
	}

	if params.PatchText == "" {
		return toolcore.NewTextErrorResponse("patch_text is required"), nil
	}

	// Identify all files needed for the patch and verify they've been read
	filesToRead := diff.IdentifyFilesNeeded(params.PatchText)
	for _, filePath := range filesToRead {
		absPath, err := p.workspace.Resolve(filePath)
		if err != nil {
			return toolcore.NewTextErrorResponse(err.Error()), nil
		}

		if fileutil.GetLastReadTime(absPath).IsZero() {
			return toolcore.NewTextErrorResponse(fmt.Sprintf("you must read the file %s before patching it. Use the FileRead tool first", filePath)), nil
		}

		fileInfo, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return toolcore.NewTextErrorResponse(fmt.Sprintf("file not found: %s", absPath)), nil
			}
			return toolcore.ToolResponse{}, fmt.Errorf("failed to access file: %w", err)
		}

		if fileInfo.IsDir() {
			return toolcore.NewTextErrorResponse(fmt.Sprintf("path is a directory, not a file: %s", absPath)), nil
		}

		modTime := fileInfo.ModTime()
		lastRead := fileutil.GetLastReadTime(absPath)
		if modTime.After(lastRead) {
			return toolcore.NewTextErrorResponse(
				fmt.Sprintf("file %s has been modified since it was last read (mod time: %s, last read: %s)",
					absPath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
				)), nil
		}
	}

	// Check for new files to ensure they don't already exist
	filesToAdd := diff.IdentifyFilesAdded(params.PatchText)
	for _, filePath := range filesToAdd {
		absPath, err := p.workspace.Resolve(filePath)
		if err != nil {
			return toolcore.NewTextErrorResponse(err.Error()), nil
		}

		_, err = os.Stat(absPath)
		if err == nil {
			return toolcore.NewTextErrorResponse(fmt.Sprintf("file already exists and cannot be added: %s", absPath)), nil
		} else if !os.IsNotExist(err) {
			return toolcore.ToolResponse{}, fmt.Errorf("failed to check file: %w", err)
		}
	}

	// Load all required files
	currentFiles := make(map[string]string)
	for _, filePath := range filesToRead {
		absPath, err := p.workspace.Resolve(filePath)
		if err != nil {
			return toolcore.NewTextErrorResponse(err.Error()), nil
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return toolcore.ToolResponse{}, fmt.Errorf("failed to read file %s: %w", absPath, err)
		}
		currentFiles[filePath] = string(content)
	}

	// Process the patch
	patch, fuzz, err := diff.TextToPatch(params.PatchText, currentFiles)
	if err != nil {
		return toolcore.NewTextErrorResponse(fmt.Sprintf("failed to parse patch: %s", err)), nil
	}

	if fuzz > 3 {
		return toolcore.NewTextErrorResponse(fmt.Sprintf("patch contains fuzzy matches (fuzz level: %d). Please make your context lines more precise", fuzz)), nil
	}

	// Convert patch to commit
	commit, err := diff.PatchToCommit(patch, currentFiles)
	if err != nil {
		return toolcore.NewTextErrorResponse(fmt.Sprintf("failed to create commit from patch: %s", err)), nil
	}

	// Get session ID and message ID
	sessionID, messageID := toolcore.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return toolcore.ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a patch")
	}

	// Request permission for all changes
	for path, change := range commit.Changes {
		switch change.Type {
		case diff.ActionAdd:
			absPath, err := p.workspace.Resolve(path)
			if err != nil {
				return toolcore.NewTextErrorResponse(err.Error()), nil
			}
			dir := filepath.Dir(absPath)
			patchDiff, _, _ := diff.GenerateDiff("", *change.NewContent, path, p.workspace.Root)
			p := p.permissions.Request(
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        dir,
					ToolName:    PatchToolName,
					Action:      "create",
					Description: fmt.Sprintf("Create file %s", absPath),
					Params: EditPermissionsParams{
						FilePath: absPath,
						Diff:     patchDiff,
					},
				},
			)
			if !p {
				return toolcore.ToolResponse{}, permission.ErrorPermissionDenied
			}
		case diff.ActionUpdate:
			currentContent := ""
			if change.OldContent != nil {
				currentContent = *change.OldContent
			}
			newContent := ""
			if change.NewContent != nil {
				newContent = *change.NewContent
			}
			patchDiff, _, _ := diff.GenerateDiff(currentContent, newContent, path, p.workspace.Root)
			absPath, err := p.workspace.Resolve(path)
			if err != nil {
				return toolcore.NewTextErrorResponse(err.Error()), nil
			}
			dir := filepath.Dir(absPath)
			p := p.permissions.Request(
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        dir,
					ToolName:    PatchToolName,
					Action:      "update",
					Description: fmt.Sprintf("Update file %s", absPath),
					Params: EditPermissionsParams{
						FilePath: absPath,
						Diff:     patchDiff,
					},
				},
			)
			if !p {
				return toolcore.ToolResponse{}, permission.ErrorPermissionDenied
			}
		case diff.ActionDelete:
			absPath, err := p.workspace.Resolve(path)
			if err != nil {
				return toolcore.NewTextErrorResponse(err.Error()), nil
			}
			dir := filepath.Dir(absPath)
			patchDiff, _, _ := diff.GenerateDiff(*change.OldContent, "", path, p.workspace.Root)
			p := p.permissions.Request(
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        dir,
					ToolName:    PatchToolName,
					Action:      "delete",
					Description: fmt.Sprintf("Delete file %s", absPath),
					Params: EditPermissionsParams{
						FilePath: absPath,
						Diff:     patchDiff,
					},
				},
			)
			if !p {
				return toolcore.ToolResponse{}, permission.ErrorPermissionDenied
			}
		}
	}

	// Apply the changes to the filesystem
	err = diff.ApplyCommit(commit, func(path string, content string) error {
		absPath, err := p.workspace.Resolve(path)
		if err != nil {
			return err
		}

		// Create parent directories if needed
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create parent directories for %s: %w", absPath, err)
		}

		return os.WriteFile(absPath, []byte(content), 0o644)
	}, func(path string) error {
		absPath, err := p.workspace.Resolve(path)
		if err != nil {
			return err
		}
		return os.Remove(absPath)
	})
	if err != nil {
		return toolcore.NewTextErrorResponse(fmt.Sprintf("failed to apply patch: %s", err)), nil
	}

	// Update file history for all modified files
	changedFiles := []string{}
	totalAdditions := 0
	totalRemovals := 0
	hookResults := []toolcore.HookResult{}

	for path, change := range commit.Changes {
		absPath, err := p.workspace.Resolve(path)
		if err != nil {
			return toolcore.NewTextErrorResponse(err.Error()), nil
		}
		changedFiles = append(changedFiles, absPath)

		oldContent := ""
		if change.OldContent != nil {
			oldContent = *change.OldContent
		}

		newContent := ""
		if change.NewContent != nil {
			newContent = *change.NewContent
		}

		// Calculate diff statistics
		patchDiff, additions, removals := diff.GenerateDiff(oldContent, newContent, path, p.workspace.Root)
		totalAdditions += additions
		totalRemovals += removals

		// Update history
		file, err := p.files.GetByPathAndSession(ctx, absPath, sessionID)
		if err != nil && change.Type != diff.ActionAdd {
			// If not adding a file, create history entry for existing file
			_, err = p.files.Create(ctx, sessionID, absPath, oldContent)
			if err != nil {
				logging.Debug("Error creating file history", "error", err)
			}
		}

		if err == nil && change.Type != diff.ActionAdd && file.Content != oldContent {
			// User manually changed content, store intermediate version
			_, err = p.files.CreateVersion(ctx, sessionID, absPath, oldContent)
			if err != nil {
				logging.Debug("Error creating file history version", "error", err)
			}
		}

		// Store new version
		if change.Type == diff.ActionDelete {
			_, err = p.files.CreateVersion(ctx, sessionID, absPath, "")
		} else {
			_, err = p.files.CreateVersion(ctx, sessionID, absPath, newContent)
		}
		if err != nil {
			logging.Debug("Error creating file history version", "error", err)
		}

		// Record file operations
		fileutil.RecordFileWrite(absPath)
		fileutil.RecordFileRead(absPath)

		hookResults = append(hookResults, p.hooks.Dispatch(ctx, toolcore.FileEvent{
			Type:       toolcore.FilePatched,
			ToolName:   PatchToolName,
			ToolCallID: call.ID,
			SessionID:  sessionID,
			MessageID:  messageID,
			Path:       absPath,
			Paths:      changedFiles,
			OldContent: oldContent,
			NewContent: newContent,
			Diff:       patchDiff,
			Metadata: map[string]any{
				"change_type": string(change.Type),
				"additions":   additions,
				"removals":    removals,
			},
		})...)
	}

	result := fmt.Sprintf("Patch applied successfully. %d files changed, %d additions, %d removals",
		len(changedFiles), totalAdditions, totalRemovals)

	response := toolcore.WithResponseMetadata(
		toolcore.NewTextResponse(result),
		PatchResponseMetadata{
			FilesChanged: changedFiles,
			Additions:    totalAdditions,
			Removals:     totalRemovals,
		})
	return toolcore.WithHookResults(response, hookResults), nil
}
