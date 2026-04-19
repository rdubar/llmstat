// Package cursor reads Cursor's state.vscdb for tier info.
// Token usage is cloud-only; we show membership tier and detect installation.
package cursor

import (
	"database/sql"
	"os"
	"runtime"

	"github.com/rdubar/llmstat/internal/provider"
	_ "modernc.org/sqlite"
	"time"
)

type Provider struct{}

func (Provider) Name() string { return "cursor" }

func (Provider) Detect() bool {
	_, err := os.Stat(dbPath())
	return err == nil
}

func (Provider) Collect(cfg provider.ProviderConfig, since time.Time) (provider.Summary, error) {
	tier := cfg.Tier
	if tier == "" {
		tier = readTier()
	}
	extra := "no local usage data"
	if tier != "" {
		extra = tier + " · no local usage data"
	}
	return provider.Summary{
		Name:     "cursor",
		LimitPct: -1,
		Extra:    extra,
	}, nil
}

func readTier() string {
	db, err := sql.Open("sqlite", dbPath()+"?mode=ro")
	if err != nil {
		return ""
	}
	defer db.Close()
	var tier string
	_ = db.QueryRow(`SELECT value FROM ItemTable WHERE key='cursorAuth/stripeMembershipType'`).Scan(&tier)
	return tier
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return home + "/Library/Application Support/Cursor/User/globalStorage/state.vscdb"
	}
	return home + "/.config/Cursor/User/globalStorage/state.vscdb"
}
