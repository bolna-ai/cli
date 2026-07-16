package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bolna-ai/bolna-cli/internal/api"
	"github.com/bolna-ai/bolna-cli/internal/auth"
	"github.com/bolna-ai/bolna-cli/internal/config"
	"github.com/bolna-ai/bolna-cli/internal/tui"
	"github.com/spf13/cobra"
)

func newDoctorCmd(a *appCtx) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config, keychain, network, and API key health",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := doctorChecks(a.profile)
			theme := a.Theme()

			var allOK bool
			var err error
			if IsTTY() {
				allOK, err = tui.RunDoctor(checks, theme)
			} else {
				allOK = tui.RunDoctorPlain(checks, theme)
			}
			if err != nil {
				return err
			}
			if !allOK {
				return fmt.Errorf("one or more checks failed")
			}
			return nil
		},
	}
}

func doctorChecks(profile string) []tui.Check {
	return []tui.Check{
		{
			Name: "Config directory is writable",
			Run: func() (bool, string) {
				if _, err := config.Load(); err != nil {
					return false, err.Error()
				}
				if err := config.Save(mustLoadConfig()); err != nil {
					return false, err.Error()
				}
				return true, ""
			},
		},
		{
			Name: "OS keychain is accessible",
			Run: func() (bool, string) {
				const probe = "__bolna_doctor_probe__"
				if err := auth.Store(probe, "probe"); err != nil {
					return false, err.Error()
				}
				defer auth.Delete(probe)
				return true, ""
			},
		},
		{
			Name: "API key is configured",
			Run: func() (bool, string) {
				_, source, err := auth.Resolve(profile)
				if err != nil {
					return false, err.Error()
				}
				if source == auth.SourceNone {
					return false, "run `bolna login` or set " + auth.EnvVar
				}
				return true, "source: " + string(source)
			},
		},
		{
			Name: "Bolna API is reachable",
			Run: func() (bool, string) {
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(api.DefaultBaseURL)
				if err != nil {
					return false, err.Error()
				}
				defer resp.Body.Close()
				return true, fmt.Sprintf("HTTP %d from %s", resp.StatusCode, api.DefaultBaseURL)
			},
		},
		{
			Name: "API key is valid",
			Run: func() (bool, string) {
				key, source, err := auth.Resolve(profile)
				if err != nil || source == auth.SourceNone {
					return false, "skipped — no API key configured"
				}
				client := api.New(key)
				info, err := client.GetUserInfo()
				if err != nil {
					if apiErr, ok := err.(*api.APIError); ok {
						return false, apiErr.Friendly("")
					}
					return false, err.Error()
				}
				name := info.Name()
				if name == "" {
					name = info.Email()
				}
				return true, "authenticated as " + name
			},
		},
		{
			Name: "Terminal supports an interactive TUI",
			Run: func() (bool, string) {
				if !IsTTY() {
					return false, "stdout is not a TTY — TUI/color output disabled, falling back to plain/JSON"
				}
				return true, ""
			},
		},
	}
}

func mustLoadConfig() config.Config {
	cfg, _ := config.Load()
	return cfg
}
