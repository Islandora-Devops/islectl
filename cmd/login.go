/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"os/exec"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

// login runs drush uli
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: `Runs "drush uli" to provide a direct login link`,
	Run: func(cmd *cobra.Command, args []string) {
		f := cmd.Flags()
		context, err := config.CurrentContext(f)
		if err != nil {
			utils.ExitOnError(err)
		}
		uid, err := f.GetUint("uid")
		if err != nil {
			utils.ExitOnError(err)
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
			fmt.Sprintf("drush uli --uri=$DRUPAL_DRUSH_URI --uid=%d", uid),
		}...)
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		err = context.RunCommand(c)
		if err != nil {
			utils.ExitOnError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().Uint("uid", 1, "Drupal user ID to provide a direct login link for")

}
