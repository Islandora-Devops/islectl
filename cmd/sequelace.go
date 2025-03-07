/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

var sequelAceCmd = &cobra.Command{
	Use:   "sequelace",
	Short: "Connect to your ISLE database using Sequel Ace (Mac OS only)",
	Run: func(cmd *cobra.Command, args []string) {
		bc, err := isle.NewBuildkitCommand(cmd)
		if err != nil {
			utils.ExitOnError(err)
		}

		sequelAcePath, err := cmd.Flags().GetString("sequel-ace-path")
		if err != nil {
			utils.ExitOnError(err)
		}
		cmdArgs := []string{
			fmt.Sprintf("%s?%s", bc.MySqlUri, bc.SshInfo),
			"-a",
			sequelAcePath,
		}
		openCmd := exec.Command("open", cmdArgs...)
		if err := openCmd.Run(); err != nil {
			slog.Error("Could not open sequelace.", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(sequelAceCmd)

	sequelAceCmd.Flags().String("sequel-ace-path", "/Applications/Sequel Ace.app/Contents/MacOS/Sequel Ace", "Full path to your Sequel Ace app (default /Applications/Sequel Ace.app/Contents/MacOS/Sequel Ace)")
}
