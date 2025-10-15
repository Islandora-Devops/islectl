package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v3"
)

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

func LoadFromFlags(f *pflag.FlagSet, context Context) (*Context, error) {
	t := reflect.TypeOf(Context{})
	exists := context.DockerSocket != ""
	slog.Debug("Loading context from flags", "exists", exists)
	m := make(map[string]interface{}, t.NumField())
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" || tag == "name" || tag == "-" {
			continue
		}
		tag = strings.Split(tag, ",")[0]

		// Skip map types as they're not supported as flags
		if field.Type.Kind() == reflect.Map {
			continue
		}

		// if we're loading flags for an existing context
		// do not add default values
		if exists && !f.Changed(tag) {
			continue
		}

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

		slog.Debug("Setting tag", "tag", tag, "value", value)
		m[tag] = value
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	cc := context
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

func SetCommandFlags(flags *pflag.FlagSet) {
	path, err := os.Getwd()
	if err != nil {
		slog.Error("Unable to get current working directory", "err", err)
		os.Exit(1)
	}
	env := filepath.Join(path, ".env")
	_ = godotenv.Load(env)

	key := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")

	// NB: these flags must match the corresponding config.Context yaml struct tag
	// though we can add additional flags that have no match for additional functionality
	// in the command logic (e.g. default)
	flags.String("docker-socket", "/var/run/docker.sock", "Path to Docker socket")
	flags.String("type", "local", "Type of context: local or remote")
	flags.String("ssh-hostname", "islandora.dev", "Remote contexts DNS name for the host.")
	flags.Uint("ssh-port", 2222, "Port number")
	flags.String("ssh-user", "nginx", "SSH user for remote context")
	flags.String("ssh-key", "", "Path to SSH private key for remote context. e.g. "+key)
	flags.String("project-dir", "", "Path to docker compose project directory")
	flags.String("project-name", "isle-site-template", "Name of the docker compose project")
	flags.String("profile", "dev", "docker compose profile")
	flags.String("site", "default", "drupal multisite")
	flags.Bool("sudo", false, "for remote contexts, run commands as sudo")
	flags.StringSlice("env-file", []string{}, "when running remote docker commands, the --env-file paths to pass to docker compose")
}
