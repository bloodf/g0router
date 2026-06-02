package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/spf13/cobra"
)

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
	cmd.AddCommand(newAuthLoginCommand("login"))
	cmd.AddCommand(newAuthLogoutCommand(dataDir))
	return cmd
}

func newLoginCommand() *cobra.Command {
	cmd := newAuthLoginCommand("login")
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

func newAuthLoginCommand(use string) *cobra.Command {
	var device bool
	var key bool

	cmd := &cobra.Command{
		Use:   use + " <provider>",
		Short: "Start provider authentication",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if device && key {
				return fmt.Errorf("choose either --device or --key")
			}
			if key {
				fmt.Fprintf(cmd.OutOrStdout(), "API key login for %s: run g0router keys add <name> or add credentials in the web UI.\n", args[0])
				return nil
			}

			flow, err := newOAuthFlow(args[0])
			if err != nil {
				return err
			}

			session, err := flow.Start(context.Background())
			if err != nil {
				return fmt.Errorf("start oauth flow: %w", err)
			}

			printAuthSession(cmd, session)
			return nil
		},
	}
	cmd.Flags().BoolVar(&device, "device", false, "start device authorization flow")
	cmd.Flags().BoolVar(&key, "key", false, "start API key credential flow")
	return cmd
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

			connections, err := s.GetConnections(args[0])
			if err != nil {
				return fmt.Errorf("list provider connections: %w", err)
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
	names := make([]string, 0, len(oauthFlowFactories()))
	for name := range oauthFlowFactories() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func newOAuthFlow(provider string) (oauth.Flow, error) {
	factory, ok := oauthFlowFactories()[strings.ToLower(provider)]
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
		"github":      func() oauth.Flow { return oauth.NewGitHubCopilotFlow(oauth.GitHubCopilotFlowConfig{}) },
		"gitlab":      func() oauth.Flow { return oauth.NewGitLabFlow(oauth.GitLabConfig{}) },
		"kimi":        func() oauth.Flow { return oauth.NewKimiFlow(oauth.KimiFlowConfig{}) },
		"kiro":        func() oauth.Flow { return oauth.NewKiroFlow(oauth.KiroConfig{}) },
		"minimax":     func() oauth.Flow { return oauth.NewMiniMaxFlow() },
		"xai":         func() oauth.Flow { return oauth.NewXAIFlow(oauth.XAIConfig{}) },
		"xiaomi":      func() oauth.Flow { return oauth.NewXiaomiFlow(oauth.XiaomiConfig{}) },
		"zhipu":       func() oauth.Flow { return oauth.NewZhipuFlow() },
	}
}
