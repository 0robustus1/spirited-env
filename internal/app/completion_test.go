package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionInstallPathFish(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	path, err := completionInstallPath("fish")
	if err != nil {
		t.Fatalf("completionInstallPath() error = %v", err)
	}

	expected := filepath.Join(tmp, "fish", "completions", "spirited-env.fish")
	if path != expected {
		t.Fatalf("path = %q, want %q", path, expected)
	}
}

func TestCompletionInstallPathFishFallsBackToDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	path, err := completionInstallPath("fish")
	if err != nil {
		t.Fatalf("completionInstallPath() error = %v", err)
	}

	expected := filepath.Join(home, ".config", "fish", "completions", "spirited-env.fish")
	if path != expected {
		t.Fatalf("path = %q, want %q", path, expected)
	}
}

func TestCompletionInstallPathRejectsUnsupportedShell(t *testing.T) {
	if _, err := completionInstallPath("bash"); err == nil {
		t.Fatal("expected error for unsupported shell")
	}
}

func TestCompletionInstallCmdRunWritesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cmd := CompletionInstallCmd{Shell: "fish"}
	if err := cmd.Run(nil); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	installed := filepath.Join(tmp, "fish", "completions", "spirited-env.fish")
	content, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "complete -c spirited-env") {
		t.Fatalf("expected fish completion content, got: %s", string(content))
	}
}
