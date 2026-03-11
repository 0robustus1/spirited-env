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

	"github.com/0robustus1/spirited-env/internal/discovery"
	"github.com/0robustus1/spirited-env/internal/dotenv"
	"github.com/0robustus1/spirited-env/internal/loader"
	"github.com/0robustus1/spirited-env/internal/pathmap"
	"github.com/0robustus1/spirited-env/internal/shell"
	"github.com/0robustus1/spirited-env/internal/version"
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
	envPath, err := rt.Mapper.EnvFileForDir(c.Dir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(envPath), dirMode); err != nil {
		return fmt.Errorf("create mapping directory: %w", err)
	}

	if _, statErr := os.Stat(envPath); errors.Is(statErr, os.ErrNotExist) {
		file, createErr := os.OpenFile(envPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
		if createErr != nil {
			return fmt.Errorf("create env file: %w", createErr)
		}
		_ = file.Close()
	}

	if chmodErr := os.Chmod(envPath, fileMode); chmodErr != nil {
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
	managed := loader.ParseManagedKeys(os.Getenv(loader.ManagedKeysEnv))

	envFile, found, err := discovery.FindNearestEnvFile(c.Dir, rt.Mapper)
	if err != nil {
		return err
	}

	if !found {
		emitted, emitErr := loader.Emit(loader.Shell(c.Shell), managed, map[string]string{})
		if emitErr != nil {
			return emitErr
		}
		fmt.Print(emitted)
		return nil
	}

	vars, parseErr := parseEnvFile(envFile)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "spirited-env: parse error in %s: %v\n", envFile, parseErr)
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
	canonical, err := pathmap.CanonicalizeDir(c.Dir)
	if err != nil {
		return err
	}

	fmt.Printf("directory: %s\n", canonical)

	envFile, found, err := discovery.FindNearestEnvFile(c.Dir, rt.Mapper)
	if err != nil {
		return err
	}

	if !found {
		fmt.Println("status: no environment file found in ancestor mappings")
		return nil
	}

	fmt.Printf("env file: %s\n", envFile)

	vars, parseErr := parseEnvFile(envFile)
	if parseErr != nil {
		fmt.Printf("parse: error (%v)\n", parseErr)
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

	if err := os.MkdirAll(filepath.Dir(destination), dirMode); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := moveFile(source, destination); err != nil {
		return err
	}

	if err := os.Chmod(destination, fileMode); err != nil {
		return fmt.Errorf("enforce destination mode: %w", err)
	}

	fmt.Printf("moved %s -> %s\n", source, destination)
	return nil
}

type InitCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell to print integration snippet for."`
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
	fmt.Printf("store root: %s\n", rt.Mapper.Root)
	if err := os.MkdirAll(rt.Mapper.Root, dirMode); err != nil {
		return fmt.Errorf("ensure store root exists: %w", err)
	}

	info, err := os.Stat(rt.Mapper.Root)
	if err != nil {
		return err
	}

	fmt.Printf("store root mode: %04o\n", info.Mode().Perm())
	if info.Mode().Perm() != dirMode {
		fmt.Printf("warning: expected mode %04o\n", dirMode)
	}

	if envFile, found, findErr := discovery.FindNearestEnvFile("", rt.Mapper); findErr != nil {
		fmt.Printf("lookup: error (%v)\n", findErr)
	} else if !found {
		fmt.Println("lookup: no active mapping for current directory")
	} else {
		fmt.Printf("lookup: active env file %s\n", envFile)
		if _, parseErr := parseEnvFile(envFile); parseErr != nil {
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

func moveFile(source, destination string) error {
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
