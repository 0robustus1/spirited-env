package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0robustus1/spirited-env/internal/loader"
)

func requireSmoke(t *testing.T) {
	t.Helper()
	if os.Getenv("SPIRITED_ENV_SMOKE") != "1" {
		t.Skip("set SPIRITED_ENV_SMOKE=1 to run smoke tests")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("cannot locate repo root from %s: %v", wd, err)
	}
	return root
}

func buildCLIForSmoke(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	binary := filepath.Join(t.TempDir(), "spirited-env-smoke")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/spirited-env")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build for smoke failed: %v, output=%s", err, out)
	}
	return binary
}

func TestNoEnvExecSmoke_RestoresAndUnsetsManagedState(t *testing.T) {
	requireSmoke(t)

	originals := loader.Originals{
		"A": {Set: true, Value: "orig-a"},
		"B": {Set: false},
	}
	encoded, err := loader.EncodeOriginals(originals)
	if err != nil {
		t.Fatalf("EncodeOriginals() error = %v", err)
	}

	binary := buildCLIForSmoke(t)
	cmd := exec.Command(binary, "no-env-exec", "sh", "-c", "printf 'A=%s\nB=%s\nC=%s\nKEYS=%s\nORIG=%s\n' \"${A-__UNSET__}\" \"${B-__UNSET__}\" \"${C-__UNSET__}\" \"${SPIRITED_ENV_KEYS-__UNSET__}\" \"${SPIRITED_ENV_ORIGINALS-__UNSET__}\"")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"A=managed-a",
		"B=managed-b",
		"C=managed-c",
		loader.ManagedKeysEnv+"=A,B,C",
		loader.OriginalsEnv+"="+encoded,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v, output=%s", err, out)
	}
	text := string(out)

	expect := []string{
		"A=orig-a",
		"B=__UNSET__",
		"C=__UNSET__",
		"KEYS=__UNSET__",
		"ORIG=__UNSET__",
	}
	for _, line := range expect {
		if !strings.Contains(text, line) {
			t.Fatalf("missing %q in output:\n%s", line, text)
		}
	}
}

func TestNoEnvExecSmoke_ExitCodePropagation(t *testing.T) {
	requireSmoke(t)

	binary := buildCLIForSmoke(t)
	cmd := exec.Command(binary, "no-env-exec", "sh", "-c", "exit 23")
	cmd.Dir = repoRoot(t)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	if exitErr.ExitCode() != 23 {
		t.Fatalf("exit code = %d, want 23", exitErr.ExitCode())
	}
}
