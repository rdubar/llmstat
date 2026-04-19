package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type providerSetup struct {
	key      string
	label    string
	choices  []string
	detectFn func() bool
}

var providers = []providerSetup{
	{
		key:     "claude",
		label:   "Claude Code",
		choices: []string{"free", "pro", "max"},
		detectFn: func() bool {
			home, _ := os.UserHomeDir()
			_, err := os.Stat(home + "/.claude/projects")
			return err == nil
		},
	},
	{
		key:     "codex",
		label:   "OpenAI Codex",
		choices: []string{"free", "plus", "pro"},
		detectFn: func() bool {
			home, _ := os.UserHomeDir()
			_, err := os.Stat(home + "/.codex/state_5.sqlite")
			return err == nil
		},
	},
	{
		key:     "gemini",
		label:   "Gemini CLI",
		choices: []string{"free", "advanced"},
		detectFn: func() bool {
			home, _ := os.UserHomeDir()
			_, err := os.Stat(home + "/.gemini/telemetry.log")
			return err == nil
		},
	},
	{
		key:     "cursor",
		label:   "Cursor",
		choices: []string{"hobby", "pro", "business"},
		detectFn: func() bool {
			home, _ := os.UserHomeDir()
			paths := []string{
				home + "/Library/Application Support/Cursor/User/globalStorage/state.vscdb",
				home + "/.config/Cursor/User/globalStorage/state.vscdb",
			}
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					return true
				}
			}
			return false
		},
	},
}

// RunSetup runs the interactive setup wizard, writing to cfgPath.
func RunSetup(cfgPath string) error {
	existing, _ := Load(cfgPath)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("llmstat setup")
	fmt.Println(strings.Repeat("─", 41))

	var detected []providerSetup
	for _, p := range providers {
		if p.detectFn() {
			detected = append(detected, p)
		}
	}

	if len(detected) == 0 {
		fmt.Println("No AI tools detected on this machine.")
		fmt.Println("Install Claude Code, Codex, Gemini CLI, or Cursor and run setup again.")
		return nil
	}

	names := make([]string, len(detected))
	for i, p := range detected {
		names[i] = p.label
	}
	fmt.Printf("Detected: %s\n\n", strings.Join(names, ", "))

	updated := existing

	for _, p := range detected {
		current := tierFor(&updated, p.key)
		prompt := fmt.Sprintf("%s subscription  [%s / enter to skip]",
			p.label, strings.Join(p.choices, " / "))
		if current != "" {
			prompt += fmt.Sprintf(" (current: %s)", current)
		}
		fmt.Printf("%s: ", prompt)

		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}
		setTier(&updated, p.key, input)
	}

	if err := Save(cfgPath, updated); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("\nWrote %s\n", cfgPath)
	fmt.Println("Run `llmstat` to see your usage.")
	return nil
}

func tierFor(cfg *Config, key string) string {
	switch key {
	case "claude":
		return cfg.Claude.Tier
	case "codex":
		return cfg.Codex.Tier
	case "gemini":
		return cfg.Gemini.Tier
	case "cursor":
		return cfg.Cursor.Tier
	}
	return ""
}

func setTier(cfg *Config, key, tier string) {
	switch key {
	case "claude":
		cfg.Claude.Tier = tier
	case "codex":
		cfg.Codex.Tier = tier
	case "gemini":
		cfg.Gemini.Tier = tier
	case "cursor":
		cfg.Cursor.Tier = tier
	}
}
