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

func Setup(context *config.Context, bt, ss, sn string) error {
	if context.DockerHostType == config.ContextRemote {
		return fmt.Errorf("Currently setup is only supported on local machines")
	}

	fmt.Printf("Site doesn't appear to exist at %s.\nProceed creating it there? Y/n: ", context.ProjectDir)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}
	input = strings.TrimSpace(input)
	if input != "" && !strings.EqualFold(input, "Y") {
		return fmt.Errorf("cancelling install operation")
	}

	fmt.Println("Creating site...")
	tmpFileName := downloadSetup()

	// supply the child directory passed as what we'll call the site
	name := filepath.Base(context.ProjectDir)
	if sn != "" {
		name = sn
	}
	cmdArgs := []string{
		tmpFileName,
		fmt.Sprintf("--buildkit-tag=%s", bt),
		fmt.Sprintf("--starter-site-branch=%s", ss),
		fmt.Sprintf("--site-name=%s", name),
	}
	c := exec.Command("bash", cmdArgs...)

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %v", err)
	}

	c.Dir = context.ProjectDir
	if wd != context.ProjectDir && sn == "" {
		c.Dir = filepath.Dir(context.ProjectDir)
	}

	c.Env = os.Environ()
	c.Stdin = os.Stdin
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error writing to stdout: %v", err)
	}
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return fmt.Errorf("error starting command %s: %v", c.String(), err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdout %s: %v", c.String(), err)
	}

	if err := c.Wait(); err != nil {
		return fmt.Errorf("error running command %s: %v", c.String(), err)
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
