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
__spirited_env_hook() {
  [[ $- == *i* ]] || return

  local current="$PWD"
  if [ "$current" = "$SPIRITED_ENV_LAST_PWD" ]; then
    return
  fi
  SPIRITED_ENV_LAST_PWD="$current"

  local output
  output="$(spirited-env load --shell bash --interactive)" || return
  eval "$output"
}

if [[ ";${PROMPT_COMMAND};" != *";__spirited_env_hook;"* ]]; then
  PROMPT_COMMAND="__spirited_env_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi
`

const zshSnippet = `# spirited-env (zsh)
__spirited_env_hook() {
  [[ -o interactive ]] || return

  local current="$PWD"
  if [[ "$current" == "$SPIRITED_ENV_LAST_PWD" ]]; then
    return
  fi
  SPIRITED_ENV_LAST_PWD="$current"

  local output
  output="$(spirited-env load --shell zsh --interactive)" || return
  eval "$output"
}

autoload -U add-zsh-hook
add-zsh-hook chpwd __spirited_env_hook
add-zsh-hook precmd __spirited_env_hook
`

const fishSnippet = `function __spirited_env_hook --on-variable PWD;
  status is-interactive; or return;
  set -l output (spirited-env load --shell fish --interactive);
  if test $status -ne 0;
    return;
  end;
  eval (string join \n -- $output);
end
`
