package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/auth"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored session cookie from the OS keychain",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.DeleteCookie(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}
