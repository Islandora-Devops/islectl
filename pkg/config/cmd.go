package config

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func (c *Context) RunCommand(cmd *exec.Cmd) ([]string, error) {
	var output []string
	if c.DockerHostType == ContextLocal {
		cmd.Env = os.Environ()
		cmd.Stdin = os.Stdin
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("error writing to stdout command %s: %v", cmd.String(), err)
		}
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("error starting command %s: %v", cmd.String(), err)
		}

		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			t := scanner.Text()
			fmt.Println(t)
			output = append(output, strings.TrimSpace(t))
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Error reading stdout", "err", err)
		}

		if err := cmd.Wait(); err != nil {
			return nil, fmt.Errorf("error running command %s: %v", cmd.String(), err)
		}

		return output, nil
	}

	// at this point, we know it's a remote context
	sshClient, err := c.DialSSH()
	if err != nil {
		return nil, fmt.Errorf("error establishing SSH connection: %v", err)
	}
	defer sshClient.Close()

	// Build the remote command string.
	// We assume cmd.Path is the command and cmd.Args contains arguments.
	remoteCmd := fmt.Sprintf("cd %s &&", c.ProjectDir)
	if c.RunSudo {
		remoteCmd += " sudo"
	}

	remoteCmd += " " + cmd.Args[0]
	if len(cmd.Args) > 1 {
		remoteCmd += " " + shellquote.Join(cmd.Args[1:]...)
	}

	slog.Info("Running remote command", "host", c.SSHHostname, "cmd", remoteCmd)

	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("error creating SSH session: %v", err)
	}
	defer session.Close()

	// create a pseudo terminal incase the command needs input
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.ECHOCTL:       0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = 80
		height = 40
	}
	if err := session.RequestPty("xterm", width, height, modes); err != nil {
		return nil, fmt.Errorf("error requesting pseudo terminal: %w", err)
	}

	// Prepare stdin pipe for the session.
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating stdin pipe: %v", err)
	}
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error obtaining stdout pipe: %v", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("error obtaining stderr pipe: %v", err)
	}
	combined := io.MultiReader(stdoutPipe, stderrPipe)

	// Start the remote command.
	if err := session.Start(remoteCmd); err != nil {
		return nil, fmt.Errorf("error starting remote command %q: %v", remoteCmd, err)
	}

	buf := make([]byte, 1024)
	prompted := false
	for {
		n, err := combined.Read(buf)
		if n > 0 {
			line := string(buf[:n])
			fmt.Print(line)
			output = append(output, line)
			if !prompted && strings.Contains(line, "[sudo] password for") {
				prompted = true
				pwd, err := promptPassword()
				if err != nil {
					slog.Error("Error reading password", "err", err)
				} else {
					fmt.Fprintln(stdinPipe, pwd)
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			slog.Error("Error reading remote output", "err", err)
			break
		}
	}
	stdinPipe.Close()
	// Wait for the remote command to complete.
	if err := session.Wait(); err != nil {
		return nil, fmt.Errorf("error running remote command %q: %v", remoteCmd, err)
	}

	return output, nil
}

func promptPassword() (string, error) {
	pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(pwdBytes), nil
}

func SetCommandFlags(flags *pflag.FlagSet) {
	path, err := os.Getwd()
	if err != nil {
		slog.Error("Unable to get current working directory", "err", err)
		os.Exit(1)
	}
	env := filepath.Join(path, ".env")
	_ = godotenv.Load(env)

	key := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")

	// NB: these flags must match the corresponding config.Context yaml struct tag
	// though we can add additional flags that have no match for additional functionality
	// in the command logic (e.g. default)
	flags.String("docker-socket", "/var/run/docker.sock", "Path to Docker socket")
	flags.String("type", "local", "Type of context: local or remote")
	flags.String("ssh-hostname", "islandora.dev", "Remote contexts DNS name for the host.")
	flags.Uint("ssh-port", 2222, "Port number")
	flags.String("ssh-user", "nginx", "SSH user for remote context")
	flags.String("ssh-key", "", "Path to SSH private key for remote context. e.g. "+key)
	flags.String("project-dir", "", "Path to docker compose project directory")
	flags.String("project-name", "isle-site-template", "Name of the docker compose project")
	flags.String("profile", "dev", "docker compose profile")
	flags.String("site", "default", "drupal multisite")
	flags.Bool("sudo", false, "for remote contexts, run commands as sudo")
	flags.StringSlice("env-file", []string{}, "when running remote docker commands, the --env-file paths to pass to docker compose")
}
