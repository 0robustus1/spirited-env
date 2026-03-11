package discovery

import (
	"os"
	"path/filepath"

	"github.com/0robustus1/spirited-env/internal/pathmap"
)

func FindNearestEnvFile(startDir string, mapper pathmap.Mapper) (string, bool, error) {
	canonical, err := pathmap.CanonicalizeDir(startDir)
	if err != nil {
		return "", false, err
	}

	current := canonical
	for {
		candidate := mapper.EnvFileForCanonicalDir(current)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, true, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", false, nil
}

func FindLayeredEnvFiles(startDir string, mapper pathmap.Mapper) ([]string, error) {
	canonical, err := pathmap.CanonicalizeDir(startDir)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0)
	current := canonical
	for {
		candidate := mapper.EnvFileForCanonicalDir(current)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			matches = append(matches, candidate)
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	for i, j := 0, len(matches)-1; i < j; i, j = i+1, j-1 {
		matches[i], matches[j] = matches[j], matches[i]
	}

	return matches, nil
}
