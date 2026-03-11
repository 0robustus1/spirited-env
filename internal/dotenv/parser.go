package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

func Parse(r io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(r)
	result := make(map[string]string)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		eq := strings.IndexRune(line, '=')
		if eq < 1 {
			return nil, &ParseError{Line: lineNo, Msg: "missing '=' assignment"}
		}

		key := strings.TrimSpace(line[:eq])
		if !envKeyPattern.MatchString(key) {
			return nil, &ParseError{Line: lineNo, Msg: fmt.Sprintf("invalid key %q", key)}
		}

		rawVal := strings.TrimSpace(line[eq+1:])
		parsedVal, err := parseValue(rawVal)
		if err != nil {
			return nil, &ParseError{Line: lineNo, Msg: err.Error()}
		}

		result[key] = parsedVal
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	if strings.HasPrefix(raw, "\"") {
		if !strings.HasSuffix(raw, "\"") || len(raw) < 2 {
			return "", fmt.Errorf("unterminated double-quoted value")
		}
		v, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("invalid double-quoted value: %w", err)
		}
		return v, nil
	}

	if strings.HasPrefix(raw, "'") {
		if !strings.HasSuffix(raw, "'") || len(raw) < 2 {
			return "", fmt.Errorf("unterminated single-quoted value")
		}
		return raw[1 : len(raw)-1], nil
	}

	return stripInlineComment(raw), nil
}

func stripInlineComment(raw string) string {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '#' {
			if i == 0 {
				return ""
			}
			if raw[i-1] == ' ' || raw[i-1] == '\t' {
				return strings.TrimSpace(raw[:i])
			}
		}
	}

	return strings.TrimSpace(raw)
}
