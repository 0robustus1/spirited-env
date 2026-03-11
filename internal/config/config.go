package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	EnvConfigHome = "SPIRITED_ENV_CONFIG_HOME"
	EnvEnvirons   = "SPIRITED_ENV_HOME"
	EnvXDGConfig  = "XDG_CONFIG_HOME"
)

type MergeStrategy string

const (
	MergeLayered MergeStrategy = "layered"
	MergeNearest MergeStrategy = "nearest"
)

type Paths struct {
	BaseConfigDir string
	EnvironsDir   string
	ConfigFile    string
}

type Settings struct {
	MergeStrategy MergeStrategy
	DirectoryMode os.FileMode
	FileMode      os.FileMode
}

func ResolvePaths() (Paths, error) {
	base, err := resolveBaseConfigDir()
	if err != nil {
		return Paths{}, err
	}

	environs := strings.TrimSpace(os.Getenv(EnvEnvirons))
	if environs == "" {
		environs = filepath.Join(base, "environs")
	}

	base = filepath.Clean(base)
	environs = filepath.Clean(environs)

	return Paths{
		BaseConfigDir: base,
		EnvironsDir:   environs,
		ConfigFile:    filepath.Join(base, "config.yaml"),
	}, nil
}

func resolveBaseConfigDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(EnvConfigHome)); custom != "" {
		return filepath.Clean(custom), nil
	}

	if xdg := strings.TrimSpace(os.Getenv(EnvXDGConfig)); xdg != "" {
		return filepath.Join(filepath.Clean(xdg), "spirited-env"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}

	return filepath.Join(home, ".config", "spirited-env"), nil
}

func DefaultSettings() Settings {
	return Settings{
		MergeStrategy: MergeLayered,
		DirectoryMode: 0o700,
		FileMode:      0o600,
	}
}

func LoadSettings(configFile string) (Settings, error) {
	settings := DefaultSettings()

	content, err := os.ReadFile(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return settings, nil
		}
		return Settings{}, fmt.Errorf("read config %s: %w", configFile, err)
	}

	var raw struct {
		MergeStrategy string `yaml:"merge_strategy"`
		DirectoryMode string `yaml:"directory_mode"`
		FileMode      string `yaml:"file_mode"`
	}

	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Settings{}, fmt.Errorf("parse YAML config %s: %w", configFile, err)
	}

	if raw.MergeStrategy != "" {
		strategy := MergeStrategy(strings.TrimSpace(raw.MergeStrategy))
		switch strategy {
		case MergeLayered, MergeNearest:
			settings.MergeStrategy = strategy
		default:
			return Settings{}, fmt.Errorf("invalid merge_strategy %q (expected layered or nearest)", raw.MergeStrategy)
		}
	}

	if raw.DirectoryMode != "" {
		mode, err := parseOctalMode(raw.DirectoryMode)
		if err != nil {
			return Settings{}, fmt.Errorf("invalid directory_mode: %w", err)
		}
		settings.DirectoryMode = mode
	}

	if raw.FileMode != "" {
		mode, err := parseOctalMode(raw.FileMode)
		if err != nil {
			return Settings{}, fmt.Errorf("invalid file_mode: %w", err)
		}
		settings.FileMode = mode
	}

	return settings, nil
}

func parseOctalMode(v string) (os.FileMode, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return 0, fmt.Errorf("empty value")
	}

	parsed, err := strconv.ParseUint(trimmed, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %q as octal: %w", v, err)
	}

	mode := os.FileMode(parsed)
	if mode > 0o777 {
		return 0, fmt.Errorf("mode %q exceeds permission bits", v)
	}

	return mode, nil
}
