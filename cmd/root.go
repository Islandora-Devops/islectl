/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "islectl",
	Short: "Interact with your ISLE site",
	Long:  `Interact with your ISLE site`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func SetVersionInfo(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit)
}

func init() {
	c, err := config.Current()
	if err != nil {
		slog.Error("Unable to fetch current context", "err", err)
	}

	rootCmd.PersistentFlags().StringP("site", "s", "default", "The name of the site. If yr not using multi-site don't worry about this.")
	rootCmd.PersistentFlags().StringP("context", "c", c, "The ISLE context to use. See islectl config --help for more info")
}
