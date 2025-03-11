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

func (c *Context) RunCommand(cmd *exec.Cmd) (string, error) {
	var output string
	if c.DockerHostType == ContextLocal {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		cmd.Env = os.Environ()
		cmd.Stdin = os.Stdin
		cmd.Dir = c.ProjectDir
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return "", fmt.Errorf("error creating stdout pipe for command %s: %v", cmd.String(), err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("error starting command %s: %v", cmd.String(), err)
		}
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			output = strings.TrimSpace(line)
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Error reading stdout", "err", err)
		}
		if err := cmd.Wait(); err != nil {
			return "", fmt.Errorf("error waiting for command %s: %v", cmd.String(), err)
		}
		return output, nil
	}

	sshClient, err := c.DialSSH()
	if err != nil {
		return "", fmt.Errorf("error establishing SSH connection: %v", err)
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
		return "", fmt.Errorf("error creating SSH session: %v", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = 80
		height = 40
	}
	if err := session.RequestPty("xterm", width, height, modes); err != nil {
		return "", fmt.Errorf("error requesting pseudo terminal: %w", err)
	}

	// set terminal to raw for easier stdin/out/err handling
	// between the os and ssh session
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return "", fmt.Errorf("failed to set terminal to raw mode: %v", err)
		}
		defer func() {
			if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
				slog.Error("Unable to return terminal to original state.", "err", err)
			}
		}()
	}

	// setup some stdout/err pipes so we can capture output
	session.Stdin = os.Stdin
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error obtaining stdout pipe: %v", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("error obtaining stderr pipe: %v", err)
	}
	combined := io.MultiReader(stdoutPipe, stderrPipe)

	// call ssh foo@host.tld "remoteCmd"
	if err := session.Start(remoteCmd); err != nil {
		return "", fmt.Errorf("error starting remote command %q: %v", remoteCmd, err)
	}

	// save the output from the command so we can return it
	buf := make([]byte, 1024)
	for {
		n, err := combined.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			fmt.Print(chunk)
			output = chunk
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			slog.Error("Error reading remote output", "err", err)
			break
		}
	}

	if err = session.Wait(); err != nil {
		// do not mark error on sigint
		if exitErr, ok := err.(*ssh.ExitError); ok && exitErr.ExitStatus() == 130 {
			return output, nil
		}
		return "", fmt.Errorf("error waiting for remote command %q: %v", remoteCmd, err)
	}

	return output, nil
}
