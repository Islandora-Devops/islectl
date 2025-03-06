package utils

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func GetRootFlags(cmd *cobra.Command) (string, string, error) {
	dir, err := cmd.Root().PersistentFlags().GetString("dir")
	if err != nil {
		return "", "", fmt.Errorf("error getting --dir (%s): %v", dir, err)
	}
	profile, err := cmd.Root().PersistentFlags().GetString("profile")
	if err != nil {
		return "", "", fmt.Errorf("error getting --profile (%s): %v", profile, err)
	}

	return dir, profile, nil
}

func RunCommand(c *exec.Cmd, input ...string) {
	c.Env = os.Environ()

	// for commands that just need one answer from stdin we could
	// maybe support it automatic answer like so
	if len(input) > 0 && input[0] != "" {
		stdinPipe, err := c.StdinPipe()
		if err != nil {
			slog.Error("Error obtaining stdin pipe", "err", err)
			os.Exit(1)
		}
		go func() {
			defer stdinPipe.Close()
			_, err := stdinPipe.Write([]byte(input[0]))
			if err != nil {
				slog.Error("Error writing input", "err", err)
			}
		}()
		// but for the most part, just prompt the user
	} else {
		c.Stdin = os.Stdin
	}

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		slog.Error("Error writing to stdout", "err", err)
		os.Exit(1)
	}
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		slog.Error("Error running command", "cmd", c.String(), "err", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		slog.Error("Error reading stdout", "err", err)
	}

	if err := c.Wait(); err != nil {
		slog.Error("Error running command", "cmd", c.String(), "err", err)
		os.Exit(1)
	}
}
