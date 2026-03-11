package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupPathForSource(t *testing.T) {
	backupRoot := "/tmp/spirited/backups"
	source := "/Users/test/work/project/.envrc"

	path, err := backupPathForSource(backupRoot, source)
	if err != nil {
		t.Fatalf("backupPathForSource() error = %v", err)
	}

	expected := filepath.Join(backupRoot, "Users", "test", "work", "project", ".envrc")
	if path != expected {
		t.Fatalf("path = %q, want %q", path, expected)
	}
}

func TestRenderEnvFileSorted(t *testing.T) {
	content := renderEnvFile(map[string]string{"B": "2", "A": "1"})
	if !strings.HasPrefix(content, "A=") {
		t.Fatalf("unexpected render order: %s", content)
	}
}

func TestEnsureUniqueBackupPath(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, ".envrc")
	if err := os.WriteFile(base, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	unique, err := ensureUniqueBackupPath(base)
	if err != nil {
		t.Fatalf("ensureUniqueBackupPath() error = %v", err)
	}
	if unique == base {
		t.Fatal("expected unique path different from base")
	}
}
