/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

// run docker compose build
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Drupal container.",
	Run: func(cmd *cobra.Command, args []string) {
		f := cmd.Flags()
		context, err := config.CurrentContext(f)
		if err != nil {
			utils.ExitOnError(err)
		}

		cmdArgs := []string{
			"compose",
			"--profile",
			context.Profile,
		}
		for _, env := range context.EnvFile {
			cmdArgs = append(cmdArgs, "--env-file", env)
		}
		cmdArgs = append(cmdArgs, "build", "--pull", "--quiet")
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		err = context.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
