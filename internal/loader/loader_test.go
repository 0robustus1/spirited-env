package loader

import (
	"encoding/base64"
	"os/exec"
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
	next := map[string]string{"A": "1", "B": "two words", "C": "$HOME/bin"}

	out, err := Emit(ShellFish, previous, next, Originals{}, map[string]string{}, false)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if !strings.Contains(out, "set -e OLD;\n") {
		t.Fatalf("expected unset command terminator in output: %s", out)
	}
	if !strings.Contains(out, "set -gx A '1';\n") {
		t.Fatalf("expected export command terminator for A: %s", out)
	}
	if !strings.Contains(out, "set -gx B 'two words';\n") {
		t.Fatalf("expected export command terminator for B: %s", out)
	}
	if !strings.Contains(out, "set -gx C '$HOME/bin';\n") {
		t.Fatalf("expected literal dollar value export for C: %s", out)
	}
	if !strings.Contains(out, "set -gx "+ManagedKeysEnv+" 'A,B,C';\n") {
		t.Fatalf("expected managed keys command terminator: %s", out)
	}
	if !strings.Contains(out, "set -gx "+OriginalsEnv) {
		t.Fatalf("expected originals command in output: %s", out)
	}
	if !strings.HasSuffix(out, ";\n") {
		t.Fatalf("expected trailing command terminator: %q", out)
	}
}

func TestFishQuoteEscapesSingleQuoteAndPreservesLiterals(t *testing.T) {
	if got := fishQuote("$HOME/bin"); got != "'$HOME/bin'" {
		t.Fatalf("fishQuote($) = %q", got)
	}
	if got := fishQuote("it's"); got != "'it'\\''s'" {
		t.Fatalf("fishQuote(apostrophe) = %q", got)
	}
	if got := fishQuote("line1\nline2"); got != "'line1\\nline2'" {
		t.Fatalf("fishQuote(newline) = %q", got)
	}
}

func TestEmitFishRoundTripSpecialValues(t *testing.T) {
	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not available in PATH")
	}

	next := map[string]string{
		"DOLLAR": "$HOME/bin",
		"QUOTE":  "it's",
		"MULTI":  "line1\\nline2",
	}

	out, err := Emit(ShellFish, nil, next, Originals{}, map[string]string{}, false)
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	cmd := exec.Command("fish", "-c", "set -l output (cat); eval (string join \\n -- $output); printf 'DOLLAR=%s\\n' (printf '%s' \"$DOLLAR\" | base64); printf 'QUOTE=%s\\n' (printf '%s' \"$QUOTE\" | base64); printf 'MULTI=%s\\n' (printf '%s' \"$MULTI\" | base64)")
	cmd.Stdin = strings.NewReader(out)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish round-trip command failed: %v, output=%s", err, raw)
	}

	text := string(raw)
	if !strings.Contains(text, "DOLLAR="+base64.StdEncoding.EncodeToString([]byte(next["DOLLAR"]))) {
		t.Fatalf("DOLLAR mismatch, got output: %s", text)
	}
	if !strings.Contains(text, "QUOTE="+base64.StdEncoding.EncodeToString([]byte(next["QUOTE"]))) {
		t.Fatalf("QUOTE mismatch, got output: %s", text)
	}
	if !strings.Contains(text, "MULTI="+base64.StdEncoding.EncodeToString([]byte(next["MULTI"]))) {
		t.Fatalf("MULTI mismatch, got output: %s", text)
	}
}
