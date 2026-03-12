package loader

import (
	"strings"
	"testing"
)

func TestEmitRestoresOriginalValues(t *testing.T) {
	previous := []string{"OLD", "KEEP"}
	next := map[string]string{"KEEP": "yes", "NEW": "x y"}
	originals := Originals{
		"OLD": {Set: true, Value: "initial"},
	}
	current := map[string]string{"NEW": "before-new"}

	out, err := Emit(ShellBash, previous, next, originals, current, true)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if !strings.Contains(out, "export OLD='initial'") {
		t.Fatalf("expected OLD restoration in output: %s", out)
	}
	if !strings.Contains(out, "export KEEP='yes'") {
		t.Fatalf("expected KEEP export in output: %s", out)
	}
	if !strings.Contains(out, "export NEW='x y'") {
		t.Fatalf("expected NEW export in output: %s", out)
	}
}

func TestEmitUnsetsWithoutRestore(t *testing.T) {
	previous := []string{"OLD"}
	next := map[string]string{}
	originals := Originals{"OLD": {Set: true, Value: "initial"}}

	out, err := Emit(ShellBash, previous, next, originals, map[string]string{}, false)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if !strings.Contains(out, "unset OLD") {
		t.Fatalf("expected OLD unset in output: %s", out)
	}
	if strings.Contains(out, "export OLD='initial'") {
		t.Fatalf("did not expect OLD restoration when disabled: %s", out)
	}
}

func TestOriginalsRoundTrip(t *testing.T) {
	originals := Originals{
		"A": {Set: true, Value: "v"},
		"B": {Set: false, Value: ""},
	}

	encoded, err := EncodeOriginals(originals)
	if err != nil {
		t.Fatalf("EncodeOriginals() error = %v", err)
	}

	decoded, err := ParseOriginals(encoded)
	if err != nil {
		t.Fatalf("ParseOriginals() error = %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("len(decoded) = %d, want 2", len(decoded))
	}
	if got := decoded["A"]; !got.Set || got.Value != "v" {
		t.Fatalf("decoded[A] = %+v", got)
	}
	if got := decoded["B"]; got.Set || got.Value != "" {
		t.Fatalf("decoded[B] = %+v", got)
	}
}

func TestEmitFishTerminatesCommands(t *testing.T) {
	previous := []string{"OLD"}
	next := map[string]string{"A": "1", "B": "two words"}

	out, err := Emit(ShellFish, previous, next, Originals{}, map[string]string{}, false)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if !strings.Contains(out, "set -e OLD;\n") {
		t.Fatalf("expected unset command terminator in output: %s", out)
	}
	if !strings.Contains(out, "set -gx A \"1\";\n") {
		t.Fatalf("expected export command terminator for A: %s", out)
	}
	if !strings.Contains(out, "set -gx B \"two words\";\n") {
		t.Fatalf("expected export command terminator for B: %s", out)
	}
	if !strings.Contains(out, "set -gx "+ManagedKeysEnv+" \"A,B\";\n") {
		t.Fatalf("expected managed keys command terminator: %s", out)
	}
	if !strings.Contains(out, "set -gx "+OriginalsEnv) {
		t.Fatalf("expected originals command in output: %s", out)
	}
	if !strings.HasSuffix(out, ";\n") {
		t.Fatalf("expected trailing command terminator: %q", out)
	}
}
