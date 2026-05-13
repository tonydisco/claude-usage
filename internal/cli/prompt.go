package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPromptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Compact one-line output for embedding in PS1 (e.g. [51%/13%])",
		Long: `Compact output suitable for shell prompts.

Designed to be fast and silent on failure: if no credential is configured or
the network is unreachable, prints nothing and exits 0 so it never breaks
your prompt.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mock, _ := cmd.Flags().GetBool("mock")
			noColor, _ := cmd.Flags().GetBool("no-color")

			u, cfg, err := snapshot(cmd.Context(), mock)
			if err != nil {
				// Prompt commands must never break the user's shell. Stay silent.
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), renderPrompt(u, cfg.WarnThreshold, cfg.AlertThreshold, !noColor))
			return nil
		},
	}
	cmd.Flags().Bool("no-color", false, "disable ANSI color escapes")
	return cmd
}
