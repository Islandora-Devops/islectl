/*
Copyright © 2025 Islandora Foundation
*/
package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"

	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ISLE command configuration",
	Long: `An ISLE config can have multiple contexts.
A context is an ISLE docker compose project running somewhere. Somewhere could be:
- on your laptop (--type local)
- on a remote server (--type remote).

It's assummed where you're running islectl from has SSH access to where that context is running. i.e. on a remote server, your local machine has an ssh key that is tied to a user on the remote server`,
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
	Run: func(cmd *cobra.Command, args []string) {
		f := cmd.Flags()
		cc, err := config.LoadFromFlags(f)
		if err != nil {
			fmt.Printf("Error reading default flag: %v\n", err)
			return
		}
		cc.Name = args[0]

		defaultContext, err := f.GetBool("default")
		if err != nil {
			fmt.Printf("Error reading default flag: %v\n", err)
			return
		}

		// override local defaults for remote environments
		if cc.DockerHostType == config.ContextRemote {
			if cc.SSHHostname == "islandora.dev" {
				question := []string{
					"You should not be setting SSH hostname to islandora.dev?\n",
					"You may have forgot to pass --ssh-hostname",
					"Instead you can enter the remote server domain name here: ",
				}
				h, err := config.GetInput(question...)
				if err != nil || h == "" {
					slog.Error("Error reading input")
					os.Exit(1)
				}
				cc.SSHHostname = h

			}
			if cc.SSHUser == "nginx" {
				u, err := user.Current()
				if err != nil {
					slog.Error("Unable to determine current user", "err", err)
					os.Exit(1)
				}
				cc.SSHUser = u.Username
				slog.Warn("You may need to pass --ssh-user for the remote host. Defaulting to your username to connect to the remote host", "name", u.Username)
			}

			if cc.Profile == "dev" {
				question := []string{
					"Are you sure you want \"dev\" for the docker compose profile on the remote context?\n",
					"Enter the profile here, enter nothing to keep dev: [dev]: ",
				}
				p, err := config.GetInput(question...)
				if err != nil {
					slog.Error("Error reading input")
					os.Exit(1)
				}
				if p != "" {
					slog.Info("Setting profile", "profile", p)
					cc.Profile = p
				}
			}
		} else if cc.DockerHostType == config.ContextLocal {
			cc.SSHKeyPath = ""
			cc.DockerSocket = config.GetDefaultLocalDockerSocket(cc.DockerSocket)
		} else {
			slog.Error("Unknown context type", "type", cc.DockerHostType)
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			log.Fatal(err)
		}

		updated := false
		for i, ctx := range cfg.Contexts {
			if ctx.Name == cc.Name {
				cfg.Contexts[i] = *cc

				updated = true
				break
			}
		}

		if !updated {
			cfg.Contexts = append(cfg.Contexts, *cc)
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = cc.Name
			}
			fmt.Printf("Added new context: %s\n", cc.Name)
		} else {
			fmt.Printf("Updated context: %s\n", cc.Name)
		}

		if defaultContext {
			cfg.CurrentContext = cc.Name
		}

		if err = config.Save(cfg); err != nil {
			log.Fatal(err)
		}
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
	path, err := os.Getwd()
	if err != nil {
		slog.Error("Unable to get current working directory", "err", err)
		os.Exit(1)
	}
	env := filepath.Join(path, ".env")
	_ = godotenv.Load(env)

	key := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")

	flags := setContextCmd.Flags()

	// NB: these flags must match the corresponding config.Context yaml struct tag
	// though we can add additional flags that have no match for additional functionality
	// in the command logic (e.g. default)
	flags.String("docker-socket", "/var/run/docker.sock", "Path to Docker socket")
	flags.String("type", "local", "Type of context: local or remote")
	flags.String("ssh-hostname", "islandora.dev", "Remote contexts DNS name for the host.")
	flags.Uint("ssh-port", 2222, "Port number")
	flags.String("ssh-user", "nginx", "SSH user for remote context")
	flags.String("ssh-key", key, "Path to SSH private key for remote context")
	flags.String("project-dir", path, "Path to docker compose project directory")
	flags.String("project-name", "isle-site-template", "Name of the docker compose project")
	flags.String("profile", "dev", "docker compose profile")
	flags.String("site", "default", "drupal multisite")
	flags.Bool("sudo", false, "for remote contexts, run commands as sudo")
	flags.StringSlice("env-file", []string{}, "when running remote docker commands, the --env-file paths to pass to docker compose")

	flags.Bool("default", false, "set to default context")

	configCmd.AddCommand(viewConfigCmd)
	configCmd.AddCommand(currentContextCmd)
	configCmd.AddCommand(getContextsCmd)
	configCmd.AddCommand(setContextCmd)
	configCmd.AddCommand(useContextCmd)
	configCmd.AddCommand(deleteContextCmd)
	rootCmd.AddCommand(configCmd)
}
