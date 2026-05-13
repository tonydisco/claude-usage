package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tonydisco/claude-usage/internal/auth"
	"github.com/tonydisco/claude-usage/internal/config"
	"github.com/tonydisco/claude-usage/internal/fetcher"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Print current Claude.ai plan usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			mock, _ := cmd.Flags().GetBool("mock")
			noColor, _ := cmd.Flags().GetBool("no-color")

			u, cfg, err := snapshot(cmd.Context(), mock)
			if err != nil {
				return err
			}
			color := !noColor && term.IsTerminal(int(os.Stdout.Fd()))
			fmt.Fprintln(cmd.OutOrStdout(), renderStatus(u, cfg.WarnThreshold, cfg.AlertThreshold, color))
			return nil
		},
	}
	cmd.Flags().Bool("no-color", false, "disable ANSI colors even when stdout is a TTY")
	return cmd
}

// snapshot loads config + credential and fetches one Usage value.
// Shared by status, prompt, watch.
//
// When cfg.OrgID is empty and we're not in mock mode, snapshot calls
// /api/organizations once, persists the discovered UUID to config, and
// continues. Subsequent calls skip the lookup.
func snapshot(ctx context.Context, mock bool) (*fetcher.Usage, config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cfg, err
	}

	client := fetcher.New("", cfg.OrgID)
	client.Mock = mock

	if !mock {
		cookie, err := auth.LoadCookie()
		if err != nil {
			return nil, cfg, err
		}
		client.SessionCookie = cookie

		if cfg.OrgID == "" {
			orgID, err := client.FetchOrgID(ctx)
			if err != nil {
				return nil, cfg, fmt.Errorf("auto-detect org_id: %w (set manually with `claude-usage config set org_id <id>`)", err)
			}
			cfg.OrgID = orgID
			client.OrgID = orgID
			if saveErr := config.Save(cfg); saveErr != nil {
				// Non-fatal: detection worked, just won't be cached.
				fmt.Fprintf(os.Stderr, "warning: could not save org_id to config: %v\n", saveErr)
			}
		}
	}

	u, err := client.Fetch(ctx)
	return u, cfg, err
}
