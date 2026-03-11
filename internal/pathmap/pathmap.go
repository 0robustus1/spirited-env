package pathmap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const HomeOverrideEnv = "SPIRITED_ENV_HOME"
const XDGConfigHomeEnv = "XDG_CONFIG_HOME"

type Mapper struct {
	Root string
}

func New(root string) (Mapper, error) {
	if root == "" {
		resolved, err := defaultRoot()
		if err != nil {
			return Mapper{}, err
		}
		root = resolved
	}

	return Mapper{Root: root}, nil
}

func defaultRoot() (string, error) {
	if custom := os.Getenv(HomeOverrideEnv); custom != "" {
		return filepath.Clean(custom), nil
	}

	if xdg := strings.TrimSpace(os.Getenv(XDGConfigHomeEnv)); xdg != "" {
		return filepath.Join(filepath.Clean(xdg), "spirited-env", "environs"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}

	return filepath.Join(home, ".config", "spirited-env", "environs"), nil
}

func CanonicalizeDir(dir string) (string, error) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get cwd: %w", err)
		}
		dir = cwd
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %q: %w", dir, err)
	}

	cleaned := filepath.Clean(abs)
	realPath, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", fmt.Errorf("canonicalize path %q: %w", cleaned, err)
	}

	return realPath, nil
}

func (m Mapper) EnvFileForDir(dir string) (string, error) {
	canonical, err := CanonicalizeDir(dir)
	if err != nil {
		return "", err
	}

	relative := strings.TrimPrefix(canonical, string(filepath.Separator))
	if relative == "" {
		return filepath.Join(m.Root, ".env"), nil
	}

	return filepath.Join(m.Root, relative, ".env"), nil
}

func (m Mapper) EnvFileForCanonicalDir(canonical string) string {
	relative := strings.TrimPrefix(canonical, string(filepath.Separator))
	if relative == "" {
		return filepath.Join(m.Root, ".env")
	}

	return filepath.Join(m.Root, relative, ".env")
}
