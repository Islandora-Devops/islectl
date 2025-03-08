/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"log/slog"
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

// pull brings an ISLE site online
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Fetches the latest images from the registry.",
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
		cmdArgs = append(cmdArgs, "pull")
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		err = context.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
		slog.Info("pull complete")
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
