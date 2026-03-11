package loader

import (
	"fmt"
	"sort"
	"strings"
)

const ManagedKeysEnv = "SPIRITED_ENV_KEYS"

type Shell string

const (
	ShellBash Shell = "bash"
	ShellZsh  Shell = "zsh"
	ShellFish Shell = "fish"
)

func ParseManagedKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		k := strings.TrimSpace(part)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

func Emit(shell Shell, previous []string, next map[string]string) (string, error) {
	nextKeys := sortedKeys(next)
	nextSet := make(map[string]struct{}, len(nextKeys))
	for _, k := range nextKeys {
		nextSet[k] = struct{}{}
	}

	toUnset := make([]string, 0)
	for _, k := range previous {
		if _, exists := nextSet[k]; !exists {
			toUnset = append(toUnset, k)
		}
	}

	sort.Strings(toUnset)
	managed := strings.Join(nextKeys, ",")

	switch shell {
	case ShellBash, ShellZsh:
		return emitPosix(toUnset, next, nextKeys, managed), nil
	case ShellFish:
		return emitFish(toUnset, next, nextKeys, managed), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func emitPosix(toUnset []string, values map[string]string, orderedKeys []string, managed string) string {
	var b strings.Builder
	for _, key := range toUnset {
		b.WriteString("unset ")
		b.WriteString(key)
		b.WriteString("\n")
	}

	for _, key := range orderedKeys {
		b.WriteString("export ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(posixQuote(values[key]))
		b.WriteString("\n")
	}

	b.WriteString("export ")
	b.WriteString(ManagedKeysEnv)
	b.WriteString("=")
	b.WriteString(posixQuote(managed))
	b.WriteString("\n")

	return b.String()
}

func emitFish(toUnset []string, values map[string]string, orderedKeys []string, managed string) string {
	var b strings.Builder
	for _, key := range toUnset {
		b.WriteString("set -e ")
		b.WriteString(key)
		b.WriteString("\n")
	}

	for _, key := range orderedKeys {
		b.WriteString("set -gx ")
		b.WriteString(key)
		b.WriteString(" ")
		b.WriteString(fishQuote(values[key]))
		b.WriteString("\n")
	}

	b.WriteString("set -gx ")
	b.WriteString(ManagedKeysEnv)
	b.WriteString(" ")
	b.WriteString(fishQuote(managed))
	b.WriteString("\n")

	return b.String()
}

func posixQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func fishQuote(s string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`"`, `\\"`,
		"$", `\\$`,
		"\n", `\\n`,
	)
	return `"` + replacer.Replace(s) + `"`
}
