package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/islandora-devops/islectl/pkg/config"
	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yaml structure
type ComposeFile struct {
	Version  string                 `yaml:"version,omitempty"`
	Services map[string]interface{} `yaml:"services,omitempty"`
	Networks map[string]interface{} `yaml:"networks,omitempty"`
	Volumes  map[string]interface{} `yaml:"volumes,omitempty"`
	Secrets  map[string]interface{} `yaml:"secrets,omitempty"`
}

// ServiceManager handles docker-compose service management
type ServiceManager struct {
	context *config.Context
}

// NewServiceManager creates a new ServiceManager for the given context
func NewServiceManager(ctx *config.Context) *ServiceManager {
	return &ServiceManager{context: ctx}
}

// GetComposePath returns the path to docker-compose.yml
func (sm *ServiceManager) GetComposePath() string {
	return filepath.Join(sm.context.ProjectDir, "docker-compose.yml")
}

// ReadComposeFile reads and parses the docker-compose.yml file
func (sm *ServiceManager) ReadComposeFile() (*ComposeFile, error) {
	composePath := sm.GetComposePath()

	// Read file content (works for both local and remote via context abstraction)
	var fileContent string
	if sm.context.DockerHostType == config.ContextLocal {
		data, err := os.ReadFile(composePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read compose file: %w", err)
		}
		fileContent = string(data)
	} else {
		// For remote contexts, read via SSH
		fileContent = sm.context.ReadSmallFile(composePath)
		if fileContent == "" {
			return nil, fmt.Errorf("failed to read compose file from remote host")
		}
	}

	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(fileContent), &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &compose, nil
}

// WriteComposeFile writes the compose file back to disk
func (sm *ServiceManager) WriteComposeFile(compose *ComposeFile) error {
	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}

	composePath := sm.GetComposePath()

	if sm.context.DockerHostType == config.ContextLocal {
		// Write locally
		if err := os.WriteFile(composePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write compose file: %w", err)
		}
	} else {
		// For remote contexts, create temp file and upload via SFTP
		tmpFile, err := os.CreateTemp("", "docker-compose-*.yml")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()

		if err := sm.context.UploadFile(tmpFile.Name(), composePath); err != nil {
			return fmt.Errorf("failed to upload compose file: %w", err)
		}
	}

	return nil
}

// ListServices returns a list of all service names in the compose file
func (sm *ServiceManager) ListServices() ([]string, error) {
	compose, err := sm.ReadComposeFile()
	if err != nil {
		return nil, err
	}

	services := make([]string, 0, len(compose.Services))
	for name := range compose.Services {
		services = append(services, name)
	}

	return services, nil
}

// ServiceExists checks if a service exists in the compose file
func (sm *ServiceManager) ServiceExists(serviceName string) (bool, error) {
	compose, err := sm.ReadComposeFile()
	if err != nil {
		return false, err
	}

	_, exists := compose.Services[serviceName]
	return exists, nil
}

// DisableService removes a service from the docker-compose.yml file
// It also removes any dependent volumes that are only used by this service
// and executes Drupal configuration cleanup if a hook is defined
func (sm *ServiceManager) DisableService(serviceName string) error {
	compose, err := sm.ReadComposeFile()
	if err != nil {
		return err
	}

	// Check if service exists
	if _, exists := compose.Services[serviceName]; !exists {
		return fmt.Errorf("service '%s' not found in compose file", serviceName)
	}

	// Execute pre-disable hook (Drupal config cleanup)
	// This MUST run before removing the service from docker-compose
	// so the drupal container is still running
	if err := sm.ExecuteDisableHook(serviceName); err != nil {
		return fmt.Errorf("failed to execute disable hook: %w", err)
	}

	// Remove the service
	delete(compose.Services, serviceName)

	// Clean up orphaned volumes (volumes only used by this service)
	// This is optional and could be made configurable
	sm.cleanupOrphanedVolumes(compose, serviceName)

	// Write the modified compose file
	if err := sm.WriteComposeFile(compose); err != nil {
		return err
	}

	return nil
}

// EnableService adds a service back to the docker-compose.yml file
// This requires a backup or template of the original service definition
// For now, this returns an error - it would need implementation with service templates
func (sm *ServiceManager) EnableService(serviceName string) error {
	return fmt.Errorf("enable service not yet implemented - service definitions need to be stored/templated")
}

// cleanupOrphanedVolumes removes volumes that are no longer referenced by any service
func (sm *ServiceManager) cleanupOrphanedVolumes(compose *ComposeFile, removedService string) {
	if compose.Volumes == nil {
		return
	}

	// Build a set of volumes still in use
	volumesInUse := make(map[string]bool)
	for _, svc := range compose.Services {
		svcMap, ok := svc.(map[string]interface{})
		if !ok {
			continue
		}

		volumes, ok := svcMap["volumes"].([]interface{})
		if !ok {
			continue
		}

		for _, vol := range volumes {
			volStr, ok := vol.(string)
			if !ok {
				continue
			}

			// Parse volume string (format: "volume-name:/path" or "/path")
			parts := strings.Split(volStr, ":")
			if len(parts) >= 2 {
				volumeName := parts[0]
				// Only consider named volumes, not bind mounts
				if !strings.HasPrefix(volumeName, "/") && !strings.HasPrefix(volumeName, ".") {
					volumesInUse[volumeName] = true
				}
			}
		}
	}

	// Remove unused volumes
	for volumeName := range compose.Volumes {
		if !volumesInUse[volumeName] {
			delete(compose.Volumes, volumeName)
		}
	}
}

// GetServiceInfo returns information about a specific service
func (sm *ServiceManager) GetServiceInfo(serviceName string) (map[string]interface{}, error) {
	compose, err := sm.ReadComposeFile()
	if err != nil {
		return nil, err
	}

	service, exists := compose.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service '%s' not found", serviceName)
	}

	svcMap, ok := service.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid service definition for '%s'", serviceName)
	}

	return svcMap, nil
}
