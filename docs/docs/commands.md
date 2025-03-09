# Commands

Below is more information about the commands to interact with your ISLE site(s).

Running these commands assummes you've already created contexts for the ISLE installs you want to use `islectl` to help manage. If you haven't done that yet, take a look at the [Getting Started](./install/) documentation.

```
$ islectl --help
Interact with your ISLE site

Usage:
  islectl [command]

Available Commands:
  build       Build the Drupal container.
  completion  Generate the autocompletion script for the specified shell
  config      Manage ISLE command configuration
  create-site Create an ISLE site and its islectl context.
  down        Run docker compose down
  drush       Run drush commands on ISLE contexts
  help        Help about any command
  make        Run custom make commands
  pull        Fetches the latest images from the registry.
  sequelace   Connect to your ISLE database using Sequel Ace (Mac OS only)
  up          Brings up the containers

Flags:
      --context string     The ISLE context to use. See islectl config --help for more info (default "dev")
  -h, --help               help for islectl
      --log-level string   The logging level for the command (default "INFO")
  -v, --version            version for islectl
```

Each command has a `--help` flag that provide what flags can be passed to the given command.

Some of the commands are self-evident with the name of the command and the description in `--help`. For those that need some more information, you can find that below:

### sequelace

Open Sequel Ace and connect to your ISLE database (Mac OS only)

![sequelace command screencast](./assets/img/sequelace.gif)


### make

Until all the isle-dc command have been migrated into this CLI, the current isle-dc make commands can be ran like so*

```
islectl make set_admin_password
```

This `islectl make` command could also support any custom make commands that are not able to be implemented in this CLI.*

\* Would require the given make command to be compatible with ISLE site template assumptions
