/*
Copyright © 2025 Islandora Foundation
*/
package drupal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:                "exec [COMMAND]",
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Short:              "Execute commands in the Drupal container",
	Long: `Execute arbitrary commands in the Drupal container.

If no command is provided, opens an interactive bash shell in the container.
This is useful for debugging, running composer commands, or performing file operations.

Examples:
  islectl drupal exec                              # Open interactive bash shell
  islectl drupal exec ls -la /var/www/drupal/web   # List files
  islectl drupal exec composer require drupal/devel # Install a module
  islectl drupal exec "drush cr && drush status"   # Run multiple commands`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// since we're disabling flag parsing to make easy passing of flags to docker compose
		// handle the context flag
		filteredArgs, isleContext, err := utils.GetContextFromArgs(cmd, args)
		if err != nil {
			return err
		}
		context, err := config.GetContext(isleContext)
		if err != nil {
			return err
		}

		if context.DockerHostType == config.ContextLocal {
			path := filepath.Join(context.ProjectDir, "docker-compose.yml")
			_, err = os.Stat(path)
			if err != nil {
				utils.ExitOnError(fmt.Errorf("docker-compose.yml not found at %s: %v", path, err))
			}
		}

		cli, err := isle.GetDockerCli(&context)
		if err != nil {
			return err
		}
		drupalContainer, err := cli.GetContainerName(&context, "drupal", false)
		if err != nil {
			return err
		}
		cmdArgs := []string{
			"exec",
			"-i",
			drupalContainer,
		}

		if len(filteredArgs) == 0 {
			filteredArgs = []string{"bash"}
		}

		cmdArgs = append(cmdArgs, filteredArgs...)
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		_, err = context.RunCommand(c)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(execCmd)
}
