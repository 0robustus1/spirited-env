package shell

import "fmt"

func Completion(name string) (string, error) {
	switch name {
	case "fish":
		return fishCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell %q", name)
	}
}

const fishCompletion = `# fish completion for spirited-env
complete -c spirited-env -f

set -l __spirited_env_commands path edit load status move import migrate config state init completion doctor version

complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a path -d "Print mapped env file path"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a edit -d "Open mapped env file in $EDITOR"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a load -d "Emit shell commands for loading env"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a status -d "Show discovered env file and key info"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a move -d "Move mapped env file to another directory mapping"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a import -d "Import env assignments from existing file"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a migrate -d "Import env assignments and back up source"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a config -d "Show effective configuration"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a state -d "Inspect or reset internal shell state"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a init -d "Print shell integration snippet"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a completion -d "Print or install shell completions"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a doctor -d "Run health checks"
complete -c spirited-env -n "not __fish_seen_subcommand_from $__spirited_env_commands" -a version -d "Print version information"

complete -c spirited-env -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from show" -a show -d "Print effective configuration as YAML"
complete -c spirited-env -n "__fish_seen_subcommand_from state; and not __fish_seen_subcommand_from show reset" -a show -d "Print current internal state"
complete -c spirited-env -n "__fish_seen_subcommand_from state; and not __fish_seen_subcommand_from show reset" -a reset -d "Reset internal state variables"
complete -c spirited-env -n "__fish_seen_subcommand_from completion; and not __fish_seen_subcommand_from fish install" -a fish -d "Print fish completion script"
complete -c spirited-env -n "__fish_seen_subcommand_from completion; and not __fish_seen_subcommand_from fish install" -a install -d "Install shell completion script"
complete -c spirited-env -n "__fish_seen_subcommand_from completion; and __fish_seen_subcommand_from install" -a fish -d "Install fish completion"

complete -c spirited-env -n "__fish_seen_subcommand_from init" -a "bash zsh fish" -d "Shell"

complete -c spirited-env -n "__fish_seen_subcommand_from load" -l shell -r -f -a "bash zsh fish" -d "Shell syntax to emit"
complete -c spirited-env -n "__fish_seen_subcommand_from state; and __fish_seen_subcommand_from reset" -l shell -r -f -a "bash zsh fish" -d "Shell syntax to emit"
complete -c spirited-env -n "__fish_seen_subcommand_from move" -l force -d "Overwrite existing destination env file"
complete -c spirited-env -n "__fish_seen_subcommand_from import" -l from -r -F -d "Source file to import"
complete -c spirited-env -n "__fish_seen_subcommand_from import" -l replace -d "Replace destination env file"
complete -c spirited-env -n "__fish_seen_subcommand_from migrate" -l from -r -F -d "Source file to migrate"
complete -c spirited-env -n "__fish_seen_subcommand_from migrate" -l replace -d "Replace destination env file"

complete -c spirited-env -n "__fish_seen_subcommand_from path edit load status import migrate move" -a "(__fish_complete_directories)" -d "Directory"
`
