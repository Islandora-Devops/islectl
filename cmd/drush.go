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
	Use:                "drush [COMMAND]",
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Short:              "Run drush commands on ISLE contexts",
	Long: `Run drush commands on ISLE contexts.

This is a shorthand for "islectl compose exec drupal drush" with automatic --uri handling.
The DRUPAL_DRUSH_URI environment variable is automatically passed unless you specify --uri or -l.

Special subcommands:
  uli - Generate and auto-open a one-time login link in your browser

Examples:
  islectl drush status                      # Check Drupal status
  islectl drush cr                          # Clear all caches
  islectl drush cex                         # Export configuration
  islectl drush cim                         # Import configuration
  islectl drush uli                         # Generate login link and open in browser
  islectl drush uli --uid=2                 # Login link for user ID 2
  islectl drush sqlq "SHOW TABLES"          # Run SQL query
  islectl drush --context prod status       # Check status on prod context`,
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
	Short: "Generate a one-time login link",
	Long: `Generate a one-time login link and automatically open it in your default browser.

This runs 'drush uli' in the Drupal container and opens the resulting URL.

Examples:
  islectl drush uli              # Login as admin (user 1)
  islectl drush uli --uid=2      # Login as user ID 2
  islectl drush uli --uri=https://example.com  # Use specific URI`,
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
