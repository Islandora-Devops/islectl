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
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

// upCmd brings an ISLE site online
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Brings up the containers or builds starter if no containers were found.",
	Run: func(cmd *cobra.Command, args []string) {
		f := cmd.Flags()
		context, err := config.CurrentContext(f)
		if err != nil {
			utils.ExitOnError(err)
		}

		path := filepath.Join(context.ProjectDir, "docker-compose.yml")
		_, err = os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			slog.Error("Error checking for docker-compose.yml", "path", path, "err", err)
			os.Exit(1)
		}

		if context.DockerHostType == config.ContextLocal && os.IsNotExist(err) {
			bt, err := cmd.Flags().GetString("buildkit-tag")
			if err != nil {
				utils.ExitOnError(err)
			}

			ss, err := cmd.Flags().GetString("starter-site")
			if err != nil {
				utils.ExitOnError(err)
			}

			sn, err := cmd.Flags().GetString("site-name")
			if err != nil {
				utils.ExitOnError(err)
			}
			err = isle.Setup(context, bt, ss, sn)
			if err != nil {
				utils.ExitOnError(err)
			}
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
		err = context.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().String("buildkit-tag", "main", "isle-buildkit tag to install. Only used on creation")
	upCmd.Flags().String("starter-site", "main", "starter-site to install. Only used on creation")
	upCmd.Flags().String("site-name", "", "site name to install. Only used on creation")
}
