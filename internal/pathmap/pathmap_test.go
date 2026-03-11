package pathmap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvFileForDir(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "store")
	mapper, err := New(root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	project := filepath.Join(base, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	mapped, err := mapper.EnvFileForDir(project)
	if err != nil {
		t.Fatalf("EnvFileForDir() error = %v", err)
	}

	canonical, err := CanonicalizeDir(project)
	if err != nil {
		t.Fatalf("CanonicalizeDir() error = %v", err)
	}

	expected := filepath.Join(root, strings.TrimPrefix(canonical, string(filepath.Separator)), ".env")
	if mapped != expected {
		t.Fatalf("mapped = %q, want %q", mapped, expected)
	}
}
