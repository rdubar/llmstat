# llmstat

Track your local AI tool usage in one line.

```
claude  ████████░░  78% of 5hr window  │ 42.0M tok  1.9M/5min
codex   ░░░░░░░░░░                     │ 5.4M tok  3 sessions
gemini  ░░░░░░░░░░                     │ 7.8k tok  1 sessions
cursor  ░░░░░░░░░░                     │ enterprise · no local usage data
```

Reads local data files directly — no API calls, no accounts, works offline.

**Supported tools:** Claude (Anthropic), Codex (OpenAI), Gemini CLI, Cursor

> **Accuracy note:** llmstat is an activity pulse, not a precise billing meter. Token counts
> are approximate — see [Known limitations](#known-limitations) below.

---

## Install

Requires [Go 1.21+](https://go.dev/doc/install).

```sh
go install github.com/rdubar/llmstat/cmd/llmstat@latest
```

Make sure `$GOPATH/bin` is on your PATH (one-time setup if not already done):

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc
```

Then run setup to configure your tier (Claude Pro, Max, etc.):

```sh
llmstat --setup
```

## Usage

```sh
llmstat            # today's usage for all detected tools
llmstat -w         # this week's usage
llmstat -m         # this month's usage
llmstat claude     # single provider detail
llmstat --json     # machine-readable JSON output
llmstat -u         # upgrade to latest version
llmstat -v         # version and build info
llmstat --credits  # credits and provenance
```

## Upgrade

```sh
llmstat --upgrade   # or: llmstat -u
```

## How it works

| Tool   | Data source |
|--------|-------------|
| Claude | `~/.claude/projects/**/*.jsonl` |
| Codex  | `~/.codex/state_5.sqlite` |
| Gemini | `~/.gemini/tmp/*/chats/session-*.json` — token counts per response |
| Cursor | `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` — tier only (usage is cloud-side) |

Each provider is auto-detected. If the data file exists, it appears in output.

Tier limits (tokens per 5-hour window) are read from a bundled `tiers.toml` and calibrated periodically against published limits. Run `llmstat --setup` to set your tier. The bar turns yellow at 60% and red at 85%.

## Known limitations

llmstat reads whatever local telemetry each tool happens to write. That data wasn't designed for accounting, so there are real accuracy gaps:

**Claude** — The JSONL logs contain repeated assistant events with identical usage payloads
milliseconds apart (likely streaming artifacts). llmstat deduplicates by exact
`(timestamp, token counts)` within each file, which reduces inflated totals significantly,
but the deduplication heuristic may still miss some cases or over-deduplicate in unusual
situations. Treat Claude token counts as directionally correct, not precise.
`costUSD` fields are zero in local logs, so the daily budget feature shows no spend even
when tokens are high.

**Codex** — `~/.codex/state_5.sqlite` stores a cumulative `tokens_used` total per thread,
not per-message events. llmstat can only count threads that were *created* within the
requested window — threads started before the window but still active are excluded.
The 5-minute rate shown by some other tools is not reported here because it would just
be the full lifetime token count of any recently-created thread, which is meaningless as
a rate.

**Gemini** — Token counts are per model response and are filtered by message timestamp,
so they should be reasonably accurate for the selected window. The main caveat is that
session files with no `timestamp` fields on messages are excluded from period filtering
and may contribute zero or all tokens depending on file modification time.

**Cursor** — No local usage data is available. llmstat can only detect your subscription
tier from the local state database. Actual token usage is cloud-side only.

**General** — "Today" is computed in your local timezone. Weekly (`-w`) and monthly (`-m`)
windows are also local-time anchored.

## Config

Config lives at `~/.config/llmstat/config.toml`. Running `--setup` creates it interactively. You can also edit it directly:

```toml
[claude]
tier = "max"

[codex]
tier = "plus"
```

## Contributing

Contributions welcome — bug reports, tier data corrections, and new provider implementations especially. Please open an issue before starting significant work so we can align on approach.

## Credits

Built and maintained by [Roger Dubar](https://github.com/rdubar), with development assistance from Claude (Anthropic) and Codex (OpenAI).

With thanks to [Alphapet Ventures](https://alpha.pet).

## License

MIT — see [LICENSE](LICENSE).
