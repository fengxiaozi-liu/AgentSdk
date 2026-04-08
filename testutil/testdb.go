package testutil

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	agentdb "github.com/opencode-ai/opencode/agent/infra/db"
	internalconfig "github.com/opencode-ai/opencode/agent/config"
)

type DBHarness struct {
	DB         *sql.DB
	Queries    *agentdb.Queries
	WorkingDir string
	DataDir    string
}

func NewDBHarness(t *testing.T) *DBHarness {
	t.Helper()

	workingDir := t.TempDir()
	dataDir := filepath.Join(workingDir, "data")

	t.Setenv("HOME", workingDir)
	t.Setenv("USERPROFILE", workingDir)
	t.Setenv("XDG_CONFIG_HOME", workingDir)
	t.Setenv("LOCALAPPDATA", workingDir)

	cfgPath := filepath.Join(workingDir, ".opencode.json")
	cfgData, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"directory": dataDir,
		},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := internalconfig.Load(workingDir, false); err != nil {
		t.Fatalf("load config: %v", err)
	}

	conn, err := agentdb.Connect()
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return &DBHarness{
		DB:         conn,
		Queries:    agentdb.New(conn),
		WorkingDir: workingDir,
		DataDir:    dataDir,
	}
}
