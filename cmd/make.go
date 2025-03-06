/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/spf13/cobra"
)

// makeCmd support deprecated custom make commands
var makeCmd = &cobra.Command{
	Use:   "make",
	Short: "Run custom make commands",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		dir, profile, err := utils.GetRootFlags(cmd)
		if err != nil {
			slog.Error("Error getting root flags", "dir", dir, "profile", profile, "err", err)
			os.Exit(1)
		}

		c := exec.Command("make", args...)
		c.Dir = dir
		utils.RunCommand(c)
	},
}

func init() {
	rootCmd.AddCommand(makeCmd)
}
