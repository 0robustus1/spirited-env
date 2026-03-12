package shell

import (
	"os/exec"
	"strings"
	"testing"
)

func TestFishSnippetIsEvalSafe(t *testing.T) {
	snippet, err := Snippet("fish")
	if err != nil {
		t.Fatalf("Snippet() error = %v", err)
	}

	if strings.HasPrefix(strings.TrimSpace(snippet), "#") {
		t.Fatalf("fish snippet must not start with a comment: %q", snippet)
	}
	if !strings.Contains(snippet, "function spirited_env_hook --on-variable PWD;") {
		t.Fatalf("missing function declaration: %s", snippet)
	}
	if !strings.Contains(snippet, "eval (string join \\n -- $output);") {
		t.Fatalf("missing eval-safe output join: %s", snippet)
	}

	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not available in PATH")
	}

	cmd := exec.Command("fish", "-c", "set -l output (cat); eval $output; functions -q spirited_env_hook; and echo defined; or echo missing")
	cmd.Stdin = strings.NewReader(snippet)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish eval test failed: %v, output=%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "defined" {
		t.Fatalf("expected defined function, got %q", strings.TrimSpace(string(out)))
	}
}

func TestFishSnippetCanBeSourced(t *testing.T) {
	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not available in PATH")
	}

	snippet, err := Snippet("fish")
	if err != nil {
		t.Fatalf("Snippet() error = %v", err)
	}

	cmd := exec.Command("fish", "-c", "source; functions -q spirited_env_hook; and echo defined; or echo missing")
	cmd.Stdin = strings.NewReader(snippet)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish source test failed: %v, output=%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "defined" {
		t.Fatalf("expected defined function, got %q", strings.TrimSpace(string(out)))
	}
}

func TestBashSnippetIsEvalSafe(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available in PATH")
	}

	snippet, err := Snippet("bash")
	if err != nil {
		t.Fatalf("Snippet() error = %v", err)
	}

	if !strings.Contains(snippet, "output=\"$(spirited-env load --shell bash)\"") {
		t.Fatalf("missing bash load call: %s", snippet)
	}
	if !strings.Contains(snippet, "eval \"$output\"") {
		t.Fatalf("missing quoted eval for bash output: %s", snippet)
	}

	cmd := exec.Command("bash", "-lc", "eval \"$1\"; type spirited_env_hook >/dev/null 2>&1 && echo defined || echo missing", "_", snippet)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash eval test failed: %v, output=%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "defined" {
		t.Fatalf("expected defined function, got %q", strings.TrimSpace(string(out)))
	}
}

func TestZshSnippetIsEvalSafe(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available in PATH")
	}

	snippet, err := Snippet("zsh")
	if err != nil {
		t.Fatalf("Snippet() error = %v", err)
	}

	if !strings.Contains(snippet, "output=\"$(spirited-env load --shell zsh)\"") {
		t.Fatalf("missing zsh load call: %s", snippet)
	}
	if !strings.Contains(snippet, "eval \"$output\"") {
		t.Fatalf("missing quoted eval for zsh output: %s", snippet)
	}

	cmd := exec.Command("zsh", "-c", "eval \"$1\"; typeset -f spirited_env_hook >/dev/null 2>&1 && echo defined || echo missing", "_", snippet)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh eval test failed: %v, output=%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "defined" {
		t.Fatalf("expected defined function, got %q", strings.TrimSpace(string(out)))
	}
}
