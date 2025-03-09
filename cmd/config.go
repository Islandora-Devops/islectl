/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/islandora-devops/islectl/internal/utils"
	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ISLE command configuration",
	Long: `
An islectl config can have multiple contexts.

An islectl context is an ISLE docker compose project running somewhere. "Somewhere" meaning:

- on your laptop (--type local)
- on a remote server (--type remote).

Remote contexts require SSH access to the remote server from where islectl is being ran from.
When creating a context the remote server DNS name, SSH port, SSH username, and your SSH private key will need to be set in the context configuration.

You can have a default context which will be used when running islectl commands, unless the context is overridden with the --context flag.`,
}

var viewConfigCmd = &cobra.Command{
	Use:   "view",
	Short: "Print your islectl config",
	Run: func(cmd *cobra.Command, args []string) {
		path := config.ConfigFilePath()
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("File %q does not exist.\n", path)
				return
			}
			log.Fatalf("Error checking file: %v", err)
		}

		// Check if it's a regular file.
		if !info.Mode().IsRegular() {
			log.Fatalf("%q is not a regular file", path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}

		fmt.Println(string(data))
	},
}

var currentContextCmd = &cobra.Command{
	Use:   "current-context",
	Short: "Display the current ISLE context",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.Current()
		if err != nil {
			log.Fatal(err)
		}
		if c == "" {
			fmt.Println("No current context is set")
		} else {
			fmt.Println("Current context:", c)
		}
	},
}

var getContextsCmd = &cobra.Command{
	Use:   "get-contexts",
	Short: "List all ISLE contexts",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			log.Fatal(err)
		}
		if len(cfg.Contexts) == 0 {
			fmt.Println("No contexts available")
			return
		}
		for _, ctx := range cfg.Contexts {
			activeMark := " "
			if ctx.Name == cfg.CurrentContext {
				activeMark = "*"
			}
			fmt.Printf("%s %s (type: %s)\n", activeMark, ctx.Name, ctx.DockerHostType)
		}
	},
}

var setContextCmd = &cobra.Command{
	Use:   "set-context [context-name]",
	Short: "Set or update properties of a context. Creates a new context if it does not exist.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		context, err := config.GetContext(args[0])
		if err != nil {
			return err
		}

		f := cmd.Flags()
		cc, err := config.LoadFromFlags(f, context)
		if err != nil {
			return err
		}
		cc.Name = args[0]

		defaultContext, err := f.GetBool("default")
		if err != nil {
			return err
		}

		// override local defaults for remote environments
		if cc.DockerHostType == config.ContextRemote {
			cc.VerifyRemoteInput(true)
		} else if cc.DockerHostType == config.ContextLocal {
			cc.SSHKeyPath = ""
			cc.DockerSocket = config.GetDefaultLocalDockerSocket(cc.DockerSocket)
		} else {
			slog.Error("Unknown context type", "type", cc.DockerHostType)
			os.Exit(1)
		}

		if err = config.SaveContext(cc, defaultContext); err != nil {
			utils.ExitOnError(err)
		}

		return nil
	},
}

var useContextCmd = &cobra.Command{
	Use:   "use-context [context-name]",
	Short: "Switch to the specified context",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			log.Fatal(err)
		}
		found := false
		for _, ctx := range cfg.Contexts {
			if ctx.Name == name {
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("Context %s not found", name)
		}
		cfg.CurrentContext = name
		if err = config.Save(cfg); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Switched to context:", name)
	},
}

var deleteContextCmd = &cobra.Command{
	Use:   "delete-context [context-name]",
	Short: "Delete an ISLE context",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			log.Fatal(err)
		}

		if cfg.CurrentContext == name {
			slog.Error("Cannot delete the current context. You can update it or create a new context with `isletctl config set-context`")
			return
		}

		found := false
		var newContexts []config.Context
		for _, ctx := range cfg.Contexts {
			if ctx.Name == name {
				found = true
				continue
			}
			newContexts = append(newContexts, ctx)
		}
		if !found {
			log.Fatalf("Context %s not found", name)
		}
		cfg.Contexts = newContexts

		if err = config.Save(cfg); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Deleted context: %s\n", name)
	},
}

func init() {
	flags := setContextCmd.Flags()
	config.SetCommandFlags(flags)
	flags.Bool("default", false, "set to default context")

	configCmd.AddCommand(viewConfigCmd)
	configCmd.AddCommand(currentContextCmd)
	configCmd.AddCommand(getContextsCmd)
	configCmd.AddCommand(setContextCmd)
	configCmd.AddCommand(useContextCmd)
	configCmd.AddCommand(deleteContextCmd)
	rootCmd.AddCommand(configCmd)
}
