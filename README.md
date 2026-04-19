# llmstat

Track your local AI tool usage in one line.

```
claude  ████████░░  78% of 5hr window  │ 28.3M tok  $4.20  1.2M/5min
codex   ░░░░░░░░░░                     │ 19.5M tok  4 sessions
```

Reads local data files directly — no API calls, no accounts, works offline.

**Supported tools:** Claude (Anthropic), Codex (OpenAI), Gemini CLI *(coming soon)*, Cursor *(coming soon)*

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
llmstat claude     # single provider detail
llmstat --json     # machine-readable JSON output
```

## Upgrade

```sh
go install github.com/rdubar/llmstat/cmd/llmstat@latest
```

## How it works

| Tool   | Data source |
|--------|-------------|
| Claude | `~/.claude/projects/**/*.jsonl` |
| Codex  | `~/.codex/state_5.sqlite` |
| Gemini | `~/.gemini/telemetry.log` *(coming soon)* |
| Cursor | `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` *(coming soon)* |

Each provider is auto-detected. If the data file exists, it appears in output.

Tier limits (tokens per 5-hour window) are read from a bundled `tiers.toml` and calibrated periodically against published limits. Run `llmstat --setup` to set your tier. The bar turns yellow at 60% and red at 85%.

## Config

Config lives at `~/.config/llmstat/config.toml`. Running `--setup` creates it interactively. You can also edit it directly:

```toml
[claude]
tier = "max"

[codex]
tier = "plus"
```

## License

MIT — see [LICENSE](LICENSE).
