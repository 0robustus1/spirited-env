package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0robustus1/spirited-env/internal/config"
	"github.com/0robustus1/spirited-env/internal/discovery"
	"github.com/0robustus1/spirited-env/internal/dotenv"
	"github.com/0robustus1/spirited-env/internal/importer"
	"github.com/0robustus1/spirited-env/internal/loader"
	"github.com/0robustus1/spirited-env/internal/pathmap"
	"github.com/0robustus1/spirited-env/internal/shell"
	"github.com/0robustus1/spirited-env/internal/version"
	"gopkg.in/yaml.v3"
)

type PathCmd struct {
	Dir string `arg:"" optional:"" help:"Directory to resolve (default: current directory)." type:"path"`
}

func (c *PathCmd) Run(rt *Runtime) error {
	envPath, err := rt.Mapper.EnvFileForDir(c.Dir)
	if err != nil {
		return err
	}
	fmt.Println(envPath)
	return nil
}

type EditCmd struct {
	Dir string `arg:"" optional:"" help:"Directory to edit mapping for (default: current directory)." type:"path"`
}

func (c *EditCmd) Run(rt *Runtime) error {
	if rt.ConfigErr != nil {
		return fmt.Errorf("load config: %w", rt.ConfigErr)
	}

	envPath, err := rt.Mapper.EnvFileForDir(c.Dir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(envPath), rt.Settings.DirectoryMode); err != nil {
		return fmt.Errorf("create mapping directory: %w", err)
	}

	if _, statErr := os.Stat(envPath); errors.Is(statErr, os.ErrNotExist) {
		file, createErr := os.OpenFile(envPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, rt.Settings.FileMode)
		if createErr != nil {
			return fmt.Errorf("create env file: %w", createErr)
		}
		_ = file.Close()
	}

	if chmodErr := os.Chmod(envPath, rt.Settings.FileMode); chmodErr != nil {
		return fmt.Errorf("enforce permissions on %s: %w", envPath, chmodErr)
	}

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, envPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open editor %q: %w", editor, err)
	}

	return nil
}

type LoadCmd struct {
	Dir         string `arg:"" optional:"" help:"Directory to load for (default: current directory)." type:"path"`
	Shell       string `required:"" enum:"bash,zsh,fish" help:"Shell syntax to emit."`
	Interactive bool   `hidden:"" help:"Enable interactive-only reporting output."`
}

func (c *LoadCmd) Run(rt *Runtime) error {
	if rt.ConfigErr != nil {
		fmt.Fprintf(os.Stderr, "spirited-env: config error: %v\n", rt.ConfigErr)
		return nil
	}

	managed := loader.ParseManagedKeys(os.Getenv(loader.ManagedKeysEnv))
	originals, originalsErr := loader.ParseOriginals(os.Getenv(loader.OriginalsEnv))
	if originalsErr != nil {
		fmt.Fprintf(os.Stderr, "spirited-env: warning: invalid %s (%v); refusing to modify environment\n", loader.OriginalsEnv, originalsErr)
		fmt.Fprintf(os.Stderr, "spirited-env: warning: recover with eval \"$(spirited-env state reset --shell %s)\"\n", c.Shell)
		return nil
	}

	current := currentEnvMap()
	vars, _, err := resolveVariables(c.Dir, rt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spirited-env: %v\n", err)
		return nil
	}

	emitted, emitErr := loader.Emit(loader.Shell(c.Shell), managed, vars, originals, current, rt.Settings.RestoreOriginalValues)
	if emitErr != nil {
		return emitErr
	}

	if rt.Settings.ReportEnvChanges && c.Interactive {
		summary := summarizeEnvChange(managed, vars, current)
		if summary.Changed {
			fmt.Fprintf(os.Stderr, "spirited-env: loaded variables: %s\n", formatKeyList(summary.Loaded))
			if len(summary.Unloaded) > 0 {
				fmt.Fprintf(os.Stderr, "spirited-env: unloaded variables: %s\n", formatKeyList(summary.Unloaded))
			}
		}
	}

	if c.Interactive && rt.Settings.MigrationSuggestion != config.MigrationSuggestionOff {
		targetDir, dirErr := pathmap.CanonicalizeDir(c.Dir)
		if dirErr == nil {
			sourcePath := filepath.Join(targetDir, ".envrc")
			migratable := isMigratableSourceFile(sourcePath)
			mappedExists := hasMappedEnvFile(targetDir, rt)
			if shouldSuggestMigration(rt.Settings.MigrationSuggestion, migratable, mappedExists) {
				fmt.Fprintf(os.Stderr, "spirited-env: detected migratable env file %s\n", sourcePath)
				fmt.Fprintf(os.Stderr, "spirited-env: run spirited-env migrate %s to import and back up it\n", strconv.Quote(targetDir))
			}
		}
	}

	fmt.Print(emitted)
	return nil
}

type StatusCmd struct {
	Dir string `arg:"" optional:"" help:"Directory to inspect (default: current directory)." type:"path"`
}

func (c *StatusCmd) Run(rt *Runtime) error {
	if rt.ConfigErr != nil {
		return fmt.Errorf("load config: %w", rt.ConfigErr)
	}

	canonical, err := pathmap.CanonicalizeDir(c.Dir)
	if err != nil {
		return err
	}

	fmt.Printf("directory: %s\n", canonical)
	fmt.Printf("strategy: %s\n", rt.Settings.MergeStrategy)
	fmt.Printf("mode: directory=%04o file=%04o\n", rt.Settings.DirectoryMode, rt.Settings.FileMode)

	files, err := resolvedEnvFiles(c.Dir, rt)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("status: no environment file found in ancestor mappings")
		return nil
	}

	if rt.Settings.MergeStrategy == config.MergeNearest {
		fmt.Printf("env file: %s\n", files[0])
	} else {
		fmt.Printf("env files (%d): %s\n", len(files), strings.Join(files, ", "))
	}

	vars, _, resolveErr := resolveVariables(c.Dir, rt)
	if resolveErr != nil {
		fmt.Printf("parse: error (%v)\n", resolveErr)
		return nil
	}

	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Printf("parse: ok (%d keys)\n", len(keys))
	if len(keys) > 0 {
		fmt.Printf("keys: %s\n", strings.Join(keys, ", "))
	}

	return nil
}

type MoveCmd struct {
	OldDir string `arg:"" help:"Original directory mapping." type:"path"`
	NewDir string `arg:"" help:"Destination directory mapping." type:"path"`
	Force  bool   `help:"Overwrite existing destination env file."`
}

type ImportCmd struct {
	Dir     string `arg:"" optional:"" help:"Directory whose mapping should receive imported variables (default: current directory)." type:"path"`
	From    string `help:"Source file to import (default: <dir>/.envrc)." type:"path"`
	Replace bool   `help:"Replace destination env file instead of merging."`
}

type MigrateCmd struct {
	Dir     string `arg:"" optional:"" help:"Directory whose mapping should receive migrated variables (default: current directory)." type:"path"`
	From    string `help:"Source file to migrate (default: <dir>/.envrc)." type:"path"`
	Replace bool   `help:"Replace destination env file instead of merging."`
}

func (c *MoveCmd) Run(rt *Runtime) error {
	if rt.ConfigErr != nil {
		return fmt.Errorf("load config: %w", rt.ConfigErr)
	}

	source, err := rt.Mapper.EnvFileForDir(c.OldDir)
	if err != nil {
		return err
	}
	destination, err := rt.Mapper.EnvFileForDir(c.NewDir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(source); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("source mapping does not exist: %s", source)
		}
		return err
	}

	if _, err := os.Stat(destination); err == nil {
		if !c.Force {
			return fmt.Errorf("destination exists: %s (use --force to overwrite)", destination)
		}
		if removeErr := os.Remove(destination); removeErr != nil {
			return fmt.Errorf("remove destination for overwrite: %w", removeErr)
		}
	}

	if err := os.MkdirAll(filepath.Dir(destination), rt.Settings.DirectoryMode); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := moveFile(source, destination, rt.Settings.FileMode); err != nil {
		return err
	}

	if err := os.Chmod(destination, rt.Settings.FileMode); err != nil {
		return fmt.Errorf("enforce destination mode: %w", err)
	}

	fmt.Printf("moved %s -> %s\n", source, destination)
	return nil
}

func (c *ImportCmd) Run(rt *Runtime) error {
	return runImportOrMigrate(rt, c.Dir, c.From, c.Replace, false)
}

func (c *MigrateCmd) Run(rt *Runtime) error {
	return runImportOrMigrate(rt, c.Dir, c.From, c.Replace, true)
}

type InitCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell to print integration snippet for."`
}

type CompletionCmd struct {
	Fish    CompletionFishCmd    `cmd:"" help:"Print fish completion script."`
	Install CompletionInstallCmd `cmd:"" help:"Install shell completion script."`
}

type CompletionFishCmd struct{}

type CompletionInstallCmd struct {
	Shell string `arg:"" enum:"fish" help:"Shell completion to install."`
}

type ConfigCmd struct {
	Show ConfigShowCmd `cmd:"" help:"Print effective configuration as YAML."`
}

type ConfigShowCmd struct{}

func (c *ConfigShowCmd) Run(rt *Runtime) error {
	settings := rt.Settings
	if rt.ConfigErr != nil {
		settings = config.DefaultSettings()
		fmt.Fprintf(os.Stderr, "spirited-env: config error in %s: %v\n", rt.Paths.ConfigFile, rt.ConfigErr)
		fmt.Fprintln(os.Stderr, "spirited-env: printing default config values")
	}

	output := struct {
		MergeStrategy        string `yaml:"merge_strategy"`
		DirectoryMode        string `yaml:"directory_mode"`
		FileMode             string `yaml:"file_mode"`
		RestoreOriginalValue bool   `yaml:"restore_original_values"`
		ReportEnvChanges     bool   `yaml:"report_env_changes"`
		MigrationSuggestion  string `yaml:"migration_suggestion_mode"`
	}{
		MergeStrategy:        string(settings.MergeStrategy),
		DirectoryMode:        fmt.Sprintf("%04o", settings.DirectoryMode),
		FileMode:             fmt.Sprintf("%04o", settings.FileMode),
		RestoreOriginalValue: settings.RestoreOriginalValues,
		ReportEnvChanges:     settings.ReportEnvChanges,
		MigrationSuggestion:  string(settings.MigrationSuggestion),
	}

	content, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	fmt.Print(string(content))
	return nil
}

func (c *InitCmd) Run(*Runtime) error {
	snippet, err := shell.Snippet(c.Shell)
	if err != nil {
		return err
	}
	fmt.Print(snippet)
	return nil
}

func (c *CompletionFishCmd) Run(*Runtime) error {
	completion, err := shell.Completion("fish")
	if err != nil {
		return err
	}
	fmt.Print(completion)
	return nil
}

func (c *CompletionInstallCmd) Run(*Runtime) error {
	completion, err := shell.Completion(c.Shell)
	if err != nil {
		return err
	}

	installPath, err := completionInstallPath(c.Shell)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(installPath), 0o755); err != nil {
		return fmt.Errorf("create completion directory: %w", err)
	}
	if err := os.WriteFile(installPath, []byte(completion), 0o644); err != nil {
		return fmt.Errorf("write completion file %s: %w", installPath, err)
	}

	fmt.Printf("installed %s completion to %s\n", c.Shell, installPath)
	return nil
}

type DoctorCmd struct{}

func (c *DoctorCmd) Run(rt *Runtime) error {
	fmt.Printf("config file: %s\n", rt.Paths.ConfigFile)
	fmt.Printf("config base: %s\n", rt.Paths.BaseConfigDir)
	fmt.Printf("store root: %s\n", rt.Mapper.Root)
	if rt.ConfigErr != nil {
		fmt.Printf("config: error (%v)\n", rt.ConfigErr)
		return nil
	}
	fmt.Printf("config: ok (strategy=%s directory_mode=%04o file_mode=%04o restore_original_values=%t report_env_changes=%t migration_suggestion_mode=%s)\n", rt.Settings.MergeStrategy, rt.Settings.DirectoryMode, rt.Settings.FileMode, rt.Settings.RestoreOriginalValues, rt.Settings.ReportEnvChanges, rt.Settings.MigrationSuggestion)

	if err := os.MkdirAll(rt.Mapper.Root, rt.Settings.DirectoryMode); err != nil {
		return fmt.Errorf("ensure store root exists: %w", err)
	}

	info, err := os.Stat(rt.Mapper.Root)
	if err != nil {
		return err
	}

	fmt.Printf("store root mode: %04o\n", info.Mode().Perm())
	if info.Mode().Perm() != rt.Settings.DirectoryMode {
		fmt.Printf("warning: expected mode %04o\n", rt.Settings.DirectoryMode)
	}

	if files, findErr := resolvedEnvFiles("", rt); findErr != nil {
		fmt.Printf("lookup: error (%v)\n", findErr)
	} else if len(files) == 0 {
		fmt.Println("lookup: no active mapping for current directory")
	} else {
		if rt.Settings.MergeStrategy == config.MergeNearest {
			fmt.Printf("lookup: active env file %s\n", files[0])
		} else {
			fmt.Printf("lookup: active env files (%d) %s\n", len(files), strings.Join(files, ", "))
		}
		if _, _, parseErr := resolveVariables("", rt); parseErr != nil {
			fmt.Printf("parse: error (%v)\n", parseErr)
		} else {
			fmt.Println("parse: ok")
		}
	}

	fmt.Println("init: print shell snippet via `spirited-env init bash|zsh|fish`")
	return nil
}

type VersionCmd struct{}

func (c *VersionCmd) Run(*Runtime) error {
	fmt.Printf("version=%s commit=%s date=%s\n", version.Version, version.Commit, version.Date)
	return nil
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values, err := dotenv.Parse(file)
	if err != nil {
		return nil, err
	}

	return values, nil
}

func moveFile(source, destination string, fileMode os.FileMode) error {
	if err := os.Rename(source, destination); err == nil {
		return nil
	}

	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	if err := os.Remove(source); err != nil {
		return err
	}

	return nil
}

func resolvedEnvFiles(dir string, rt *Runtime) ([]string, error) {
	if rt.Settings.MergeStrategy == config.MergeNearest {
		envFile, found, err := discovery.FindNearestEnvFile(dir, rt.Mapper)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		return []string{envFile}, nil
	}

	return discovery.FindLayeredEnvFiles(dir, rt.Mapper)
}

func resolveVariables(dir string, rt *Runtime) (map[string]string, []string, error) {
	files, err := resolvedEnvFiles(dir, rt)
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 {
		return map[string]string{}, nil, nil
	}

	merged := map[string]string{}
	for _, file := range files {
		vars, parseErr := parseEnvFile(file)
		if parseErr != nil {
			return nil, files, fmt.Errorf("parse error in %s: %w", file, parseErr)
		}
		for k, v := range vars {
			merged[k] = v
		}
	}

	return merged, files, nil
}

type StateCmd struct {
	Reset StateResetCmd `cmd:"" help:"Print shell commands to reset internal state variables."`
	Show  StateShowCmd  `cmd:"" help:"Print current internal state."`
}

type StateResetCmd struct {
	Shell string `required:"" enum:"bash,zsh,fish" help:"Shell syntax to emit."`
}

func (c *StateResetCmd) Run(*Runtime) error {
	emitted, err := loader.EmitReset(loader.Shell(c.Shell))
	if err != nil {
		return err
	}
	fmt.Print(emitted)
	return nil
}

type StateShowCmd struct{}

func (c *StateShowCmd) Run(*Runtime) error {
	originals, err := loader.ParseOriginals(os.Getenv(loader.OriginalsEnv))
	if err != nil {
		return fmt.Errorf("parse %s: %w", loader.OriginalsEnv, err)
	}

	output := struct {
		ManagedKeys string           `yaml:"managed_keys"`
		Originals   loader.Originals `yaml:"originals"`
	}{
		ManagedKeys: os.Getenv(loader.ManagedKeysEnv),
		Originals:   originals,
	}

	content, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	fmt.Print(string(content))
	return nil
}

func currentEnvMap() map[string]string {
	env := map[string]string{}
	for _, item := range os.Environ() {
		eq := strings.IndexByte(item, '=')
		if eq <= 0 {
			continue
		}
		env[item[:eq]] = item[eq+1:]
	}
	return env
}

type envChangeSummary struct {
	Changed  bool
	Loaded   []string
	Unloaded []string
}

func summarizeEnvChange(previous []string, next map[string]string, current map[string]string) envChangeSummary {
	nextKeys := make([]string, 0, len(next))
	for key := range next {
		nextKeys = append(nextKeys, key)
	}
	sort.Strings(nextKeys)

	prevSet := make(map[string]struct{}, len(previous))
	for _, key := range previous {
		prevSet[key] = struct{}{}
	}

	nextSet := make(map[string]struct{}, len(nextKeys))
	for _, key := range nextKeys {
		nextSet[key] = struct{}{}
	}

	unloaded := make([]string, 0)
	for _, key := range previous {
		if _, ok := nextSet[key]; !ok {
			unloaded = append(unloaded, key)
		}
	}
	sort.Strings(unloaded)

	changed := len(unloaded) > 0 || len(previous) != len(nextKeys)
	if !changed {
		for _, key := range nextKeys {
			if _, ok := prevSet[key]; !ok {
				changed = true
				break
			}
		}
	}

	if !changed {
		for _, key := range nextKeys {
			currentValue, ok := current[key]
			if !ok || currentValue != next[key] {
				changed = true
				break
			}
		}
	}

	return envChangeSummary{
		Changed:  changed,
		Loaded:   nextKeys,
		Unloaded: unloaded,
	}
}

func formatKeyList(keys []string) string {
	if len(keys) == 0 {
		return "(none)"
	}
	return strings.Join(keys, ", ")
}

func shouldSuggestMigration(mode config.MigrationSuggestionMode, migratable bool, mappedExists bool) bool {
	if !migratable {
		return false
	}

	switch mode {
	case config.MigrationSuggestionAlways:
		return true
	case config.MigrationSuggestionIfUnmapped:
		return !mappedExists
	default:
		return false
	}
}

func isMigratableSourceFile(sourcePath string) bool {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return false
	}

	values, issues, parseErr := importer.ParseAssignmentsAll(string(content))
	if parseErr != nil {
		return false
	}
	if len(issues) > 0 {
		return false
	}

	return len(values) > 0
}

func hasMappedEnvFile(dir string, rt *Runtime) bool {
	mappedPath, err := rt.Mapper.EnvFileForDir(dir)
	if err != nil {
		return false
	}

	info, statErr := os.Stat(mappedPath)
	if statErr != nil {
		return false
	}

	return !info.IsDir()
}

func completionInstallPath(shellName string) (string, error) {
	switch shellName {
	case "fish":
		configDir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if configDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve user home directory: %w", err)
			}
			configDir = filepath.Join(homeDir, ".config")
		}

		configDir, err := filepath.Abs(configDir)
		if err != nil {
			return "", fmt.Errorf("resolve completion directory: %w", err)
		}
		return filepath.Join(configDir, "fish", "completions", "spirited-env.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shellName)
	}
}

func runImportOrMigrate(rt *Runtime, dirArg, fromArg string, replace bool, migrate bool) error {
	if rt.ConfigErr != nil {
		return fmt.Errorf("load config: %w", rt.ConfigErr)
	}

	targetDir, err := pathmap.CanonicalizeDir(dirArg)
	if err != nil {
		return err
	}

	sourcePath, err := resolveSourcePath(targetDir, fromArg)
	if err != nil {
		return err
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source file %s: %w", sourcePath, err)
	}

	importedValues, issues, err := importer.ParseAssignmentsAll(string(sourceContent))
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		var b strings.Builder
		b.WriteString("failed to import: unsupported or invalid lines in source\n")
		for _, issue := range issues {
			b.WriteString(fmt.Sprintf("line %d: %s | %s\n", issue.Line, issue.Reason, issue.Content))
		}
		return errors.New(strings.TrimSuffix(b.String(), "\n"))
	}

	destinationPath, err := rt.Mapper.EnvFileForDir(targetDir)
	if err != nil {
		return err
	}

	mergedValues := importedValues
	if !replace {
		existingValues, readErr := readExistingEnvFile(destinationPath)
		if readErr != nil {
			return readErr
		}
		mergedValues = existingValues
		for k, v := range importedValues {
			mergedValues[k] = v
		}
	}

	if err := writeEnvMapping(destinationPath, mergedValues, rt.Settings.DirectoryMode, rt.Settings.FileMode); err != nil {
		return err
	}

	modeLabel := "merge"
	if replace {
		modeLabel = "replace"
	}

	if !migrate {
		fmt.Printf("imported %d keys from %s -> %s (mode=%s)\n", len(importedValues), sourcePath, destinationPath, modeLabel)
		return nil
	}

	canonicalSource, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return fmt.Errorf("canonicalize source for backup: %w", err)
	}

	backupPath, err := backupPathForSource(rt.Paths.BackupDir, canonicalSource)
	if err != nil {
		return err
	}
	backupPath, err = ensureUniqueBackupPath(backupPath)
	if err != nil {
		return err
	}

	if err := moveSourceToBackup(sourcePath, backupPath, rt.Settings.DirectoryMode); err != nil {
		return err
	}

	fmt.Printf("migrated %d keys from %s -> %s (mode=%s, backup=%s)\n", len(importedValues), sourcePath, destinationPath, modeLabel, backupPath)
	return nil
}

func resolveSourcePath(targetDir, fromArg string) (string, error) {
	if strings.TrimSpace(fromArg) == "" {
		return filepath.Join(targetDir, ".envrc"), nil
	}
	abs, err := filepath.Abs(fromArg)
	if err != nil {
		return "", fmt.Errorf("resolve source path %q: %w", fromArg, err)
	}
	return filepath.Clean(abs), nil
}

func readExistingEnvFile(path string) (map[string]string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	values, err := parseEnvFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse existing destination env file %s: %w", path, err)
	}
	return values, nil
}

func writeEnvMapping(path string, values map[string]string, dirMode os.FileMode, fileMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	content := renderEnvFile(values)
	if err := os.WriteFile(path, []byte(content), fileMode); err != nil {
		return fmt.Errorf("write destination env file %s: %w", path, err)
	}
	if err := os.Chmod(path, fileMode); err != nil {
		return fmt.Errorf("enforce destination env file mode on %s: %w", path, err)
	}
	return nil
}

func renderEnvFile(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(strconv.Quote(values[key]))
		b.WriteString("\n")
	}
	return b.String()
}

func backupPathForSource(backupRoot, sourcePath string) (string, error) {
	abs, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve backup source path %q: %w", sourcePath, err)
	}

	relative := strings.TrimPrefix(filepath.Clean(abs), string(filepath.Separator))
	if relative == "" {
		return "", fmt.Errorf("cannot derive backup path for source %q", sourcePath)
	}

	return filepath.Join(backupRoot, relative), nil
}

func ensureUniqueBackupPath(path string) (string, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path, nil
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	candidate := fmt.Sprintf("%s.%s.bak", base, time.Now().Format("20060102-150405"))
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, nil
	}

	for i := 1; i <= 1000; i++ {
		withCounter := fmt.Sprintf("%s.%03d", candidate, i)
		if _, err := os.Stat(withCounter); errors.Is(err, os.ErrNotExist) {
			return withCounter, nil
		}
	}

	return "", fmt.Errorf("unable to allocate unique backup path for %s", path)
}

func moveSourceToBackup(sourcePath, backupPath string, dirMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(backupPath), dirMode); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	if err := os.Rename(sourcePath, backupPath); err == nil {
		return nil
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source before backup: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source for backup: %w", err)
	}
	defer source.Close()

	target, err := os.OpenFile(backupPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("copy source to backup: %w", err)
	}
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("remove source after backup: %w", err)
	}
	return nil
}
