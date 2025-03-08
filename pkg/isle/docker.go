package isle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/islandora-devops/islectl/pkg/config"
	"golang.org/x/crypto/ssh"
)

type DockerClient struct {
	CLI     *client.Client
	sshConn *ssh.Client
}

func (d *DockerClient) Close() error {
	if d.sshConn != nil {
		return d.sshConn.Close()
	}
	return nil
}

// GetDockerCli returns a DockerClient wrapper.
// If the context is remote, it creates an SSH tunnel; otherwise, it uses the local Docker socket.
func GetDockerCli(activeCtx *config.Context) *DockerClient {
	if activeCtx.DockerHostType == config.ContextLocal {
		cli, err := client.NewClientWithOpts(
			client.WithHost("unix://"+activeCtx.DockerSocket),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating local Docker client: %v\n", err)
			os.Exit(1)
		}
		return &DockerClient{CLI: cli}
	}

	sshConn, err := activeCtx.DialSSH()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error establishing SSH connection: %v\n", err)
		os.Exit(1)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshConn.Dial("unix", activeCtx.DockerSocket)
		},
	}
	httpClient := &http.Client{
		Transport: transport,
	}
	cli, err := client.NewClientWithOpts(
		client.WithHost("http://docker"),
		client.WithHTTPClient(httpClient),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Docker client over SSH: %v\n", err)
		os.Exit(1)
	}

	return &DockerClient{
		CLI:     cli,
		sshConn: sshConn,
	}
}

func GetSecret(ctx context.Context, cli *client.Client, c *config.Context, containerName, secret string) (string, error) {
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", err
	}

	expectedTarget := fmt.Sprintf("/run/secrets/%s", secret)
	for _, mount := range containerJSON.HostConfig.Mounts {
		if mount.Target == expectedTarget {
			secret := filepath.Join(c.ProjectDir, "secrets", secret)
			return c.ReadSmallFile(secret), nil
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
