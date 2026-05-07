package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"ferryman-agent/history"
	"ferryman-agent/permission"
	toolcore "ferryman-agent/tools/core"
	mcptools "ferryman-agent/tools/mcp"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewWorkspace, NewContainerToolset)

type Workspace struct {
	Root string
}

func NewWorkspace(root string) Workspace {
	return Workspace{Root: root}
}

func NewContainerToolset(
	workspace Workspace,
	permissions permission.Service,
	historySvc history.Service,
) []toolcore.BaseTool {
	baseTools := []toolcore.BaseTool{
		NewGlobTool(workspace),
		NewGrepTool(workspace),
		NewLsTool(workspace),
		NewSourcegraphTool(),
		NewViewTool(workspace),
		NewEditTool(workspace, permissions, historySvc),
		NewWriteTool(workspace, permissions, historySvc),
		NewPatchTool(workspace, permissions, historySvc),
		NewBashTool(workspace, permissions),
		NewFetchTool(workspace, permissions),
	}
	return append(
		baseTools,
		mcptools.GetMcpTools(context.Background(), permissions)...,
	)
}

func (w Workspace) Resolve(path string) (string, error) {
	root, err := filepath.Abs(w.Root)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("workspace root is required")
	}

	target := path
	if target == "" {
		target = root
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace %q", path, root)
	}
	return target, nil
}

func (w Workspace) Contains(path string) bool {
	_, err := w.Resolve(path)
	return err == nil
}
