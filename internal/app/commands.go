package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0robustus1/spirited-env/internal/config"
	"github.com/0robustus1/spirited-env/internal/discovery"
	"github.com/0robustus1/spirited-env/internal/dotenv"
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
	Dir   string `arg:"" optional:"" help:"Directory to load for (default: current directory)." type:"path"`
	Shell string `required:"" enum:"bash,zsh,fish" help:"Shell syntax to emit."`
}

func (c *LoadCmd) Run(rt *Runtime) error {
	if rt.ConfigErr != nil {
		fmt.Fprintf(os.Stderr, "spirited-env: config error: %v\n", rt.ConfigErr)
		return nil
	}

	managed := loader.ParseManagedKeys(os.Getenv(loader.ManagedKeysEnv))
	vars, _, err := resolveVariables(c.Dir, rt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spirited-env: %v\n", err)
		return nil
	}

	if len(vars) == 0 {
		emitted, emitErr := loader.Emit(loader.Shell(c.Shell), managed, map[string]string{})
		if emitErr != nil {
			return emitErr
		}
		fmt.Print(emitted)
		return nil
	}

	emitted, emitErr := loader.Emit(loader.Shell(c.Shell), managed, vars)
	if emitErr != nil {
		return emitErr
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

type InitCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell to print integration snippet for."`
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
		MergeStrategy string `yaml:"merge_strategy"`
		DirectoryMode string `yaml:"directory_mode"`
		FileMode      string `yaml:"file_mode"`
	}{
		MergeStrategy: string(settings.MergeStrategy),
		DirectoryMode: fmt.Sprintf("%04o", settings.DirectoryMode),
		FileMode:      fmt.Sprintf("%04o", settings.FileMode),
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

type DoctorCmd struct{}

func (c *DoctorCmd) Run(rt *Runtime) error {
	fmt.Printf("config file: %s\n", rt.Paths.ConfigFile)
	fmt.Printf("config base: %s\n", rt.Paths.BaseConfigDir)
	fmt.Printf("store root: %s\n", rt.Mapper.Root)
	if rt.ConfigErr != nil {
		fmt.Printf("config: error (%v)\n", rt.ConfigErr)
		return nil
	}
	fmt.Printf("config: ok (strategy=%s directory_mode=%04o file_mode=%04o)\n", rt.Settings.MergeStrategy, rt.Settings.DirectoryMode, rt.Settings.FileMode)

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
