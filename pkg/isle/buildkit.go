package isle

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/islandora-devops/islectl/pkg/config"
	yaml "gopkg.in/yaml.v3"
)

func GetUris(c *config.Context) (string, string, error) {
	cli := GetDockerCli(c)

	containerName := fmt.Sprintf("%s-mariadb-%s-1", c.ProjectName, c.Profile)
	ctx := context.Background()

	var err error
	vars := []string{
		"DB_ROOT_USER",
		"DB_ROOT_PASSWORD",
		"DB_MYSQL_HOST",
		"DB_MYSQL_PORT",
	}
	envs := make(map[string]string, len(vars))
	for _, v := range vars {
		envs[v], err = GetSecret(ctx, cli.CLI, c, containerName, v)
		if err != nil {
			return "", "", err
		}
	}

	mysqlUri := fmt.Sprintf("mysql://%s:%s@%s:%s/%s", envs["DB_ROOT_USER"], envs["DB_ROOT_PASSWORD"], envs["DB_MYSQL_HOST"], envs["DB_MYSQL_PORT"], fmt.Sprintf("drupal_%s", c.Site))
	sshUri := fmt.Sprintf("ssh_host=%s&ssh_port=%d&ssh_user=%s&ssh_keyLocation=%s&ssh_keyLocationEnabled=1", c.SSHHostname, c.SSHPort, c.SSHUser, c.SSHKeyPath)
	if c.DockerHostType == config.ContextLocal {
		containerName = fmt.Sprintf("%s-ide-1", c.ProjectName)
		idePass, err := GetConfigEnv(ctx, cli.CLI, containerName, "CODE_SERVER_PASSWORD")
		if err != nil {
			return "", "", err
		}
		sshUri = fmt.Sprintf("ssh_host=%s&ssh_port=%d&ssh_user=%s&ssh_password=%s", c.SSHHostname, c.SSHPort, c.SSHUser, idePass)
	} else if c.DockerHostType == config.ContextRemote {
		// on remote hosts we need to get the IP address
		// mariadb is exposed at in the network namespace
		containerJSON, err := cli.CLI.ContainerInspect(ctx, containerName)
		if err != nil {
			return "", "", fmt.Errorf("error inspecting container %q: %v", containerName, err)
		}

		networkName := fmt.Sprintf("%s_default", c.ProjectName)
		network, ok := containerJSON.NetworkSettings.Networks[networkName]
		if !ok {
			return "", "", fmt.Errorf("network %q not found in container %q", networkName, containerName)
		}

		mysqlUri = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", envs["DB_ROOT_USER"], envs["DB_ROOT_PASSWORD"], network.IPAddress, envs["DB_MYSQL_PORT"], fmt.Sprintf("drupal_%s", c.Site))

	}

	return mysqlUri, sshUri, nil
}

func Setup(context *config.Context, defaultContext, confirmed bool, bt, ss string) error {
	out, err := yaml.Marshal(context)
	if err != nil {
		slog.Error("Unable to parse context")
		return err
	}
	fmt.Println("\nHere is the context that will be created")
	fmt.Println(string(out))

	fmt.Println("\nAnd these buildkit/starter site flags")
	flags := []string{
		fmt.Sprintf("--buildkit-tag=%s", bt),
		fmt.Sprintf("--starter-site-branch=%s", ss),
		fmt.Sprintf("--site-name=%s", context.ProjectName),
	}
	for _, f := range flags {
		fmt.Println(f)
	}
	if !confirmed {
		fmt.Printf("\nAre you sure you want to proceed creating the site? y/N: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}
		input = strings.TrimSpace(input)
		if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
			return fmt.Errorf("cancelling install operation")
		}
	}

	fmt.Println("Creating site...")
	tmpFileName := downloadSetup()
	if context.DockerHostType == config.ContextRemote {
		destination := "/tmp/isle-setup.sh"
		err = context.UploadFile(tmpFileName, "/tmp/isle-setup.sh")
		if err != nil {
			return fmt.Errorf("unable to upload file to %s: %v", destination, err)
		}
		tmpFileName = destination
	}

	cmdArgs := []string{
		"-l",
		tmpFileName,
	}
	cmdArgs = append(cmdArgs, flags...)
	c := exec.Command("bash", cmdArgs...)
	originalDir := context.ProjectDir

	// need to cd into the base dir of the project
	context.ProjectDir = filepath.Dir(context.ProjectDir)
	if _, err = context.RunCommand(c); err != nil {
		slog.Error("Error installing site", "err", err)
		os.Exit(1)
	}

	// put project dir back to its original value before saving the context
	context.ProjectDir = originalDir
	if err = config.SaveContext(context, defaultContext); err != nil {
		slog.Error("Error saving context.", "err", err)
		os.Exit(1)
	}
	fmt.Println("Site created!")

	return nil
}

func downloadSetup() string {
	url := "https://raw.githubusercontent.com/Islandora-Devops/isle-site-template/support-flags/setup.sh"
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("failed to download install script", "err", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "setup-*.sh")
	if err != nil {
		slog.Error("failed to create temp file", "err", err)
		os.Exit(1)
	}
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		slog.Error("failed to write to temp file", "err", err)
		os.Exit(1)
	}
	if err := tmpFile.Chmod(0755); err != nil {
		slog.Error("failed to set executable permissions", "err", err)
		os.Exit(1)
	}
	tmpFile.Close()

	return tmpFile.Name()
}
