/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

// createCmd brings an ISLE site online
var createCmd = &cobra.Command{
	Use:   "create-site",
	Short: "Create an ISLE site and its islectl context.",
	Run: func(cmd *cobra.Command, args []string) {
		f := cmd.Flags()
		context, err := config.LoadFromFlags(f)
		if err != nil {
			utils.ExitOnError(err)
		}

		trustFlags, err := cmd.Flags().GetBool("yes")
		if err != nil {
			utils.ExitOnError(err)
		}

		context.Name, err = cmd.Flags().GetString("context-name")
		if err != nil {
			utils.ExitOnError(err)
		}

		if !trustFlags {
			if context.Name == "" {
				slog.Error("Can not have a blank context name")
				os.Exit(1)
			}

			if cexists, err := config.ContextExists(context.Name); cexists || err != nil {
				slog.Error("Can not overwrite existing context", "err", err)
				os.Exit(1)
			}

			t, err := config.GetInput(fmt.Sprintf("Is the context local or remote? [%s]: ", string(context.DockerHostType)))
			if err != nil {
				utils.ExitOnError(err)
			}
			if t != "" {
				context.DockerHostType = config.ContextType(t)
			}
			dir, err := config.GetInput(fmt.Sprintf("Where would you like to install the project? [%s]: ", context.ProjectDir))
			if err != nil {
				utils.ExitOnError(err)
			}
			if dir != "" {
				context.ProjectDir = dir
			}

			if context.DockerHostType == config.ContextRemote {
				context.VerifyRemoteInput()
			}
		}

		exists, err := context.ProjectDirExists()
		if err != nil {
			utils.ExitOnError(err)
		}
		if exists {
			slog.Error("Project directory exists. Must create a new directory", "dir", context.ProjectDir, "err", err)
			os.Exit(1)
		}

		bt, err := cmd.Flags().GetString("buildkit-tag")
		if err != nil {
			utils.ExitOnError(err)
		}

		ss, err := cmd.Flags().GetString("starter-site")
		if err != nil {
			utils.ExitOnError(err)
		}

		defaultContext, err := f.GetBool("default")
		if err != nil {
			fmt.Printf("Error reading default flag: %v\n", err)
			return
		}

		err = isle.Setup(context, defaultContext, trustFlags, bt, ss)
		if err != nil {
			utils.ExitOnError(err)
		}

		slog.Info(fmt.Sprintf("Creation succeed. You may want to run `islectl up --context %s`", context.Name))
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	flags := createCmd.Flags()
	config.SetCommandFlags(flags)
	flags.Bool("yes", false, "Skip asking questions and just do the thing")
	flags.String("context-name", "", "Name of the context")
	flags.String("buildkit-tag", "main", "isle-buildkit tag to install")
	flags.String("starter-site", "main", "starter-site to install")
	flags.Bool("default", false, "set to default context")

	err := createCmd.MarkFlagRequired("context-name")
	if err != nil {
		slog.Error("Could not set context-name flag as required", "err", err)
	}
}
