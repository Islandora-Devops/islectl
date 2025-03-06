/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/spf13/cobra"
)

// upCmd brings an ISLE site online
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Brings up the containers or builds starter if no containers were found.",
	Run: func(cmd *cobra.Command, args []string) {
		dir, profile, err := utils.GetRootFlags(cmd)
		if err != nil {
			slog.Error("Error getting root flags", "dir", dir, "profile", profile, "err", err)
			os.Exit(1)
		}

		path := filepath.Join(dir, "docker-compose.yml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Site doesn't appear to exist at %s.\nProceed creating it there? Y/n: ", dir)
			input, err := reader.ReadString('\n')
			if err != nil {
				slog.Error("Error reading input", "err", err)
				os.Exit(1)
			}
			input = strings.TrimSpace(input)
			if input == "" || strings.EqualFold(input, "Y") {
				fmt.Println("Creating site...")
				tmpFileName := downloadSetup()

				// supply the child directory passed as what we'll call the site
				name := filepath.Base(dir)
				cmdArgs := []string{
					tmpFileName,
					"--buildkit-tag=main",
					"--starter-site-branch=main",
					fmt.Sprintf("--site-name=%s", name),
				}
				c := exec.Command("bash", cmdArgs...)
				// checkout isle-site-template in the basedir
				c.Dir = filepath.Dir(dir)
				utils.RunCommand(c)
				slog.Info("Site created!")
			} else {
				fmt.Println("Cancelling...")
				return
			}
		} else if err != nil {
			slog.Error("Error checking for docker-compose.yml", "path", path, "err", err)
			os.Exit(1)
		}

		cmdArgs := []string{
			"compose",
			"--profile",
			profile,
			"up",
			"-d",
			"--remove-orphans",
		}
		c := exec.Command("docker", cmdArgs...)
		c.Dir = dir
		utils.RunCommand(c)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func downloadSetup() string {
	url := "https://raw.githubusercontent.com/Islandora-Devops/isle-site-template/support-flags/setup.sh"
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("failed to download install script", "err", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "setup-*.sh")
	if err != nil {
		slog.Error("failed to create temp file", "err", err)
		os.Exit(1)
	}
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		slog.Error("failed to write to temp file", "err", err)
		os.Exit(1)
	}
	if err := tmpFile.Chmod(0755); err != nil {
		slog.Error("failed to set executable permissions", "err", err)
		os.Exit(1)
	}
	tmpFile.Close()

	return tmpFile.Name()
}
