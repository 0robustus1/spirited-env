package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0robustus1/spirited-env/internal/config"
	"github.com/0robustus1/spirited-env/internal/pathmap"
)

func TestShouldSuggestMigrationModes(t *testing.T) {
	tests := []struct {
		name        string
		mode        config.MigrationSuggestionMode
		migratable  bool
		mappedExist bool
		want        bool
	}{
		{name: "off", mode: config.MigrationSuggestionOff, migratable: true, mappedExist: false, want: false},
		{name: "always true", mode: config.MigrationSuggestionAlways, migratable: true, mappedExist: true, want: true},
		{name: "always false when not migratable", mode: config.MigrationSuggestionAlways, migratable: false, mappedExist: false, want: false},
		{name: "if_unmapped true", mode: config.MigrationSuggestionIfUnmapped, migratable: true, mappedExist: false, want: true},
		{name: "if_unmapped false", mode: config.MigrationSuggestionIfUnmapped, migratable: true, mappedExist: true, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSuggestMigration(tc.mode, tc.migratable, tc.mappedExist)
			if got != tc.want {
				t.Fatalf("shouldSuggestMigration() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestIsMigratableSourceFile(t *testing.T) {
	tmp := t.TempDir()
	valid := filepath.Join(tmp, "valid.envrc")
	invalid := filepath.Join(tmp, "invalid.envrc")
	empty := filepath.Join(tmp, "empty.envrc")

	if err := os.WriteFile(valid, []byte("export A=1\nB=two\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(valid) error = %v", err)
	}
	if err := os.WriteFile(invalid, []byte("eval $(echo nope)\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid) error = %v", err)
	}
	if err := os.WriteFile(empty, []byte("# comment\n\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(empty) error = %v", err)
	}

	if !isMigratableSourceFile(valid) {
		t.Fatal("expected valid source to be migratable")
	}
	if isMigratableSourceFile(invalid) {
		t.Fatal("expected invalid source to be non-migratable")
	}
	if isMigratableSourceFile(empty) {
		t.Fatal("expected empty source to be non-migratable")
	}
	if isMigratableSourceFile(filepath.Join(tmp, "missing.envrc")) {
		t.Fatal("expected missing source to be non-migratable")
	}
}

func TestHasMappedEnvFile(t *testing.T) {
	root := t.TempDir()
	mapper, err := pathmap.New(root)
	if err != nil {
		t.Fatalf("pathmap.New() error = %v", err)
	}

	rt := &Runtime{Mapper: mapper}
	project := t.TempDir()

	if hasMappedEnvFile(project, rt) {
		t.Fatal("expected no mapped env file")
	}

	mappedPath, err := mapper.EnvFileForDir(project)
	if err != nil {
		t.Fatalf("EnvFileForDir() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(mappedPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(mappedPath, []byte("A=1\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !hasMappedEnvFile(project, rt) {
		t.Fatal("expected mapped env file to exist")
	}
}
