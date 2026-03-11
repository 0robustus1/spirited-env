package loader

import (
	"strings"
	"testing"
)

func TestEmitPosix(t *testing.T) {
	previous := []string{"OLD", "KEEP"}
	next := map[string]string{"KEEP": "yes", "NEW": "x y"}

	out, err := Emit(ShellBash, previous, next)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if !strings.Contains(out, "unset OLD") {
		t.Fatalf("expected unset OLD in output: %s", out)
	}
	if !strings.Contains(out, "export KEEP='yes'") {
		t.Fatalf("expected KEEP export in output: %s", out)
	}
	if !strings.Contains(out, "export NEW='x y'") {
		t.Fatalf("expected NEW export in output: %s", out)
	}
}
