// Package cli wires up the cobra command tree.
package cli

import (
	"github.com/spf13/cobra"
)

// Execute parses argv and runs the appropriate subcommand.
func Execute(version string) error {
	root := &cobra.Command{
		Use:           "claude-usage",
		Short:         "Show how much of your Claude.ai plan you've used",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().Bool("mock", false, "read from samples/usage-response.json instead of claude.ai (for development)")

	root.AddCommand(
		newStatusCmd(),
		newPromptCmd(),
		newLoginCmd(),
		newLogoutCmd(),
		newConfigCmd(),
	)
	return root.Execute()
}
