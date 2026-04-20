# llmstat

> **Experimental & lightweight** — a minimal, offline utility for a quick activity glance. Not a billing meter or quota tracker. See [what it does and doesn't do](#what-it-does-and-doesnt-do).
>
> If you need accurate cross-machine usage or real-time limit tracking, use [OpenUsage](https://github.com/janekbaraniewski/openusage) instead.

A single-command activity summary for your local AI coding tools.

```
claude  ░░░░░░░░░░                               │ 13.7M tok  1.6M/5min  2 sessions
codex   ██░░░░░░░░  0% of 5h window [ou] ↺ 00:37 │ 3 sessions
gemini  ░░░░░░░░░░                               │ 7.8k tok  1 session
cursor  ░░░░░░░░░░                               │ enterprise · no local usage data
```

Reads local data files directly — no API calls, no accounts, works offline.

If [OpenUsage](https://github.com/janekbaraniewski/openusage) is installed and running, llmstat uses its daemon as a backend for real server-side rate limit data (marked `[ou]`), while still reading local files for token counts and session data.

**Supported tools:** Claude (Anthropic), Codex (OpenAI), Gemini CLI, Cursor

---

## What it does and doesn't do

### What it does

- Tells you **roughly how much** you've been using each AI tool today, this week, or this month
- Shows your current **token burn rate** (tokens in the last 5 minutes) as a proxy for current activity
- Counts **sessions** started in the period
- Works **offline**, with no accounts or API keys required
- Outputs **JSON** for scripting
- **Enriches with real rate limits** if [OpenUsage](https://github.com/janekbaraniewski/openusage) is running — server-confirmed percentages and reset times, marked `[ou]`

### What it doesn't do

**It cannot tell you how close you are to your usage limit** — unless OpenUsage is running. Provider limits are enforced server-side and not exposed in local files. Without OpenUsage, the progress bar is only shown where a verified limit is known (e.g. a configured daily budget). With OpenUsage, providers that expose rate-limit headers (currently Codex) show real percentages.

**Token counts are approximate**, not billing-accurate. Local logs were not designed for accounting:
- Claude logs contain duplicate records that must be deduplicated by message ID
- Codex stores cumulative thread totals, not per-message events
- Cache reads (which can dwarf actual input/output tokens) are excluded from counts since they don't appear to count against limits the same way
- Counts are per-device — if you use the same tool on multiple machines, each machine only sees its own activity

**If you need accurate cross-machine usage data**, see [OpenUsage](#see-also), which takes a daemon + API approach instead.

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

Then run setup to configure your tier:

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
| Claude | `~/.claude/projects/**/*.jsonl` (tokens), `~/.claude/sessions/` (session count) |
| Codex  | `~/.codex/state_5.sqlite` |
| Gemini | `~/.gemini/tmp/*/chats/session-*.json` — token counts per response |
| Cursor | `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` — tier only |

Each provider is auto-detected. If the data file/directory exists, it appears in output.

If the [OpenUsage](https://github.com/janekbaraniewski/openusage) daemon is running at `~/.local/state/openusage/telemetry.sock`, llmstat queries it for server-side rate limit data and uses it to fill in the progress bar where available. This is optional — llmstat works fully without it.

## Config

Config lives at `~/.config/llmstat/config.toml`. Running `--setup` creates it interactively. You can also edit it directly:

```toml
[claude]
tier = "pro"   # pro | max5x | max20x

[codex]
tier = "plus"
```

## Known data quality issues

**Claude — duplicate log records**
The JSONL logs emit multiple records per API call (one per content block — thinking block,
then text response), each carrying the full usage payload. llmstat deduplicates by
`message.id`. Without this, totals are roughly 2× inflated.

**Claude — no limit bar**
Anthropic publishes no token limits for Claude Code CLI. Community estimates derived from
the claude.ai web interface are empirically wrong for Claude Code — heavy users don't hit
limits at those figures. The bar is omitted rather than showing a misleading number.

**Claude — cache reads excluded from counts**
`cache_read_input_tokens` can be 20–30× larger than actual input+output tokens (large
contexts are re-read on every request). These are excluded from displayed totals since
they don't appear to count against rolling limits the same way.

**Codex — cumulative token totals**
`~/.codex/state_5.sqlite` stores a running `tokens_used` per thread, not per-message
events. Only threads *created* within the window are counted — threads started before
the window but still active are missed.

**Gemini — session file filtering**
Token counts are filtered by per-message timestamp, which is accurate. Files with no
timestamps fall back to file modification time.

**Cursor — no usage data**
Subscription tier is detected locally; actual usage is cloud-side only.

**All providers — per-device only**
Counts reflect this machine only. No cross-machine aggregation.

## See also

[OpenUsage](https://github.com/janekbaraniewski/openusage) — runs a local daemon and pulls
usage from provider APIs (17+ providers). Responds instantly (pre-computed), aggregates
across machines, and shows rolling window data. A better choice if you need accuracy over
simplicity. llmstat can use it as a backend when it's running.

## Contributing

Contributions welcome — bug reports, tier data corrections, and new provider implementations especially. Please open an issue before starting significant work so we can align on approach.

## Credits

Built and maintained by [Roger Dubar](https://github.com/rdubar), with development assistance from Claude (Anthropic) and Codex (OpenAI).

With thanks to [Alphapet Ventures](https://alpha.pet).

## License

MIT — see [LICENSE](LICENSE).
