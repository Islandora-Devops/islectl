# Getting Started

## Install

### Homebrew

You can install `islectl` using homebrew

```bash
brew tap islandora-devops/tap
brew install islandora-devops/tap/islectl
```

### Download Binary

Instead of homebrew, you can download a binary for your system from [the latest release](https://github.com/islandora-devops/islectl/releases/latest)

Then put the binary in a directory that is in your `$PATH`

## Setup

Now that you have `islectl` installed, you'll need to configure `islectl` so it knows how to run commands against your various ISLE installations. Your `islectl` config is located at `$HOME/.islectl/config.yaml`

### On contexts

Contexts are how `islectl` can be ran from one location to manage several ISLE installations.

You can learn more about contexts with the config `--help` command

```
$ islectl config --help

An islectl config can have multiple contexts.

An islectl context is an ISLE docker compose project running somewhere. "Somewhere" meaning:

- on your laptop (--type local)
- on a remote server (--type remote).

Remote contexts require SSH access to the remote server from where islectl is being ran from.
When creating a context the remote server DNS name, SSH port, SSH username, and your SSH private key will need to be set in the context configuration.

You can have a default context which will be used when running islectl commands, unless the context is overridden with the --context flag.

Usage:
  islectl config [command]

Available Commands:
  current-context Display the current ISLE context
  delete-context  Delete an ISLE context
  get-contexts    List all ISLE contexts
  set-context     Set or update properties of a context. Creates a new context if it does not exist.
  use-context     Switch to the specified context
  view            Print your islectl config

Flags:
  -h, --help   help for config

Global Flags:
  -c, --context string   The ISLE context to use. See islectl config --help for more info (default "dev")
  -s, --site string      The name of the site. If yr not using multi-site don't worry about this. (default "default")
```

### Creating new ISLE sites

You can install an ISLE site on your local machine or a remote server with the command  `islectl create context [context-name]`. The command sets up `isle-site-template` and your `islectl` context for the install.

#### Create a local context

Below is an example command that will install an ISLE site at `/home/vivek/isle`

```
$ islectl create context dev \
  --type local \
  --profile dev \
  --project-dir /home/vivek/isle \
  --project-name isle
```

#### Create a remote context

Below is an example command that will install an ISLE on the server `islandora.YOUR-INSTITUTION.edu`. It assumes you can SSH into that server on port 22 with the SSH key at `$HOME/.ssh/id_rsa` with the remote user `vivek`. Change the flags to match your local environment and see `islectl create context --help` for more information.

```
$ islectl create context stage \
  --type remote \
  --profile dev \
  --project-dir /opt/islandora \
  --project-name YOUR-INSITUTION \
  --ssh-hostname islandora.YOUR-INSTITUTION.edu \
  --ssh-port 22 \
  --ssh-user vivek \
  --ssh-key $(pwd).ssh/id_rsa
```


### Creating context(s) for existing installations

If you already have sites based on isle-site-template running, you can configure `islectl` for them with `islectl create config [context-name]`

#### Configure a local context

For ISLE installs running on your local machine, you can create a local context. An example command for that would be:

```bash
islectl create config dev \
  --type local \
  --default \
  --profile dev \
  --project-dir /Users/vivek/isle-site-template
```

#### Create a remote context

For ISLE installs on remote servers, you can setup a remote context. An example command for that would be:

```bash
islectl create config stage \
  --type remote \
  --ssh-hostname isle.myinstitution.edu \
  --profile prod \
  --project-dir /path/to/your/isle/site/template/directory \
  --project-name custom-project-name \
  --ssh-port 22 \
  --env-file .env \
  --env-file /path/to/another/.env \
  --sudo=true
```


### Using different contexts

In the two examples above, a local context and remote context where created.

The local context was named `dev`, and we also made it our default context with the `--default` flag. The remote context was named `stage`

I can see all the contexts with
```
$ islectl config get-contexts
* dev (type: local)
  stage (type: remote)
```

The asterisk indicates `dev` will be used to run commands . So when running the `login` command, it will be ran against the `dev` context.

```bash
$ islectl login
https://islandora.dev/user/reset/1/1741453534/JuSMZIM_aCvsJR7gMgOcUxHkEL-YDMVL1_klQoYxhkQ/login
```

Though if the `--context` flag is passed the default contexet can be overriden.
```bash
$ islectl login --context stage
https://isle.myinstitution.edu/user/reset/1/1741453647/cdscdsc-YDMVL1_mdwkpamc2/login
```

The default context can also be switched permananetly with 

```
islectl set-context stage
```

## Updating islectl

### Homebrew

If homebrew was used, you can simply upgrade the homebrew formulae for islectl

```
brew update && brew upgrade islectl
```

### Download Binary

If the binary was downloaded and added to the `$PATH` updating islectl could look as follows. Requires [gh](https://cli.github.com/manual/installation) and `tar`

```bash
# update for your architecture
ARCH="islectl_Linux_x86_64.tar.gz"
TAG=$(gh release list --exclude-pre-releases --exclude-drafts --limit 1 --repo islandora-devops/islectl | awk '{print $3}')
gh release download $TAG --repo islandora-devops/islectl --pattern $ARCH
tar -zxvf $ARCH
mv islectl /directory/in/path/binary/was/placed
rm $ARCH
```
