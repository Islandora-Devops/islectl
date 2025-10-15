/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/islandora-devops/islectl/pkg/compose"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage ISLE docker-compose services",
	Long: `Manage services in the docker-compose.yml file for ISLE installations.

You can list, disable, and enable services in your ISLE installation.
This is useful for removing optional services like blazegraph to reduce
resource usage or customize your installation.

Examples:
  # List all services in the current context
  islectl service list

  # Disable blazegraph service
  islectl service disable blazegraph

  # Disable multiple services
  islectl service disable blazegraph solr

  # Disable with confirmation prompt skipped
  islectl service disable blazegraph --yes`,
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services in the docker-compose.yml",
	Long: `List all services defined in the docker-compose.yml file.

This shows all services currently configured in your ISLE installation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		context, err := config.CurrentContext(cmd.Flags())
		if err != nil {
			return err
		}

		sm := compose.NewServiceManager(context)
		services, err := sm.ListServices()
		if err != nil {
			return fmt.Errorf("failed to list services: %w", err)
		}

		sort.Strings(services)

		fmt.Printf("Services in context '%s':\n", context.Name)
		for _, svc := range services {
			fmt.Printf("  - %s\n", svc)
		}

		return nil
	},
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable [service-name...]",
	Short: "Disable one or more services in docker-compose.yml",
	Long: `Disable services by removing them from the docker-compose.yml file.

This will:
  1. Remove related Drupal configuration (if applicable)
  2. Run 'drush config:import' to apply configuration changes
  3. Remove the service definition from docker-compose.yml
  4. Remove any orphaned volumes that were only used by the service

For services with Drupal integration (like blazegraph), this automatically
cleans up configuration files like actions and contexts.

WARNING: This operation modifies your docker-compose.yml and Drupal config.
Make sure you have a backup or your changes are committed to version control.

Examples:
  # Disable blazegraph service with confirmation prompt
  islectl service disable blazegraph

  # Disable multiple services
  islectl service disable blazegraph solr

  # Skip confirmation prompt (useful for automation)
  islectl service disable blazegraph --yes`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		context, err := config.CurrentContext(cmd.Flags())
		if err != nil {
			return err
		}

		sm := compose.NewServiceManager(context)

		// Check if services exist
		for _, serviceName := range args {
			exists, err := sm.ServiceExists(serviceName)
			if err != nil {
				return fmt.Errorf("failed to check service '%s': %w", serviceName, err)
			}
			if !exists {
				return fmt.Errorf("service '%s' not found in compose file", serviceName)
			}
		}

		// Get confirmation unless --yes flag is set
		skipConfirm, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return err
		}

		if !skipConfirm {
			fmt.Printf("You are about to disable the following service(s):\n")
			for _, svc := range args {
				fmt.Printf("  - %s\n", svc)
				// Show if service has Drupal config cleanup
				hook := compose.GetServiceHook(svc)
				if hook != nil && len(hook.ConfigFilesToRemove) > 0 {
					fmt.Printf("    → Will remove %d Drupal config file(s)\n", len(hook.ConfigFilesToRemove))
				}
			}
			fmt.Printf("\nThis will modify:\n")
			fmt.Printf("  - %s\n", sm.GetComposePath())
			fmt.Printf("  - config/sync/ (Drupal configuration)\n")
			fmt.Print("\nAre you sure you want to continue? [y/N]: ")

			input, err := config.GetInput("")
			if err != nil {
				return err
			}

			if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
				fmt.Println("Operation cancelled")
				return nil
			}
		}

		// Disable each service
		for _, serviceName := range args {
			slog.Info("Disabling service", "service", serviceName)
			if err := sm.DisableService(serviceName); err != nil {
				return fmt.Errorf("failed to disable service '%s': %w", serviceName, err)
			}
			fmt.Printf("✓ Service '%s' disabled\n", serviceName)
		}

		fmt.Println("\nServices successfully disabled!")
		fmt.Println("Run 'islectl compose down && islectl compose up' to apply changes.")

		return nil
	},
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable [service-name]",
	Short: "Enable a previously disabled service",
	Long: `Enable a service by restoring it to the docker-compose.yml file.

NOTE: This feature is not yet implemented. To re-enable a service,
you'll need to manually restore the service definition from your
version control system or a backup.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("enable command not yet implemented - please restore service definitions from version control or backup")
	},
}

var serviceInfoCmd = &cobra.Command{
	Use:   "info [service-name]",
	Short: "Show information about a service",
	Long: `Display detailed information about a specific service from docker-compose.yml.

This shows the raw service definition including image, ports, volumes, etc.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		context, err := config.CurrentContext(cmd.Flags())
		if err != nil {
			return err
		}

		serviceName := args[0]
		sm := compose.NewServiceManager(context)

		info, err := sm.GetServiceInfo(serviceName)
		if err != nil {
			return err
		}

		fmt.Printf("Service: %s\n", serviceName)
		fmt.Println("---")

		// Print service information
		if image, ok := info["image"]; ok {
			fmt.Printf("Image: %v\n", image)
		}

		if ports, ok := info["ports"].([]interface{}); ok {
			fmt.Println("Ports:")
			for _, port := range ports {
				fmt.Printf("  - %v\n", port)
			}
		}

		if volumes, ok := info["volumes"].([]interface{}); ok {
			fmt.Println("Volumes:")
			for _, vol := range volumes {
				fmt.Printf("  - %v\n", vol)
			}
		}

		if networks, ok := info["networks"].([]interface{}); ok {
			fmt.Println("Networks:")
			for _, net := range networks {
				fmt.Printf("  - %v\n", net)
			}
		}

		return nil
	},
}

func init() {
	// Add global flags
	flags := serviceCmd.PersistentFlags()
	flags.StringP("context", "c", "", "Context to use")

	// Disable command flags
	disableFlags := serviceDisableCmd.Flags()
	disableFlags.Bool("yes", false, "Skip confirmation prompt")

	// Add subcommands
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceInfoCmd)

	// Add to root command
	rootCmd.AddCommand(serviceCmd)
}
