package shell

import "fmt"

func Snippet(name string) (string, error) {
	switch name {
	case "bash":
		return bashSnippet, nil
	case "zsh":
		return zshSnippet, nil
	case "fish":
		return fishSnippet, nil
	default:
		return "", fmt.Errorf("unsupported shell %q", name)
	}
}

const bashSnippet = `# spirited-env (bash)
spirited_env_hook() {
  local current="$PWD"
  if [ "$current" = "$SPIRITED_ENV_LAST_PWD" ]; then
    return
  fi
  SPIRITED_ENV_LAST_PWD="$current"

  local output
  output="$(spirited-env load --shell bash)" || return
  eval "$output"
}

if [[ ";${PROMPT_COMMAND};" != *";spirited_env_hook;"* ]]; then
  PROMPT_COMMAND="spirited_env_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi
`

const zshSnippet = `# spirited-env (zsh)
spirited_env_hook() {
  local current="$PWD"
  if [[ "$current" == "$SPIRITED_ENV_LAST_PWD" ]]; then
    return
  fi
  SPIRITED_ENV_LAST_PWD="$current"

  local output
  output="$(spirited-env load --shell zsh)" || return
  eval "$output"
}

autoload -U add-zsh-hook
add-zsh-hook chpwd spirited_env_hook
add-zsh-hook precmd spirited_env_hook
`

const fishSnippet = `function spirited_env_hook --on-variable PWD;
  set -l output (spirited-env load --shell fish);
  if test $status -ne 0;
    return;
  end;
  eval (string join \n -- $output);
end
`
