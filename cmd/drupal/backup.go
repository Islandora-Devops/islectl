/*
Copyright © 2025 Islandora Foundation
*/
package drupal

import (
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "backup drupal stateful files",
	RunE: func(cmd *cobra.Command, args []string) error {
		f := cmd.Flags()
		context, err := config.CurrentContext(f)
		if err != nil {
			utils.ExitOnError(err)
		}

		cli, err := isle.GetDockerCli(context)
		if err != nil {
			return err
		}
		drupalContainer, err := cli.GetContainerName(context, "drupal", false)
		if err != nil {
			return err
		}

		cmdArgs := []string{
			"exec",
			drupalContainer,
			"drush",
			"sql-dump",
			"-y",
			"--skip-tables-list=cache,cache_*,watchdog",
			"--structure-tables-list=cache,cache_*,watchdog",
			"--debug",
			"--gzip",
			"--result-file=/tmp/db.tar.gz",
		}

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
	backupCmd.Flags().StringSlice("file", []string{"database"}, "components to backup")
	backupCmd.Flags().StringSlice("component", []string{"database"}, "components to backup")

	RootCmd.AddCommand(backupCmd)
}
