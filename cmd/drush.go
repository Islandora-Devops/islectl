/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os/exec"
	"slices"
	"strings"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

var drushCmd = &cobra.Command{
	Use:                "drush",
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Short:              "Run drush commands on ISLE contexts",
	Long: `
Short hand for "islectl compose exec drupal drush".

This allows us to easily add additional features around common drush commands.

e.g. islectl drush uli auto-opens the reset link in the default web browser.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// since we're disabling flag parsing to make easy passing of flags to docker compose
		// handle the context flag
		filteredArgs, isleContext, err := utils.GetContextFromArgs(cmd, args)
		if err != nil {
			return err
		}

		drush := "drush"
		if !slices.Contains(filteredArgs, "--uri") && !slices.Contains(filteredArgs, "-l") {
			drush = drush + " --uri $DRUPAL_DRUSH_URI"
		}

		context, err := config.GetContext(isleContext)
		if err != nil {
			return err
		}

		cmdArgs := []string{
			"compose",
			"--profile",
			context.Profile,
			"exec",
			fmt.Sprintf("drupal-%s", context.Profile),
			"bash",
			"-c",
			fmt.Sprintf("%s %s", drush, shellquote.Join(filteredArgs...)),
		}
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		_, err = context.RunCommand(c)
		if err != nil {
			return err
		}

		return nil
	},
}

// login runs drush uli
var loginCmd = &cobra.Command{
	Use:   "uli",
	Short: `Runs "drush uli" to provide a direct login link`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := cmd.Flags()
		context, err := config.CurrentContext(f)
		if err != nil {
			return err
		}
		uid, err := f.GetUint("uid")
		if err != nil {
			return err
		}
		uri, err := f.GetString("uri")
		if err != nil {
			return err
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
			"exec",
			fmt.Sprintf("drupal-%s", context.Profile),
			"bash",
			"-c",
			fmt.Sprintf("drush uli --uri=%s --uid=%d", uri, uid),
		}...)
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		output, err := context.RunCommand(c)
		if err != nil {
			return err
		}

		if strings.HasPrefix(output, "http") {
			err := utils.OpenURL(output)
			if err != nil {
				slog.Warn("Error opening URL", "err", err)
			}
		}

		return nil
	},
}

func init() {
	loginCmd.Flags().Uint("uid", 1, "Drupal user ID to provide a direct login link for")
	loginCmd.Flags().String("uri", "$DRUPAL_DRUSH_URI", "--uri flag to pass to drush")

	drushCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(drushCmd)
}
