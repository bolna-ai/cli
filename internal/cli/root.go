package cli

import (
	"fmt"

	"github.com/bolna-ai/cli/internal/tui"
	"github.com/spf13/cobra"
)

// Execute builds and runs the full bolna command tree.
func Execute(build BuildInfo) error {
	a := &appCtx{build: build}

	root := &cobra.Command{
		Use:   "bolna",
		Short: "The Bolna Voice AI CLI — manage agents, calls, numbers, and batches",
		Long: "bolna is a CLI and full-screen TUI for Bolna Voice AI.\n\n" +
			"Run `bolna` with no arguments in a terminal to open the interactive\n" +
			"dashboard. Every dashboard view has an equivalent scriptable command\n" +
			"below — run `bolna <command> --help` (or `bolna help <command>`) for\n" +
			"that command's own flags. Every command supports -o/--output\n" +
			"table|json|csv, and `list` commands also support -q/--quiet for\n" +
			"bare-ID output — handy for piping into xargs or other scripts.\n\n" +
			"First time here? Just run `bolna` — it'll offer to log you in.",
		Example: "  bolna                                          open the dashboard\n" +
			"  bolna login                                    store an API key in the OS keychain\n" +
			"  bolna agents list -o json                      list agents as JSON\n" +
			"  bolna agents list -q | xargs -I{} bolna agents view {}\n" +
			"  bolna calls list <agent-id>                    an agent's recent call history\n" +
			"  bolna call start <agent-id> --to +14155552671  place a call",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			switch a.output {
			case "", "table", "json", "csv":
				return nil
			default:
				return fmt.Errorf("invalid --output %q: must be table, json, or csv", a.output)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !IsTTY() {
				return cmd.Help()
			}
			client, err := a.ClientOrLogin()
			if err != nil {
				return fmt.Errorf("%w\n\nRun `bolna login` first, or see `bolna --help`.", err)
			}
			return tui.RunDashboard(client, a.Theme())
		},
	}

	root.PersistentFlags().StringVar(&a.profile, "profile", "", "named credential profile (default: \"default\")")
	root.PersistentFlags().StringVarP(&a.output, "output", "o", "table", "output format: table, json, or csv")
	root.PersistentFlags().BoolVar(&a.json, "json", false, "shorthand for --output json")
	root.PersistentFlags().BoolVar(&a.noColor, "no-color", false, "disable colored output")
	root.PersistentFlags().BoolVarP(&a.verbose, "verbose", "v", false, "verbose logging")

	root.AddGroup(
		&cobra.Group{ID: "account", Title: "Account:"},
		&cobra.Group{ID: "agents", Title: "Agents:"},
		&cobra.Group{ID: "calls", Title: "Calls:"},
		&cobra.Group{ID: "resources", Title: "Resources:"},
		&cobra.Group{ID: "utility", Title: "Utility:"},
	)
	root.SetHelpCommandGroupID("utility")
	root.SetCompletionCommandGroupID("utility")

	grouped := func(groupID string, cmds ...*cobra.Command) []*cobra.Command {
		for _, c := range cmds {
			c.GroupID = groupID
		}
		return cmds
	}

	root.AddCommand(grouped("account", newLoginCmd(a), newLogoutCmd(a), newWhoamiCmd(a))...)
	root.AddCommand(grouped("agents", newAgentsCmd(a))...)
	root.AddCommand(grouped("calls", newCallCmd(a), newCallsCmd(a))...)
	root.AddCommand(grouped("resources", newNumbersCmd(a), newBatchesCmd(a))...)
	root.AddCommand(grouped("utility", newDoctorCmd(a), newVersionCmd(a))...)

	return root.Execute()
}
