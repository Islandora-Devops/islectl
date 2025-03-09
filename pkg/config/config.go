package config

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	yaml "gopkg.in/yaml.v3"
)

type ContextType string

const (
	ContextLocal  ContextType = "local"
	ContextRemote ContextType = "remote"
)

type Context struct {
	Name           string      `yaml:"name"`
	DockerHostType ContextType `mapstructure:"type" yaml:"type"`
	DockerSocket   string      `yaml:"docker-socket"`
	ProjectName    string      `yaml:"project-name"`
	Profile        string      `yaml:"profile"`
	ProjectDir     string      `yaml:"project-dir"`
	SSHUser        string      `yaml:"ssh-user"`
	SSHHostname    string      `yaml:"ssh-hostname,omitempty"`
	SSHPort        uint        `yaml:"ssh-port,omitempty"`
	SSHKeyPath     string      `yaml:"ssh-key,omitempty"`
	Site           string      `yaml:"site"`
	EnvFile        []string    `yaml:"env-file"`
	RunSudo        bool        `yaml:"sudo"`
}

type Config struct {
	CurrentContext string    `yaml:"current-context"`
	Contexts       []Context `yaml:"contexts"`
}

func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Unable to detect home directory", "err", err)
		os.Exit(1)
	}

	baseDir := filepath.Join(home, ".islectl")
	_, err = os.Stat(baseDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(baseDir, 0700)
		if err != nil {
			slog.Error("Unable to create ~/.islectl directory", "err", err)
			os.Exit(1)
		}
	}

	return filepath.Join(baseDir, "config.yaml")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigFilePath())
	if err != nil {
		return &Config{}, nil
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}

func ContextExists(name string) (bool, error) {
	c, err := Load()
	if err != nil {
		return false, err
	}

	for _, context := range c.Contexts {
		if strings.EqualFold(context.Name, name) {
			return true, nil
		}
	}

	return false, nil
}

func Save(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFilePath(), data, 0644)
}

func Current() (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	if cfg.CurrentContext == "" {
		return "", nil
	}

	return cfg.CurrentContext, nil
}

func SaveContext(ctx *Context, setDefault bool) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	updated := false
	for i, c := range cfg.Contexts {
		if c.Name == ctx.Name {
			cfg.Contexts[i] = *ctx

			updated = true
			break
		}
	}

	if !updated {
		cfg.Contexts = append(cfg.Contexts, *ctx)
		if cfg.CurrentContext == "" {
			cfg.CurrentContext = ctx.Name
		}
		fmt.Printf("Added new context: %s\n", ctx.Name)
	} else {
		fmt.Printf("Updated context: %s\n", ctx.Name)
	}

	if setDefault {
		cfg.CurrentContext = ctx.Name
	}

	return Save(cfg)
}

func CurrentContext(f *pflag.FlagSet) (*Context, error) {
	c, err := f.GetString("context")
	if err != nil {
		return nil, fmt.Errorf("error getting context flag: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("unable to load islectl config. Have you ran `islectl config set-context`?")
	}

	if c == "default" {
		c = cfg.CurrentContext
	}
	for _, context := range cfg.Contexts {
		if context.Name == c {
			return &context, nil
		}
	}

	return nil, fmt.Errorf("unable to set current context. Have you ran `islectl config use-context`?")
}

func GetInput(question ...string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	lastItemIndex := len(question) - 1
	for i := range question {
		if i == lastItemIndex {
			fmt.Print(question[i])
			continue
		}
		fmt.Println(question[i])
	}
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("unable to readon from stdin: %v", err)
	}
	input = strings.TrimSpace(input)
	fmt.Println()
	return input, nil
}

func LoadFromFlags(f *pflag.FlagSet) (*Context, error) {
	t := reflect.TypeOf(Context{})
	m := make(map[string]interface{}, t.NumField())
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" || tag == "name" {
			continue
		}
		tag = strings.Split(tag, ",")[0]
		var value interface{}
		switch field.Type.Kind() {
		case reflect.Bool:
			v, err := f.GetBool(tag)
			if err != nil {
				return nil, fmt.Errorf("error getting flag %q: %w", tag, err)
			}
			value = v

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v, err := f.GetUint(tag)
			if err != nil {
				return nil, fmt.Errorf("error getting flag %q: %w", tag, err)
			}
			value = v
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.String {
				v, err := f.GetStringSlice(tag)
				if err != nil {
					return nil, fmt.Errorf("error getting string slice flag %q: %w", tag, err)
				}
				value = v
			}
		default:
			v, err := f.GetString(tag)
			if err != nil {
				return nil, fmt.Errorf("error getting flag %q: %w", tag, err)
			}
			value = v
		}

		m[tag] = value
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	var cc Context
	if err := yaml.Unmarshal(data, &cc); err != nil {
		return nil, err
	}

	return &cc, nil
}

// for local contexts, try a bunch of common paths grab the docker socket
// this is mostly needed for Mac OS
func GetDefaultLocalDockerSocket(dockerSocket string) string {
	macOsSocket := filepath.Join(os.Getenv("HOME"), ".docker/run/docker.sock")
	if isDockerSocketAlive(macOsSocket) {
		return macOsSocket
	}

	tried := []string{macOsSocket}
	if isDockerSocketAlive(dockerSocket) {
		return strings.TrimPrefix(dockerSocket, "unix://")
	}

	dockerSocket = os.Getenv("DOCKER_HOST")
	if isDockerSocketAlive(dockerSocket) {
		return strings.TrimPrefix(dockerSocket, "unix://")
	}

	tried = append(tried, dockerSocket)
	slog.Error("Unable to determine docker socket from any common paths", "testedSockets", tried)
	return ""
}

func isDockerSocketAlive(socket string) bool {
	socket = strings.TrimPrefix(socket, "unix://")
	conn, err := net.DialTimeout("unix", socket, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (c *Context) ReadSmallFile(filename string) string {
	if c.DockerHostType == ContextLocal {
		data, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("Error reading file", "file", filename, "err", err)
			return ""
		}

		return string(data)
	}
	client, err := c.DialSSH()
	if err != nil {
		slog.Error("Error establishing SSH connection", "err", err)
		return ""
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		slog.Error("Error creating SSH session", "err", err)
		return ""
	}
	defer session.Close()

	// Run "cat" on the remote host to read the file.
	output, err := session.Output(fmt.Sprintf("cat %s", filename))
	if err != nil {
		slog.Error("Error reading remote file", "file", filename, "err", err)
		return ""
	}

	return string(output)
}

func (c *Context) DialSSH() (*ssh.Client, error) {
	key, err := os.ReadFile(c.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading SSH key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("error parsing SSH key: %w", err)
	}

	knownHostsPath := filepath.Join(filepath.Dir(c.SSHKeyPath), "known_hosts")
	slog.Debug("Setting known_hosts", "known_hosts", knownHostsPath)
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("error creating known_hosts callback: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: c.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         5 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", c.SSHHostname, c.SSHPort)
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {

		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			if len(keyErr.Want) == 0 {
				fmt.Println("The host key for your remote context is not known.")
				fmt.Println("This means your SSH known_hosts file doesn't have an entry for this host.")
			} else {
				fmt.Println("The host key for your remote context does not match the expected key.")
				fmt.Println("This might indicate that the host's key has changed or that there could be a security issue.")
				fmt.Println("Please verify the new key with your host administrator.")
				fmt.Println("If the change is legitimate, update your known_hosts file by removing the old key and adding the new one.")
			}
			fmt.Printf("\nTry running `ssh -p %d -t %s@%s` and trying again\n\n", c.SSHPort, c.SSHUser, c.SSHHostname)

		}
		return nil, fmt.Errorf("error dialing SSH at %s: %w", sshAddr, err)
	}

	return client, nil
}

func (c *Context) ProjectDirExists() (bool, error) {
	if c.DockerHostType == ContextLocal {
		_, err := os.Stat(c.ProjectDir)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return true, err
		}

		return !os.IsNotExist(err), nil
	}

	client, err := c.DialSSH()
	if err != nil {
		slog.Error("Error establishing SSH connection", "err", err)
		return false, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		slog.Error("Error creating SSH session", "err", err)
		return false, err
	}
	defer session.Close()

	_, err = session.Output(fmt.Sprintf("test -e %s", c.ProjectDir))
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (cc *Context) VerifyRemoteInput() {
	if cc.SSHHostname == "islandora.dev" {
		question := []string{
			"You should not be setting SSH hostname to islandora.dev",
			"You may have forgot to pass --ssh-hostname",
			"Instead you can enter the remote server domain name here: ",
		}
		h, err := GetInput(question...)
		if err != nil || h == "" {
			slog.Error("Error reading input")
			os.Exit(1)
		}
		cc.SSHHostname = h
	}

	if cc.SSHUser == "nginx" {
		u, err := user.Current()
		if err != nil {
			slog.Error("Unable to determine current user", "err", err)
			os.Exit(1)
		}
		cc.SSHUser = u.Username
		slog.Warn("You may need to pass --ssh-user for the remote host.")
		slog.Warn("Defaulting to your username to connect to the remote host", "name", u.Username)
	}

	if cc.SSHPort == 2222 {
		question := []string{
			"You may have forgot to pass --ssh-port",
			"The default value is 2222, which is a good default for local contexts",
			"You can enter the port to connect to [2222]: ",
		}
		p, err := GetInput(question...)
		if err != nil {
			slog.Error("Error reading input")
			os.Exit(1)
		}
		if p != "" {
			port, err := strconv.Atoi(p)
			if err != nil {
				slog.Error("Unable to convert input to int")
				os.Exit(1)

			}
			cc.SSHPort = uint(port)
		}
	}

	if cc.Profile == "dev" {
		question := []string{
			"Are you sure you want \"dev\" for the docker compose profile on the remote context?",
			"Enter the profile here, enter nothing to keep dev: [dev]: ",
		}
		p, err := GetInput(question...)
		if err != nil {
			slog.Error("Error reading input")
			os.Exit(1)
		}
		if p != "" {
			slog.Info("Setting profile", "profile", p)
			cc.Profile = p
		}
	}
}

func (c *Context) UploadFile(source, destination string) error {
	client, err := c.DialSSH()
	if err != nil {
		slog.Error("Error establishing SSH connection", "err", err)
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		log.Fatal(err)
	}
	defer sftpClient.Close()

	localFile, err := os.Open(source)
	if err != nil {
		log.Fatal(err)
	}
	defer localFile.Close()

	remoteFile, err := sftpClient.Create(destination)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	_, err = remoteFile.ReadFrom(localFile)
	if err != nil {
		return err
	}

	return nil
}
