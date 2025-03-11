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
	cli, err := GetDockerCli(c)
	if err != nil {
		return "", "", err
	}

	containerName, err := cli.GetContainerName(c, "mariadb", false)
	if err != nil {
		return "", "", err
	}
	ctx := context.Background()

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

	mysqlUri := fmt.Sprintf("mysql://%s:%s@", envs["DB_ROOT_USER"], envs["DB_ROOT_PASSWORD"])
	sshUri := fmt.Sprintf("ssh_host=%s&ssh_port=%d&ssh_user=%s", c.SSHHostname, c.SSHPort, c.SSHUser)
	if c.DockerHostType == config.ContextLocal {
		containerName, err := cli.GetContainerName(c, "ide", true)
		if err != nil {
			return "", "", err
		}
		idePass, err := GetConfigEnv(ctx, cli.CLI, containerName, "CODE_SERVER_PASSWORD")
		if err != nil {
			return "", "", err
		}
		// for local contexts, we'll SSH into codeserver and use docker internal DNS to access mariadb
		mysqlUri = mysqlUri + fmt.Sprintf("%s:%s/%s", envs["DB_MYSQL_HOST"], envs["DB_MYSQL_PORT"], fmt.Sprintf("drupal_%s", c.Site))
		sshUri = sshUri + fmt.Sprintf("&ssh_password=%s", idePass)
	} else if c.DockerHostType == config.ContextRemote {
		// on remote hosts we need to get the IP address
		// mariadb is exposed at in the network namespace
		serviceIp, err := cli.GetServiceIp(ctx, c, containerName)
		if err != nil {
			return "", "", err
		}

		// for remote contexts, we'll SSH into the remote server
		// and use the docker network namespace IP:port for mariadb
		mysqlUri = mysqlUri + fmt.Sprintf("%s:%s/%s", serviceIp, envs["DB_MYSQL_PORT"], fmt.Sprintf("drupal_%s", c.Site))
		sshUri = sshUri + fmt.Sprintf("&ssh_keyLocation=%s&ssh_keyLocationEnabled=1", c.SSHKeyPath)
	}

	return mysqlUri, sshUri, nil
}

func Setup(context *config.Context, defaultContext, confirmed bool, bt, ss string) error {
	contextStr, err := context.String()
	if err != nil {
		return err
	}
	fmt.Println("\nHere is the context that will be created")
	fmt.Println(contextStr)

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
	url := "https://raw.githubusercontent.com/Islandora-Devops/isle-site-template/76ccdbc95919b63b2536f25441fc901aa1d71ba3/setup.sh"
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
