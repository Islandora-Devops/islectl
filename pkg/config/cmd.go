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

	shellquote "github.com/kballard/go-shellquote"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func (c *Context) RunCommand(cmd *exec.Cmd, input ...string) error {
	if c.DockerHostType == ContextLocal {
		cmd.Env = os.Environ()
		cmd.Stdin = os.Stdin
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("error writing to stdout command %s: %v", cmd.String(), err)
		}
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error starting command %s: %v", cmd.String(), err)
		}

		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Error reading stdout", "err", err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("error running command %s: %v", cmd.String(), err)
		}

		return nil
	}

	sshClient, err := c.DialSSH()
	if err != nil {
		return fmt.Errorf("error establishing SSH connection: %v", err)
	}
	defer sshClient.Close()

	// Build the remote command string.
	// We assume cmd.Path is the command and cmd.Args contains arguments.
	remoteCmd := fmt.Sprintf("cd %s &&", c.ProjectDir)
	if c.RunSudo {
		remoteCmd += " sudo"
	}
	// just get the binary name since local/remote paths are probably not the same
	remoteCmd += " " + filepath.Base(cmd.Path)
	if len(cmd.Args) > 1 {
		remoteCmd += " " + shellquote.Join(cmd.Args[1:]...)
	}

	slog.Info("Running remote command", "host", c.SSHHostname, "cmd", remoteCmd)

	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("error creating SSH session: %v", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.ECHOCTL:       0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return fmt.Errorf("error requesting pseudo terminal: %w", err)
	}

	// Prepare stdin pipe for the session.
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating stdin pipe: %v", err)
	}
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error obtaining stdout pipe: %v", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("error obtaining stderr pipe: %v", err)
	}
	combined := io.MultiReader(stdoutPipe, stderrPipe)

	// Start the remote command.
	if err := session.Start(remoteCmd); err != nil {
		return fmt.Errorf("error starting remote command %q: %v", remoteCmd, err)
	}

	buf := make([]byte, 1024)
	prompted := false
	for {
		n, err := combined.Read(buf)
		if n > 0 {
			output := string(buf[:n])
			fmt.Print(output)
			if !prompted && strings.Contains(output, "[sudo] password for") {
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
		return fmt.Errorf("error running remote command %q: %v", remoteCmd, err)
	}
	return nil
}

func promptPassword() (string, error) {
	pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(pwdBytes), nil
}
