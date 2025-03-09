/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

// upCmd brings an ISLE site online
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Brings up the containers",
	Run: func(cmd *cobra.Command, args []string) {
		f := cmd.Flags()
		context, err := config.CurrentContext(f)
		if err != nil {
			utils.ExitOnError(err)
		}

		path := filepath.Join(context.ProjectDir, "docker-compose.yml")
		_, err = os.Stat(path)
		if err != nil {
			slog.Error("Error checking for docker-compose.yml", "path", path, "err", err)
			os.Exit(1)
		}

		cmdArgs := []string{
			"compose",
			"--profile",
			context.Profile,
		}
		for _, env := range context.EnvFile {
			cmdArgs = append(cmdArgs, "--env-file", env)
		}
		cmdArgs = append(cmdArgs, []string{
			"up",
			"-d",
			"--remove-orphans",
		}...)
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		_, err = context.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
