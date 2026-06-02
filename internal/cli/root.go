package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/store"
	"github.com/spf13/cobra"
)

type rootConfig struct {
	Version string
	Serve   serveRunner
}

type serveConfig struct {
	Port    int
	DataDir string
	Version string
}

type serveRunner func(context.Context, serveConfig) error

// NewRootCommand builds the g0router CLI command tree.
func NewRootCommand(version string) *cobra.Command {
	return newRootCommand(rootConfig{
		Version: version,
		Serve:   runServer,
	})
}

func newRootCommand(config rootConfig) *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:           "g0router",
		Short:         "LLM gateway and provider router",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintf(cmd.OutOrStdout(), "g0router %s\n", config.Version)
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&showVersion, "version", false, "print version and exit")
	cmd.AddCommand(NewAuthCommand())
	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(NewInstallCommand())
	cmd.AddCommand(newServeCommand(config.Version, config.Serve))

	return cmd
}

func newServeCommand(version string, serve serveRunner) *cobra.Command {
	var port int
	var dataDir string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serve == nil {
				return fmt.Errorf("serve runner unavailable")
			}
			return serve(cmd.Context(), serveConfig{
				Port:    port,
				DataDir: dataDir,
				Version: version,
			})
		},
	}
	cmd.Flags().IntVar(&port, "port", 20128, "HTTP port")
	cmd.Flags().StringVar(&dataDir, "data-dir", "~/.g0router", "data directory")
	return cmd
}

func runServer(ctx context.Context, config serveConfig) error {
	dataDir, err := expandServeDataDir(config.DataDir)
	if err != nil {
		return err
	}
	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer s.Close()

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(config.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", config.Port, err)
	}

	server := api.NewServer(api.ServerConfig{
		Port:       config.Port,
		Version:    config.Version,
		Store:      s,
		UsageStore: s,
	})

	go func() {
		<-ctx.Done()
		_ = server.Stop()
	}()

	if err := server.Serve(ln); err != nil {
		return fmt.Errorf("run server: %w", err)
	}
	return nil
}

func expandServeDataDir(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home dir: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
