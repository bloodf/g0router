package cli

import (
	"context"
	"fmt"
	"net/url"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/spf13/cobra"
)

func newMCPCommand(dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP instances",
	}
	cmd.AddCommand(newMCPAuthCommand(dataDir))
	return cmd
}

func newMCPAuthCommand(dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage MCP authentication",
	}
	cmd.AddCommand(newMCPAuthCompleteCommand(dataDir))
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
