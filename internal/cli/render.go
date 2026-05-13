package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/tonydisco/claude-usage/internal/fetcher"
)

// renderStatus formats a Usage snapshot as a multi-line table for `status`.
func renderStatus(u *fetcher.Usage, warn, alert int, color bool) string {
	var b strings.Builder
	for _, nb := range u.Buckets() {
		fmt.Fprintln(&b, renderRow(nb, warn, alert, color))
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderPrompt formats the two key buckets compactly for shell PS1.
// Example: [51%/13%]
func renderPrompt(u *fetcher.Usage, warn, alert int, color bool) string {
	s := fmt.Sprintf("[%.0f%%/%.0f%%]", u.Session.PercentUsed, u.Weekly.PercentUsed)
	if !color {
		return s
	}
	worst := max64(u.Session.PercentUsed, u.Weekly.PercentUsed)
	return colorFor(worst, warn, alert).Render(s)
}

func renderRow(nb fetcher.NamedBucket, warn, alert int, color bool) string {
	const labelWidth = 8
	const barWidth = 20

	label := pad(nb.Name, labelWidth)
	bar := progressBar(nb.PercentUsed, barWidth)
	pct := fmt.Sprintf("%3.0f%%", nb.PercentUsed)
	reset := resetHint(nb.ResetsAt)

	row := fmt.Sprintf("%s  %s  %s  %s", label, bar, pct, reset)
	if !color {
		return row
	}
	return colorFor(nb.PercentUsed, warn, alert).Render(row)
}

func progressBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100.0 * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func resetHint(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Until(t)
	if d < 0 {
		return "(reset overdue)"
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("(resets in %s)", roundDuration(d))
	}
	return fmt.Sprintf("(resets %s)", t.Local().Format("Mon 3:04 PM"))
}

func roundDuration(d time.Duration) string {
	if d >= time.Hour {
		h := int(d / time.Hour)
		m := int((d % time.Hour) / time.Minute)
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", int(d/time.Minute))
}

func colorFor(pct float64, warn, alert int) lipgloss.Style {
	switch {
	case pct >= float64(alert):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	case pct >= float64(warn):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	}
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
