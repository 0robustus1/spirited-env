package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0robustus1/spirited-env/internal/pathmap"
)

func TestFindLayeredEnvFiles(t *testing.T) {
	base := t.TempDir()
	store := filepath.Join(base, "store")
	mapper, err := pathmap.New(store)
	if err != nil {
		t.Fatalf("pathmap.New() error = %v", err)
	}

	project := filepath.Join(base, "project")
	nested := filepath.Join(project, "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	projectEnv, err := mapper.EnvFileForDir(project)
	if err != nil {
		t.Fatalf("EnvFileForDir(project) error = %v", err)
	}
	nestedEnv, err := mapper.EnvFileForDir(nested)
	if err != nil {
		t.Fatalf("EnvFileForDir(nested) error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(projectEnv), 0o755); err != nil {
		t.Fatalf("MkdirAll(projectEnv) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(nestedEnv), 0o755); err != nil {
		t.Fatalf("MkdirAll(nestedEnv) error = %v", err)
	}
	if err := os.WriteFile(projectEnv, []byte("A=project\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(projectEnv) error = %v", err)
	}
	if err := os.WriteFile(nestedEnv, []byte("A=nested\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(nestedEnv) error = %v", err)
	}

	files, err := FindLayeredEnvFiles(nested, mapper)
	if err != nil {
		t.Fatalf("FindLayeredEnvFiles() error = %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0] != projectEnv {
		t.Fatalf("files[0] = %q, want %q", files[0], projectEnv)
	}
	if files[1] != nestedEnv {
		t.Fatalf("files[1] = %q, want %q", files[1], nestedEnv)
	}
}
