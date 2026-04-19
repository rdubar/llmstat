package display

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/rdubar/llmstat/internal/provider"
	"golang.org/x/term"
)

const barWidth = 10

// isTTY reports whether stdout is a terminal.
func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

type colour struct{ open, close string }

var (
	green  = colour{"\033[32m", "\033[0m"}
	yellow = colour{"\033[33m", "\033[0m"}
	red    = colour{"\033[31m", "\033[0m"}
	bold   = colour{"\033[1m", "\033[0m"}
	dim    = colour{"\033[2m", "\033[0m"}
	reset  = "\033[0m"
)

func wrap(c colour, s string) string {
	if !isTTY() {
		return s
	}
	return c.open + s + c.close
}

// Render prints all summaries aligned in columns.
func Render(summaries []provider.Summary) {
	if len(summaries) == 0 {
		return
	}

	// Compute column widths
	nameWidth := 0
	pctWidth := 0
	for _, s := range summaries {
		if len(s.Name) > nameWidth {
			nameWidth = len(s.Name)
		}
		if p := pctStr(s); len(p) > pctWidth {
			pctWidth = len(p)
		}
	}

	for _, s := range summaries {
		fmt.Println(renderLine(s, nameWidth, pctWidth))
	}
}

func renderLine(s provider.Summary, nameWidth, pctWidth int) string {
	name := fmt.Sprintf("%-*s", nameWidth, s.Name)

	if s.Err != nil {
		return fmt.Sprintf("%s  %s",
			wrap(bold, name),
			wrap(dim, fmt.Sprintf("[unavailable: %v]", s.Err)),
		)
	}

	bar := renderBar(s.LimitPct)
	pct := fmt.Sprintf("%-*s", pctWidth, pctStr(s))

	data := renderData(s)

	return fmt.Sprintf("%s  %s  %s  %s %s",
		wrap(bold, name),
		bar,
		pct,
		wrap(dim, "│"),
		data,
	)
}

func renderBar(pct float64) string {
	if pct < 0 {
		// No limit known — show a neutral empty bar
		return wrap(dim, strings.Repeat("░", barWidth))
	}
	filled := int(math.Round(math.Min(pct, 1.0) * barWidth))
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	c := green
	switch {
	case pct >= 0.85:
		c = red
	case pct >= 0.60:
		c = yellow
	}
	return wrap(c, bar)
}

func pctStr(s provider.Summary) string {
	if s.LimitPct < 0 || s.LimitSource == "" {
		return ""
	}
	label := s.LimitLabel
	if label == "" {
		label = s.LimitSource
	}
	pct := int(math.Round(s.LimitPct * 100))
	if pct > 100 {
		return fmt.Sprintf(">100%% of %s", label)
	}
	return fmt.Sprintf("%d%% of %s", pct, label)
}

func renderData(s provider.Summary) string {
	parts := []string{}

	if s.TokensToday > 0 {
		parts = append(parts, fmtTokens(s.TokensToday)+" tok")
	}
	if s.CostUSD > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f", s.CostUSD))
	}
	if s.RatePer5Min > 0 {
		parts = append(parts, fmtTokens(s.RatePer5Min)+"/5min")
	}
	if s.Sessions > 0 {
		parts = append(parts, fmt.Sprintf("%d sessions", s.Sessions))
	}
	if s.Extra != "" {
		parts = append(parts, s.Extra)
	}

	return strings.Join(parts, "  ")
}

func fmtTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// PrintWarnings prints stale-tier warnings to stderr.
func PrintWarnings(warnings []string) {
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, wrap(yellow, "⚠  "+w))
	}
}

// ensure reset on any output
var _ = reset
