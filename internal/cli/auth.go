package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	providercred "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/spf13/cobra"
)

type oauthFlowFactory func(provider string) (oauth.Flow, error)

// NewAuthCommand builds OAuth credential commands.
func NewAuthCommand() *cobra.Command {
	dataDir := "~/.g0router"
	return newAuthCommand(&dataDir)
}

func newAuthCommand(dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage provider authentication",
	}
	cmd.AddCommand(newAuthListCommand())
	cmd.AddCommand(newAuthLoginCommand("login", dataDir, newOAuthFlow))
	cmd.AddCommand(newAuthLogoutCommand(dataDir))
	return cmd
}

func newLoginCommand(dataDir *string) *cobra.Command {
	cmd := newAuthLoginCommand("login", dataDir, newOAuthFlow)
	cmd.Short = "Start provider authentication"
	return cmd
}

func newLogoutCommand(dataDir *string) *cobra.Command {
	cmd := newAuthLogoutCommand(dataDir)
	cmd.Short = "Remove provider credentials"
	return cmd
}

func newAuthListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List supported auth providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, provider := range supportedProviderNames() {
				fmt.Fprintln(cmd.OutOrStdout(), provider)
			}
			return nil
		},
	}
}

func newAuthLoginCommand(use string, dataDir *string, flowFactory oauthFlowFactory) *cobra.Command {
	var device bool
	var key bool
	var apiKeyValue string
	var connectionName string

	cmd := &cobra.Command{
		Use:   use + " <provider>",
		Short: "Start provider authentication",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if device && key {
				return fmt.Errorf("choose either --device or --key")
			}
			if key {
				if apiKeyValue == "" {
					fmt.Fprintf(cmd.OutOrStdout(), "API key login for %s: rerun with --api-key or add provider credentials in the web UI.\n", args[0])
					return nil
				}
				if err := persistAPIKeyLogin(*dataDir, args[0], connectionName, apiKeyValue); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "stored API key connection for %s\n", providercred.CanonicalProviderID(args[0]))
				return nil
			}

			flow, err := flowFactory(args[0])
			if err != nil {
				return err
			}

			session, err := flow.Start(cmd.Context())
			if err != nil {
				return fmt.Errorf("start oauth flow: %w", err)
			}

			printAuthSession(cmd, session)
			if device {
				status, err := completeDeviceLogin(cmd.Context(), *dataDir, args[0], flow, session)
				if err != nil {
					return err
				}
				if status != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Device login status: %s\n", status)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&device, "device", false, "start device authorization flow")
	cmd.Flags().BoolVar(&key, "key", false, "start API key credential flow")
	cmd.Flags().StringVar(&apiKeyValue, "api-key", "", "provider API key to store with --key")
	cmd.Flags().StringVar(&connectionName, "name", "", "connection name to store with --key")
	return cmd
}

func persistAPIKeyLogin(dataDir, provider, name, apiKeyValue string) error {
	canonicalProvider := providercred.CanonicalProviderID(provider)
	entry, ok := providercred.ProviderMatrix().Provider(canonicalProvider)
	if !ok || !authTypesInclude(entry.AuthTypes, "api_key") {
		return fmt.Errorf("provider %s does not support API-key auth", canonicalProvider)
	}
	if strings.TrimSpace(apiKeyValue) == "" {
		return fmt.Errorf("api key is required")
	}
	if strings.TrimSpace(name) == "" {
		name = canonicalProvider
	}

	s, err := openCLIStore(dataDir)
	if err != nil {
		return err
	}
	defer s.Close()

	conn := &store.Connection{
		Provider: canonicalProvider,
		Name:     name,
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKeyValue,
		IsActive: true,
	}
	if err := s.CreateConnection(conn); err != nil {
		return fmt.Errorf("create api key connection: %w", err)
	}
	return nil
}

func authTypesInclude(authTypes []string, want string) bool {
	for _, authType := range authTypes {
		if authType == want {
			return true
		}
	}
	return false
}

func completeDeviceLogin(ctx context.Context, dataDir string, runtimeProvider string, flow oauth.Flow, session oauth.AuthSession) (oauth.PollStatus, error) {
	if session.UserCode == "" && session.Verification == "" {
		return "", nil
	}
	result, err := flow.Poll(ctx, session)
	if err != nil {
		return "", fmt.Errorf("poll oauth flow: %w", err)
	}
	if result.Status != oauth.PollStatusComplete || result.Token == nil {
		return result.Status, nil
	}

	s, err := openCLIStore(dataDir)
	if err != nil {
		return "", err
	}
	defer s.Close()

	conn := providercred.ConnectionFromOAuthTokenForProvider(*result.Token, "", runtimeProvider)
	if err := s.CreateConnection(conn); err != nil {
		return "", fmt.Errorf("create oauth connection: %w", err)
	}
	return result.Status, nil
}

func newAuthLogoutCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "logout <provider>",
		Short: "Remove provider credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()

			var connections []*store.Connection
			for _, provider := range providercred.ProviderAliases(args[0]) {
				providerConnections, err := s.GetConnections(provider)
				if err != nil {
					return fmt.Errorf("list provider connections: %w", err)
				}
				connections = append(connections, providerConnections...)
			}
			for _, conn := range connections {
				if err := s.DeleteConnection(conn.ID); err != nil {
					return fmt.Errorf("delete connection %s: %w", conn.ID, err)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %d connection(s) for %s\n", len(connections), args[0])
			return nil
		},
	}
}

func printAuthSession(cmd *cobra.Command, session oauth.AuthSession) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Provider: %s\n", session.Provider)
	if session.AuthURL != "" {
		fmt.Fprintf(out, "Open this URL: %s\n", session.AuthURL)
	}
	if session.UserCode != "" {
		fmt.Fprintf(out, "User code: %s\n", session.UserCode)
	}
	if session.Verification != "" {
		fmt.Fprintf(out, "Verification URL: %s\n", session.Verification)
	}
	if session.SessionID != "" {
		fmt.Fprintf(out, "Session ID: %s\n", session.SessionID)
	}
	fmt.Fprintln(out, "Paste the resulting code with the callback flow or finish in the web UI.")
}

func supportedProviderNames() []string {
	seen := make(map[string]bool, len(oauthFlowFactories()))
	for _, factory := range oauthFlowFactories() {
		seen[factory().ProviderID().String()] = true
	}
	for _, entry := range providercred.ProviderMatrix().Entries() {
		if len(entry.AuthTypes) > 0 {
			seen[entry.G0RouterID] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func newOAuthFlow(provider string) (oauth.Flow, error) {
	canonical := oauth.CanonicalFlowProviderID(oauth.ProviderID(provider))
	factory, ok := oauthFlowFactories()[canonical.String()]
	if !ok {
		return nil, fmt.Errorf("unknown oauth provider: %s", provider)
	}
	return factory(), nil
}

func oauthFlowFactories() map[string]func() oauth.Flow {
	return map[string]func() oauth.Flow{
		"alibaba":     func() oauth.Flow { return oauth.NewAlibabaFlow() },
		"anthropic":   func() oauth.Flow { return oauth.NewAnthropicFlow() },
		"antigravity": func() oauth.Flow { return oauth.NewAntigravityFlow(oauth.AntigravityConfig{}) },
		"codex":       func() oauth.Flow { return oauth.NewCodexFlow(oauth.CodexFlowConfig{}) },
		"cursor":      func() oauth.Flow { return oauth.NewCursorFlow() },
		"deepseek":    func() oauth.Flow { return oauth.NewDeepSeekFlow(oauth.DeepSeekConfig{}) },
		"gemini":      func() oauth.Flow { return oauth.NewGeminiFlow(oauth.GeminiConfig{}) },
		"github-copilot": func() oauth.Flow {
			return oauth.NewGitHubCopilotFlow(oauth.GitHubCopilotFlowConfig{})
		},
		"gitlab-duo": func() oauth.Flow {
			return oauth.NewGitLabFlow(oauth.GitLabConfig{})
		},
		"kimi":    func() oauth.Flow { return oauth.NewKimiFlow(oauth.KimiFlowConfig{}) },
		"kiro":    func() oauth.Flow { return oauth.NewKiroFlow(oauth.KiroConfig{}) },
		"minimax": func() oauth.Flow { return oauth.NewMiniMaxFlow() },
		"qianfan": func() oauth.Flow {
			return oauth.NewQianfanFlow()
		},
		"xai":    func() oauth.Flow { return oauth.NewXAIFlow(oauth.XAIConfig{}) },
		"xiaomi": func() oauth.Flow { return oauth.NewXiaomiFlow(oauth.XiaomiConfig{}) },
		"zhipu":  func() oauth.Flow { return oauth.NewZhipuFlow() },
	}
}
