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

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

type BuildkitCompose struct {
	WorkingDirectory    string
	ComposeProfile      string
	ComposeProject      string
	DrupalMultisiteName string
	MySqlUri            string
	SshInfo             string
}

func NewBuildkitCommand(cmd *cobra.Command) (*BuildkitCompose, error) {
	dir, err := cmd.Root().PersistentFlags().GetString("dir")
	if err != nil {
		return nil, fmt.Errorf("error getting --dir (%s): %v", dir, err)
	}
	profile, err := cmd.Root().PersistentFlags().GetString("profile")
	if err != nil {
		return nil, fmt.Errorf("error getting --profile (%s): %v", profile, err)
	}

	site, err := cmd.Root().PersistentFlags().GetString("site")
	if err != nil {
		return nil, fmt.Errorf("error getting --site (%s): %v", site, err)
	}

	project, err := cmd.Root().PersistentFlags().GetString("compose-project")
	if err != nil {
		return nil, fmt.Errorf("error getting --compose-project (%s): %v", project, err)
	}
	// if --dir was passed
	// we might not have gotten the COMPOSE_PROJECT_NAME passed
	// so try grabbing it
	if project == "" {
		env := filepath.Join(dir, ".env")
		_ = godotenv.Load(env)
		project = os.Getenv("COMPOSE_PROJECT_NAME")
	}

	bkc := &BuildkitCompose{
		WorkingDirectory:    dir,
		ComposeProfile:      profile,
		ComposeProject:      project,
		DrupalMultisiteName: site,
	}

	err = bkc.loadEnv()
	if err != nil {
		return nil, fmt.Errorf("not able to load environment. Do you need to run islectl up?\n%v", err)
	}

	return bkc, nil
}

func (bc *BuildkitCompose) loadEnv() error {

	path := filepath.Join(bc.WorkingDirectory, "docker-compose.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	cli := GetDockerCli()

	containerName := fmt.Sprintf("%s-mariadb-%s-1", bc.ComposeProject, bc.ComposeProfile)
	ctx := context.Background()

	var err error
	vars := []string{
		"DB_ROOT_USER",
		"DB_ROOT_PASSWORD",
		"DB_MYSQL_HOST",
		"DB_MYSQL_PORT",
	}
	config := make(map[string]string, len(vars))
	for _, v := range vars {
		config[v], err = GetSecret(ctx, cli, bc.WorkingDirectory, containerName, v)
		if err != nil {
			return err
		}
	}
	containerName = fmt.Sprintf("%s-ide-1", bc.ComposeProject)
	idePass, err := GetConfigEnv(ctx, cli, containerName, "CODE_SERVER_PASSWORD")
	if err != nil {
		return err
	}

	bc.MySqlUri = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", config["DB_ROOT_USER"], config["DB_ROOT_PASSWORD"], config["DB_MYSQL_HOST"], config["DB_MYSQL_PORT"], fmt.Sprintf("drupal_%s", bc.DrupalMultisiteName))
	bc.SshInfo = fmt.Sprintf("ssh_host=%s&ssh_port=%d&ssh_user=%s&ssh_password=%s", os.Getenv("DOMAIN"), 2222, "nginx", idePass)

	return nil
}

func (bc *BuildkitCompose) Setup(path, bt, ss, sn string) error {
	fmt.Printf("Site doesn't appear to exist at %s.\nProceed creating it there? Y/n: ", bc.WorkingDirectory)

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
	name := filepath.Base(bc.WorkingDirectory)
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

	// by default, assummes we're naming the site the basepath of --dir
	c.Dir = bc.WorkingDirectory
	// but if --dir was passed and a sitename was not
	// assumme we're naming the site
	if wd != bc.WorkingDirectory && sn == "" {
		c.Dir = filepath.Dir(bc.WorkingDirectory)
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
