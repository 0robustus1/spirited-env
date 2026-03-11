# spirited-env

`spirited-env` loads environment variables based on the directory you are currently in, while storing env files in a central location.

## Install

```bash
go install github.com/0robustus1/spirited-env/cmd/spirited-env@latest
```

## How Mapping Works

- Project paths are canonicalized (`Abs + Clean + EvalSymlinks`).
- Canonical path `/Users/tim/work/app` maps to:
  - `~/.config/spirited-env/environs/Users/tim/work/app/.env`
- Default behavior is layered merge (parent directories first, current directory overrides).

Config path precedence:

1. `SPIRITED_ENV_CONFIG_HOME`
2. `XDG_CONFIG_HOME/spirited-env`
3. `~/.config/spirited-env`

Environs storage precedence:

1. `SPIRITED_ENV_HOME`
2. `<config-base>/environs`

## Config File

`spirited-env` reads optional configuration from `<config-base>/config.yaml`.

```yaml
merge_strategy: layered
directory_mode: "0700"
file_mode: "0600"
restore_original_values: true
```

Options:

- `merge_strategy`: `layered` (default) or `nearest`.
- `directory_mode`: octal permission string for created mapping directories.
- `file_mode`: octal permission string for created/enforced `.env` files.
- `restore_original_values`: `true` (default) restores pre-existing shell values when a key stops being managed.

When `config.yaml` is missing, defaults are used (`layered`, `0700`, `0600`).

To print the effective configuration as valid YAML (useful as a bootstrap file):

```bash
spirited-env config show
```

## Commands

```bash
spirited-env path [dir]
spirited-env edit [dir]
spirited-env load [dir] --shell bash|zsh|fish
spirited-env status [dir]
spirited-env move <old-dir> <new-dir> [--force]
spirited-env config show
spirited-env state show
spirited-env state reset --shell bash|zsh|fish
spirited-env init bash|zsh|fish
spirited-env doctor
spirited-env version
```

## Shell Integration (Print-Only)

Print the snippet and append it to your shell config manually.

```bash
spirited-env init bash
spirited-env init zsh
spirited-env init fish
```

## Parser Behavior

Supported dotenv subset:

- `KEY=VALUE`
- `export KEY=VALUE`
- comments and blank lines
- single and double quoted values

`spirited-env` does not execute shell code from env files.
