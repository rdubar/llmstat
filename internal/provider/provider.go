package provider

import "time"

// Summary is the standard result returned by every provider.
type Summary struct {
	Name        string
	TokensToday int64
	CostUSD     float64
	RatePer5Min int64   // tokens in the last 5 minutes
	Sessions    int
	LimitPct    float64 // 0.0–1.0; -1 means unknown
	LimitSource string  // "live", "tier", "budget", "avg", or ""
	LimitLabel  string  // human label, e.g. "5hr window (Max est.)"
	Extra       string  // shown after │ in the output line
	Err         error   // non-nil renders as [unavailable: ...]
}

// ProviderConfig is the per-provider section from config.toml.
type ProviderConfig struct {
	Tier           string
	DailyBudgetUSD float64
	LogPath        string
	Enabled        *bool // nil = auto-detect from data path
}

// Provider is implemented by each AI tool integration.
type Provider interface {
	Name() string

	// Detect returns false if the provider's data path does not exist.
	// A false result causes the provider to be silently skipped.
	Detect() bool

	// Collect gathers usage data for the period starting at since.
	// Errors in one provider must not affect others — return err in Summary.
	Collect(cfg ProviderConfig, since time.Time) (Summary, error)
}
