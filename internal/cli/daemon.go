package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/config"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Background tray icon (start/stop/status)",
	}
	cmd.AddCommand(daemonStartCmd(), daemonStopCmd(), daemonStatusCmd())
	return cmd
}

// pidFilePath returns the path to the PID file used to track a running
// daemon. Lives alongside config so it survives across shells.
func pidFilePath() (string, error) {
	cfgPath, err := config.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfgPath), "daemon.pid"), nil
}

func daemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Launch the tray icon as a detached background process",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !trayAvailable() {
				return errNoTray
			}
			pidPath, err := pidFilePath()
			if err != nil {
				return err
			}
			if pid, alive := readPID(pidPath); alive {
				return fmt.Errorf("daemon already running (pid %d)", pid)
			}

			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate executable: %w", err)
			}
			c := exec.Command(exe, "tray")
			// Detach from this terminal so closing the shell doesn't
			// kill the tray. Platform-specific (see daemon_*.go).
			detach(c)
			c.Stdout = nil
			c.Stderr = nil
			c.Stdin = nil
			if err := c.Start(); err != nil {
				return fmt.Errorf("spawn tray: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(pidPath, []byte(strconv.Itoa(c.Process.Pid)), 0o644); err != nil {
				return fmt.Errorf("write pid file: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Started daemon (pid %d).\n", c.Process.Pid)
			// Detach: don't wait.
			go func() { _ = c.Process.Release() }()
			return nil
		},
	}
}

func daemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running tray daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			pidPath, err := pidFilePath()
			if err != nil {
				return err
			}
			pid, alive := readPID(pidPath)
			if !alive {
				_ = os.Remove(pidPath)
				return errors.New("daemon not running")
			}
			p, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			if err := terminate(p); err != nil {
				return fmt.Errorf("signal pid %d: %w", pid, err)
			}
			// Give it up to 2s to exit cleanly.
			for i := 0; i < 20; i++ {
				time.Sleep(100 * time.Millisecond)
				if _, alive := readPID(pidPath); !alive {
					break
				}
			}
			_ = os.Remove(pidPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Stopped daemon (pid %d).\n", pid)
			return nil
		},
	}
}

func daemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether the tray daemon is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			pidPath, err := pidFilePath()
			if err != nil {
				return err
			}
			pid, alive := readPID(pidPath)
			if !alive {
				fmt.Fprintln(cmd.OutOrStdout(), "not running")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "running (pid %d)\n", pid)
			return nil
		},
	}
}

// readPID returns (pid, true) if the PID file exists and the named
// process is alive; otherwise (0, false).
func readPID(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil || pid <= 0 {
		return 0, false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	if !alive(p) {
		return 0, false
	}
	return pid, true
}
