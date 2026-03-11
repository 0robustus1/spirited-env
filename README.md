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
- Mapping directories are created with mode `0700`.
- `.env` files are created/enforced with mode `0600`.

Config root precedence:

1. `SPIRITED_ENV_HOME`
2. `XDG_CONFIG_HOME/spirited-env/environs`
3. `~/.config/spirited-env/environs`

## Commands

```bash
spirited-env path [dir]
spirited-env edit [dir]
spirited-env load [dir] --shell bash|zsh|fish
spirited-env status [dir]
spirited-env move <old-dir> <new-dir> [--force]
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
