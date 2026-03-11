package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsUsesConfigOverride(t *testing.T) {
	t.Setenv(EnvConfigHome, "/tmp/spirited-config")
	t.Setenv(EnvXDGConfig, "/tmp/xdg")
	t.Setenv(EnvEnvirons, "")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.BaseConfigDir != "/tmp/spirited-config" {
		t.Fatalf("BaseConfigDir = %q", paths.BaseConfigDir)
	}
	if paths.EnvironsDir != "/tmp/spirited-config/environs" {
		t.Fatalf("EnvironsDir = %q", paths.EnvironsDir)
	}
	if paths.BackupDir != "/tmp/spirited-config/backups" {
		t.Fatalf("BackupDir = %q", paths.BackupDir)
	}
}

func TestResolvePathsUsesXDGAndEnvironsOverride(t *testing.T) {
	t.Setenv(EnvConfigHome, "")
	t.Setenv(EnvXDGConfig, "/tmp/xdg")
	t.Setenv(EnvEnvirons, "/tmp/custom-environs")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.BaseConfigDir != "/tmp/xdg/spirited-env" {
		t.Fatalf("BaseConfigDir = %q", paths.BaseConfigDir)
	}
	if paths.EnvironsDir != "/tmp/custom-environs" {
		t.Fatalf("EnvironsDir = %q", paths.EnvironsDir)
	}
	if paths.BackupDir != "/tmp/xdg/spirited-env/backups" {
		t.Fatalf("BackupDir = %q", paths.BackupDir)
	}
}

func TestResolvePathsFallsBackToDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvConfigHome, "")
	t.Setenv(EnvXDGConfig, "")
	t.Setenv(EnvEnvirons, "")
	t.Setenv("HOME", home)

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.BaseConfigDir != filepath.Join(home, ".config", "spirited-env") {
		t.Fatalf("BaseConfigDir = %q", paths.BaseConfigDir)
	}
	if paths.BackupDir != filepath.Join(home, ".config", "spirited-env", "backups") {
		t.Fatalf("BackupDir = %q", paths.BackupDir)
	}
}

func TestLoadSettingsDefaultsWhenMissing(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	settings, err := LoadSettings(file)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if settings.MergeStrategy != MergeLayered {
		t.Fatalf("MergeStrategy = %q", settings.MergeStrategy)
	}
	if settings.DirectoryMode != 0o700 {
		t.Fatalf("DirectoryMode = %04o", settings.DirectoryMode)
	}
	if settings.FileMode != 0o600 {
		t.Fatalf("FileMode = %04o", settings.FileMode)
	}
	if !settings.RestoreOriginalValues {
		t.Fatalf("RestoreOriginalValues = %t, want true", settings.RestoreOriginalValues)
	}
}

func TestLoadSettingsReadsValues(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("merge_strategy: nearest\ndirectory_mode: \"0750\"\nfile_mode: \"0640\"\nrestore_original_values: false\n")
	if err := os.WriteFile(file, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := LoadSettings(file)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if settings.MergeStrategy != MergeNearest {
		t.Fatalf("MergeStrategy = %q", settings.MergeStrategy)
	}
	if settings.DirectoryMode != 0o750 {
		t.Fatalf("DirectoryMode = %04o", settings.DirectoryMode)
	}
	if settings.FileMode != 0o640 {
		t.Fatalf("FileMode = %04o", settings.FileMode)
	}
	if settings.RestoreOriginalValues {
		t.Fatalf("RestoreOriginalValues = %t, want false", settings.RestoreOriginalValues)
	}
}

func TestLoadSettingsInvalidStrategy(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("merge_strategy: invalid\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := LoadSettings(file); err == nil {
		t.Fatal("expected error for invalid merge strategy")
	}
}
