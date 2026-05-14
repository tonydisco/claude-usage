package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/config"
	"github.com/tonydisco/claude-usage/internal/fetcher"
)

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Live TUI that auto-refreshes Claude.ai plan usage",
		Long:  "Open a live dashboard that refreshes every poll_interval_seconds. Press 'r' to refresh now, 'q' or Ctrl-C to quit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			mock, _ := cmd.Flags().GetBool("mock")
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			interval := time.Duration(cfg.PollIntervalSeconds) * time.Second
			if interval < 5*time.Second {
				interval = 60 * time.Second
			}
			m := watchModel{
				cfg:      cfg,
				mock:     mock,
				interval: interval,
			}
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(cmd.Context()))
			_, err = p.Run()
			return err
		},
	}
	return cmd
}

type watchModel struct {
	cfg      config.Config
	mock     bool
	interval time.Duration

	usage     *fetcher.Usage
	err       error
	lastFetch time.Time
	loading   bool
	width     int
	height    int
}

type fetchedMsg struct {
	u   *fetcher.Usage
	err error
	at  time.Time
}

type tickMsg time.Time

func (m watchModel) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), tick(m.interval))
}

func (m watchModel) fetchCmd() tea.Cmd {
	mock := m.mock
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		u, _, err := snapshot(ctx, mock)
		return fetchedMsg{u: u, err: err, at: time.Now()}
	}
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			if !m.loading {
				m.loading = true
				return m, m.fetchCmd()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case fetchedMsg:
		m.loading = false
		m.usage = msg.u
		m.err = msg.err
		m.lastFetch = msg.at
	case tickMsg:
		m.loading = true
		return m, tea.Batch(m.fetchCmd(), tick(m.interval))
	}
	return m, nil
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13")).MarginBottom(1)
	footerStyle = lipgloss.NewStyle().Faint(true).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
)

func (m watchModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("claude-usage — live") + "\n")

	switch {
	case m.err != nil:
		b.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n")
	case m.usage == nil:
		b.WriteString("Loading…\n")
	default:
		b.WriteString(renderStatus(m.usage, m.cfg.WarnThreshold, m.cfg.AlertThreshold, true) + "\n")
	}

	status := "idle"
	if m.loading {
		status = "refreshing…"
	}
	when := "never"
	if !m.lastFetch.IsZero() {
		when = m.lastFetch.Local().Format("15:04:05")
	}
	footer := fmt.Sprintf("[r] refresh  [q] quit   ·   last: %s   ·   %s   ·   every %s",
		when, status, m.interval)
	b.WriteString(footerStyle.Render(footer))
	return b.String()
}
