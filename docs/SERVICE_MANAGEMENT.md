# Service Management Architecture

This document describes the service management system in `islectl`, which allows users to customize their ISLE installations by enabling or disabling docker-compose services.

## Overview

The service management system provides a flexible way to:

1. **Disable services during initial installation** via `create context` command
2. **Manage services on existing installations** via standalone `service` commands
3. **Work seamlessly with both local and remote contexts**

This is particularly useful for:
- Reducing resource usage by disabling optional services (e.g., blazegraph, solr)
- Customizing ISLE installations for specific use cases
- Testing different configurations

## Architecture Components

### 1. Docker Compose Manipulation Package (`pkg/compose`)

**Location**: `pkg/compose/compose.go`

The core of the service management system is a package that handles reading, parsing, and modifying `docker-compose.yml` files. It provides context-aware operations that work for both local and remote installations.

**Key Components**:

- **`ServiceManager`**: Main struct that wraps a `config.Context` and provides service management operations
- **`ComposeFile`**: Struct representing the docker-compose.yaml structure (services, networks, volumes, secrets)

**Key Methods**:

```go
// Create a new service manager for a context
func NewServiceManager(ctx *config.Context) *ServiceManager

// Read and parse docker-compose.yml (local or remote)
func (sm *ServiceManager) ReadComposeFile() (*ComposeFile, error)

// Write modified compose file back (local or remote via SFTP)
func (sm *ServiceManager) WriteComposeFile(compose *ComposeFile) error

// List all services in the compose file
func (sm *ServiceManager) ListServices() ([]string, error)

// Check if a service exists
func (sm *ServiceManager) ServiceExists(serviceName string) (bool, error)

// Remove a service from docker-compose.yml
func (sm *ServiceManager) DisableService(serviceName string) error

// Get detailed information about a service
func (sm *ServiceManager) GetServiceInfo(serviceName string) (map[string]interface{}, error)
```

**Design Principles**:

1. **Context Abstraction**: All file operations use the existing `config.Context` abstraction:
   - For local contexts: Direct filesystem operations
   - For remote contexts: SSH/SFTP operations via context methods

2. **Safe YAML Manipulation**: Uses `gopkg.in/yaml.v3` for proper YAML parsing and serialization

3. **Orphaned Resource Cleanup**: When disabling a service, the system automatically removes:
   - Volumes that were only used by that service
   - (Future: networks, secrets that are no longer referenced)

4. **Drupal Configuration Hooks**: Services can define hooks to clean up Drupal configuration when disabled:
   - Removes config files from `config/sync` directory
   - Runs `drush config:import` to apply changes
   - Works for both local and remote contexts

### 2. Standalone Service Command (`cmd/service.go`)

**Location**: `cmd/service.go`

Provides a CLI interface for managing services on existing installations.

**Subcommands**:

```bash
# List all services
islectl service list

# Disable one or more services
islectl service disable blazegraph
islectl service disable blazegraph solr --yes

# Get information about a service
islectl service info blazegraph

# Enable a service (not yet implemented)
islectl service enable blazegraph
```

**Command Structure**:

- `service` - Root command for service management
  - `list` - List all services in docker-compose.yml
  - `disable [service-name...]` - Disable services (with confirmation)
  - `enable [service-name]` - Re-enable services (planned)
  - `info [service-name]` - Show service details

**Flags**:

- `--context, -c`: Specify which context to operate on
- `--yes`: Skip confirmation prompts (useful for automation/scripts)

**Safety Features**:

1. **Confirmation Prompts**: By default, shows what will be changed and asks for confirmation
2. **Validation**: Checks that services exist before attempting to disable them
3. **Clear Messaging**: Shows the path being modified and services being affected

### 3. Integration with Create Context Command

**Location**: `cmd/create.go` and `pkg/isle/buildkit.go`

Service management is integrated into the `create context` workflow, allowing users to customize installations during initial setup.

**New Flag**:

```bash
--disable-service=SERVICE_NAME
```

This flag can be repeated to disable multiple services:

```bash
islectl create context mysite \
  --disable-service=blazegraph \
  --disable-service=solr \
  --yes
```

**Workflow**:

1. User runs `create context` with optional `--disable-service` flags
2. The setup script downloads and installs ISLE site template as normal
3. **After installation completes**, the system:
   - Reads the installed docker-compose.yml
   - Removes specified services
   - Cleans up orphaned volumes
   - Writes the modified compose file back

4. Context is saved to config

**User Experience**:

- **With `--yes` flag**: Services are automatically disabled, no prompts
- **Without `--yes` flag**: Shows all settings including services to be disabled, asks for confirmation
- **Interactive mode**: User is prompted for all settings, sees services that will be disabled before confirming

## Usage Examples

### Example 1: Create New Site Without Blazegraph

```bash
# Using flags (no prompts)
islectl create context mysite \
  --type local \
  --profile dev \
  --project-dir /home/user/mysite \
  --disable-service=blazegraph \
  --yes

# Interactive mode (prompted for each setting)
islectl create context mysite
# Is the context local or remote? [local]: local
# Where would you like to install the project? [/home/user/mysite]:
#
# Optional services:
#   Include blazegraph? (RDF triplestore for Linked Data and SPARQL queries) [Y/n]: n
#
# Here is the context that will be created
# ... (shows context details) ...
#
# Services to be disabled after installation:
#   - blazegraph
#
# Are you sure you want to proceed creating the site? y/N: y
```

**Behavior**:
- If no `--disable-service` flags are provided and `--yes` is not used, the tool asks about each optional service
- Each service shows its description to help users decide
- Default responses are indicated (Y/n means Yes is default, y/N means No is default)
- Pressing Enter accepts the default
- If `--disable-service` flags ARE provided, no prompts are shown (explicit flags take precedence)

### Example 2: Disable Service on Existing Installation

```bash
# On current context
islectl service disable blazegraph

# Output:
# You are about to disable the following service(s):
#   - blazegraph
#     → Will remove 8 Drupal config file(s)
#
# This will modify:
#   - /path/to/site/docker-compose.yml
#   - config/sync/ (Drupal configuration)
#
# Are you sure you want to continue? [y/N]: y
#
# Cleaning up Drupal configuration for 'blazegraph'...
#   ✓ Removed search_api.server.blazegraph.yml
#   ✓ Removed context.context.blazegraph_index.yml
#   ✓ Removed system.action.index_node_in_blazegraph.yml
#   ... (more config files)
#
# Importing configuration changes...
#   ✓ Configuration imported successfully
# ✓ Service 'blazegraph' disabled
#
# Services successfully disabled!
# Run 'islectl compose down && islectl compose up' to apply changes.

# On specific context
islectl service disable blazegraph --context prod

# Multiple services, skip confirmation
islectl service disable blazegraph solr --yes

# Then apply the changes
islectl compose down
islectl compose up
```

### Example 3: List and Inspect Services

```bash
# List all services
islectl service list

# Get details about a specific service
islectl service info blazegraph
```

### Example 4: Remote Context

All commands work identically for remote contexts:

```bash
# Create remote site without optional services
islectl create context prod \
  --type remote \
  --ssh-hostname isle.example.com \
  --ssh-user deploy \
  --project-dir /opt/isle \
  --disable-service=blazegraph \
  --disable-service=solr \
  --yes

# Manage services on remote installation
islectl service list --context prod
islectl service disable fedora --context prod --yes
```

## Implementation Details

### File Operations Across Local and Remote Contexts

The `ServiceManager` abstracts file operations to work with both local and remote contexts:

**Local Context**:
```go
// Read
data, err := os.ReadFile(composePath)

// Write
err := os.WriteFile(composePath, data, 0644)
```

**Remote Context**:
```go
// Read
fileContent := sm.context.ReadSmallFile(composePath)

// Write (via temp file + SFTP)
tmpFile, _ := os.CreateTemp("", "docker-compose-*.yml")
tmpFile.Write(data)
sm.context.UploadFile(tmpFile.Name(), composePath)
```

### YAML Parsing and Manipulation

The system uses `gopkg.in/yaml.v3` for YAML operations:

```go
type ComposeFile struct {
    Version  string                 `yaml:"version,omitempty"`
    Services map[string]interface{} `yaml:"services,omitempty"`
    Networks map[string]interface{} `yaml:"networks,omitempty"`
    Volumes  map[string]interface{} `yaml:"volumes,omitempty"`
    Secrets  map[string]interface{} `yaml:"secrets,omitempty"`
}

// Parse
yaml.Unmarshal([]byte(fileContent), &compose)

// Modify
delete(compose.Services, serviceName)

// Serialize
yaml.Marshal(compose)
```

### Orphaned Volume Cleanup

When a service is disabled, volumes that are no longer referenced by any service are automatically removed:

```go
func cleanupOrphanedVolumes(compose *ComposeFile, removedService string) {
    // Build set of volumes still in use by remaining services
    volumesInUse := make(map[string]bool)
    for _, svc := range compose.Services {
        // Parse volume definitions: "volume-name:/path"
        // Add to volumesInUse set
    }

    // Remove unused volumes
    for volumeName := range compose.Volumes {
        if !volumesInUse[volumeName] {
            delete(compose.Volumes, volumeName)
        }
    }
}
```

### Drupal Configuration Hooks

**Location**: `pkg/compose/hooks.go`

Services can define hooks that specify Drupal configuration files to remove when the service is disabled. This ensures that Drupal configuration stays in sync with the available services.

**Hook Registry**:

```go
var ServiceHookRegistry = map[string]*ServiceHook{
    "blazegraph": {
        ServiceName: "blazegraph",
        Description: "RDF triplestore for Linked Data and SPARQL queries",
        ConfigFilesToRemove: []string{
            "search_api.server.blazegraph.yml",
            "context.context.blazegraph_index.yml",
            "system.action.index_node_in_blazegraph.yml",
            "system.action.index_media_in_blazegraph.yml",
            "system.action.index_file_in_blazegraph.yml",
            "system.action.delete_node_from_blazegraph.yml",
            "system.action.delete_media_from_blazegraph.yml",
            "system.action.delete_file_from_blazegraph.yml",
        },
    },
}
```

**Hook Execution Flow**:

1. **Check for hook**: When disabling a service, check if a hook is defined
2. **Remove config files**: Delete specified files from `config/sync/` directory
3. **Import configuration**: Run `drush config:import -y` via docker compose exec
4. **Remove service**: Continue with docker-compose.yml modification

This happens **before** the service is removed from docker-compose.yml, ensuring the drupal container is still running and can execute drush commands.

**Adding New Service Hooks**:

To add a hook for a new service, add an entry to `ServiceHookRegistry` in `pkg/compose/hooks.go`:

```go
"fedora": {
    ServiceName:     "fedora",
    Description:     "Fedora repository for digital object storage",
    IsOptional:      true,       // Will be offered during interactive install
    DefaultEnabled:  true,        // Default answer when prompted
    ConfigFilesToRemove: []string{
        "islandora.settings.yml",  // May contain Fedora URL
        "flysystem.settings.yml",  // Fedora flysystem configuration
        // Add other related config files
    },
},
```

**Service Hook Fields**:
- `ServiceName`: Docker compose service name
- `Description`: Human-readable description shown during prompts
- `IsOptional`: If true, users will be asked about this service during interactive installation
- `DefaultEnabled`: Default answer for optional services (Y/n vs y/N)
- `ConfigFilesToRemove`: Drupal config files to remove when disabling

The config file names should match the filenames in the `config/sync/` directory of islandora-starter-site.

## Future Enhancements

### 1. Service Templates for Re-enabling

Currently, `service enable` is not implemented because it requires storing the original service definitions. Potential solutions:

- **Version control**: Encourage users to commit docker-compose.yml before modifications
- **Backup files**: Automatically create `.bak` files before modifications
- **Service templates**: Maintain a library of default service definitions
- **Git integration**: Use git to restore previous versions

### 2. Service Dependencies

Add dependency checking to prevent disabling services that other services depend on:

```go
// Define service dependencies
var serviceDeps = map[string][]string{
    "drupal": {"mariadb", "solr"},
    "matomo": {"mariadb"},
}

// Check before disabling
func (sm *ServiceManager) CheckDependencies(serviceName string) error
```

### 3. Preset Configurations

Allow users to apply preset configurations:

```bash
islectl create context mysite --preset=minimal
islectl service apply-preset minimal
```

Presets could be defined as:
```yaml
presets:
  minimal:
    disable: [blazegraph, fedora, fits]
  development:
    disable: [matomo]
  production:
    disable: [ide]
```

### 4. Interactive Service Selection

During `create context`, prompt user to select which services to include:

```
Which optional services would you like to include?
[x] Blazegraph (triplestore)
[x] Fedora (repository)
[ ] FITS (file characterization)
[x] Matomo (analytics)
```

### 5. Service Status Checking

Add ability to check if disabled services have running containers:

```bash
islectl service status blazegraph
# Output: Service 'blazegraph' is disabled in docker-compose.yml but has running containers
```

## Testing

### Manual Testing

1. **Test service disable on local context**:
   ```bash
   islectl create context test --type local --profile dev --project-dir /tmp/test
   islectl service list --context test
   islectl service disable blazegraph --context test
   islectl compose down --context test
   islectl compose up --context test
   ```

2. **Test service disable during creation**:
   ```bash
   islectl create context test2 \
     --type local \
     --project-dir /tmp/test2 \
     --disable-service=blazegraph \
     --yes
   ```

3. **Test with remote context**: Same commands with `--type remote` and SSH flags

### Unit Tests

Key test cases needed:

- `TestReadComposeFile`: Test parsing docker-compose.yml
- `TestDisableService`: Test service removal
- `TestCleanupOrphanedVolumes`: Test volume cleanup logic
- `TestListServices`: Test service listing
- `TestServiceExists`: Test service existence checking

Example test structure:
```go
func TestDisableService(t *testing.T) {
    tests := []struct {
        name        string
        service     string
        wantErr     bool
        checkVolume string
        volumeGone  bool
    }{
        {"disable blazegraph", "blazegraph", false, "blazegraph-data", true},
        {"disable nonexistent", "fake", true, "", false},
    }
    // ... test implementation
}
```

## Related Documentation

- **Project Instructions**: `CLAUDE.md` - General project overview
- **Go Conventions**: `docs/GO_CONVENTIONS.md` - Coding standards
- **User Documentation**:
  - `docs/docs/commands.md` - Command reference
  - `docs/docs/install.md` - Installation guide

## Conclusion

The service management system provides a flexible, safe way to customize ISLE installations both during initial setup and for existing installations. By building on the existing context abstraction, it works seamlessly with both local and remote installations, and provides a foundation for future enhancements like service templates, presets, and dependency management.
