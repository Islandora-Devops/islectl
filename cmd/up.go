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
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

// upCmd brings an ISLE site online
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Brings up the containers or builds starter if no containers were found.",
	Run: func(cmd *cobra.Command, args []string) {
		bc, err := isle.NewBuildkitCommand(cmd)
		if err != nil {
			utils.ExitOnError(err)
		}

		bt, err := cmd.Flags().GetString("buildkit-tag")
		if err != nil {
			utils.ExitOnError(err)
		}

		path := filepath.Join(bc.WorkingDirectory, "docker-compose.yml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			ss, err := cmd.Flags().GetString("starter-site")
			if err != nil {
				utils.ExitOnError(err)
			}

			sn, err := cmd.Flags().GetString("site-name")
			if err != nil {
				utils.ExitOnError(err)
			}
			err = bc.Setup(path, bt, ss, sn)
			if err != nil {
				utils.ExitOnError(err)
			}
		} else if err != nil {
			slog.Error("Error checking for docker-compose.yml", "path", path, "err", err)
			os.Exit(1)
		}

		cmdArgs := []string{
			"compose",
			"--profile",
			bc.ComposeProfile,
			"up",
			"-d",
			"--remove-orphans",
		}
		c := exec.Command("docker", cmdArgs...)
		c.Dir = bc.WorkingDirectory
		err = utils.RunCommand(c)
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
