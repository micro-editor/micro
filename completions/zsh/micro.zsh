#compdef micro                -*- shell-script -*-

# Save this file as _micro in /usr/local/share/zsh/site-functions or in any
# other folder in $fpath.  E.g. save it in a folder called ~/.zfunc and add a
# line containing `fpath=(~/.zfunc $fpath)` somewhere before `compinit` in your
# ~/.zshrc.

__micro() {
    # Give completions using the `_arguments` utility function with
    # `-s` for option stacking like `micro -ab` for `micro -a -b` and
    # `-S` for delimiting options with `--` like in `micro -- -a`.
    _arguments -s -S \
        "(- *)"{-v,-version}"[Show the version number and information]" \
        "(- *)"-help"[Show list of command-line options]" \
        -clean"[Cleans the configuration directory]" \
        -config-dir"[Specify a custom location for the configuration directory]" \
        -option"[Show all option help]" \
        -debug"[Enable debug mode (enables logging to ./log.txt)]" \
        -plugin"[Manage micro plugins]:(action):(remove update search list available)" \
        '*:filename:_files'
}
        #$(micro -options | grep -oE '^-[[:alnum:]@-]+' | tr '\n' ' ') \

__micro
