/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
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
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	env := filepath.Join(path, ".env")
	_ = godotenv.Load(env)

	rootCmd.PersistentFlags().StringP("profile", "p", "dev", "docker compose profile (dev or prod)")
	rootCmd.PersistentFlags().StringP("dir", "d", path, "path to isle-site-template for your site. Defaults to current directory.")
	rootCmd.PersistentFlags().String("compose-project", os.Getenv("COMPOSE_PROJECT_NAME"), "Docker compose project name")
	rootCmd.PersistentFlags().StringP("site", "s", "default", "The name of the site, in reference to isle-buildkit's drupal multisite support.")

}
