package loader

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ManagedKeysEnv = "SPIRITED_ENV_KEYS"
const OriginalsEnv = "SPIRITED_ENV_ORIGINALS"

type OriginalValue struct {
	Set   bool   `json:"set"`
	Value string `json:"value"`
}

type Originals map[string]OriginalValue

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

func ParseOriginals(raw string) (Originals, error) {
	if strings.TrimSpace(raw) == "" {
		return Originals{}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode originals: %w", err)
	}

	var originals Originals
	if err := json.Unmarshal(decoded, &originals); err != nil {
		return nil, fmt.Errorf("unmarshal originals JSON: %w", err)
	}
	if originals == nil {
		return Originals{}, nil
	}

	return originals, nil
}

func EncodeOriginals(originals Originals) (string, error) {
	if originals == nil {
		originals = Originals{}
	}
	encoded, err := json.Marshal(originals)
	if err != nil {
		return "", fmt.Errorf("marshal originals JSON: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encoded), nil
}

func Emit(shell Shell, previous []string, next map[string]string, originals Originals, current map[string]string, restoreOriginals bool) (string, error) {
	if originals == nil {
		originals = Originals{}
	}

	nextKeys := sortedKeys(next)
	prevSet := make(map[string]struct{}, len(previous))
	for _, k := range previous {
		prevSet[k] = struct{}{}
	}
	nextSet := make(map[string]struct{}, len(nextKeys))
	for _, k := range nextKeys {
		nextSet[k] = struct{}{}
	}

	toRelease := make([]string, 0)
	for _, k := range previous {
		if _, exists := nextSet[k]; !exists {
			toRelease = append(toRelease, k)
		}
	}

	toAcquire := make([]string, 0)
	for _, k := range nextKeys {
		if _, exists := prevSet[k]; !exists {
			toAcquire = append(toAcquire, k)
		}
	}

	sort.Strings(toRelease)
	managed := strings.Join(nextKeys, ",")

	toUnset := make([]string, 0, len(toRelease))
	toRestore := map[string]string{}
	if restoreOriginals {
		for _, key := range toAcquire {
			if _, exists := originals[key]; exists {
				continue
			}
			if value, ok := current[key]; ok {
				originals[key] = OriginalValue{Set: true, Value: value}
			} else {
				originals[key] = OriginalValue{Set: false}
			}
		}

		for _, key := range toRelease {
			if original, exists := originals[key]; exists {
				if original.Set {
					toRestore[key] = original.Value
				} else {
					toUnset = append(toUnset, key)
				}
				delete(originals, key)
				continue
			}
			toUnset = append(toUnset, key)
		}
	} else {
		toUnset = append(toUnset, toRelease...)
		for _, key := range toRelease {
			delete(originals, key)
		}
	}
	sort.Strings(toUnset)

	encodedOriginals, err := EncodeOriginals(originals)
	if err != nil {
		return "", err
	}

	switch shell {
	case ShellBash, ShellZsh:
		return emitPosix(toUnset, toRestore, next, nextKeys, managed, encodedOriginals), nil
	case ShellFish:
		return emitFish(toUnset, toRestore, next, nextKeys, managed, encodedOriginals), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func EmitReset(shell Shell) (string, error) {
	switch shell {
	case ShellBash, ShellZsh:
		return "unset " + ManagedKeysEnv + "\nunset " + OriginalsEnv + "\n", nil
	case ShellFish:
		return "set -e " + ManagedKeysEnv + "\nset -e " + OriginalsEnv + "\n", nil
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

func emitPosix(toUnset []string, toRestore map[string]string, values map[string]string, orderedKeys []string, managed string, encodedOriginals string) string {
	var b strings.Builder
	for _, key := range toUnset {
		b.WriteString("unset ")
		b.WriteString(key)
		b.WriteString("\n")
	}

	restoreKeys := sortedKeys(toRestore)
	for _, key := range restoreKeys {
		b.WriteString("export ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(posixQuote(toRestore[key]))
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
	b.WriteString("export ")
	b.WriteString(OriginalsEnv)
	b.WriteString("=")
	b.WriteString(posixQuote(encodedOriginals))
	b.WriteString("\n")

	return b.String()
}

func emitFish(toUnset []string, toRestore map[string]string, values map[string]string, orderedKeys []string, managed string, encodedOriginals string) string {
	var b strings.Builder
	for _, key := range toUnset {
		b.WriteString("set -e ")
		b.WriteString(key)
		b.WriteString(";\n")
	}

	restoreKeys := sortedKeys(toRestore)
	for _, key := range restoreKeys {
		b.WriteString("set -gx ")
		b.WriteString(key)
		b.WriteString(" ")
		b.WriteString(fishQuote(toRestore[key]))
		b.WriteString(";\n")
	}

	for _, key := range orderedKeys {
		b.WriteString("set -gx ")
		b.WriteString(key)
		b.WriteString(" ")
		b.WriteString(fishQuote(values[key]))
		b.WriteString(";\n")
	}

	b.WriteString("set -gx ")
	b.WriteString(ManagedKeysEnv)
	b.WriteString(" ")
	b.WriteString(fishQuote(managed))
	b.WriteString(";\n")
	b.WriteString("set -gx ")
	b.WriteString(OriginalsEnv)
	b.WriteString(" ")
	b.WriteString(fishQuote(encodedOriginals))
	b.WriteString(";\n")

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
