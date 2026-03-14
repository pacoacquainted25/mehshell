package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	TransientPrompt bool
	InstantPrompt   bool
	ViMode          bool
	Icons           bool
	Style           string
	Segments        map[string]bool
}

func defaultConfig() config {
	return config{
		TransientPrompt: true,
		InstantPrompt:   true,
		ViMode:          true,
		Icons:           true,
		Style:           "lean",
		Segments: map[string]bool{
			"os": true, "dir": true, "git": true, "node": true,
			"python": true, "go": true, "rust": true, "ruby": true,
			"java": true, "conda": true, "venv": true, "k8s": true,
			"terraform": true, "docker": true, "aws": true, "azure": true,
			"gcloud": true, "battery": true, "duration": true, "time": true,
		},
	}
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "mehshell", "config")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mehshell", "config")
}

func loadConfig() config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		on := val == "true" || val == "1" || val == "yes"
		switch key {
		case "transient_prompt":
			cfg.TransientPrompt = on
		case "instant_prompt":
			cfg.InstantPrompt = on
		case "vi_mode":
			cfg.ViMode = on
		case "icons":
			cfg.Icons = on
		case "style":
			switch val {
			case "lean", "classic", "rainbow":
				cfg.Style = val
			}
		default:
			if _, ok := cfg.Segments[key]; ok {
				cfg.Segments[key] = on
			}
		}
	}
	return cfg
}

const defaultConfigFile = `# mehshell configuration
#
# After changing, restart your shell or run: source <(mehshell init zsh)
# All options default to true if not specified.

# ── Features ────────────────────────────────────────────

# Collapse previous prompts to a simple ">" on Enter.
# Set to false to preserve full prompts with timestamps in scrollback.
transient_prompt = true

# Show cached prompt immediately on shell startup.
instant_prompt = true

# Swap prompt character on vi keymap change.
vi_mode = true

# Show Nerd Font icons. Set to false for text labels instead.
icons = true

# Prompt style: lean, classic, or rainbow.
#   lean    - colored text, no backgrounds (default)
#   classic - powerline arrows with segment-colored backgrounds
#   rainbow - powerline arrows with cycling rainbow backgrounds
style = lean

# ── Left segments ───────────────────────────────────────

os = true
dir = true
git = true
node = true
python = true
go = true
rust = true
ruby = true
java = true

# ── Right segments ──────────────────────────────────────

conda = true
venv = true
k8s = true
terraform = true
docker = true
aws = true
azure = true
gcloud = true
battery = true
duration = true
time = true
`
