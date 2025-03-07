package utils

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

func RunCommand(c *exec.Cmd, input ...string) error {
	c.Env = os.Environ()
	c.Stdin = os.Stdin
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error writing to stdout command %s: %v", c.String(), err)
	}
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return fmt.Errorf("error starting command %s: %v", c.String(), err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		slog.Error("Error reading stdout", "err", err)
	}

	if err := c.Wait(); err != nil {
		return fmt.Errorf("error running command %s: %v", c.String(), err)
	}

	return nil
}
