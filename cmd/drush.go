/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os/exec"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		// since we're disabling flag parsing to make easy passing of flags to drush
		// handle the context flag
		isleContext, err := cmd.Root().PersistentFlags().GetString("context")
		if err != nil {
			return err
		}

		// remove --context flag from the args if it exits
		// and set it as the default context if it does
		filteredArgs := []string{}
		skipNext := false
		for _, arg := range args {
			if arg == "--context" {
				skipNext = true
				continue
			}
			if strings.HasPrefix(arg, "--context=") {
				components := strings.Split(arg, "=")
				isleContext = components[1]
				continue
			}
			if skipNext {
				isleContext = arg
				skipNext = false
				continue
			}
			filteredArgs = append(filteredArgs, arg)
		}

		f := cmd.Flags()
		err = f.Set("context", isleContext)
		if err != nil {
			return err
		}
		context, err := config.CurrentContext(f)
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
			fmt.Sprintf("drush %s", shellquote.Join(filteredArgs...)),
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

		len := len(output)
		if len > 0 {
			err := utils.OpenURL(output[len-1])
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
