package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var Version = "dev"

// ── Zsh prompt colors ────────────────────────────────────────────

const (
	cCyan    = 75
	cBlue    = 39
	cMagenta = 170
	cGreen   = 76
	cRed     = 196
	cYellow  = 220
	cOrange  = 208
	cGray    = 242
)

func fc(color int, s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "%", "%%")
	return fmt.Sprintf("%%F{%d}%s%%f", color, s)
}

// ── Segment with sort order ──────────────────────────────────────

type seg struct {
	text  string
	order int
	bg    int
}

// ── Config ───────────────────────────────────────────────────────

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

// ── Zsh init ─────────────────────────────────────────────────────

func zshInitScript(cfg config) string {
	bin := "mehshell"
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			bin = resolved
		} else {
			bin = exe
		}
	}

	var b strings.Builder
	b.WriteString("zmodload zsh/datetime 2>/dev/null\n")
	b.WriteString("typeset -gi _mehshell_ts=0\n")
	if cfg.InstantPrompt {
		b.WriteString("[[ -r \"${XDG_CACHE_HOME:-$HOME/.cache}/mehshell-prompt-cache\" ]] && source \"${XDG_CACHE_HOME:-$HOME/.cache}/mehshell-prompt-cache\"\n")
	}
	b.WriteString("_mehshell_preexec() { _mehshell_ts=$EPOCHSECONDS }\n")
	b.WriteString("_mehshell_precmd() {\n")
	b.WriteString("  local e=$? d=0\n")
	b.WriteString("  (( _mehshell_ts > 0 )) && d=$(( EPOCHSECONDS - _mehshell_ts ))\n")
	b.WriteString("  _mehshell_ts=0\n")
	b.WriteString(fmt.Sprintf("  local _out=\"$(%s $e $d $COLUMNS)\"\n", bin))
	b.WriteString("  eval \"$_out\"\n")
	if cfg.InstantPrompt {
		b.WriteString("  print -r -- \"$_out\" >| \"${XDG_CACHE_HOME:-$HOME/.cache}/mehshell-prompt-cache\" 2>/dev/null\n")
	}
	b.WriteString("}\n")
	if cfg.ViMode {
		b.WriteString("_mehshell_zle_keymap_select() {\n")
		b.WriteString("  [[ $KEYMAP == vicmd ]] && PROMPT=\"${PROMPT/❯/❮}\" || PROMPT=\"${PROMPT/❮/❯}\"\n")
		b.WriteString("  zle reset-prompt\n")
		b.WriteString("}\n")
		b.WriteString("zle -N zle-keymap-select _mehshell_zle_keymap_select\n")
	}
	if cfg.TransientPrompt {
		b.WriteString("_mehshell_accept_line() {\n")
		b.WriteString("  PROMPT=$'%F{76}❯%f '\n")
		b.WriteString("  zle reset-prompt\n")
		b.WriteString("  zle .accept-line\n")
		b.WriteString("}\n")
		b.WriteString("zle -N accept-line _mehshell_accept_line\n")
	}
	b.WriteString("preexec_functions+=(_mehshell_preexec)\n")
	b.WriteString("precmd_functions+=(_mehshell_precmd)")
	return b.String()
}

// ── Style rendering ──────────────────────────────────────────────

func stripColors(s string) string {
	for _, prefix := range []string{"%F{", "%K{"} {
		for {
			idx := strings.Index(s, prefix)
			if idx == -1 {
				break
			}
			end := strings.Index(s[idx:], "}")
			if end == -1 {
				break
			}
			s = s[:idx] + s[idx+end+1:]
		}
	}
	for _, esc := range []string{"%f", "%k"} {
		s = strings.ReplaceAll(s, esc, "")
	}
	s = strings.ReplaceAll(s, "%%", "%")
	return s
}

func escPercent(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}

func contrastFg(bg int) int {
	switch bg {
	case 178, 136, 220, 3, 76, 2, 208:
		return 0
	default:
		return 255
	}
}

func renderPowerlineLeft(segs []seg, rainbow bool) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	if rainbow {
		for i, s := range segs {
			fg := contrastFg(s.bg)
			plain := stripColors(s.text)
			b.WriteString(fmt.Sprintf("%%K{%d}%%F{%d} %s ", s.bg, fg, escPercent(plain)))
			if i < len(segs)-1 {
				b.WriteString(fmt.Sprintf("%%F{%d}%%K{%d}\ue0b0", s.bg, segs[i+1].bg))
			} else {
				b.WriteString(fmt.Sprintf("%%f%%k%%F{%d}\ue0b0%%f", s.bg))
			}
		}
	} else {
		b.WriteString(fmt.Sprintf("%%K{%d}", 238))
		for i, s := range segs {
			b.WriteString(fmt.Sprintf(" %s ", s.text))
			if i < len(segs)-1 {
				b.WriteString(fmt.Sprintf("%%F{%d}\ue0b1%%f", 246))
			}
		}
		b.WriteString(fmt.Sprintf("%%f%%k%%F{%d}\ue0b0%%f", 238))
	}
	return b.String()
}

func renderPowerlineRight(segs []seg, rainbow bool) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	if rainbow {
		for i, s := range segs {
			fg := contrastFg(s.bg)
			plain := stripColors(s.text)
			if i == 0 {
				b.WriteString(fmt.Sprintf("%%F{%d}\ue0b2%%K{%d}%%F{%d} %s %%f", s.bg, s.bg, fg, escPercent(plain)))
			} else {
				b.WriteString(fmt.Sprintf("%%F{%d}\ue0b2%%K{%d}%%F{%d} %s %%f", s.bg, s.bg, fg, escPercent(plain)))
			}
		}
		b.WriteString("%k")
	} else {
		b.WriteString(fmt.Sprintf("%%F{%d}\ue0b2%%K{%d}", 238, 238))
		for i, s := range segs {
			b.WriteString(fmt.Sprintf(" %s ", s.text))
			if i < len(segs)-1 {
				b.WriteString(fmt.Sprintf("%%F{%d}\ue0b3%%f", 246))
			}
		}
		b.WriteString("%f%k")
	}
	return b.String()
}

// ── Main ─────────────────────────────────────────────────────────

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("mehshell", Version)
			return
		case "init":
			if len(os.Args) > 2 && os.Args[2] == "zsh" {
				cfg := loadConfig()
				fmt.Println(zshInitScript(cfg))
			} else {
				fmt.Fprintln(os.Stderr, "usage: mehshell init zsh")
				os.Exit(1)
			}
			return
		case "config":
			if len(os.Args) > 2 && os.Args[2] == "init" {
				p := configPath()
				if _, err := os.Stat(p); err == nil {
					fmt.Fprintln(os.Stderr, "config already exists:", p)
					os.Exit(1)
				}
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					os.Exit(1)
				}
				if err := os.WriteFile(p, []byte(defaultConfigFile), 0644); err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					os.Exit(1)
				}
				fmt.Println("created", p)
			} else if len(os.Args) > 2 && os.Args[2] == "path" {
				fmt.Println(configPath())
			} else {
				fmt.Fprintln(os.Stderr, "usage: mehshell config [init|path]")
				os.Exit(1)
			}
			return
		}
	}

	cfg := loadConfig()
	on := cfg.Segments

	exitCode := 0
	duration := 0
	columns := 80

	if len(os.Args) > 1 {
		exitCode, _ = strconv.Atoi(os.Args[1])
	}
	if len(os.Args) > 2 {
		duration, _ = strconv.Atoi(os.Args[2])
	}
	if len(os.Args) > 3 {
		columns, _ = strconv.Atoi(os.Args[3])
	}
	if columns <= 0 {
		columns = 80
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var left, right []seg

	add := func(dst *[]seg, order int, bg int, fn func() string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if t := fn(); t != "" {
				mu.Lock()
				*dst = append(*dst, seg{t, order, bg})
				mu.Unlock()
			}
		}()
	}

	icons := cfg.Icons
	if on["os"] {
		add(&left, 0, 24, func() string { return segOS(icons) })
	}
	if on["dir"] {
		add(&left, 1, 31, func() string { return segDir(cwd, home) })
	}
	if on["git"] {
		add(&left, 2, 97, func() string { return segGit(cwd, icons) })
	}
	if on["node"] {
		add(&left, 3, 34, func() string { return segNode(cwd, icons) })
	}
	if on["python"] {
		add(&left, 4, 136, func() string { return segPython(cwd, icons) })
	}
	if on["go"] {
		add(&left, 5, 37, func() string { return segGo(cwd, icons) })
	}
	if on["rust"] {
		add(&left, 6, 130, func() string { return segRust(cwd, icons) })
	}
	if on["ruby"] {
		add(&left, 7, 124, func() string { return segRuby(cwd, icons) })
	}
	if on["java"] {
		add(&left, 8, 94, func() string { return segJava(cwd, icons) })
	}
	if on["conda"] {
		add(&right, 0, 34, func() string { return segConda(icons) })
	}
	if on["venv"] {
		add(&right, 1, 34, func() string { return segVenv(icons) })
	}
	if on["k8s"] {
		add(&right, 2, 25, func() string { return segK8s(cwd, home, icons) })
	}
	if on["terraform"] {
		add(&right, 3, 57, func() string { return segTerraform(cwd, icons) })
	}
	if on["docker"] {
		add(&right, 4, 25, func() string { return segDocker(cwd, icons) })
	}
	if on["aws"] {
		add(&right, 5, 166, func() string { return segAWS(icons) })
	}
	if on["azure"] {
		add(&right, 6, 25, func() string { return segAzure(icons) })
	}
	if on["gcloud"] {
		add(&right, 7, 34, func() string { return segGCloud(icons) })
	}
	if on["battery"] {
		add(&right, 8, 22, func() string { return segBattery(icons) })
	}
	if on["duration"] {
		add(&right, 9, 136, func() string { return segDuration(duration) })
	}
	if on["time"] {
		add(&right, 10, 238, func() string { return segTime() })
	}

	wg.Wait()

	sort.Slice(left, func(i, j int) bool { return left[i].order < left[j].order })
	sort.Slice(right, func(i, j int) bool { return right[i].order < right[j].order })

	var line1, line2 string

	charColor := cGreen
	if exitCode != 0 {
		charColor = cRed
	}

	switch cfg.Style {
	case "classic", "rainbow":
		rainbow := cfg.Style == "rainbow"
		lStr := renderPowerlineLeft(left, rainbow)
		rStr := renderPowerlineRight(right, rainbow)
		pad := columns - visibleWidth(lStr) - visibleWidth(rStr)
		if pad < 1 {
			pad = 1
		}
		line1 = lStr + strings.Repeat(" ", pad) + rStr
		line2 = fc(charColor, "❯")
	default:
		leftStr := joinSegs(left, " ")
		rightStr := joinSegs(right, "  ")
		prefix := fc(cCyan, "╭─") + " "
		prefixVis := 3
		pad := columns - prefixVis - visibleWidth(leftStr) - visibleWidth(rightStr)
		if pad < 1 {
			pad = 1
		}
		line1 = prefix + leftStr + strings.Repeat(" ", pad) + rightStr
		line2 = fc(cCyan, "╰─") + fc(charColor, "❯")
	}

	l1 := escShell(line1)
	l2 := escShell(line2)
	fmt.Printf("PROMPT=$'\\n%s\\n%s '\n", l1, l2)
}

// ── Segments ─────────────────────────────────────────────────────

func segOS(icons bool) string {
	if !icons {
		return ""
	}
	icon := "\uf17c"
	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lower := strings.ToLower(string(data))
			switch {
			case strings.Contains(lower, "arch"):
				icon = "\uf303"
			case strings.Contains(lower, "ubuntu"):
				icon = "\uf31b"
			case strings.Contains(lower, "fedora"):
				icon = "\uf30a"
			case strings.Contains(lower, "debian"):
				icon = "\uf306"
			case strings.Contains(lower, "nixos"):
				icon = "\uf313"
			}
		}
	case "darwin":
		icon = "\uf179"
	}
	return fc(cBlue, icon)
}

func segDir(cwd, home string) string {
	dir := cwd
	if strings.HasPrefix(dir, home) {
		dir = "~" + dir[len(home):]
	}
	if dir == "~/" {
		dir = "~"
	}
	return fc(cBlue, dir)
}

func segGit(cwd string, icons bool) string {
	branch, repoDir := gitBranch(cwd)
	if branch == "" {
		return ""
	}

	dirty := gitDirty(repoDir)
	var result string
	if icons {
		result = fc(cBlue, "\uf126 ") + fc(cMagenta, branch)
	} else {
		result = fc(cMagenta, branch)
	}
	if dirty != "" {
		result += " " + dirty
	}
	return result
}

func segNode(cwd string, icons bool) string {
	if _, err := os.Stat(filepath.Join(cwd, "package.json")); err != nil {
		return ""
	}
	prefix := "node "
	if icons {
		prefix = "\ue718 "
	}
	for _, f := range []string{".node-version", ".nvmrc"} {
		if data, err := os.ReadFile(filepath.Join(cwd, f)); err == nil {
			ver := strings.TrimSpace(string(data))
			if ver != "" {
				return fc(cGreen, prefix+ver)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "node", "--version").Output(); err == nil {
		return fc(cGreen, prefix+strings.TrimSpace(string(out)))
	}
	return ""
}

func segPython(cwd string, icons bool) string {
	if !hasMarkerUp(cwd, []string{".python-version", "pyproject.toml", "setup.py", "Pipfile", "requirements.txt"}) {
		return ""
	}
	prefix := "py "
	if icons {
		prefix = "\ue73c "
	}
	if data, err := os.ReadFile(filepath.Join(cwd, ".python-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cYellow, prefix+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "python", "--version").Output(); err == nil {
		ver := strings.TrimPrefix(strings.TrimSpace(string(out)), "Python ")
		return fc(cYellow, prefix+ver)
	}
	return ""
}

func segGo(cwd string, icons bool) string {
	gomod := filepath.Join(cwd, "go.mod")
	data, err := os.ReadFile(gomod)
	if err != nil {
		return ""
	}
	prefix := "go "
	if icons {
		prefix = "\ue627 "
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") {
			ver := strings.Fields(line)[1]
			return fc(cCyan, prefix+ver)
		}
	}
	return ""
}

func segConda(icons bool) string {
	env := os.Getenv("CONDA_DEFAULT_ENV")
	if env == "" || env == "base" {
		return ""
	}
	prefix := "conda "
	if icons {
		prefix = "\ue73c "
	}
	return fc(cGreen, prefix+env)
}

func segVenv(icons bool) string {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return ""
	}
	prefix := "venv "
	if icons {
		prefix = "\ue73c "
	}
	return fc(cGreen, prefix+filepath.Base(venv))
}

func segK8s(cwd, home string, icons bool) string {
	markers := []string{"skaffold.yaml", "helmfile.yaml", "Chart.yaml", "kustomization.yaml"}
	hasMarker := false
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(cwd, m)); err == nil {
			hasMarker = true
			break
		}
	}
	if !hasMarker {
		return ""
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	data, err := os.ReadFile(kubeconfig)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "current-context:") {
			ctx := strings.TrimSpace(strings.TrimPrefix(line, "current-context:"))
			if ctx != "" && ctx != "\"\"" {
				prefix := "k8s "
				if icons {
					prefix = "\u2388 "
				}
				return fc(cBlue, prefix+ctx)
			}
		}
	}
	return ""
}

func segAWS(icons bool) string {
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		return ""
	}
	prefix := "aws "
	if icons {
		prefix = "\uf52c "
	}
	return fc(cOrange, prefix+profile)
}

func segDuration(secs int) string {
	if secs < 3 {
		return ""
	}
	var s string
	switch {
	case secs >= 3600:
		s = fmt.Sprintf("%dh%dm", secs/3600, (secs%3600)/60)
	case secs >= 60:
		s = fmt.Sprintf("%dm%ds", secs/60, secs%60)
	default:
		s = fmt.Sprintf("%ds", secs)
	}
	return fc(cYellow, s)
}

func segTime() string {
	return fc(cGray, time.Now().Format("03:04:05 PM"))
}

// ── Additional segments ─────────────────────────────────────────

func segRust(cwd string, icons bool) string {
	if _, err := os.Stat(filepath.Join(cwd, "Cargo.toml")); err != nil {
		return ""
	}
	prefix := "rs "
	if icons {
		prefix = "\ue7a8 "
	}
	for _, f := range []string{"rust-toolchain", ".rust-version"} {
		if data, err := os.ReadFile(filepath.Join(cwd, f)); err == nil {
			ver := strings.TrimSpace(string(data))
			if ver != "" {
				return fc(cOrange, prefix+ver)
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(cwd, "rust-toolchain.toml")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "channel") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					ver := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
					if ver != "" {
						return fc(cOrange, prefix+ver)
					}
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "rustc", "--version").Output(); err == nil {
		ver := strings.TrimPrefix(strings.TrimSpace(string(out)), "rustc ")
		if i := strings.Index(ver, " "); i != -1 {
			ver = ver[:i]
		}
		return fc(cOrange, prefix+ver)
	}
	return ""
}

func segRuby(cwd string, icons bool) string {
	if !hasMarkerUp(cwd, []string{".ruby-version", "Gemfile", "Rakefile"}) {
		return ""
	}
	prefix := "rb "
	if icons {
		prefix = "\ue739 "
	}
	if data, err := os.ReadFile(filepath.Join(cwd, ".ruby-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cRed, prefix+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "ruby", "--version").Output(); err == nil {
		ver := strings.TrimPrefix(strings.TrimSpace(string(out)), "ruby ")
		if i := strings.Index(ver, " "); i != -1 {
			ver = ver[:i]
		}
		return fc(cRed, prefix+ver)
	}
	return ""
}

func segJava(cwd string, icons bool) string {
	markers := []string{"pom.xml", "build.gradle", "build.gradle.kts", ".java-version"}
	found := false
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(cwd, m)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return ""
	}
	prefix := "java "
	if icons {
		prefix = "\ue738 "
	}
	if data, err := os.ReadFile(filepath.Join(cwd, ".java-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cOrange, prefix+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "java", "-version").CombinedOutput(); err == nil {
		line := strings.SplitN(string(out), "\n", 2)[0]
		if start := strings.Index(line, "\""); start != -1 {
			if end := strings.Index(line[start+1:], "\""); end != -1 {
				return fc(cOrange, prefix+line[start+1:start+1+end])
			}
		}
	}
	return ""
}

func segTerraform(cwd string, icons bool) string {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return ""
	}
	hasTF := false
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tf") {
			hasTF = true
			break
		}
	}
	if !hasTF {
		return ""
	}
	prefix := "tf "
	if icons {
		prefix = "\uf0ac "
	}
	if data, err := os.ReadFile(filepath.Join(cwd, ".terraform-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cMagenta, prefix+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "terraform", "version").Output(); err == nil {
		line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		line = strings.TrimPrefix(line, "Terraform ")
		return fc(cMagenta, prefix+line)
	}
	return ""
}

func segDocker(cwd string, icons bool) string {
	markers := []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	found := false
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(cwd, m)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return ""
	}
	prefix := "docker"
	if icons {
		prefix = "\uf308"
	}
	if ctx := os.Getenv("DOCKER_CONTEXT"); ctx != "" && ctx != "default" {
		return fc(cCyan, prefix+" "+ctx)
	}
	if name := os.Getenv("DOCKER_MACHINE_NAME"); name != "" {
		return fc(cCyan, prefix+" "+name)
	}
	return fc(cCyan, prefix)
}

func segAzure(icons bool) string {
	acct := os.Getenv("AZURE_DEFAULTS_GROUP")
	if acct == "" {
		return ""
	}
	prefix := "az "
	if icons {
		prefix = "\ufd03 "
	}
	return fc(cBlue, prefix+acct)
}

func segGCloud(icons bool) string {
	project := os.Getenv("CLOUDSDK_CORE_PROJECT")
	if project == "" {
		project = os.Getenv("GCLOUD_PROJECT")
	}
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if project == "" {
		return ""
	}
	prefix := "gcp "
	if icons {
		prefix = "\uf1a0 "
	}
	return fc(cGreen, prefix+project)
}

func segBattery(icons bool) string {
	switch runtime.GOOS {
	case "linux":
		for _, name := range []string{"BAT0", "BAT1", "battery"} {
			data, err := os.ReadFile(filepath.Join("/sys/class/power_supply", name, "capacity"))
			if err != nil {
				continue
			}
			pct, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				continue
			}
			statusData, _ := os.ReadFile(filepath.Join("/sys/class/power_supply", name, "status"))
			status := strings.TrimSpace(string(statusData))

			icon := "\uf240"
			color := cGreen
			switch {
			case pct <= 10:
				icon = "\uf244"
				color = cRed
			case pct <= 25:
				icon = "\uf243"
				color = cOrange
			case pct <= 50:
				icon = "\uf242"
				color = cYellow
			case pct <= 75:
				icon = "\uf241"
				color = cGreen
			}
			if status == "Charging" {
				icon = "\uf0e7"
			}
			if icons {
				return fc(color, icon+" "+strconv.Itoa(pct)+"%")
			}
			return fc(color, strconv.Itoa(pct)+"%")
		}
	case "darwin":
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		out, err := exec.CommandContext(ctx, "pmset", "-g", "batt").Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, "InternalBattery") {
				continue
			}
			idx := strings.Index(line, "%")
			if idx == -1 {
				continue
			}
			start := idx - 1
			for start >= 0 && line[start] >= '0' && line[start] <= '9' {
				start--
			}
			pct, err := strconv.Atoi(strings.TrimSpace(line[start+1 : idx]))
			if err != nil {
				continue
			}
			icon := "\uf240"
			color := cGreen
			switch {
			case pct <= 10:
				icon = "\uf244"
				color = cRed
			case pct <= 25:
				icon = "\uf243"
				color = cOrange
			case pct <= 50:
				icon = "\uf242"
				color = cYellow
			case pct <= 75:
				icon = "\uf241"
				color = cGreen
			}
			if strings.Contains(line, "charging") && !strings.Contains(line, "discharging") {
				icon = "\uf0e7"
			}
			if icons {
				return fc(color, icon+" "+strconv.Itoa(pct)+"%")
			}
			return fc(color, strconv.Itoa(pct)+"%")
		}
	}
	return ""
}

// ── Git helpers ──────────────────────────────────────────────────

func gitBranch(cwd string) (branch string, repoDir string) {
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			var headPath string
			if info.IsDir() {
				headPath = filepath.Join(gitPath, "HEAD")
			} else {
				// Worktree/submodule: .git is a file
				data, err := os.ReadFile(gitPath)
				if err != nil {
					return "", ""
				}
				gitDir := strings.TrimSpace(strings.TrimPrefix(string(data), "gitdir: "))
				if !filepath.IsAbs(gitDir) {
					gitDir = filepath.Join(dir, gitDir)
				}
				headPath = filepath.Join(gitDir, "HEAD")
			}
			data, err := os.ReadFile(headPath)
			if err != nil {
				return "", ""
			}
			head := strings.TrimSpace(string(data))
			if strings.HasPrefix(head, "ref: refs/heads/") {
				return head[16:], dir
			}
			if len(head) >= 8 {
				return head[:8], dir
			}
			return "", ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
}

func gitDirty(repoDir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		// Timeout or error — show branch only, no lag
		return ""
	}

	staged, modified, untracked := false, false, false
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		switch {
		case x == '?':
			untracked = true
		default:
			if x != ' ' {
				staged = true
			}
			if y != ' ' {
				modified = true
			}
		}
	}

	if !staged && !modified && !untracked {
		return fc(cGreen, "✓")
	}

	var parts []string
	if staged {
		parts = append(parts, fc(cGreen, "+"))
	}
	if modified {
		parts = append(parts, fc(cRed, "!"))
	}
	if untracked {
		parts = append(parts, fc(cYellow, "?"))
	}
	return strings.Join(parts, "")
}

// ── Helpers ──────────────────────────────────────────────────────

func hasMarkerUp(cwd string, markers []string) bool {
	dir := cwd
	for {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func visibleWidth(s string) int {
	stripped := s
	for _, prefix := range []string{"%F{", "%K{"} {
		for {
			idx := strings.Index(stripped, prefix)
			if idx == -1 {
				break
			}
			end := strings.Index(stripped[idx:], "}")
			if end == -1 {
				break
			}
			stripped = stripped[:idx] + stripped[idx+end+1:]
		}
	}
	for _, esc := range []string{"%f", "%k"} {
		stripped = strings.ReplaceAll(stripped, esc, "")
	}
	stripped = strings.ReplaceAll(stripped, "%%", "%")
	return utf8.RuneCountInString(stripped)
}

func escShell(s string) string {
	// Escape for zsh $'...' quoting
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func joinSegs(segs []seg, sep string) string {
	parts := make([]string, len(segs))
	for i, s := range segs {
		parts[i] = s.text
	}
	return strings.Join(parts, sep)
}
