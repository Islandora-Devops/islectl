package config

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestLoadFromFlags(t *testing.T) {
	// Create a new flag set.
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("docker-socket", "/var/run/docker.sock", "Path to Docker socket")
	flags.String("type", "local", "Context type: local or remote")
	flags.String("profile", "default", "Profile name")
	flags.String("ssh-hostname", "example.com", "SSH host for remote context")
	flags.Uint("ssh-port", 22, "port")
	flags.String("ssh-user", "user", "SSH user for remote context")
	flags.String("ssh-key", "/path/to/ssh-key", "Path to SSH private key for remote context")
	flags.String("project-dir", "/path/to/project", "Project directory")
	flags.String("project-name", "foo", "Composer Project Name")
	flags.String("site", "foo", "Composer Project Name")
	flags.Bool("sudo", false, "Run commands on remote hosts as sudo")
	flags.StringSlice("env-file", []string{}, "path to env files to pass to docker compose")

	// Define test arguments to override defaults.
	args := []string{
		"--docker-socket", "/custom/docker.sock",
		"--type", "remote",
		"--profile", "prod",
		"--ssh-hostname", "remote.example.com",
		"--ssh-port", "123",
		"--ssh-user", "remoteuser",
		"--ssh-key", "/custom/ssh-key",
		"--project-dir", "/custom/project",
		"--project-name", "bar",
		"--site", "default",
		"--sudo", "true",
		"--env-file", ".env",
		"--env-file", "/tmp/.env",
	}
	if err := flags.Parse(args); err != nil {
		t.Fatalf("Error parsing flags: %v", err)
	}

	c := Context{}
	ctx, err := LoadFromFlags(flags, c)
	if err != nil {
		t.Fatalf("Error loading from flags: %v", err)
	}

	// Verify that each field is set as expected.
	if ctx.DockerSocket != "/custom/docker.sock" {
		t.Errorf("Expected docker-socket '/custom/docker.sock', got %q", ctx.DockerSocket)
	}
	if ctx.DockerHostType != "remote" {
		t.Errorf("Expected type 'remote', got %q", ctx.DockerHostType)
	}
	if ctx.Profile != "prod" {
		t.Errorf("Expected profile 'prod', got %q", ctx.Profile)
	}
	if ctx.SSHHostname != "remote.example.com" {
		t.Errorf("Expected ssh-host 'remote.example.com', got %q", ctx.SSHHostname)
	}
	if ctx.SSHPort != 123 {
		t.Errorf("Expected port 123, got %d", ctx.SSHPort)
	}
	if ctx.SSHUser != "remoteuser" {
		t.Errorf("Expected ssh-user 'remoteuser', got %q", ctx.SSHUser)
	}
	if ctx.SSHKeyPath != "/custom/ssh-key" {
		t.Errorf("Expected ssh-key '/custom/ssh-key', got %q", ctx.SSHKeyPath)
	}
	if ctx.ProjectDir != "/custom/project" {
		t.Errorf("Expected project-dir '/custom/project', got %q", ctx.ProjectDir)
	}
	if ctx.ProjectName != "bar" {
		t.Errorf("Expected project-name 'bar', got %q", ctx.ProjectName)
	}
	if ctx.Site != "default" {
		t.Errorf("Expected site 'default', got %q", ctx.ProjectName)
	}
	if ctx.RunSudo != true {
		t.Errorf("Expected site 'true', got %t", ctx.RunSudo)
	}
	expectedSlice := []string{".env", "/tmp/.env"}
	if !reflect.DeepEqual(ctx.EnvFile, expectedSlice) {
		t.Errorf("expected env-file slice %v but got %v", expectedSlice, ctx.EnvFile)
	}
}

func TestGetInput(t *testing.T) {
	input := "hello\n"
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create input pipe: %v", err)
	}
	_, err = inW.WriteString(input)
	if err != nil {
		t.Fatalf("failed to write input: %v", err)
	}
	inW.Close()
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	os.Stdin = inR
	origStdout := os.Stdout
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create output pipe: %v", err)
	}
	os.Stdout = outW

	result, err := GetInput("Enter input: ")
	if err != nil {
		t.Fatalf("GetInput error: %v", err)
	}
	if result != "hello" {
		t.Fatalf("expected %q, got %q", "hello", result)
	}

	outW.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, outR)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	os.Stdout = origStdout

	output := buf.String()
	expectedPrompt := "Enter input: "
	if !strings.Contains(output, expectedPrompt) {
		t.Fatalf("expected output to contain %q, got %q", expectedPrompt, output)
	}
}
