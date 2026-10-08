export extern "micro" [
    -version(-v)              # Show version of eza
    -help                     # Show list of command-line options
    -clean                    # Cleans the configuration directory
    -config-dir               # Specify a custom location for the configuration directory
    -option                   # Show all option help
    -debug                    # Enable debug mode (enables logging to ./log.txt)
    -plugin                   # Manage micro plugins. Use either of "remove", "update", "search", "list", or "available" subcommands
]
