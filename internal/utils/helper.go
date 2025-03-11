package utils

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func ExitOnError(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}

// open a URL from the terminal
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unknown runtime command to open URL")
	}

	return cmd.Start()
}

// for cobra commands that allow arbitrary args to facilitate passing flags to other commands
// strip out islectl's context flag from the args if it was passed
func GetContextFromArgs(cmd *cobra.Command, args []string) ([]string, string, error) {
	isleContext, err := cmd.Root().PersistentFlags().GetString("context")
	if err != nil {
		return nil, "", err
	}

	// remove --context flag from the args if it exists
	// and set it as the default context if it was passed as a flag
	filteredArgs := []string{}
	skipNext := false
	for _, arg := range args {
		if arg == "--context" {
			skipNext = true
			continue
		}
		if strings.HasPrefix(arg, "--context=") {
			components := strings.Split(arg, "=")
			isleContext = components[1]
			continue
		}
		if skipNext {
			isleContext = arg
			skipNext = false
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}

	isleContext = strings.Trim(isleContext, `" `)

	return filteredArgs, isleContext, nil
}
