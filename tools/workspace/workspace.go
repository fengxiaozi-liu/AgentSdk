package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewWorkspace)

type Workspace struct {
	Root string
}

func NewWorkspace(root string) Workspace {
	return Workspace{Root: root}
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
