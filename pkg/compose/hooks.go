package compose

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/islandora-devops/islectl/pkg/config"
)

// ServiceHook defines Drupal configuration changes needed when disabling/enabling a service
type ServiceHook struct {
	// ServiceName is the docker-compose service name
	ServiceName string

	// ConfigFilesToRemove are config YAML files to delete from config/sync when disabling
	// These should be relative to the config/sync directory (e.g., "search_api.server.blazegraph.yml")
	ConfigFilesToRemove []string

	// ConfigFilesToAdd are config YAML files to restore when enabling (future feature)
	ConfigFilesToAdd []string

	// Description explains what this service does
	Description string

	// IsOptional indicates if this service should be offered as optional during installation
	IsOptional bool

	// DefaultEnabled is the default state when prompting (if IsOptional is true)
	DefaultEnabled bool
}

// ServiceHookRegistry maps service names to their hooks
var ServiceHookRegistry = map[string]*ServiceHook{
	"blazegraph": {
		ServiceName:    "blazegraph",
		Description:    "RDF triplestore for Linked Data and SPARQL queries",
		IsOptional:     true,
		DefaultEnabled: true,
		ConfigFilesToRemove: []string{
			// Search API server configuration
			"search_api.server.blazegraph.yml",
			// Context configurations that trigger blazegraph indexing
			"context.context.blazegraph_index.yml",
			// Actions that index to blazegraph
			"system.action.index_node_in_blazegraph.yml",
			"system.action.index_media_in_blazegraph.yml",
			"system.action.index_file_in_blazegraph.yml",
			"system.action.delete_node_from_blazegraph.yml",
			"system.action.delete_media_from_blazegraph.yml",
			"system.action.delete_file_from_blazegraph.yml",
		},
	},
	// Future: add hooks for other services
	// "fedora": {...},
	// "fits": {...},
	// "solr": {...},
}

// GetServiceHook returns the hook for a service, or nil if no hook exists
func GetServiceHook(serviceName string) *ServiceHook {
	return ServiceHookRegistry[serviceName]
}

// GetOptionalServices returns a list of all optional services
func GetOptionalServices() []*ServiceHook {
	optional := make([]*ServiceHook, 0)
	for _, hook := range ServiceHookRegistry {
		if hook.IsOptional {
			optional = append(optional, hook)
		}
	}
	return optional
}

// ExecuteDisableHook runs the Drupal configuration cleanup when disabling a service
// This should be called BEFORE removing the service from docker-compose.yml
// so the drupal container is still running and we can execute drush commands
func (sm *ServiceManager) ExecuteDisableHook(serviceName string) error {
	hook := GetServiceHook(serviceName)
	if hook == nil {
		slog.Debug("No configuration hook defined for service", "service", serviceName)
		return nil
	}

	if len(hook.ConfigFilesToRemove) == 0 {
		slog.Debug("No config files to remove for service", "service", serviceName)
		return nil
	}

	slog.Info("Running Drupal configuration cleanup", "service", serviceName)
	fmt.Printf("\nCleaning up Drupal configuration for '%s'...\n", serviceName)

	// Path to config/sync directory relative to project root
	configSyncPath := filepath.Join(sm.context.ProjectDir, "config", "sync")

	// Remove config files
	for _, configFile := range hook.ConfigFilesToRemove {
		configPath := filepath.Join(configSyncPath, configFile)

		// Check if file exists (works for local and remote)
		exists, err := sm.fileExists(configPath)
		if err != nil {
			slog.Warn("Error checking config file", "file", configFile, "err", err)
			continue
		}

		if !exists {
			slog.Debug("Config file does not exist, skipping", "file", configFile)
			continue
		}

		// Remove the file
		if err := sm.removeFile(configPath); err != nil {
			slog.Warn("Failed to remove config file", "file", configFile, "err", err)
			fmt.Printf("  ⚠ Warning: Could not remove %s: %v\n", configFile, err)
			continue
		}

		fmt.Printf("  ✓ Removed %s\n", configFile)
		slog.Info("Removed config file", "file", configFile)
	}

	// Import configuration changes with drush
	fmt.Println("\nImporting configuration changes...")
	if err := sm.runDrushConfigImport(); err != nil {
		return fmt.Errorf("failed to import configuration: %w", err)
	}

	fmt.Println("  ✓ Configuration imported successfully")
	return nil
}

// ExecuteEnableHook runs the Drupal configuration restoration when enabling a service
// This is a placeholder for future implementation
func (sm *ServiceManager) ExecuteEnableHook(serviceName string) error {
	hook := GetServiceHook(serviceName)
	if hook == nil {
		return nil
	}

	// TODO: Implement config file restoration
	// This would require storing original config files or pulling them from
	// islandora-starter-site repository
	return fmt.Errorf("enable hook not yet implemented for service: %s", serviceName)
}

// fileExists checks if a file exists (works for local and remote contexts)
func (sm *ServiceManager) fileExists(path string) (bool, error) {
	if sm.context.DockerHostType == config.ContextLocal {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// For remote contexts, use test -f
	cmd := exec.Command("test", "-f", path)
	cmd.Dir = sm.context.ProjectDir
	_, err := sm.context.RunCommand(cmd)
	return err == nil, nil
}

// removeFile removes a file (works for local and remote contexts)
func (sm *ServiceManager) removeFile(path string) error {
	if sm.context.DockerHostType == config.ContextLocal {
		return os.Remove(path)
	}

	// For remote contexts, use rm
	cmd := exec.Command("rm", path)
	cmd.Dir = sm.context.ProjectDir
	_, err := sm.context.RunCommand(cmd)
	return err
}

// runDrushConfigImport runs `drush config:import -y` via docker compose exec
func (sm *ServiceManager) runDrushConfigImport() error {
	// Determine the drupal service name (includes profile suffix)
	drupalService := fmt.Sprintf("drupal-%s", sm.context.Profile)

	// Build the drush command
	drushCmd := "drush config:import -y"

	// Execute via docker compose exec
	args := []string{
		"compose",
		"exec",
		"-T", // Disable pseudo-TTY allocation
		drupalService,
		"bash",
		"-c",
		drushCmd,
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = sm.context.ProjectDir

	output, err := sm.context.RunCommand(cmd)
	if err != nil {
		slog.Error("Failed to run drush config:import", "output", output, "err", err)
		return fmt.Errorf("drush config:import failed: %w\nOutput: %s", err, output)
	}

	slog.Debug("Drush config:import output", "output", output)
	return nil
}
