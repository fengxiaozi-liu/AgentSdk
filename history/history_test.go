package history

import (
	"context"
	"testing"

	datadb "ferryman-agent/data/db"
	"ferryman-agent/data/repo"
)

func TestServiceUsesHistoryRepoVersions(t *testing.T) {
	ctx := context.Background()
	repos := repo.NewRepositories(datadb.NewSource())
	service := NewService(repos.History)

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
