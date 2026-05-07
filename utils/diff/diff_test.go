package diff

import (
	"path/filepath"
	"strings"
	"testing"
)

func testWorkingDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestGenerateDiffCountsAdditionsAndRemovals(t *testing.T) {
	workingDir := testWorkingDir(t)
	filePath := filepath.Join(workingDir, "file.txt")

	diffText, additions, removals := GenerateDiff("one\ntwo\n", "one\nthree\nfour\n", filePath, workingDir)

	if additions != 2 {
		t.Fatalf("expected 2 additions, got %d\n%s", additions, diffText)
	}
	if removals != 1 {
		t.Fatalf("expected 1 removal, got %d\n%s", removals, diffText)
	}
	if strings.Contains(diffText, workingDir) {
		t.Fatalf("diff should not include absolute working directory: %s", diffText)
	}
	if !strings.Contains(diffText, "--- a/file.txt") || !strings.Contains(diffText, "+++ b/file.txt") {
		t.Fatalf("diff should use normalized a/b paths: %s", diffText)
	}
}

func TestPatchParseAndApplyUpdate(t *testing.T) {
	patchText := `*** Begin Patch
*** Update File: file.txt
@@
 hello
-old
+new
 bye
*** End Patch`
	orig := map[string]string{"file.txt": "hello\nold\nbye"}

	patch, fuzz, err := TextToPatch(patchText, orig)
	if err != nil {
		t.Fatalf("TextToPatch: %v", err)
	}
	if fuzz != 0 {
		t.Fatalf("expected fuzz 0, got %d", fuzz)
	}

	commit, err := PatchToCommit(patch, orig)
	if err != nil {
		t.Fatalf("PatchToCommit: %v", err)
	}

	writes := map[string]string{}
	if err := ApplyCommit(commit, func(path string, content string) error {
		writes[path] = content
		return nil
	}, func(path string) error {
		t.Fatalf("unexpected remove call for %s", path)
		return nil
	}); err != nil {
		t.Fatalf("ApplyCommit: %v", err)
	}

	if writes["file.txt"] != "hello\nnew\nbye" {
		t.Fatalf("unexpected patched content: %q", writes["file.txt"])
	}
}
