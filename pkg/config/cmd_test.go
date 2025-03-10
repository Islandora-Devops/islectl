package config

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRunCommandLocal(t *testing.T) {
	ctx := &Context{
		DockerHostType: ContextLocal,
	}
	cmd := exec.Command("echo", "hello")
	output, err := ctx.RunCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) == 0 || !strings.Contains(output[0], "hello") {
		t.Fatalf("expected output to contain 'hello', got %v", output)
	}
}
