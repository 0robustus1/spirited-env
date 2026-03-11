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

func TestDefaultRootUsesOverride(t *testing.T) {
	t.Setenv(HomeOverrideEnv, "/tmp/custom-spirited")
	t.Setenv(XDGConfigHomeEnv, "/tmp/xdg")

	root, err := defaultRoot()
	if err != nil {
		t.Fatalf("defaultRoot() error = %v", err)
	}

	if root != "/tmp/custom-spirited" {
		t.Fatalf("root = %q, want %q", root, "/tmp/custom-spirited")
	}
}

func TestDefaultRootUsesXDGConfigHome(t *testing.T) {
	t.Setenv(HomeOverrideEnv, "")
	t.Setenv(XDGConfigHomeEnv, "/tmp/xdg")

	root, err := defaultRoot()
	if err != nil {
		t.Fatalf("defaultRoot() error = %v", err)
	}

	expected := filepath.Join("/tmp/xdg", "spirited-env", "environs")
	if root != expected {
		t.Fatalf("root = %q, want %q", root, expected)
	}
}

func TestDefaultRootFallsBackToDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv(HomeOverrideEnv, "")
	t.Setenv(XDGConfigHomeEnv, "")
	t.Setenv("HOME", home)

	root, err := defaultRoot()
	if err != nil {
		t.Fatalf("defaultRoot() error = %v", err)
	}

	expected := filepath.Join(home, ".config", "spirited-env", "environs")
	if root != expected {
		t.Fatalf("root = %q, want %q", root, expected)
	}
}
