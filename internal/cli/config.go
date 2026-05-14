package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View or change configuration values",
	}
	cmd.AddCommand(configShowCmd(), configSetCmd(), configPathCmd())
	return cmd
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(),
				"poll_interval_seconds = %d\nwarn_threshold        = %d\nalert_threshold       = %d\nnotify                = %v\norg_id                = %q\n",
				c.PollIntervalSeconds, c.WarnThreshold, c.AlertThreshold, c.Notify, c.OrgID,
			)
			return nil
		},
	}
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value (poll_interval_seconds, warn_threshold, alert_threshold, notify, org_id)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return err
			}
			key, val := args[0], args[1]
			switch key {
			case "poll_interval_seconds":
				n, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("%s must be an integer", key)
				}
				c.PollIntervalSeconds = n
			case "warn_threshold":
				n, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("%s must be an integer", key)
				}
				c.WarnThreshold = n
			case "alert_threshold":
				n, err := strconv.Atoi(val)
				if err != nil {
					return fmt.Errorf("%s must be an integer", key)
				}
				c.AlertThreshold = n
			case "notify":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("%s must be true or false", key)
				}
				c.Notify = b
			case "org_id":
				c.OrgID = val
			default:
				return fmt.Errorf("unknown key %q", key)
			}
			return config.Save(c)
		},
	}
}
