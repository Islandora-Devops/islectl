package utils

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
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
