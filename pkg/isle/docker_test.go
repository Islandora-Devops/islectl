package isle

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/islandora-devops/islectl/pkg/config"
)

// FakeDockerClient implements the DockerAPI interface for testing.
type FakeDockerClient struct {
	InspectFunc func(ctx context.Context, container string) (dockercontainer.InspectResponse, error)
}

var _ DockerAPI = (*FakeDockerClient)(nil)

func (f *FakeDockerClient) ContainerInspect(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
	return f.InspectFunc(ctx, container)
}

func (f *FakeDockerClient) ContainerList(ctx context.Context, options dockercontainer.ListOptions) ([]dockercontainer.Summary, error) {
	return nil, fmt.Errorf("Not implemented")
}

func TestGetConfigEnv_VariableFound(t *testing.T) {
	fake := &FakeDockerClient{
		InspectFunc: func(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
			return dockercontainer.InspectResponse{
				Config: &dockercontainer.Config{
					Env: []string{"TEST_ENV=value123", "OTHER=foo"},
				},
			}, nil
		},
	}
	value, err := GetConfigEnv(context.Background(), fake, "dummyContainer", "TEST_ENV")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "value123" {
		t.Errorf("expected %q, got %q", "value123", value)
	}
}

func TestGetConfigEnv_VariableNotFound(t *testing.T) {
	fake := &FakeDockerClient{
		InspectFunc: func(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
			return dockercontainer.InspectResponse{
				Config: &dockercontainer.Config{
					Env: []string{"OTHER=foo"},
				},
			}, nil
		},
	}
	_, err := GetConfigEnv(context.Background(), fake, "dummyContainer", "MISSING")
	if err == nil {
		t.Fatal("expected an error for missing environment variable, got nil")
	}
	expected := `environment variable "MISSING" not found`
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error message to contain %q, got %q", expected, err.Error())
	}
}

func TestGetConfigEnv_MalformedEnvEntries(t *testing.T) {
	fake := &FakeDockerClient{
		InspectFunc: func(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
			return dockercontainer.InspectResponse{
				Config: &dockercontainer.Config{
					Env: []string{"MALFORMED", "TEST_ENV =valueWithSpace", "ANOTHER=valid"},
				},
			}, nil
		},
	}
	expected := `environment variable "TEST_ENV" not found in container dummyContainer`
	_, err := GetConfigEnv(context.Background(), fake, "dummyContainer", "TEST_ENV")
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error message to contain %q, got %q", expected, err.Error())
	}

}

func TestGetConfigEnv_MultipleEquals(t *testing.T) {
	fake := &FakeDockerClient{
		InspectFunc: func(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
			return dockercontainer.InspectResponse{
				Config: &dockercontainer.Config{
					Env: []string{"TEST_ENV=part1=part2"},
				},
			}, nil
		},
	}
	value, err := GetConfigEnv(context.Background(), fake, "dummyContainer", "TEST_ENV")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "part1=part2" {
		t.Errorf("expected %q, got %q", "part1=part2", value)
	}
}

func TestGetSecret_MountedSecret(t *testing.T) {
	fake := &FakeDockerClient{
		InspectFunc: func(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
			resp := dockercontainer.InspectResponse{
				Config: &dockercontainer.Config{
					Env: []string{"SECRET=envSecret"},
				},
				Mounts: []dockercontainer.MountPoint{
					{
						Destination: filepath.Join("/run/secrets", "secretName"),
					},
				},
			}
			return resp, nil
		},
	}
	fakeConfig := &config.Context{
		ProjectDir:  "/tmp/project",
		ProjectName: "test",
		ReadSmallFileFunc: func(path string) string {
			if strings.HasSuffix(path, filepath.Join("secrets", "secretName")) {
				return "fileSecret"
			}
			return ""
		},
	}
	secret, err := GetSecret(context.Background(), fake, fakeConfig, "dummyContainer", "secretName")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "fileSecret" {
		t.Errorf("expected %q, got %q", "fileSecret", secret)
	}
}

func TestGetServiceIp(t *testing.T) {
	fake := &FakeDockerClient{
		InspectFunc: func(ctx context.Context, container string) (dockercontainer.InspectResponse, error) {
			return dockercontainer.InspectResponse{
				NetworkSettings: &dockercontainer.NetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"test_default": {IPAddress: "172.17.0.3"},
					},
				},
			}, nil
		},
	}
	fakeConfig := &config.Context{
		ProjectName: "test",
	}
	dClient := &DockerClient{
		CLI: fake,
	}
	ip, err := dClient.GetServiceIp(context.Background(), fakeConfig, "dummyContainer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "172.17.0.3" {
		t.Errorf("expected %q, got %q", "172.17.0.3", ip)
	}
}
