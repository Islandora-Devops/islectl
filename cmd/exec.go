/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:                "exec [SERVICE] [COMMAND] [ARGS...]",
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Short:              "exec into compose services running in ISLE contexts",
	Long: `
Equivalent of docker compose --profile dev exec SERVICE COMMAND [ARGS...]
`,
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
		if len(filteredArgs) < 2 {
			for _, arg := range filteredArgs {
				if arg == "--help" || arg == "-h" {
					return fmt.Errorf("see help")
				}
			}
			return fmt.Errorf("bad command (%s). See islectl exec --help", strings.Join(filteredArgs, " "))
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
			fmt.Sprintf("%s-%s", filteredArgs[0], context.Profile),
			filteredArgs[1],
		}
		if len(filteredArgs) > 2 {
			cmdArgs = append(cmdArgs, shellquote.Join(filteredArgs[2:]...))
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

func init() {
	rootCmd.AddCommand(execCmd)
}
