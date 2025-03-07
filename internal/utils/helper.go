package utils

import (
	"log/slog"
	"os"
)

func ExitOnError(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}
