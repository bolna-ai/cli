package cli

import (
	"encoding/json"
	"fmt"

	"github.com/bolna-ai/bolna-cli/internal/api"
	"github.com/bolna-ai/bolna-cli/internal/auth"
	"github.com/bolna-ai/bolna-cli/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// interactiveLogin prompts for (or uses apiKeyFlag as) an API key, validates
// it against the live API, and stores it in the OS keychain. Shared by
// `bolna login` and the inline "not logged in — log in now?" flow that
// other commands fall into via appCtx.ClientOrLogin.
func interactiveLogin(a *appCtx, apiKeyFlag string) (api.UserInfo, error) {
	apiKey := apiKeyFlag
	theme := a.Theme()
	if apiKey == "" {
		if !IsTTY() {
			return nil, fmt.Errorf("no --api-key given and stdin is not a terminal; pass --api-key or set %s", auth.EnvVar)
		}
		fmt.Println(theme.Muted.Render("Get a key from the Bolna dashboard → Developers tab."))
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Title("Bolna API key").
				EchoMode(huh.EchoModePassword).
				Value(&apiKey).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API key cannot be empty")
					}
					return nil
				}),
		))
		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	client := api.New(apiKey)
	fmt.Println(theme.Muted.Render("Validating key against the Bolna API…"))
	info, err := client.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("could not validate API key: %w", err)
	}

	if err := auth.Store(a.profile, apiKey); err != nil {
		return nil, fmt.Errorf("saving key to OS keychain: %w", err)
	}
	cfg, _ := config.Load()
	if _, err := config.AddProfile(cfg, profileOrDefault(a.profile)); err != nil {
		return nil, fmt.Errorf("saving profile: %w", err)
	}
	return info, nil
}

func newLoginCmd(a *appCtx) *cobra.Command {
	var apiKeyFlag string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store a Bolna API key in the OS keychain",
		Long: "Validates a Bolna API key against the account API and stores it in the OS\n" +
			"keychain (macOS Keychain / Linux Secret Service / Windows Credential Manager)\n" +
			"under the active profile. Get a key from the Bolna dashboard's Developers tab.",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := interactiveLogin(a, apiKeyFlag)
			if err != nil {
				return err
			}
			theme := a.Theme()
			name := info.Name()
			if name == "" {
				name = info.Email()
			}
			fmt.Println(theme.Success.Render("✓ Logged in") + theme.Muted.Render(" as "+name))
			fmt.Println()
			fmt.Println(theme.Muted.Render("Try next:"))
			fmt.Println("  " + theme.Bold.Render("bolna doctor") + theme.Muted.Render("       — sanity-check config, keychain, network"))
			fmt.Println("  " + theme.Bold.Render("bolna agents list") + theme.Muted.Render("  — see your agents"))
			fmt.Println("  " + theme.Bold.Render("bolna") + theme.Muted.Render("               — open the full dashboard"))
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "API key to store (skips the interactive prompt; useful for scripting)")
	return cmd
}

func newLogoutCmd(a *appCtx) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored Bolna API key for the active profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Delete(a.profile); err != nil {
				return fmt.Errorf("removing key from OS keychain: %w", err)
			}
			fmt.Println(a.Theme().Success.Render("✓ Logged out"))
			return nil
		},
	}
}

func newWhoamiCmd(a *appCtx) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show account profile, wallet balance, and concurrency limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			info, err := client.GetUserInfo()
			if err != nil {
				return friendlyAPIErr(err, "")
			}

			if a.Format() == "json" {
				return printJSON(info)
			}

			theme := a.Theme()
			var rows []string
			if name := info.Name(); name != "" {
				rows = append(rows, theme.Bold.Render(name))
			}
			if email := info.Email(); email != "" {
				rows = append(rows, theme.Muted.Render(email))
			}
			if bal, ok := info.Balance(); ok {
				rows = append(rows, fmt.Sprintf("%s %s", theme.Subtitle.Render("Wallet balance:"), theme.Success.Render(fmt.Sprintf("$%.2f", bal))))
			}
			if current, max, ok := info.Concurrency(); ok {
				rows = append(rows, fmt.Sprintf("%s %d/%d", theme.Subtitle.Render("Concurrency:"), current, max))
			}
			if len(rows) == 0 {
				rows = append(rows, theme.Muted.Render("(no displayable fields — use --json for the raw response)"))
			}

			fmt.Println(theme.Card.Render(lipgloss.JoinVertical(lipgloss.Left, rows...)))
			return nil
		},
	}
}

func profileOrDefault(p string) string {
	if p == "" {
		return "default"
	}
	return p
}

func printJSON(v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

// friendlyAPIErr renders api.APIError with an actionable message; hint is
// appended for 404s (e.g. "call `bolna agents list` to see valid IDs").
func friendlyAPIErr(err error, hint string) error {
	if apiErr, ok := err.(*api.APIError); ok {
		return fmt.Errorf("%s", apiErr.Friendly(hint))
	}
	return err
}
