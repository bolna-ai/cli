package cli

import (
	"fmt"
	"os"

	"github.com/bolna-ai/cli/internal/api"
	"github.com/bolna-ai/cli/internal/auth"
	"github.com/bolna-ai/cli/internal/config"
	"github.com/bolna-ai/cli/internal/tui/styles"
	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
)

// BuildInfo carries version metadata injected by goreleaser's -ldflags.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// appCtx holds everything a command needs beyond its own flags: global
// output flags, the resolved profile, and lazily-built API client/theme.
type appCtx struct {
	profile string
	json    bool
	output  string
	noColor bool
	verbose bool
	build   BuildInfo
}

// Format resolves the effective output format for the current invocation:
// --json is a shorthand for -o json kept for muscle memory; -o/--output
// otherwise picks table (default)/json/csv.
func (a *appCtx) Format() string {
	if a.json {
		return "json"
	}
	switch a.output {
	case "json", "csv", "table":
		return a.output
	default:
		return "table"
	}
}

// IsTTY reports whether stdout is an interactive terminal. Every command
// uses this to decide between rich/TUI output and plain/JSON output, so
// piping or CI never triggers colors, spinners, or animation.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// Theme resolves the active color theme from saved config, honoring
// --no-color by returning plain (uncolored) styles.
func (a *appCtx) Theme() styles.Theme {
	if a.noColor {
		return styles.New(styles.Palette{}) // zero-value AdaptiveColors render as default terminal color
	}
	cfg, _ := config.Load()
	return styles.New(styles.ByName(cfg.Theme))
}

// Client resolves the API key for the active profile (env override first,
// then keychain) and returns a ready-to-use Bolna API client.
func (a *appCtx) Client() (*api.Client, error) {
	key, source, err := auth.Resolve(a.profile)
	if err != nil {
		return nil, err
	}
	if source == auth.SourceNone {
		return nil, fmt.Errorf("not logged in — run `bolna login` or set %s", auth.EnvVar)
	}
	return api.New(key), nil
}

// ClientOrLogin is like Client, but on a TTY with no key configured it
// offers to log in right there instead of just erroring — so a brand new
// `bolna` invocation goes straight from "never used this before" to a
// working dashboard/command in one prompt, rather than a dead end pointing
// at a separate `bolna login` step.
func (a *appCtx) ClientOrLogin() (*api.Client, error) {
	client, err := a.Client()
	if err == nil {
		return client, nil
	}
	if !IsTTY() {
		return nil, err
	}
	_, source, resolveErr := auth.Resolve(a.profile)
	if resolveErr != nil || source != auth.SourceNone {
		return nil, err // a real error (bad keychain, etc.), not "no key yet"
	}

	confirmed := false
	if huhErr := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("You're not logged in yet").
			Description("Log in with a Bolna API key now?").
			Value(&confirmed),
	)).Run(); huhErr != nil {
		return nil, huhErr
	}
	if !confirmed {
		return nil, err
	}

	if _, loginErr := interactiveLogin(a, ""); loginErr != nil {
		return nil, loginErr
	}
	return a.Client()
}
