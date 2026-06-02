package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the g0router CLI command tree.
func NewRootCommand(version string) *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:           "g0router",
		Short:         "LLM gateway and provider router",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintf(cmd.OutOrStdout(), "g0router %s\n", version)
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&showVersion, "version", false, "print version and exit")
	cmd.AddCommand(NewAuthCommand())
	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(NewInstallCommand())
	cmd.AddCommand(newServeCommand())

	return cmd
}

func newServeCommand() *cobra.Command {
	var port int
	var dataDir string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStderr(), "g0router serve is not yet implemented (port %d, data dir %s)\n", port, dataDir)
			return fmt.Errorf("serve not yet implemented")
		},
	}
	cmd.Flags().IntVar(&port, "port", 20128, "HTTP port")
	cmd.Flags().StringVar(&dataDir, "data-dir", "~/.g0router", "data directory")
	return cmd
}
