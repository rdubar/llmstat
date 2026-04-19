// Package codex reads usage data from ~/.codex/state_5.sqlite.
// SQLite support via modernc.org/sqlite (pure Go, no cgo).
package codex

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rdubar/llmstat/internal/config"
	"github.com/rdubar/llmstat/internal/provider"
	_ "modernc.org/sqlite"
)

type Provider struct{}

func (Provider) Name() string { return "codex" }

func (Provider) Detect() bool {
	_, err := os.Stat(dbPath())
	return err == nil
}

func (Provider) Collect(cfg provider.ProviderConfig, since time.Time) (provider.Summary, error) {
	db, err := sql.Open("sqlite", dbPath()+"?mode=ro")
	if err != nil {
		return provider.Summary{Name: "codex", Err: err}, err
	}
	defer db.Close()

	sinceUnix := since.Unix()
	now := time.Now()

	// tokens_used is a cumulative per-thread total; we can only count threads
	// that were *created* in the window — not incremental tokens added since then.
	// This understates usage for long-lived threads but avoids inflating counts.
	var tokensTotal int64
	var sessions int
	err = db.QueryRow(
		`SELECT COALESCE(SUM(tokens_used),0), COUNT(*) FROM threads WHERE created_at >= ?`,
		sinceUnix,
	).Scan(&tokensTotal, &sessions)
	if err != nil {
		return provider.Summary{Name: "codex", Err: err}, err
	}

	// 5-hour window for tier limit (same caveat: threads created in window)
	win5h := now.Add(-5 * time.Hour).Unix()
	var tokens5h int64
	_ = db.QueryRow(
		`SELECT COALESCE(SUM(tokens_used),0) FROM threads WHERE created_at >= ?`,
		win5h,
	).Scan(&tokens5h)

	// RatePer5Min is not reported: tokens_used is cumulative so a thread created
	// seconds ago would contribute its full lifetime total, not recent activity.
	s := provider.Summary{
		Name:        "codex",
		TokensToday: tokensTotal,
		Sessions:    sessions,
		LimitPct:    -1,
	}

	tier, ok := config.LookupTier("codex", cfg.Tier)
	if ok && tier.TokensPerWindow > 0 {
		s.LimitPct = float64(tokens5h) / float64(tier.TokensPerWindow)
		s.LimitSource = "5hr window"
		s.LimitLabel = fmt.Sprintf("%s limit", cfg.Tier)
	}

	return s, nil
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "state_5.sqlite")
}
