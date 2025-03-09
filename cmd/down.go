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

// run docker compose down
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Run docker compose down",
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
		cmdArgs = append(cmdArgs, "down")
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		_, err = context.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
