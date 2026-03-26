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

Backup storage path:

- `<config-base>/backups/<canonical-source-file-path>`

## Config File

`spirited-env` reads optional configuration from `<config-base>/config.yaml`.

```yaml
merge_strategy: layered
directory_mode: "0700"
file_mode: "0600"
restore_original_values: true
report_env_changes: true
migration_suggestion_mode: off
```

Options:

- `merge_strategy`: `layered` (default) or `nearest`.
- `directory_mode`: octal permission string for created mapping directories.
- `file_mode`: octal permission string for created/enforced `.env` files.
- `restore_original_values`: `true` (default) restores pre-existing shell values when a key stops being managed.
- `report_env_changes`: `true` (default) reports loaded/unloaded variable names on directory-triggered interactive loads.
- `migration_suggestion_mode`: controls interactive migration hints for `.envrc` files. Values: `off` (default), `if_unmapped`, `always`.

When `config.yaml` is missing, defaults are used (`layered`, `0700`, `0600`, `restore_original_values: true`, `report_env_changes: true`, `migration_suggestion_mode: off`).

Change reports are emitted only for interactive shell hooks and are written to stderr, so shell-eval output on stdout remains clean for scripting.

Migration suggestions are also interactive-only and written to stderr.

To print the effective configuration as valid YAML (useful as a bootstrap file):

```bash
spirited-env config show
```

## Commands

```bash
spirited-env path [dir]
spirited-env edit [dir]
spirited-env load [dir] --shell bash|zsh|fish
spirited-env refresh [dir] [--shell bash|zsh|fish]
spirited-env status [dir]
spirited-env move <old-dir> <new-dir> [--force]
spirited-env import [dir] [--from <path>] [--replace]
spirited-env migrate [dir] [--from <path>] [--replace]
spirited-env config show
spirited-env state show
spirited-env state reset --shell bash|zsh|fish
spirited-env init bash|zsh|fish
spirited-env completion fish
spirited-env completion install fish
spirited-env doctor
spirited-env version
```

`refresh` mirrors `load` behavior and auto-detects the active shell from process parentage when `--shell` is omitted.

## Migration from direnv

You can import or migrate `.envrc` files into the centralized spirited-env mapping.

```bash
spirited-env import .
spirited-env migrate .
spirited-env import . --from ./custom.envrc --replace
```

Behavior:

- `import`: reads source assignments and writes them into the mapped `.env` file.
- `migrate`: performs import, then moves the source file into centralized backups.
- supported source lines: `KEY=VALUE` and `export KEY=VALUE` (plus comments/blank lines).
- unsupported shell syntax fails hard, reports all invalid lines, and writes nothing.

## Shell Integration (Print-Only)

Print the snippet and append it to your shell config manually.

```bash
spirited-env init bash
spirited-env init zsh
spirited-env init fish
```

For fish, both of the following are supported:

```fish
spirited-env init fish | source
eval (spirited-env init fish)
```

`spirited-env init fish | source` is the preferred form.

## Fish Completions

Generate completion script:

```fish
spirited-env completion fish
```

Install completion script into fish's user completion path:

```fish
spirited-env completion install fish
```

Manual install (equivalent):

```fish
mkdir -p ~/.config/fish/completions
spirited-env completion fish > ~/.config/fish/completions/spirited-env.fish
```

To activate immediately in the current shell:

```fish
source ~/.config/fish/completions/spirited-env.fish
```

## Parser Behavior

Supported dotenv subset:

- `KEY=VALUE`
- `export KEY=VALUE`
- comments and blank lines
- single and double quoted values

`spirited-env` does not execute shell code from env files.
