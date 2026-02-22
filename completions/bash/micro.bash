#compdef micro                -*- shell-script -*-

# Shell completion for micro command
# To be installed in "/usr/share/bash-completion/completions/micro"
# and "/usr/share/zsh/site-functions/"

_micro() {
	local prev cur plugin_opts all_opts
	COMPREPLY=()

	prev="${COMP_WORDS[COMP_CWORD-1]}"
	cur="${COMP_WORDS[COMP_CWORD]}"
	plugin_opts='remove update search list available'
    all_opts='-clean -config-dir -options -debug -version -plugin'

	case "${prev}" in
		-plugin)
			COMPREPLY=( $(compgen -W "${plugin_opts}" -- "${cur}") )
			return 0
			;;
		-config-dir)
			_filedir -d
			return 0
			;;
		*)
            _filedir -d
		;;
	esac

	# Options
	case "${cur}" in
		-*)
			COMPREPLY=( $( compgen -W "${all_opts}" -- "${cur}") )
			return 0
			;;
		--*)
			COMPREPLY=( $( compgen -W "--version --help" -- "${cur}") )
			return 0
			;;
		*)
			COMPREPLY=( $( compgen -W "${all_opts}" -- "${cur}") )
			return 0
			;;
	esac
}

if [[ -n ${ZSH_VERSION} ]]; then
	autoload -U bashcompinit
	bashcompinit
fi

complete -F _micro micro
