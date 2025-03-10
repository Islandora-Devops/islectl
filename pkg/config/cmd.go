package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/kballard/go-shellquote"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func (c *Context) RunCommand(cmd *exec.Cmd) ([]string, error) {
	var output []string
	if c.DockerHostType == ContextLocal {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		cmd.Env = os.Environ()
		cmd.Stdin = os.Stdin
		cmd.Dir = c.ProjectDir
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("error creating stdout pipe for command %s: %v", cmd.String(), err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("error starting command %s: %v", cmd.String(), err)
		}
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			output = append(output, strings.TrimSpace(line))
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Error reading stdout", "err", err)
		}
		if err := cmd.Wait(); err != nil {
			return nil, fmt.Errorf("error waiting for command %s: %v", cmd.String(), err)
		}
		return output, nil
	}

	sshClient, err := c.DialSSH()
	if err != nil {
		return nil, fmt.Errorf("error establishing SSH connection: %v", err)
	}
	defer sshClient.Close()

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
	if err := session.Start(remoteCmd); err != nil {
		return nil, fmt.Errorf("error starting remote command %q: %v", remoteCmd, err)
	}

	buf := make([]byte, 1024)
	prompted := false
	for {
		n, err := combined.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			fmt.Print(chunk)
			output = append(output, chunk)
			if !prompted && strings.Contains(chunk, "[sudo] password for") {
				prompted = true
				pwd, err := promptPassword()
				if err != nil {
					slog.Error("Error reading password", "err", err)
				} else {
					if _, err := fmt.Fprintln(stdinPipe, pwd); err != nil {
						slog.Error("Error writing password to stdin", "err", err)
					}
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
	if err := session.Wait(); err != nil {
		return nil, fmt.Errorf("error waiting for remote command %q: %v", remoteCmd, err)
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
