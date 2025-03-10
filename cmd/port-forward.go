package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/islandora-devops/islectl/pkg/config"
	"github.com/islandora-devops/islectl/pkg/isle"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var portForwardCmd = &cobra.Command{
	Use:   "port-forward [LOCAL-PORT:SERVICE:REMOTE-PORT...]",
	Args:  cobra.ArbitraryArgs,
	Short: "Forward one or more local ports to a service",
	Long: `
Access remote context docker service ports.

For docker services running in remote contexts that do not have ports exposed on the host VM, accessing those services can be tricky.
The islectl port-forward command can help in these situations.

As an example, from a local machine, accessing your stage context's traefik dashboard and solr admin UI
could be done by running this command in the terminal:

islectl port-forward \
  8983:solr:8983 \
  8080:traefik:8080 \
  8161:activemq:8161 \
  --context stage

Then, while leaving the terminal open, in your web browser you can visit

http://localhost:8983/solr to see the solr admin UI
http://localhost:8080/dashboard to see the traefik dashboard (assuming it's enabled in your config)
http://localhost:8161/admin/queues.jsp to see ActiveMQ queues

Be sure to run Ctrl+c in your terminal when you are done to close the connection.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := cmd.Flags()
		c, err := config.CurrentContext(f)
		if err != nil {
			return err
		}
		if runtime.GOOS != "linux" && c.DockerHostType == config.ContextLocal {
			return fmt.Errorf("port-forwarding on non-linux local contexts is not currently supported")
		}
		cli, err := isle.GetDockerCli(c)
		if err != nil {
			return err
		}
		defer cli.Close()

		listeners := make([]net.Listener, 0, len(args))
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGTERM)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for _, arg := range args {
			parts := strings.Split(arg, ":")
			if len(parts) != 3 {
				return fmt.Errorf("invalid port forwarding spec '%s': expected format LOCAL-PORT:SERVICE:REMOTE-PORT", arg)
			}
			localPortStr, service, remotePortStr := parts[0], parts[1], parts[2]

			localPort, err := strconv.Atoi(localPortStr)
			if err != nil {
				return fmt.Errorf("invalid local port '%s': must be an integer", localPortStr)
			}
			remotePort, err := strconv.Atoi(remotePortStr)
			if err != nil {
				return fmt.Errorf("invalid remote port '%s': must be an integer", remotePortStr)
			}

			addr := fmt.Sprintf("localhost:%d", localPort)
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("local port %d appears to be in use: %v", localPort, err)
			}
			listeners = append(listeners, listener)

			containerName, err := cli.GetContainerName(c, service, false)
			if err != nil {
				return err
			}
			serviceIp, err := cli.GetServiceIp(ctx, c, containerName)
			if err != nil {
				return err
			}

			remoteEndpoint := fmt.Sprintf("%s:%d", serviceIp, remotePort)
			go func(listener net.Listener, lp, remoteAddr string) {
				defer listener.Close()
				fmt.Printf("Forwarding localhost:%s -> %s via SSH\n", lp, remoteAddr)
				for {
					localConn, err := listener.Accept()
					if err != nil {
						if strings.Contains(err.Error(), "use of closed network connection") {
							return
						}
						fmt.Fprintf(os.Stderr, "error accepting connection on port %s: %v\n", lp, err)
						return
					}
					go forward(cli.SshCli, localConn, remoteAddr)
				}
			}(listener, localPortStr, remoteEndpoint)
		}

		<-done
		fmt.Println("Shutting down port forwards...")
		for _, listener := range listeners {
			listener.Close()
		}
		return nil
	},
}

func forward(client *ssh.Client, localConn net.Conn, remoteAddr string) {
	defer localConn.Close()
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dial remote address %s: %v\n", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	go func() {
		if _, err := io.Copy(remoteConn, localConn); err != nil {
			fmt.Fprintf(os.Stderr, "error while copying local to remote: %v\n", err)
		}
	}()
	if _, err := io.Copy(localConn, remoteConn); err != nil {
		fmt.Fprintf(os.Stderr, "error while copying remote to local: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(portForwardCmd)
}
