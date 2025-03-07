/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

// makeCmd support deprecated custom make commands
var makeCmd = &cobra.Command{
	Use:   "make",
	Short: "Run custom make commands",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		bc, err := isle.NewBuildkitCommand(cmd)
		if err != nil {
			utils.ExitOnError(err)
		}

		c := exec.Command("make", args...)
		c.Dir = bc.WorkingDirectory
		err = utils.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(makeCmd)
}
