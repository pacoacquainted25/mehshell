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
	return fmt.Sprintf("%%F{%d}%s%%f", color, s)
}

// ── Segment with sort order ──────────────────────────────────────

type seg struct {
	text  string
	order int
}

// ── Main ─────────────────────────────────────────────────────────

const zshInit = `zmodload zsh/datetime 2>/dev/null
typeset -gi _mehshell_ts=0
# Instant prompt: show cached prompt while plugins load
[[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/mehshell-prompt-cache" ]] && source "${XDG_CACHE_HOME:-$HOME/.cache}/mehshell-prompt-cache"
_mehshell_preexec() { _mehshell_ts=$EPOCHSECONDS }
_mehshell_precmd() {
  local e=$? d=0
  (( _mehshell_ts > 0 )) && d=$(( EPOCHSECONDS - _mehshell_ts ))
  _mehshell_ts=0
  local _out="$(mehshell $e $d $COLUMNS)"
  eval "$_out"
  print -r -- "$_out" >| "${XDG_CACHE_HOME:-$HOME/.cache}/mehshell-prompt-cache" 2>/dev/null
}
# Vi mode: swap prompt char on keymap change
_mehshell_zle_keymap_select() {
  [[ $KEYMAP == vicmd ]] && PROMPT="${PROMPT/❯/❮}" || PROMPT="${PROMPT/❮/❯}"
  zle reset-prompt
}
# Transient prompt: simplify previous prompt on Enter
_mehshell_accept_line() {
  PROMPT=$'%F{76}❯%f '
  zle reset-prompt
  zle .accept-line
}
zle -N zle-keymap-select _mehshell_zle_keymap_select
zle -N accept-line _mehshell_accept_line
preexec_functions+=(_mehshell_preexec)
precmd_functions+=(_mehshell_precmd)`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("mehshell", Version)
			return
		case "init":
			if len(os.Args) > 2 && os.Args[2] == "zsh" {
				fmt.Println(zshInit)
			} else {
				fmt.Fprintln(os.Stderr, "usage: mehshell init zsh")
				os.Exit(1)
			}
			return
		}
	}

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

	add := func(dst *[]seg, order int, fn func() string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if t := fn(); t != "" {
				mu.Lock()
				*dst = append(*dst, seg{t, order})
				mu.Unlock()
			}
		}()
	}

	// Left segments
	add(&left, 0, func() string { return segOS() })
	add(&left, 1, func() string { return segDir(cwd, home) })
	add(&left, 2, func() string { return segGit(cwd) })
	add(&left, 3, func() string { return segNode(cwd) })
	add(&left, 4, func() string { return segPython(cwd) })
	add(&left, 5, func() string { return segGo(cwd) })
	add(&left, 6, func() string { return segRust(cwd) })
	add(&left, 7, func() string { return segRuby(cwd) })
	add(&left, 8, func() string { return segJava(cwd) })

	// Right segments
	add(&right, 0, func() string { return segConda() })
	add(&right, 1, func() string { return segVenv() })
	add(&right, 2, func() string { return segK8s(cwd, home) })
	add(&right, 3, func() string { return segTerraform(cwd) })
	add(&right, 4, func() string { return segDocker(cwd) })
	add(&right, 5, func() string { return segAWS() })
	add(&right, 6, func() string { return segAzure() })
	add(&right, 7, func() string { return segGCloud() })
	add(&right, 8, func() string { return segBattery() })
	add(&right, 9, func() string { return segDuration(duration) })
	add(&right, 10, func() string { return segTime() })

	wg.Wait()

	sort.Slice(left, func(i, j int) bool { return left[i].order < left[j].order })
	sort.Slice(right, func(i, j int) bool { return right[i].order < right[j].order })

	leftStr := joinSegs(left, " ")
	rightStr := joinSegs(right, "  ")

	// Right-align line 1: pad between left and right
	prefix := fc(cCyan, "╭─") + " "
	prefixVis := 3 // "╭─ "
	leftVis := visibleWidth(leftStr)
	rightVis := visibleWidth(rightStr)
	pad := columns - prefixVis - leftVis - rightVis
	if pad < 1 {
		pad = 1
	}

	line1 := prefix + leftStr + strings.Repeat(" ", pad) + rightStr

	// Prompt char
	char := fc(cGreen, "❯")
	if exitCode != 0 {
		char = fc(cRed, "❯")
	}
	line2 := fc(cCyan, "╰─") + char

	// Escape for $'...' quoting
	l1 := escShell(line1)
	l2 := escShell(line2)

	fmt.Printf("PROMPT=$'\\n%s\\n%s '\n", l1, l2)
}

// ── Segments ─────────────────────────────────────────────────────

func segOS() string {
	icon := "\uf17c" // linux tux
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

func segGit(cwd string) string {
	branch, repoDir := gitBranch(cwd)
	if branch == "" {
		return ""
	}

	dirty := gitDirty(repoDir)
	result := fc(cBlue, "\uf126 ") + fc(cMagenta, branch)
	if dirty != "" {
		result += " " + dirty
	}
	return result
}

func segNode(cwd string) string {
	if _, err := os.Stat(filepath.Join(cwd, "package.json")); err != nil {
		return ""
	}
	// Try reading version files first (no fork)
	for _, f := range []string{".node-version", ".nvmrc"} {
		if data, err := os.ReadFile(filepath.Join(cwd, f)); err == nil {
			ver := strings.TrimSpace(string(data))
			if ver != "" {
				return fc(cGreen, "\ue718 "+ver)
			}
		}
	}
	// Fallback: run node --version
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "node", "--version").Output(); err == nil {
		return fc(cGreen, "\ue718 "+strings.TrimSpace(string(out)))
	}
	return ""
}

func segPython(cwd string) string {
	if !hasMarkerUp(cwd, []string{".python-version", "pyproject.toml", "setup.py", "Pipfile", "requirements.txt"}) {
		return ""
	}
	if data, err := os.ReadFile(filepath.Join(cwd, ".python-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cYellow, "\ue73c "+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "python", "--version").Output(); err == nil {
		ver := strings.TrimPrefix(strings.TrimSpace(string(out)), "Python ")
		return fc(cYellow, "\ue73c "+ver)
	}
	return ""
}

func segGo(cwd string) string {
	gomod := filepath.Join(cwd, "go.mod")
	data, err := os.ReadFile(gomod)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") {
			ver := strings.Fields(line)[1]
			return fc(cCyan, "\ue724 "+ver)
		}
	}
	return ""
}

func segConda() string {
	env := os.Getenv("CONDA_DEFAULT_ENV")
	if env == "" || env == "base" {
		return ""
	}
	return fc(cGreen, "\ue73c "+env)
}

func segVenv() string {
	venv := os.Getenv("VIRTUAL_ENV")
	if venv == "" {
		return ""
	}
	return fc(cGreen, "\ue73c "+filepath.Base(venv))
}

func segK8s(cwd, home string) string {
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
				return fc(cBlue, "\u2388 "+ctx)
			}
		}
	}
	return ""
}

func segAWS() string {
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		return ""
	}
	return fc(cOrange, "\uf52c "+profile)
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

func segRust(cwd string) string {
	if _, err := os.Stat(filepath.Join(cwd, "Cargo.toml")); err != nil {
		return ""
	}
	for _, f := range []string{"rust-toolchain", ".rust-version"} {
		if data, err := os.ReadFile(filepath.Join(cwd, f)); err == nil {
			ver := strings.TrimSpace(string(data))
			if ver != "" {
				return fc(cOrange, "\ue7a8 "+ver)
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
						return fc(cOrange, "\ue7a8 "+ver)
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
		return fc(cOrange, "\ue7a8 "+ver)
	}
	return ""
}

func segRuby(cwd string) string {
	if !hasMarkerUp(cwd, []string{".ruby-version", "Gemfile", "Rakefile"}) {
		return ""
	}
	if data, err := os.ReadFile(filepath.Join(cwd, ".ruby-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cRed, "\ue739 "+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "ruby", "--version").Output(); err == nil {
		ver := strings.TrimPrefix(strings.TrimSpace(string(out)), "ruby ")
		if i := strings.Index(ver, " "); i != -1 {
			ver = ver[:i]
		}
		return fc(cRed, "\ue739 "+ver)
	}
	return ""
}

func segJava(cwd string) string {
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
	if data, err := os.ReadFile(filepath.Join(cwd, ".java-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cOrange, "\ue738 "+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "java", "-version").CombinedOutput(); err == nil {
		line := strings.SplitN(string(out), "\n", 2)[0]
		if start := strings.Index(line, "\""); start != -1 {
			if end := strings.Index(line[start+1:], "\""); end != -1 {
				return fc(cOrange, "\ue738 "+line[start+1:start+1+end])
			}
		}
	}
	return ""
}

func segTerraform(cwd string) string {
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
	if data, err := os.ReadFile(filepath.Join(cwd, ".terraform-version")); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			return fc(cMagenta, "\uf0ac "+ver)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "terraform", "version").Output(); err == nil {
		line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		line = strings.TrimPrefix(line, "Terraform ")
		return fc(cMagenta, "\uf0ac "+line)
	}
	return ""
}

func segDocker(cwd string) string {
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
	if ctx := os.Getenv("DOCKER_CONTEXT"); ctx != "" && ctx != "default" {
		return fc(cCyan, "\uf308 "+ctx)
	}
	if name := os.Getenv("DOCKER_MACHINE_NAME"); name != "" {
		return fc(cCyan, "\uf308 "+name)
	}
	return fc(cCyan, "\uf308")
}

func segAzure() string {
	acct := os.Getenv("AZURE_DEFAULTS_GROUP")
	if acct == "" {
		return ""
	}
	return fc(cBlue, "\ufd03 "+acct)
}

func segGCloud() string {
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
	return fc(cGreen, "\uf1a0 "+project)
}

func segBattery() string {
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
			return fc(color, icon+" "+strconv.Itoa(pct)+"%")
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
			return fc(color, icon+" "+strconv.Itoa(pct)+"%")
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
	// Strip zsh prompt escapes: %F{N}...%f, %B, %b, etc.
	stripped := s
	for {
		idx := strings.Index(stripped, "%F{")
		if idx == -1 {
			break
		}
		end := strings.Index(stripped[idx:], "}")
		if end == -1 {
			break
		}
		stripped = stripped[:idx] + stripped[idx+end+1:]
	}
	stripped = strings.ReplaceAll(stripped, "%f", "")
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
