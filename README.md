# sitectl CLI

Command line utility to interact with your local and remote ISLE installs.

## Install

### Homebrew

You can install islectl using homebrew

```
brew tap islandora-devops/isle
brew install islandora-devops/isle/islectl
```

### Download Binary

Instead of homebrew, you can download a binary for your system from [the latest release](https://github.com/islandora-devops/islectl/releases/latest)

Then put the binary in a directory that is in your `$PATH`

## Usage

```
$ islectl --help
Interact with your ISLE site

Usage:
  sitectl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  make        Run custom make commands
  up          Brings up the containers or builds starter if no containers were found.

Flags:
  -d, --dir string       path to isle-site-template for your site. Defaults to current directory. (default "islectl")
  -h, --help             help for sitectl
  -p, --profile string   isle-site-template profile (default "dev")
  -v, --version          version for sitectl

Use "sitectl [command] --help" for more information about a command.
```
### up

Install or bring online an isle-site-template project

```
cd /path/to/islandora-devops/isle-site-template
islectl up
```

or

```
islectl up --dir /path/to/islandora-devops/isle-site-template
```

### make

Until all the isle-dc command have been migrated into this CLI, the current isle-dc make commands can be ran like so

```
islectl make up --dir /path/to/islandora-devops/isle-dc 
```

This `islectl make` command could also support any custom make commands that are not able to be implemented in this CLI. Would require the given make command to be compatible with the given docker compose project.

## Updating

### Homebrew

If homebrew was used, you can simply upgrade the homebrew formulae for islectl

```
brew update && brew upgrade islectl
```

### Download Binary

If the binary was downloaded and added to the `$PATH` updating islectl could look as follows. Requires [gh](https://cli.github.com/manual/installation) and `tar`

```
# update for your architecture
ARCH="islectl_Linux_x86_64.tar.gz"
TAG=$(gh release list --exclude-pre-releases --exclude-drafts --limit 1 --repo islandora-devops/islectl | awk '{print $3}')
gh release download $TAG --repo islandora-devops/islectl --pattern $ARCH
tar -zxvf $ARCH
mv islectl /directory/in/path/binary/was/placed
rm $ARCH
```

## TODO

- [x] Implement `make up` Brings up the containers or builds starter if no containers were found.
- [ ] Implement `starter Make a local site with codebase directory bind mounted, using starter site unless other package specified in .env or present already.
- [ ] Implement `starter_dev Make a local site with codebase directory bind mounted, using cloned starter site.
- [ ] Implement `make pull Fetches the latest images from the registry.
- [ ] Implement `make build Create Dockerfile from example if it does not exist.
- [ ] Implement `make push-image Push your custom drupal image to dockerhub or a container registry
- [ ] Implement `make down Brings down the containers. Same as docker compose down --remove-orphans
- [ ] Implement `make env Pull in changes to the .env file.
- [ ] Implement `make set_admin_password Sets the Drupal admin password and accomodates for permissions restrictions to the secrets directory. Only runs sudo when needed.
- [ ] Implement `make clean Destroys everything beware!
- [ ] Implement `make config-export Exports the sites configuration.
- [ ] Implement `make config-import Import the sites configuration. N.B You may need to run this multiple times in succession due to errors in the configurations dependencies.
- [ ] Implement `make demo_content Helper function for demo sites: do a workbench import of sample objects
- [ ] Implement `make hydrate Reconstitute the site from environment variables.
- [ ] Implement `make login Runs "drush uli" to provide a direct login link for user 1
- [ ] Implement `make secrets_warning Check to see if the secrets directory contains default secrets.
- [ ] Implement `make fix_masonry Fix missing masonry library.
- [ ] Implement `make fix_views This fixes a know issues with views when using the make local build. The error must be triggered before this will work.
- [ ] Implement `make xdebug Turn on xdebug.
- [ ] Implement `make set-timeout Update all PHP and NGinx timeouts to TIMEOUT_VALUE
