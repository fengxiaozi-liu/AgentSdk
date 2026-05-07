package history

import (
	"context"
	"testing"

	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
)

func TestServiceUsesHistoryRepoVersions(t *testing.T) {
	ctx := context.Background()
	client, err := datadb.NewDbClient(datadb.DatabaseConfig{
		Type: datadb.DatabaseSQLite,
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := client.AutoMigrate(&repo.SessionRecord{}, &repo.MessageRecord{}, &repo.HistoryRecord{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	service := NewService(repo.NewHistoryRepo(client))

	first, err := service.Create(ctx, "s1", "file.txt", "one")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if first.Version != InitialVersion {
		t.Fatalf("unexpected initial version: %q", first.Version)
	}
	second, err := service.CreateVersion(ctx, "s1", "file.txt", "two")
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if second.Version != "v1" {
		t.Fatalf("expected v1, got %q", second.Version)
	}
	latest, err := service.GetByPathAndSession(ctx, "file.txt", "s1")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.Content != "two" {
		t.Fatalf("expected latest content two, got %q", latest.Content)
	}
}
