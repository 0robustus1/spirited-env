package app

import (
	"reflect"
	"testing"
)

func TestSummarizeEnvChangeNoChange(t *testing.T) {
	previous := []string{"A", "B"}
	next := map[string]string{"A": "1", "B": "2"}
	current := map[string]string{"A": "1", "B": "2"}

	summary := summarizeEnvChange(previous, next, current)
	if summary.Changed {
		t.Fatalf("Changed = %t, want false", summary.Changed)
	}
	if got, want := summary.Loaded, []string{"A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Loaded = %v, want %v", got, want)
	}
	if len(summary.Unloaded) != 0 {
		t.Fatalf("Unloaded = %v, want empty", summary.Unloaded)
	}
}

func TestSummarizeEnvChangeValueChanged(t *testing.T) {
	previous := []string{"A", "B"}
	next := map[string]string{"A": "1", "B": "2-new"}
	current := map[string]string{"A": "1", "B": "2"}

	summary := summarizeEnvChange(previous, next, current)
	if !summary.Changed {
		t.Fatalf("Changed = %t, want true", summary.Changed)
	}
	if got, want := summary.Loaded, []string{"A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Loaded = %v, want %v", got, want)
	}
	if len(summary.Unloaded) != 0 {
		t.Fatalf("Unloaded = %v, want empty", summary.Unloaded)
	}
}

func TestSummarizeEnvChangeLoadAndUnload(t *testing.T) {
	previous := []string{"OLD", "KEEP"}
	next := map[string]string{"KEEP": "ok", "NEW": "v"}
	current := map[string]string{"KEEP": "ok", "OLD": "old"}

	summary := summarizeEnvChange(previous, next, current)
	if !summary.Changed {
		t.Fatalf("Changed = %t, want true", summary.Changed)
	}
	if got, want := summary.Loaded, []string{"KEEP", "NEW"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Loaded = %v, want %v", got, want)
	}
	if got, want := summary.Unloaded, []string{"OLD"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Unloaded = %v, want %v", got, want)
	}
}
