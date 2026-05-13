//go:build cgo

package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/gen2brain/beeep"
	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/config"
	"github.com/tonydisco/claude-usage/internal/fetcher"
)

const dashboardURL = "https://claude.ai/settings/usage"

func newTrayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tray",
		Short: "Run as a system tray icon (foreground; use `daemon start` to detach)",
		Long: `Run a small menu-bar icon that summarizes Claude.ai plan usage and
auto-refreshes on the configured poll interval.

Clicking the icon opens a battery-style detail panel with one line
per bucket (Session / Weekly / Sonnet / Design), the next reset
time, and shortcuts to refresh or open the dashboard. A desktop
notification fires once per bucket when warn_threshold or
alert_threshold is crossed.

Quit from the tray menu or with Ctrl-C.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mock, _ := cmd.Flags().GetBool("mock")
			t := &trayApp{mock: mock, ctx: cmd.Context()}
			systray.Run(t.onReady, t.onExit)
			return nil
		},
	}
}

type trayApp struct {
	ctx  context.Context
	mock bool

	mu       sync.Mutex
	cfg      config.Config
	notified map[string]int // bucket -> last notified level (0/warn=1/alert=2)

	// Menu items — fixed slots that we mutate via SetTitle on each refresh.
	headerM    *systray.MenuItem
	bucketM    map[string]*systray.MenuItem
	lastM      *systray.MenuItem
	refreshM   *systray.MenuItem
	dashboardM *systray.MenuItem
	quitM      *systray.MenuItem
}

func (t *trayApp) onReady() {
	systray.SetTitle("CU …")
	systray.SetTooltip("claude-usage — loading")
	t.notified = map[string]int{}

	// Promote the systray-only process to a regular Dock-visible app
	// so we can paint a progress-bar icon there. No-op on Linux/Windows.
	dockShow(true)

	// Header — like "Battery 100%" on the macOS menu.
	t.headerM = systray.AddMenuItem("claude-usage — loading", "")
	t.headerM.Disable()
	systray.AddSeparator()

	// One disabled item per bucket; populated on first fetch.
	t.bucketM = map[string]*systray.MenuItem{}
	for _, name := range []string{"Session", "Weekly", "Sonnet", "Design"} {
		mi := systray.AddMenuItem(name+"  —  …", "")
		mi.Disable()
		t.bucketM[name] = mi
	}
	systray.AddSeparator()

	t.lastM = systray.AddMenuItem("Last update: —", "")
	t.lastM.Disable()
	systray.AddSeparator()

	t.refreshM = systray.AddMenuItem("Refresh now", "Fetch latest usage")
	t.dashboardM = systray.AddMenuItem("Open dashboard…", "Open claude.ai usage page in browser")
	systray.AddSeparator()
	t.quitM = systray.AddMenuItem("Quit", "Stop the tray")

	go t.loop()
	go t.menuLoop()
}

func (t *trayApp) onExit() {}

func (t *trayApp) menuLoop() {
	for {
		select {
		case <-t.refreshM.ClickedCh:
			t.fetchAndUpdate()
		case <-t.dashboardM.ClickedCh:
			_ = openURL(dashboardURL)
		case <-t.quitM.ClickedCh:
			systray.Quit()
			return
		case <-t.ctx.Done():
			systray.Quit()
			return
		}
	}
}

func (t *trayApp) loop() {
	t.fetchAndUpdate()
	cfg, _ := config.Load()
	interval := time.Duration(cfg.PollIntervalSeconds) * time.Second
	if interval < 30*time.Second {
		interval = 60 * time.Second
	}
	tk := time.NewTicker(interval)
	defer tk.Stop()
	for {
		select {
		case <-tk.C:
			t.fetchAndUpdate()
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *trayApp) fetchAndUpdate() {
	ctx, cancel := context.WithTimeout(t.ctx, 15*time.Second)
	defer cancel()
	u, cfg, err := snapshot(ctx, t.mock)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.cfg = cfg

	if err != nil {
		systray.SetTitle("CU !")
		systray.SetTooltip("claude-usage: " + err.Error())
		t.headerM.SetTitle("claude-usage — error")
		for _, mi := range t.bucketM {
			mi.SetTitle("(unavailable)")
		}
		t.lastM.SetTitle("Last update: failed — " + err.Error())
		return
	}

	worst := worstBucket(u)
	systray.SetTitle(fmt.Sprintf("%s CU %.0f%%", bandEmoji(worst.PercentUsed, cfg), worst.PercentUsed))
	systray.SetTooltip(tooltipFor(u))
	dockSetIcon(renderDockIcon(u, cfg))
	t.headerM.SetTitle(fmt.Sprintf("claude-usage  —  worst: %.0f%%", worst.PercentUsed))
	for _, nb := range u.Buckets() {
		if mi, ok := t.bucketM[nb.Name]; ok {
			mi.SetTitle(formatBucketLine(nb, cfg))
		}
	}
	t.lastM.SetTitle("Last update: " + time.Now().Local().Format("15:04:05"))

	if cfg.Notify {
		t.maybeNotify(u, cfg)
	}
}

// formatBucketLine renders one row of the detail panel:
//   "🟢 Session    16%   ·   resets in 3h 47m"
func formatBucketLine(nb fetcher.NamedBucket, cfg config.Config) string {
	label := pad(nb.Name, 8)
	pct := fmt.Sprintf("%3.0f%%", nb.PercentUsed)
	reset := resetHint(nb.ResetsAt)
	emoji := bandEmoji(nb.PercentUsed, cfg)
	if reset == "" {
		return fmt.Sprintf("%s %s %s", emoji, label, pct)
	}
	return fmt.Sprintf("%s %s %s   %s", emoji, label, pct, reset)
}

// bandEmoji picks a colored circle to match the warn/alert bands.
//   < warn   → green
//   ≥ warn   → orange
//   ≥ alert  → red
func bandEmoji(pct float64, cfg config.Config) string {
	switch {
	case pct >= float64(cfg.AlertThreshold):
		return "🔴"
	case pct >= float64(cfg.WarnThreshold):
		return "🟠"
	default:
		return "🟢"
	}
}

func tooltipFor(u *fetcher.Usage) string {
	return fmt.Sprintf("Session %.0f%%  ·  Weekly %.0f%%  ·  Sonnet %.0f%%  ·  Design %.0f%%",
		u.Session.PercentUsed, u.Weekly.PercentUsed, u.Sonnet.PercentUsed, u.Design.PercentUsed)
}

func worstBucket(u *fetcher.Usage) fetcher.Bucket {
	worst := u.Session
	for _, nb := range u.Buckets() {
		if nb.PercentUsed > worst.PercentUsed {
			worst = nb.Bucket
		}
	}
	return worst
}

func (t *trayApp) maybeNotify(u *fetcher.Usage, cfg config.Config) {
	for _, nb := range u.Buckets() {
		level := 0
		switch {
		case nb.PercentUsed >= float64(cfg.AlertThreshold):
			level = 2
		case nb.PercentUsed >= float64(cfg.WarnThreshold):
			level = 1
		}
		if level == 0 {
			t.notified[nb.Name] = 0
			continue
		}
		if t.notified[nb.Name] >= level {
			continue
		}
		t.notified[nb.Name] = level
		title := fmt.Sprintf("claude-usage: %s %.0f%%", nb.Name, nb.PercentUsed)
		msg := "Approaching limit — claude.ai may rate-limit you soon."
		if level == 2 {
			msg = "Above alert threshold — limit may already be enforced."
		}
		_ = beeep.Notify(title, msg, "")
	}
}

// openURL launches the OS's default handler for url.
func openURL(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
