// Package cursor reads Cursor's state.vscdb SQLite database.
// Schema is reverse-engineered and marked experimental.
// TODO milestone 8: implement full collection.
package cursor

import (
	"os"
	"runtime"
	"time"

	"github.com/rdubar/llmstat/internal/provider"
)

type Provider struct{}

func (Provider) Name() string { return "cursor" }

func (Provider) Detect() bool {
	_, err := os.Stat(dbPath())
	return err == nil
}

func (Provider) Collect(cfg provider.ProviderConfig, since time.Time) (provider.Summary, error) {
	return provider.Summary{
		Name:     "cursor",
		LimitPct: -1,
		Extra:    "coming soon (experimental)",
	}, nil
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return home + "/Library/Application Support/Cursor/User/globalStorage/state.vscdb"
	}
	return home + "/.config/Cursor/User/globalStorage/state.vscdb"
}
