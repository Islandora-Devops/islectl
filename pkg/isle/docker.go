package isle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/islandora-devops/islectl/pkg/config"
	"golang.org/x/crypto/ssh"
)

// DockerAPI abstracts the Docker client functionality needed by our package.
type DockerAPI interface {
	ContainerInspect(ctx context.Context, container string) (dockercontainer.InspectResponse, error)
}

type DockerClient struct {
	CLI        DockerAPI
	SshCli     *ssh.Client
	httpClient *http.Client
}

func (d *DockerClient) Close() error {
	var firstErr error
	if d.SshCli != nil {
		if err := d.SshCli.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if d.httpClient != nil {
		d.httpClient.CloseIdleConnections()
	}
	return firstErr
}

func GetDockerCli(activeCtx *config.Context) (*DockerClient, error) {
	if activeCtx.DockerHostType == config.ContextLocal {
		cli, err := client.NewClientWithOpts(
			client.WithHost("unix://"+activeCtx.DockerSocket),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating local Docker client: %v", err)
		}
		return &DockerClient{CLI: cli}, nil
	}
	sshConn, err := activeCtx.DialSSH()
	if err != nil {
		return nil, fmt.Errorf("error establishing SSH connection: %v", err)
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
		sshConn.Close()
		return nil, fmt.Errorf("error creating Docker client over SSH: %v", err)
	}
	return &DockerClient{
		CLI:        cli,
		SshCli:     sshConn,
		httpClient: httpClient,
	}, nil
}

func GetSecret(ctx context.Context, cli DockerAPI, c *config.Context, containerName, secretName string) (string, error) {
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", err
	}
	expectedTarget := filepath.Join("/run/secrets", secretName)
	for _, mount := range containerJSON.Mounts {
		if mount.Destination == expectedTarget {
			secretFilePath := filepath.Join(c.ProjectDir, "secrets", secretName)
			return c.ReadSmallFile(secretFilePath), nil
		}
	}
	return GetConfigEnv(ctx, cli, containerName, secretName)
}

func GetConfigEnv(ctx context.Context, cli DockerAPI, containerName, envName string) (string, error) {
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("error inspecting container %s: %v", containerName, err)
	}
	for _, env := range containerJSON.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && parts[0] == envName {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("environment variable %q not found in container %s", envName, containerName)
}

func (d *DockerClient) GetServiceIp(ctx context.Context, c *config.Context, containerName string) (string, error) {
	containerJSON, err := d.CLI.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("error inspecting container %q: %v", containerName, err)
	}
	networkName := fmt.Sprintf("%s_default", c.ProjectName)
	network, ok := containerJSON.NetworkSettings.Networks[networkName]
	if !ok {
		return "", fmt.Errorf("network %q not found in container %q", networkName, containerName)
	}
	return network.IPAddress, nil
}
