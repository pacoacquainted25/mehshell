# PROJECT KNOWLEDGE BASE

**Generated:** 2026-03-13
**Commit:** 4a7f0b0
**Branch:** master

## OVERVIEW

Fast, parallelized zsh prompt engine. Single Go binary, stdlib-only, ~6ms per prompt. Outputs `PROMPT` variable via `eval "$(mehshell $e $d $COLUMNS)"`. Supports instant prompt, transient prompt, and vi mode.

## STRUCTURE

```
mehshell/
├── main.go              # ALL application code (497 lines, 20 functions)
├── go.mod               # Go 1.22, zero dependencies
├── .goreleaser.yml      # Cross-platform release builds
├── .github/workflows/
│   └── release.yml      # Tag-triggered GoReleaser CI
├── README.md
└── .gitignore           # Excludes: mehshell binary, dist/
```

Single-file monolith. No packages, no internal/, no cmd/. Intentional — project is <500 lines.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add/modify prompt segment | `main.go` L128-153 (registration), segment func below | Follow `seg*()` pattern |
| Git detection | `main.go` L353-444 | Zero-fork branch via `.git/HEAD`, 150ms timeout on dirty |
| Zsh init script | `main.go` L50-61 | Embedded `precmd`/`preexec` hooks |
| Color constants | `main.go` L23-32 | Zsh 256-color codes |
| Zsh init script | `main.go` L50-61 | Embedded `precmd`/`preexec` hooks |
| Version injection | `main.go` L19 | Set via ldflags: `-X main.Version={{.Version}}` |
| Release config | `.goreleaser.yml` | Linux + macOS, amd64 + arm64, CGO off |
| CI pipeline | `.github/workflows/release.yml` | Triggers on `v*` tags |

## ARCHITECTURE

### Execution Flow
1. Zsh `precmd` hook calls `mehshell <exitCode> <duration> <columns>`
2. `main()` spawns 20 goroutines (9 left + 11 right segments) via `sync.WaitGroup`
3. Each `seg*()` returns formatted string or `""` (skip)
4. Mutex-protected append to `left`/`right` slices, sorted by `order`
5. Right-aligned line 1 + prompt char line 2 → `PROMPT=$'...'`

### Segment Registration Pattern
```go
add(&left, ORDER, func() string { return segFoo(args) })
```
- `add()` wraps goroutine + WaitGroup + mutex
- Return `""` to hide segment
- `order` int controls display position

### Segment Functions
| Function | Side | Order | Trigger |
|----------|------|-------|---------|
| `segOS` | left | 0 | Always (reads `/etc/os-release`) |
| `segDir` | left | 1 | Always |
| `segGit` | left | 2 | `.git` found walking up |
| `segNode` | left | 3 | `package.json` in cwd |
| `segPython` | left | 4 | Marker files walking up |
| `segGo` | left | 5 | `go.mod` in cwd |
| `segRust` | left | 6 | `Cargo.toml` in cwd |
| `segRuby` | left | 7 | Marker files walking up |
| `segJava` | left | 8 | `pom.xml`, `build.gradle` in cwd |
| `segConda` | right | 0 | `$CONDA_DEFAULT_ENV` set (not "base") |
| `segVenv` | right | 1 | `$VIRTUAL_ENV` set |
| `segK8s` | right | 2 | K8s manifest in cwd |
| `segTerraform` | right | 3 | `.tf` files in cwd |
| `segDocker` | right | 4 | Dockerfile/compose in cwd |
| `segAWS` | right | 5 | `$AWS_PROFILE` set |
| `segAzure` | right | 6 | `$AZURE_DEFAULTS_GROUP` set |
| `segGCloud` | right | 7 | `$CLOUDSDK_CORE_PROJECT` / `$GCLOUD_PROJECT` set |
| `segBattery` | right | 8 | Battery present (Linux `/sys/`, macOS `pmset`) |
| `segDuration` | right | 9 | Duration ≥ 3s |
| `segTime` | right | 10 | Always |

## CONVENTIONS

- **Naming**: `seg*()` for segments, `git*()` for git helpers, short vars (`cwd`, `wg`, `mu`)
- **Colors**: Use `fc(cColor, text)` helper — never raw `%F{N}` strings
- **Error handling**: Return `""` on any error. Never panic, never log.
- **External commands**: Always wrap in `context.WithTimeout` (100-150ms)
- **File reads over forks**: Prefer `os.ReadFile` over `exec.Command` when possible
- **Section headers**: ASCII box drawing (`// ── Section ──────`)
- **No dependencies**: Stdlib only. Do not add external modules.

## ANTI-PATTERNS

- **No `as any` equivalent**: Don't use blank identifier to ignore important errors
- **No blocking commands**: Every `exec.Command` MUST have a context timeout
- **No `log.*` or `fmt.Println` to stderr**: Silent operation only (stdout is eval'd by zsh)
- **Don't print to stdout arbitrarily**: Output is `eval`'d — stray output breaks the shell

## COMMANDS

```bash
# Build
go build

# Run (simulates prompt generation)
./mehshell 0 5 120

# Init script
./mehshell init zsh

# Version
./mehshell --version

# Release (CI only, via tag)
git tag v0.X.X && git push origin v0.X.X
```

## NOTES

- **Binary in repo**: Compiled `mehshell` ELF exists at root (gitignored but present). Don't confuse with source.
- **No tests**: No `*_test.go` files exist. `go test` has nothing to run.
- **No linter config**: No golangci-lint, relies on `gofmt` defaults.
- **Nerd Font required**: Icons render as boxes without a patched font.
- **Zsh only**: `init` subcommand only supports `zsh`. No bash/fish/etc.
- **K8s segment**: Only shows when K8s manifest files exist in cwd (skaffold.yaml, Chart.yaml, etc.), not just when kubectl is available.
- **Node segment**: Only shows when `package.json` exists in cwd.
- **Python segment**: Uses `hasMarkerUp()` — walks parent dirs for markers.
