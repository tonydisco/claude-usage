package cli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/gen2brain/beeep"
	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/config"
	"github.com/tonydisco/claude-usage/internal/fetcher"
)

func newTrayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tray",
		Short: "Run as a system tray icon (foreground; use `daemon start` to detach)",
		Long: `Run a small system tray icon that shows current Claude.ai plan
usage and refreshes on the configured poll interval. Notifies once
per bucket when warn_threshold or alert_threshold is crossed.

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

	mu        sync.Mutex
	cfg       config.Config
	notified  map[string]int // bucket -> last notified level (0/warn=1/alert=2)
	quitMenu  *systray.MenuItem
	refresh   *systray.MenuItem
	statusM   *systray.MenuItem
	lastError error
}

func (t *trayApp) onReady() {
	systray.SetTitle("CU …")
	systray.SetTooltip("claude-usage — loading")
	t.statusM = systray.AddMenuItem("Loading…", "")
	t.statusM.Disable()
	systray.AddSeparator()
	t.refresh = systray.AddMenuItem("Refresh now", "Fetch latest usage")
	t.quitMenu = systray.AddMenuItem("Quit", "Stop the tray")
	t.notified = map[string]int{}

	go t.loop()
	go t.menuLoop()
}

func (t *trayApp) onExit() {}

func (t *trayApp) menuLoop() {
	for {
		select {
		case <-t.refresh.ClickedCh:
			t.fetchAndUpdate()
		case <-t.quitMenu.ClickedCh:
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
	t.lastError = err

	if err != nil {
		systray.SetTitle("CU !")
		systray.SetTooltip("claude-usage: " + err.Error())
		t.statusM.SetTitle("Error: " + err.Error())
		return
	}

	worst := worstBucket(u)
	systray.SetTitle(fmt.Sprintf("CU %.0f%%", worst.PercentUsed))
	systray.SetTooltip(tooltipFor(u))
	t.statusM.SetTitle(fmt.Sprintf("Session %.0f%%  ·  Weekly %.0f%%", u.Session.PercentUsed, u.Weekly.PercentUsed))

	if cfg.Notify {
		t.maybeNotify(u, cfg)
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
