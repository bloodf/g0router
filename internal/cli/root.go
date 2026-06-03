package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/api/handlers"
	appconfig "github.com/bloodf/g0router/internal/config"
	providerinfo "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/spf13/cobra"
)

type rootConfig struct {
	Version string
	Serve   serveRunner
}

type serveConfig struct {
	Port          int
	BindAddress   string
	DataDir       string
	Version       string
	RequireAPIKey bool
	APIKeySecret  string
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
	dataDir := envString("DATA_DIR", "~/.g0router")

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
	cmd.PersistentFlags().StringVar(&dataDir, "data-dir", dataDir, "data directory")
	cmd.AddCommand(newAuthCommand(&dataDir))
	cmd.AddCommand(newLoginCommand(&dataDir))
	cmd.AddCommand(newLogoutCommand(&dataDir))
	cmd.AddCommand(newKeysCommand(&dataDir))
	cmd.AddCommand(newMCPCommand(&dataDir))
	cmd.AddCommand(newProvidersCommand())
	cmd.AddCommand(newStatusCommand(&dataDir))
	cmd.AddCommand(newHealthcheckCommand())
	cmd.AddCommand(newVersionCommand(config.Version))
	cmd.AddCommand(NewInstallCommand())
	cmd.AddCommand(newUninstallCommand())
	cmd.AddCommand(newServeCommand(config.Version, config.Serve, &dataDir))

	return cmd
}

func newServeCommand(version string, serve serveRunner, rootDataDir *string) *cobra.Command {
	port := 20128

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serve == nil {
				return fmt.Errorf("serve runner unavailable")
			}
			restoreEnv := applyServeFlagEnv(cmd, port, *rootDataDir)
			defer restoreEnv()
			loaded, err := appconfig.Load()
			if err != nil {
				return err
			}
			return serve(cmd.Context(), serveConfig{
				Port:          loaded.Port,
				BindAddress:   loaded.BindAddress,
				DataDir:       loaded.DataDir,
				Version:       version,
				RequireAPIKey: loaded.RequireAPIKey,
				APIKeySecret:  loaded.APIKeySecret,
			})
		},
	}
	cmd.Flags().IntVar(&port, "port", port, "HTTP port")
	return cmd
}

func applyServeFlagEnv(cmd *cobra.Command, port int, dataDir string) func() {
	values := make(map[string]string)
	if commandFlagChanged(cmd, "port") {
		values["PORT"] = strconv.Itoa(port)
	}
	if commandFlagChanged(cmd, "data-dir") {
		values["DATA_DIR"] = dataDir
	}
	return setTemporaryEnv(values)
}

func commandFlagChanged(cmd *cobra.Command, name string) bool {
	flag := cmd.Flag(name)
	return flag != nil && flag.Changed
}

func setTemporaryEnv(values map[string]string) func() {
	type previousValue struct {
		value string
		ok    bool
	}
	previous := make(map[string]previousValue, len(values))
	for key, value := range values {
		old, ok := os.LookupEnv(key)
		previous[key] = previousValue{value: old, ok: ok}
		_ = os.Setenv(key, value)
	}
	return func() {
		for key, old := range previous {
			if old.ok {
				_ = os.Setenv(key, old.value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	}
}

func envString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
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

	listenAddress := net.JoinHostPort(config.BindAddress, strconv.Itoa(config.Port))
	ln, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", listenAddress, err)
	}

	server := api.NewServer(newServerConfig(config, s))

	go func() {
		<-ctx.Done()
		_ = server.Stop()
	}()

	if err := server.Serve(ln); err != nil {
		return fmt.Errorf("run server: %w", err)
	}
	return nil
}

func newServerConfig(config serveConfig, s *store.Store) api.ServerConfig {
	engine := newDefaultInferenceEngine(s)
	mcpClients, mcpTools := newDefaultMCPRuntime()

	return api.ServerConfig{
		Port:             config.Port,
		Version:          config.Version,
		RequireAPIKey:    config.RequireAPIKey,
		APIKeySecret:     config.APIKeySecret,
		APIKeyValidator:  storeAPIKeyValidator{s: s},
		InferenceEngine:  engine,
		Store:            s,
		ModelSource:      engine,
		OAuthFlows:       defaultOAuthFlows(),
		UsageStore:       s,
		QuotaFetchers:    defaultQuotaFetchers(),
		MCPClientManager: mcpClients,
		MCPToolManager:   mcpTools,
	}
}

type storeAPIKeyValidator struct {
	s *store.Store
}

func (v storeAPIKeyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	_, ok, err := v.s.ValidateAPIKey(key, secret)
	return ok, err
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

func openCLIStore(dataDir string) (*store.Store, error) {
	expanded, err := expandServeDataDir(dataDir)
	if err != nil {
		return nil, err
	}
	s, err := store.NewStore(filepath.Join(expanded, "g0router.db"))
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	return s, nil
}

func localAPIKeySecret() (string, error) {
	if secret := os.Getenv("API_KEY_SECRET"); secret != "" {
		return secret, nil
	}
	return "", fmt.Errorf("API_KEY_SECRET required to create API keys")
}

func newKeysCommand(dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage local API keys",
	}
	cmd.AddCommand(newKeysAddCommand(dataDir))
	cmd.AddCommand(newKeysListCommand(dataDir))
	cmd.AddCommand(newKeysRemoveCommand(dataDir))
	return cmd
}

func newKeysAddCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create a local API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()

			secret, err := localAPIKeySecret()
			if err != nil {
				return err
			}
			key, raw, err := s.CreateAPIKey(args[0], secret)
			if err != nil {
				return fmt.Errorf("create api key: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", key.Name, raw)
			return nil
		},
	}
}

func newKeysListCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List local API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()

			keys, err := s.ListAPIKeys()
			if err != nil {
				return fmt.Errorf("list api keys: %w", err)
			}
			for _, key := range keys {
				if key.IsActive {
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", key.Name, key.Prefix)
				}
			}
			return nil
		},
	}
}

func newKeysRemoveCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:     "rm <name>",
		Aliases: []string{"remove"},
		Short:   "Disable a local API key",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()

			keys, err := s.ListAPIKeys()
			if err != nil {
				return fmt.Errorf("list api keys: %w", err)
			}
			for _, key := range keys {
				if key.Name == args[0] && key.IsActive {
					if err := s.DeleteAPIKey(key.ID); err != nil {
						return fmt.Errorf("remove api key: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", args[0])
					return nil
				}
			}
			return fmt.Errorf("api key not found: %s", args[0])
		},
	}
}

func newProvidersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect providers",
	}
	cmd.AddCommand(newProvidersListCommand())
	cmd.AddCommand(newProvidersTestCommand())
	return cmd
}

func newProvidersListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, provider := range knownProviderNames() {
				fmt.Fprintln(cmd.OutOrStdout(), provider)
			}
			return nil
		},
	}
}

func newProvidersTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test <provider>",
		Short: "Validate provider support",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, provider := range knownProviderNames() {
				if provider == args[0] {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: configured\n", args[0])
					return nil
				}
			}
			return fmt.Errorf("unknown provider: %s", args[0])
		},
	}
}

func newStatusCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local gateway status",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()
			fmt.Fprintln(cmd.OutOrStdout(), "store: ok")
			return nil
		},
	}
}

func newHealthcheckCommand() *cobra.Command {
	var port int
	var url string

	cmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Check local server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := url
			if target == "" {
				target = "http://127.0.0.1:" + strconv.Itoa(port) + "/healthz"
			}
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get(target)
			if err != nil {
				return fmt.Errorf("healthcheck %s: %w", target, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("healthcheck %s: %s", target, resp.Status)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "healthcheck: ok")
			return nil
		},
	}
	cmd.Flags().IntVar(&port, "port", 20128, "HTTP port")
	cmd.Flags().StringVar(&url, "url", "", "healthcheck URL")
	return cmd
}

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "g0router %s\n", version)
			return nil
		},
	}
}

func knownProviderNames() []string {
	entries := providerinfo.PublicInferenceProviders()
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.G0RouterID)
	}
	return names
}

func defaultOAuthFlows() handlers.OAuthFlows {
	flows := make(handlers.OAuthFlows)
	for _, factory := range oauthFlowFactories() {
		flow := factory()
		flows[flow.ProviderID()] = flow
	}
	return flows
}

func defaultQuotaFetchers() map[providers.ModelProvider]usage.QuotaFetcher {
	fetchers := make(map[providers.ModelProvider]usage.QuotaFetcher)
	for _, provider := range knownProviderNames() {
		modelProvider := providers.ModelProvider(provider)
		fetchers[modelProvider] = usage.NewUnsupportedQuotaFetcher(modelProvider)
	}
	return fetchers
}

type staticModelSource struct{}

func (staticModelSource) ListModels(ctx context.Context) ([]providers.Model, error) {
	models := make([]providers.Model, 0, len(knownProviderNames()))
	for _, provider := range knownProviderNames() {
		modelProvider := providers.ModelProvider(provider)
		models = append(models, providers.Model{
			ID:       provider + "-default",
			Object:   "model",
			OwnedBy:  provider,
			Provider: modelProvider,
		})
	}
	return models, nil
}
