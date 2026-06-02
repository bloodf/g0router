package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewInstallCommand builds the systemd install command.
func NewInstallCommand() *cobra.Command {
	var user bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Print systemd installation targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if user {
				fmt.Fprintln(out, "Install plan: user service")
				fmt.Fprintln(out, "Unit: ~/.config/systemd/user/g0router.service")
				fmt.Fprintln(out, "Data: ~/.g0router")
				return nil
			}

			fmt.Fprintln(out, "Install plan: system service")
			fmt.Fprintln(out, "Unit: /etc/systemd/system/g0router.service")
			fmt.Fprintln(out, "Data: /var/lib/g0router")
			return nil
		},
	}
	cmd.Flags().BoolVar(&user, "user", false, "install as a user systemd service")
	return cmd
}

func newUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Print systemd removal targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Remove systemd service")
			fmt.Fprintln(out, "Data: keeps data")
			return nil
		},
	}
}
