package history

import (
	"context"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/agent/session"
	"github.com/opencode-ai/opencode/agent/testutil"
)

func TestServiceCreateVersionAndListLatest(t *testing.T) {
	ctx := context.Background()
	harness := testutil.NewDBHarness(t)

	sessions := session.NewService(harness.Queries)
	files := NewService(harness.Queries, harness.DB)

	sess, err := sessions.Create(ctx, "chat")
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	file, err := files.Create(ctx, sess.ID, "/tmp/test.txt", "v0")
	if err != nil {
		t.Fatalf("Create file history: %v", err)
	}
	if file.Version != InitialVersion {
		t.Fatalf("expected initial version, got %q", file.Version)
	}

	time.Sleep(1100 * time.Millisecond)

	updated, err := files.CreateVersion(ctx, sess.ID, "/tmp/test.txt", "v1")
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if updated.Version != "v1" {
		t.Fatalf("expected v1 version, got %q", updated.Version)
	}

	latest, err := files.ListLatestSessionFiles(ctx, sess.ID)
	if err != nil {
		t.Fatalf("ListLatestSessionFiles: %v", err)
	}

	if len(latest) != 1 {
		t.Fatalf("expected 1 latest file, got %d", len(latest))
	}
	if latest[0].Content != "v1" {
		t.Fatalf("expected latest content v1, got %q", latest[0].Content)
	}
}
