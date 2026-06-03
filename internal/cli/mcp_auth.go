package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/spf13/cobra"
)

func newMCPCommand(dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP instances",
	}
	cmd.AddCommand(newMCPAddCommand(dataDir))
	cmd.AddCommand(newMCPListCommand(dataDir))
	cmd.AddCommand(newMCPRemoveCommand(dataDir))
	cmd.AddCommand(newMCPAccountsCommand(dataDir))
	cmd.AddCommand(newMCPToolsCommand(dataDir))
	cmd.AddCommand(newMCPAuthCommand(dataDir))
	return cmd
}

func newMCPAuthCommand(dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage MCP authentication",
	}
	cmd.AddCommand(newMCPAuthStartCommand(dataDir))
	cmd.AddCommand(newMCPAuthCompleteCommand(dataDir))
	return cmd
}

func newMCPAddCommand(dataDir *string) *cobra.Command {
	var serverKey, launchType, transport, command, urlValue, accountLabel, cwd string
	var args, headers, env []string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add an MCP instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, values []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()

			instance := &store.MCPInstance{
				Name:       values[0],
				ServerKey:  serverKey,
				LaunchType: mcp.LaunchType(launchType),
				Transport:  mcp.Transport(transport),
				Args:       args,
				Headers:    parseAssignments(headers),
				Env:        parseAssignments(env),
				IsActive:   true,
			}
			if command != "" {
				instance.Command = &command
			}
			if urlValue != "" {
				instance.URL = &urlValue
			}
			if accountLabel != "" {
				instance.AccountLabel = &accountLabel
			}
			if cwd != "" {
				instance.CWD = &cwd
			}
			if err := s.CreateMCPInstance(instance); err != nil {
				return fmt.Errorf("add mcp instance: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "added mcp instance %s\n", instance.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverKey, "server-key", "", "MCP server key")
	cmd.Flags().StringVar(&launchType, "launch-type", "http", "launch type")
	cmd.Flags().StringVar(&transport, "transport", "streamable-http", "transport")
	cmd.Flags().StringVar(&command, "command", "", "command to run")
	cmd.Flags().StringArrayVar(&args, "arg", nil, "command argument")
	cmd.Flags().StringVar(&urlValue, "url", "", "MCP URL")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP header key=value")
	cmd.Flags().StringArrayVar(&env, "env", nil, "environment key=value")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory")
	cmd.Flags().StringVar(&accountLabel, "account-label", "", "account label")
	return cmd
}

func newMCPListCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List MCP instances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()
			instances, err := s.ListMCPInstances()
			if err != nil {
				return fmt.Errorf("list mcp instances: %w", err)
			}
			for _, instance := range instances {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", instance.Name, instance.ServerKey, instance.LaunchType, stringValue(instance.AccountLabel), instance.HealthStatus)
			}
			return nil
		},
	}
}

func newMCPRemoveCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <instance>",
		Short: "Remove an MCP instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()
			instance, err := findMCPInstanceByName(s, args[0])
			if err != nil {
				return err
			}
			if err := s.DeleteMCPInstance(instance.ID); err != nil {
				return fmt.Errorf("remove mcp instance: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed mcp instance %s\n", instance.Name)
			return nil
		},
	}
}

func newMCPAccountsCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "accounts <instance>",
		Short: "List MCP accounts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()
			instance, err := findMCPInstanceByName(s, args[0])
			if err != nil {
				return err
			}
			accounts, err := s.ListMCPOAuthAccounts(instance.ID)
			if err != nil {
				return fmt.Errorf("list mcp accounts: %w", err)
			}
			for _, account := range accounts {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", account.AccountLabel, account.ResourceURI, account.ExpiresAt.Format(time.RFC3339))
			}
			return nil
		},
	}
}

func newMCPToolsCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "tools <instance>",
		Short: "List MCP tools",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()
			instance, err := findMCPInstanceByName(s, args[0])
			if err != nil {
				return err
			}
			if instance.ToolManifest == nil {
				return nil
			}
			manifest, err := mcp.BuildCompactManifest(*instance.ToolManifest)
			if err != nil {
				return fmt.Errorf("build compact mcp manifest: %w", err)
			}
			for _, tool := range manifest.Tools {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", tool.Function.Name, tool.Function.Description)
			}
			return nil
		},
	}
}

func newMCPAuthStartCommand(dataDir *string) *cobra.Command {
	var authorizationURL, resourceURI, redirectURI string
	cmd := &cobra.Command{
		Use:   "start <instance>",
		Short: "Start MCP OAuth",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()
			instance, err := findMCPInstanceByName(s, args[0])
			if err != nil {
				return err
			}
			flow, err := mcp.BuildOAuthStartFlow(mcp.OAuthStartConfig{
				InstanceID:        instance.ID,
				AuthorizationURL:  authorizationURL,
				RedirectURI:       redirectURI,
				ResourceURI:       resourceURI,
				ExpirationSeconds: int((10 * time.Minute).Seconds()),
			})
			if err != nil {
				return err
			}
			if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
				InstanceID:         instance.ID,
				State:              flow.State,
				CodeVerifierSecret: flow.CodeVerifierSecret,
				RedirectURI:        flow.RedirectURI,
				AuthorizationURL:   flow.AuthorizationURL,
				ResourceURI:        flow.ResourceURI,
				ExpiresAt:          flow.ExpiresAt,
			}); err != nil {
				return fmt.Errorf("create mcp oauth flow: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), flow.AuthorizationURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&authorizationURL, "authorization-url", "", "authorization URL")
	cmd.Flags().StringVar(&resourceURI, "resource", "", "resource URI")
	cmd.Flags().StringVar(&redirectURI, "redirect-url", "", "redirect URL")
	return cmd
}

func newMCPAuthCompleteCommand(dataDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "complete <instance> <callback-url>",
		Short: "Complete MCP OAuth with a pasted callback URL",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMCPCallbackURL(args[1]); err != nil {
				return err
			}
			s, err := openCLIStore(*dataDir)
			if err != nil {
				return err
			}
			defer s.Close()

			instance, err := findMCPInstanceByName(s, args[0])
			if err != nil {
				return err
			}
			engine := mcp.NewOAuthEngine(s, nil)
			account, err := engine.CompleteCallback(context.Background(), instance.ID, args[1])
			if err != nil {
				return fmt.Errorf("complete mcp auth: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "completed mcp auth for %s account %s\n", instance.Name, account.AccountLabel)
			return nil
		},
	}
}

func findMCPInstanceByName(s *store.Store, name string) (*store.MCPInstance, error) {
	instances, err := s.ListMCPInstances()
	if err != nil {
		return nil, fmt.Errorf("list mcp instances: %w", err)
	}
	for _, instance := range instances {
		if instance.Name == name {
			return instance, nil
		}
	}
	return nil, fmt.Errorf("mcp instance %q not found", name)
}

func validateMCPCallbackURL(callbackURL string) error {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("parse callback url: %w", err)
	}
	if parsed.Query().Get("code") == "" {
		return fmt.Errorf("code is required")
	}
	if parsed.Query().Get("state") == "" {
		return fmt.Errorf("state is required")
	}
	return nil
}

func parseAssignments(values []string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	parsed := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok {
			parsed[value] = ""
			continue
		}
		parsed[key] = val
	}
	return parsed
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
