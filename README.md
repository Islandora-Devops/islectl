# islectl CLI

🚧 PoC currently under construction/consideration. This repo may be deleted at any time.

Command line utility to interact with your local and remote ISLE installs.

Documentation is available at https://islandora-devops.github.io/islectl/

## Attribution

- The `config` commands for setting contexts were heavily inspired by `kubectl`
- `ddev` was also an inspiration for this tool, with the added goal/feature of also managing production installations as well as local, development environments for ISLE repositories.
- This tool is meant to be a CLI equivalent of the `make` commands provided by [isle-dc](https://github.com/islandora-devops/isle-dc), but instead to be ran against [isle-site-template](https://github.com/islandora-devops/isle-site-template)

## TODO

- [x] Implement `make up` Brings up the containers or builds starter if no containers were found.
  - Now `islectl up` for existing sites. Use `islectl create-site` to create a new site
- [ ] Implement `make starter` Make a local site with codebase directory bind mounted, using starter site unless other package specified in .env or present already.
- [ ] Implement `starter_dev` Make a local site with codebase directory bind mounted, using cloned starter site.
- [x] Implement `make pull` Fetches the latest images from the registry.
- [x] Implement `make build` Create Dockerfile from example if it does not exist.
- [ ] Implement `make push-image` Push your custom drupal image to dockerhub or a container registry
- [x] Implement `make down` Brings down the containers. Same as docker compose down --remove-orphans
- [ ] Implement `make env` Pull in changes to the .env file.
- [ ] Implement `make set_admin_password` Sets the Drupal admin password and accomodates for permissions restrictions to the secrets directory. Only runs sudo when needed.
- [ ] Implement `make clean` Destroys everything beware!
- [x] Implement `make config-export` Exports the sites configuration.
  - Now `islectl drush cex`
- [x] Implement `make config-import` Import the sites configuration. N.B You may need to run this multiple times in succession due to errors in the configurations dependencies.
  - Now `islectl drush cim`
- [ ] Implement `make demo_content` Helper function for demo sites: do a workbench import of sample objects
- [ ] Implement `make hydrate` Reconstitute the site from environment variables.
- [x] Implement `make login` Runs "drush uli" to provide a direct login link for user 1
  - Now `islectl drush uli` with an optional `--uid` flag and automatically opens link from terminal
- [ ] Implement `make secrets_warning` Check to see if the secrets directory contains default secrets.
- [ ] Implement `make fix_masonry` Fix missing masonry library.
- [ ] Implement `make fix_views` This fixes a know issues with views when using the make local build. The error must be triggered before this will work.
- [ ] Implement `make xdebug` Turn on xdebug.
- [ ] Implement `make set-timeout` Update all PHP and NGinx timeouts to TIMEOUT_VALUE
- [ ] Support changing the docker compose project name
- [ ] Support configuring multiple isle-site-template contexts running locally
- [ ] Support moving from a docker volume to bind mount
- [ ] Support overridding conf
