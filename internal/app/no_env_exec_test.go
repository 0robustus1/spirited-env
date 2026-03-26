package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/0robustus1/spirited-env/internal/loader"
)

func TestBuildNoEnvEnvironmentRestoresAndUnsets(t *testing.T) {
	originals := loader.Originals{
		"RESTORE": {Set: true, Value: "before"},
		"DROP":    {Set: false},
	}
	encoded, err := loader.EncodeOriginals(originals)
	if err != nil {
		t.Fatalf("EncodeOriginals() error = %v", err)
	}

	input := []string{
		"RESTORE=managed-now",
		"DROP=managed-now",
		"NEW_ONLY=managed-now",
		loader.ManagedKeysEnv + "=DROP,NEW_ONLY,RESTORE",
		loader.OriginalsEnv + "=" + encoded,
		"KEEP=ok",
	}

	got, err := buildNoEnvEnvironment(input)
	if err != nil {
		t.Fatalf("buildNoEnvEnvironment() error = %v", err)
	}

	values := envListToMap(got)
	if values["RESTORE"] != "before" {
		t.Fatalf("RESTORE = %q, want before", values["RESTORE"])
	}
	if _, ok := values["DROP"]; ok {
		t.Fatal("DROP should be unset")
	}
	if _, ok := values["NEW_ONLY"]; ok {
		t.Fatal("NEW_ONLY should be unset")
	}
	if _, ok := values[loader.ManagedKeysEnv]; ok {
		t.Fatalf("%s should be removed", loader.ManagedKeysEnv)
	}
	if _, ok := values[loader.OriginalsEnv]; ok {
		t.Fatalf("%s should be removed", loader.OriginalsEnv)
	}
	if values["KEEP"] != "ok" {
		t.Fatalf("KEEP = %q, want ok", values["KEEP"])
	}
}

func TestBuildNoEnvEnvironmentInvalidOriginals(t *testing.T) {
	_, err := buildNoEnvEnvironment([]string{loader.OriginalsEnv + "=not-base64"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRunNoEnvExecMissingCommand(t *testing.T) {
	err := runNoEnvExec(nil, nil, func(string, []string, []string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "no command provided") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNoEnvExecPassesThroughArgsAndEnv(t *testing.T) {
	var gotPath string
	var gotArgv []string
	var gotEnv []string

	execFn := func(path string, argv []string, env []string) error {
		gotPath = path
		gotArgv = append([]string{}, argv...)
		gotEnv = append([]string{}, env...)
		return nil
	}

	originals := loader.Originals{"A": {Set: true, Value: "orig"}}
	encoded, err := loader.EncodeOriginals(originals)
	if err != nil {
		t.Fatalf("EncodeOriginals() error = %v", err)
	}

	env := []string{
		"A=managed",
		loader.ManagedKeysEnv + "=A",
		loader.OriginalsEnv + "=" + encoded,
	}

	err = runNoEnvExec([]string{"/bin/sh", "-c", "echo ok"}, env, execFn)
	if err != nil {
		t.Fatalf("runNoEnvExec() error = %v", err)
	}

	if gotPath != "/bin/sh" {
		t.Fatalf("path = %q, want /bin/sh", gotPath)
	}
	if want := []string{"/bin/sh", "-c", "echo ok"}; !reflect.DeepEqual(gotArgv, want) {
		t.Fatalf("argv = %v, want %v", gotArgv, want)
	}
	values := envListToMap(gotEnv)
	if values["A"] != "orig" {
		t.Fatalf("A = %q, want orig", values["A"])
	}
}

func TestRunNoEnvExecPropagatesExecError(t *testing.T) {
	want := errors.New("boom")
	err := runNoEnvExec([]string{"/bin/sh"}, nil, func(string, []string, []string) error {
		return want
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}
