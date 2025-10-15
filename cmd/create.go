/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create ISLE resources",
	Long: `Create ISLE sites and contexts.

Use 'create context' to install a new ISLE site from scratch.
Use 'create config' to add an islectl context for an existing ISLE site.`,
}

// createConfigCmd creates islectl config for existing isle-site-template installs
var createConfigCmd = &cobra.Command{
	Use:   "config [context-name]",
	Args:  cobra.ExactArgs(1),
	Short: "Create an islectl config for existing ISLE installs",
	Long: `Create an islectl context for an existing ISLE installation.

This command registers an existing ISLE site with islectl so you can manage it.
It does NOT create a new ISLE site - use 'create context' for that.

The command will interactively prompt for:
  - Whether the site is local or remote
  - Project directory path
  - Remote SSH connection details (if applicable)

Examples:
  # Create config for a local ISLE site
  islectl create config dev --type local --project-dir /home/user/isle

  # Create config for a remote ISLE site
  islectl create config prod \
    --type remote \
    --project-dir /opt/isle \
    --ssh-hostname isle.example.com \
    --ssh-user deploy \
    --ssh-key ~/.ssh/id_rsa`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cc, err := config.GetContext(args[0])
		if err != nil {
			return err
		}

		cexists := cc.DockerSocket != ""

		f := cmd.Flags()
		context, err := config.LoadFromFlags(f, cc)
		if err != nil {
			return err
		}
		context.Name = args[0]

		if cexists {
			overwrite, err := config.GetInput("The context already exists. Do you want to overwrite it? [y/N]: ")
			if err != nil {
				return err
			}
			if !strings.EqualFold(overwrite, "y") && !strings.EqualFold(overwrite, "yes") {
				fmt.Println("Cancelling...")
				os.Exit(1)
			}
		}

		t, err := config.GetInput(fmt.Sprintf("Is the context local (on this machine) or remote (on a VM)? [%s]: ", string(context.DockerHostType)))
		if err != nil {
			return err
		}
		if t != "" {
			if t != "remote" && t != "local" {
				fmt.Printf("Unknown context type (%s). Valid values are local or remote\n", t)
				os.Exit(1)
			}
			context.DockerHostType = config.ContextType(t)
		}
		dir, err := config.GetInput(fmt.Sprintf("Full directory path to the project (directory where docker-compose.yml is located) [%s]: ", context.ProjectDir))
		if err != nil {
			return err
		}
		if dir != "" {
			context.ProjectDir = dir
		}

		if context.DockerHostType == config.ContextRemote {
			err = cc.VerifyRemoteInput(true)
			if err != nil {
				return err
			}
		} else {
			if !f.Changed("docker-socket") {
				context.DockerSocket = config.GetDefaultLocalDockerSocket(context.DockerSocket)
			}
		}
		exists, err := context.ProjectDirExists()
		if err != nil {
			return err
		}
		if !exists {
			slog.Error("Project directory does not exist. You may want to create a new site and context with `islectl create context`", "dir", context.ProjectDir, "err", err)
			os.Exit(1)
		}

		defaultContext, err := f.GetBool("default")
		if err != nil {
			fmt.Printf("Error reading default flag: %v\n", err)
			return err
		}

		err = config.SaveContext(context, defaultContext)
		if err != nil {
			return err
		}

		contextStr, err := context.String()
		if err != nil {
			return err
		}
		fmt.Println("\nContext created successfully")
		fmt.Println(contextStr)

		return nil
	},
}

// createContextCmd creates an ISLE site and islectl context
var createContextCmd = &cobra.Command{
	Use:   "context [context-name]",
	Args:  cobra.ExactArgs(1),
	Short: "Create an ISLE site and islectl context",
	Long: `Create a new ISLE site from scratch and register it as an islectl context.

This command:
  1. Downloads the latest ISLE site template setup script
  2. Runs the installation (locally or remotely via SSH)
  3. Creates an islectl context to manage the new site

The installation will create a complete ISLE environment with Drupal and all required services.

Examples:
  # Create a local development site
  islectl create context dev \
    --type local \
    --profile dev \
    --project-dir /home/user/my-isle-site \
    --project-name my-site

  # Create a remote production site
  islectl create context prod \
    --type remote \
    --profile prod \
    --project-dir /opt/isle \
    --project-name my-institution \
    --ssh-hostname isle.example.com \
    --ssh-user deploy \
    --ssh-key ~/.ssh/id_rsa \
    --yes

Flags:
  --yes              Skip confirmation prompts (useful for automation)
  --buildkit-tag     ISLE buildkit tag/branch to use (default: main)
  --starter-site     Starter site branch to use (default: main)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cn := args[0]
		cc, err := config.GetContext(cn)
		if err != nil {
			return err
		}

		f := cmd.Flags()
		context, err := config.LoadFromFlags(f, cc)
		if err != nil {
			return err
		}
		context.Name = cn

		trustFlags, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return err
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
				return err
			}
			if t != "" {
				if t != "remote" && t != "local" {
					fmt.Printf("Unknown context type (%s). Valid values are local or remote\n", t)
					os.Exit(1)
				}
				context.DockerHostType = config.ContextType(t)
			}
			dir, err := config.GetInput(fmt.Sprintf("Where would you like to install the project? [%s]: ", context.ProjectDir))
			if err != nil {
				return err
			}
			if dir != "" {
				context.ProjectDir = dir
			}

			if context.DockerHostType == config.ContextRemote {
				err = cc.VerifyRemoteInput(false)
				if err != nil {
					return err
				}
			}
		}

		exists, err := context.ProjectDirExists()
		if err != nil {
			return err
		}
		if exists {
			slog.Error("Project directory exists. Must create a new directory", "dir", context.ProjectDir, "err", err)
			os.Exit(1)
		}

		bt, err := cmd.Flags().GetString("buildkit-tag")
		if err != nil {
			return err
		}

		ss, err := cmd.Flags().GetString("starter-site")
		if err != nil {
			return err
		}

		defaultContext, err := f.GetBool("default")
		if err != nil {
			fmt.Printf("Error reading default flag: %v\n", err)
			return err
		}

		err = isle.Setup(context, defaultContext, trustFlags, bt, ss)
		if err != nil {
			return err
		}

		slog.Info(fmt.Sprintf("Creation succeed. You may want to run `islectl up --context %s`", context.Name))
		return nil
	},
}

func init() {
	flags := createContextCmd.Flags()
	config.SetCommandFlags(flags)
	flags.Bool("yes", false, "Skip asking questions and just do the thing")
	flags.String("buildkit-tag", "main", "isle-buildkit tag to install")
	flags.String("starter-site", "main", "starter-site to install")
	flags.Bool("default", false, "set to default context")

	createCmd.AddCommand(createContextCmd)

	flags = createConfigCmd.Flags()
	config.SetCommandFlags(flags)
	flags.Bool("default", false, "set to default context")

	createCmd.AddCommand(createConfigCmd)

	rootCmd.AddCommand(createCmd)
}
