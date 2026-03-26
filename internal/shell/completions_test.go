package shell

import (
	"strings"
	"testing"
)

func TestCompletionFishIncludesCoreCommands(t *testing.T) {
	completion, err := Completion("fish")
	if err != nil {
		t.Fatalf("Completion() error = %v", err)
	}

	checks := []string{
		"complete -c spirited-env -f",
		"-a path",
		"-a edit",
		"-a load",
		"-a refresh",
		"-a no-env-exec",
		"-a completion",
		"-a version",
		"-a \"bash zsh fish\"",
		"-l shell -r -f -a \"bash zsh fish\"",
		"-l from -r -F",
		"__fish_complete_directories",
	}

	for _, check := range checks {
		if !strings.Contains(completion, check) {
			t.Fatalf("expected %q in completion output", check)
		}
	}
}

func TestCompletionRejectsUnsupportedShell(t *testing.T) {
	if _, err := Completion("bash"); err == nil {
		t.Fatal("expected error for unsupported completion shell")
	}
}
