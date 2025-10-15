# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 📚 Critical Documentation References
- **Go Conventions**: `./docs/GO_CONVENTIONS.md`
- **Service Management Architecture**: `./docs/SERVICE_MANAGEMENT.md` - Service customization system
- **Project Documentation**: https://islandora-devops.github.io/islectl/
- **User Documentation** (mkdocs):
  - `./docs/docs/index.md` - Overview and features
  - `./docs/docs/install.md` - Installation and setup guide (contexts, creating sites)
  - `./docs/docs/commands.md` - Command reference and examples

---

## Project Overview

`islectl` is a CLI tool for managing ISLE (Islandora) installations. It's inspired by:
- `kubectl` for context management (switching between different environments)
- `ddev` for development workflow
- `make` commands from isle-dc, but designed for isle-site-template

The tool provides a unified interface to manage both **local** and **remote** ISLE docker-compose projects.

---

## Build & Test Commands

```bash
# Build the binary
make build

# Run all tests
make test

# Run linter (required before committing)
make lint
```

---

## Architecture Overview

### Context System (kubectl-inspired)

The core concept is **contexts** - configurations for ISLE installations that can be local or remote.

**Config storage**: `~/.islectl/config.yaml` stores all contexts and the current active context.

**Context types**:
- `local`: Docker socket on the same machine (typically `/var/run/docker.sock`)
- `remote`: Docker socket accessed over SSH tunnel to a remote server

**Key files**:
- `pkg/config/config.go` - Config file I/O, loading, and saving
- `pkg/config/context.go` - Context struct and operations (SSH dialing, file reading, remote verification)
- `pkg/config/cmd.go` - Flag parsing and context loading from command flags
- `pkg/compose/compose.go` - Docker compose YAML manipulation for service management

**Critical pattern**: Almost every command needs to get the current context:
```go
context, err := config.CurrentContext(cmd.Flags())
```

### Command Structure (Cobra-based)

All commands are in `cmd/`:
- `cmd/root.go` - Root command with global flags (`--context`, `--log-level`)
- `cmd/config.go` - Context management (set-context, use-context, get-contexts, delete-context, etc.)
- `cmd/create.go` - Create new ISLE sites or configs for existing sites
- `cmd/service.go` - Service management (list, disable, enable services in docker-compose.yml)
- `cmd/compose.go` - Wrapper for docker-compose commands with context awareness
- `cmd/drush.go` - Drush command wrapper with automatic URI handling
- `cmd/drupal/` - Drupal-specific subcommands:
  - `cmd/drupal/root.go` - Drupal command group root
  - `cmd/drupal/exec.go` - Execute commands in drupal container
  - `cmd/drupal/backup.go` - Backup drupal database and files
- `cmd/port-forward.go` - Port forwarding for remote contexts
- `cmd/make.go` - Wrapper for custom make commands
- `cmd/sequelace.go` - Open Sequel Ace (Mac only) connected to ISLE database

### Docker Abstraction Layer

The `pkg/isle/docker.go` file provides abstraction over Docker API to work with both local and remote contexts:

**Key function**: `GetDockerCli(context)` - Returns a DockerClient that works locally or over SSH tunnel.

**How it works**:
- For local contexts: Direct connection to local Docker socket
- For remote contexts: HTTP client that dials `unix://docker.sock` over SSH tunnel

**Important utilities**:
- `GetContainerName()` - Find container by docker-compose project and service labels
- `GetSecret()` - Read secrets from running containers
- `GetConfigEnv()` - Read environment variables from containers
- `GetServiceIp()` - Get IP address of a service in the docker network

**Profile handling**: Service names are suffixed with profile (e.g., `drupal-dev` vs `drupal-prod`). The `GetContainerName()` function handles this automatically.

### Remote Context Execution

For remote contexts, commands execute docker/docker-compose on the remote host over SSH.

**Key Context methods** (in `pkg/config/context.go`):
- `context.RunCommand(cmd)` - Executes commands locally or via SSH based on context type
- `context.ReadSmallFile(path)` - Reads files locally or via SSH `cat`
- `context.UploadFile(src, dest)` - SFTP file upload for remote contexts
- `context.DialSSH()` - Establishes SSH connection with known_hosts verification
- `context.ProjectDirExists()` - Checks if project directory exists (locally or remotely)
- `context.VerifyRemoteInput(existingSite)` - Interactive verification/correction of remote SSH settings

**SSH Security**: The code enforces known_hosts verification and provides helpful error messages to guide users to manually SSH first if keys aren't known.

### Command Patterns

**Compose command pattern** (`cmd/compose.go`):
```go
DisableFlagParsing: true,  // Pass flags through to docker-compose
Args: cobra.ArbitraryArgs,
```
- Manually extracts `--context` flag via `utils.GetContextFromArgs()`
- Automatically adds `-d --remove-orphans` to `up` commands
- Automatically adds `--pull` to `build` commands
- Sets working directory to `context.ProjectDir`

**Drush command pattern** (`cmd/drush.go`):
- Wraps `docker compose exec drupal-{profile} bash -c "drush ..."`
- Automatically adds `--uri $DRUPAL_DRUSH_URI` unless user provides `--uri` or `-l`
- Special subcommand `uli` auto-opens login link in browser via `utils.OpenURL()`

---

## Key Design Decisions

1. **Profile awareness**: Docker compose profiles (dev/prod) are part of service names throughout the codebase. Services are named like `drupal-dev` or `drupal-prod`.

2. **SSH security**: Known_hosts verification is enforced. Error messages guide users to run `ssh` manually first to accept host keys.

3. **Flag pass-through**: Commands like `compose` and `drush` disable Cobra flag parsing to allow arbitrary flags to be passed to underlying tools.

4. **Context abstraction**: Operations should work identically for local and remote contexts by using `Context` methods rather than direct filesystem or command execution.

5. **Version info injection**: Build-time variables (`version`, `commit`, `date`) are injected via goreleaser's `-ldflags`.

6. **No sudo by default**: Commands run as the user unless `RunSudo: true` is set in the context.

---

## Common Development Patterns

### Adding a new command that needs context

1. Get the context from flags:
   ```go
   context, err := config.CurrentContext(cmd.Flags())
   if err != nil {
       return err
   }
   ```

2. For local/remote command execution:
   ```go
   c := exec.Command("some-command", args...)
   c.Dir = context.ProjectDir
   output, err := context.RunCommand(c)
   if err != nil {
       return err
   }
   ```

3. For Docker API access:
   ```go
   dockerCli, err := isle.GetDockerCli(context)
   if err != nil {
       return err
   }
   defer dockerCli.Close()

   // Use dockerCli.CLI for Docker API calls
   containerName, err := dockerCli.GetContainerName(context, "drupal", false)
   ```

### Working with docker-compose services

When you need to find a container by service name:
```go
containerName, err := dockerCli.GetContainerName(context, "drupal", false)
// The false parameter means "do prefix with profile"
// For services that never use profiles (like traefik), pass true
```

### Reading files on local or remote hosts

Always use the Context method to abstract local vs remote:
```go
content := context.ReadSmallFile("/path/to/file")
```

### Working with docker-compose.yml

Use the `ServiceManager` from `pkg/compose` to manipulate docker-compose files:
```go
sm := compose.NewServiceManager(context)

// List all services
services, err := sm.ListServices()

// Check if service exists
exists, err := sm.ServiceExists("blazegraph")

// Disable a service
err := sm.DisableService("blazegraph")
```

See `docs/SERVICE_MANAGEMENT.md` for detailed architecture and usage patterns.

### Testing

- Use table-driven tests (see `GO_CONVENTIONS.md`)
- Mock the Docker API using the `DockerAPI` interface in tests
- Example: `pkg/isle/docker_test.go` and `pkg/config/context_test.go`

---

## Project Goals (from README)

This tool aims to replace `make` commands from isle-dc for isle-site-template:
- Manage both local development and production installations
- Support multiple local contexts running simultaneously
- Provide shortcuts for common Drupal/Drush operations
- Maintain consistency across local and remote environments

See README.md TODO section for feature implementation status.