/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

var composeCmd = &cobra.Command{
	Use:                "compose COMMAND",
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Short:              "Run docker compose commands on ISLE contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		// since we're disabling flag parsing to make easy passing of flags to drush
		// handle the context flag
		isleContext, err := cmd.Root().PersistentFlags().GetString("context")
		if err != nil {
			return err
		}

		// remove --context flag from the args if it exists
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
		isleContext = strings.Trim(isleContext, `"`)

		validCommands := []string{
			"attach",
			"build",
			"commit",
			"config",
			"cp",
			"create",
			"down",
			"events",
			"exec",
			"export",
			"images",
			"kill",
			"logs",
			"ls",
			"pause",
			"port",
			"ps",
			"pull",
			"push",
			"restart",
			"rm",
			"run",
			"scale",
			"start",
			"stats",
			"stop",
			"top",
			"unpause",
			"up",
			"version",
			"wait",
			"watch",
			"-h",
			"--help",
		}
		if len(filteredArgs) == 0 || !slices.Contains(validCommands, filteredArgs[0]) {
			utils.ExitOnError(fmt.Errorf("unknown docker compose command: %s", filteredArgs[0]))
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

		if context.DockerHostType == config.ContextLocal {
			path := filepath.Join(context.ProjectDir, "docker-compose.yml")
			_, err = os.Stat(path)
			if err != nil {
				utils.ExitOnError(fmt.Errorf("docker-compose.yml not found at %s: %v", path, err))
			}
		}

		cmdArgs := []string{
			"compose",
			"--profile",
			context.Profile,
		}

		cmdArgs = append(cmdArgs, filteredArgs...)
		c := exec.Command("docker", cmdArgs...)
		c.Dir = context.ProjectDir
		_, err = context.RunCommand(c)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(composeCmd)
}
