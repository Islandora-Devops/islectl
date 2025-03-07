package isle

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
)

func GetDockerCli() *client.Client {
	// handle mac docker host issues
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///var/run/docker.sock"
		macSocketPath := filepath.Join(os.Getenv("HOME"), ".docker/run/docker.sock")
		if _, err := os.Stat(macSocketPath); err == nil {
			host = "unix://" + macSocketPath
		}
	}

	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("Unable to initialize docker client", "err", err)
		os.Exit(1)
	}

	return cli
}

func GetSecret(ctx context.Context, cli *client.Client, dir, containerName, secret string) (string, error) {
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", err
	}

	expectedTarget := fmt.Sprintf("/run/secrets/%s", secret)
	for _, mount := range containerJSON.HostConfig.Mounts {
		if mount.Target == expectedTarget {
			secret := filepath.Join(dir, "secrets", secret)
			return readSmallFile(secret), nil
		}
	}

	// if we didn't find the mounted secret, fall back to container default
	return GetConfigEnv(ctx, cli, containerName, secret)
}

func GetConfigEnv(ctx context.Context, cli *client.Client, containerName, envName string) (string, error) {
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("error inspecting container %s: %v", containerName, err)
	}
	for _, env := range containerJSON.Config.Env {
		line := strings.Split(env, "=")
		if line[0] == envName {
			return strings.Join(line[1:], "="), nil
		}
	}

	return "", nil
}

func readSmallFile(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("Error reading file", "file", filename, "err", err)
		return ""
	}

	return string(data)
}
