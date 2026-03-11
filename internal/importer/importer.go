package importer

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/0robustus1/spirited-env/internal/dotenv"
)

type Issue struct {
	Line    int
	Reason  string
	Content string
}

func ParseAssignmentsAll(content string) (map[string]string, []Issue, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	values := map[string]string{}
	issues := make([]Issue, 0)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parsed, err := dotenv.Parse(strings.NewReader(raw + "\n"))
		if err != nil {
			issues = append(issues, Issue{Line: lineNo, Reason: normalizeReason(err.Error()), Content: trimmed})
			continue
		}

		for k, v := range parsed {
			values[k] = v
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan source: %w", err)
	}

	if len(issues) > 0 {
		return nil, issues, nil
	}

	return values, nil, nil
}

func normalizeReason(reason string) string {
	return strings.TrimPrefix(reason, "line 1: ")
}
