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
// Numbers are battery-style "remaining" capacity (100% means untouched,
// 0% means at the plan ceiling). Color band is decided from raw usage
// so warn/alert thresholds keep their meaning. Example: [49%/87%]
func renderPrompt(u *fetcher.Usage, warn, alert int, color bool) string {
	s := fmt.Sprintf("[%.0f%%/%.0f%%]", 100-u.Session.PercentUsed, 100-u.Weekly.PercentUsed)
	if !color {
		return s
	}
	worst := max64(u.Session.PercentUsed, u.Weekly.PercentUsed)
	return colorFor(worst, warn, alert).Render(s)
}

func renderRow(nb fetcher.NamedBucket, warn, alert int, color bool) string {
	const labelWidth = 8
	const barWidth = 20

	// Battery-style: the visible bar represents how much capacity is
	// still LEFT in the bucket. As usage grows, the bar drains.
	remaining := 100.0 - nb.PercentUsed
	label := pad(nb.Name, labelWidth)
	bar := progressBar(remaining, barWidth)
	pct := fmt.Sprintf("%3.0f%%", remaining)
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

// Battery-style three-band coloring:
//   green   below warn_threshold
//   orange  between warn and alert
//   red     at or above alert_threshold
func colorFor(pct float64, warn, alert int) lipgloss.Style {
	switch {
	case pct >= float64(alert):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // red
	case pct >= float64(warn):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // green
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
