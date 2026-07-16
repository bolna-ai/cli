package cli

import (
	"fmt"
	"time"

	"github.com/bolna-ai/bolna-cli/internal/api"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func newCallCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Place and inspect outbound calls",
	}
	cmd.AddCommand(newCallStartCmd(a))
	return cmd
}

func newCallStartCmd(a *appCtx) *cobra.Command {
	var recipient, from string
	var yes bool
	cmd := &cobra.Command{
		Use:   "start <agent-id>",
		Short: "Place a real outbound call, spends account balance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			theme := a.Theme()

			if recipient == "" {
				if !IsTTY() {
					return fmt.Errorf("--to is required in non-interactive mode")
				}
				if err := huh.NewForm(huh.NewGroup(
					huh.NewInput().Title("Recipient phone number (E.164)").Description("e.g. +14155552671").Value(&recipient),
				)).Run(); err != nil {
					return err
				}
			}

			if !yes {
				info, infoErr := client.GetUserInfo()
				fmt.Println(theme.Warning.Render("⚠ This places a real phone call and spends account balance."))
				fmt.Printf("  Agent:     %s\n", agentID)
				fmt.Printf("  Recipient: %s\n", recipient)
				if infoErr == nil {
					if bal, ok := info.Balance(); ok {
						fmt.Printf("  Wallet:    $%.2f\n", bal)
					}
				}
				if !IsTTY() {
					return fmt.Errorf("refusing to place a call without confirmation in a non-interactive session; pass --yes")
				}
				confirmed := false
				if err := huh.NewForm(huh.NewGroup(
					huh.NewConfirm().Title("Place this call?").Value(&confirmed),
				)).Run(); err != nil {
					return err
				}
				if !confirmed {
					fmt.Println(theme.Muted.Render("Cancelled."))
					return nil
				}
			}

			result, err := client.StartCall(api.StartCallInput{
				AgentID:              agentID,
				RecipientPhoneNumber: recipient,
				FromPhoneNumber:      from,
			})
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}
			if a.Format() == "json" {
				return printJSON(result)
			}
			id, _ := result["execution_id"].(string)
			if id == "" {
				id, _ = result["id"].(string)
			}
			fmt.Println(theme.Success.Render("✓ Call started"))
			if id != "" {
				fmt.Println(theme.Muted.Render("Execution ID: " + id))
				fmt.Println(theme.Muted.Render("Track it with: bolna calls view " + id))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&recipient, "to", "", "recipient phone number, E.164 (e.g. +14155552671)")
	cmd.Flags().StringVar(&from, "from", "", "from-number override, E.164 (must belong to your account)")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the cost/confirmation prompt (scripting)")
	return cmd
}

func newCallsCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calls",
		Short: "Inspect call history and execution details",
	}
	cmd.AddCommand(newCallsListCmd(a), newCallsViewCmd(a))
	return cmd
}

func newCallsListCmd(a *appCtx) *cobra.Command {
	var from, to string
	var page, pageSize int
	var quiet bool
	cmd := &cobra.Command{
		Use:     "list <agent-id>",
		Aliases: []string{"ls"},
		Short:   "Call history for one agent, defaults to the last 7 days",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			input := api.ListExecutionsInput{AgentID: args[0], PageNumber: page, PageSize: pageSize}
			if from != "" {
				t, err := time.Parse(time.RFC3339, from)
				if err != nil {
					return fmt.Errorf("--from must be an ISO 8601 UTC timestamp: %w", err)
				}
				input.From = t
			}
			if to != "" {
				t, err := time.Parse(time.RFC3339, to)
				if err != nil {
					return fmt.Errorf("--to must be an ISO 8601 UTC timestamp: %w", err)
				}
				input.To = t
			}

			execPage, err := client.ListAgentExecutions(input)
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}

			headers := []string{"EXECUTION ID", "STATUS", "DURATION", "TO", "CREATED"}
			rows := make([][]string, len(execPage.Data))
			for i, e := range execPage.Data {
				to := ""
				if e.TelephonyData != nil {
					to = e.TelephonyData.ToNumber
				}
				rows[i] = []string{e.ID, e.Status, fmtDuration(e.ConversationDuration), orDash(to), e.CreatedAt}
			}
			if err := a.renderList(headers, rows, 0, 1, execPage, quiet); err != nil {
				return err
			}
			if !quiet && a.Format() == "table" && execPage.HasMore {
				theme := a.Theme()
				fmt.Println(theme.Muted.Render(fmt.Sprintf("\n… %d more (page %d of ~%d, page-size %d). Use --page to page further.",
					execPage.Total-len(execPage.Data), execPage.PageNumber, (execPage.Total+execPage.PageSize-1)/execPage.PageSize, execPage.PageSize)))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start of window, ISO 8601 UTC (default: 7 days ago)")
	cmd.Flags().StringVar(&to, "to", "", "end of window, ISO 8601 UTC (default: now)")
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "page size, max 50")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print only execution IDs, one per line (for scripting)")
	return cmd
}

func newCallsViewCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <execution-id>",
		Short: "Full call detail — transcript, status, cost, telephony data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			execution, err := client.GetExecution(args[0])
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna calls list <agent-id>` to see valid execution IDs.")
			}
			if a.Format() == "json" {
				return printJSON(execution)
			}

			theme := a.Theme()
			header := theme.Title.Render(execution.ID()) + "  " + theme.StatusColor(execution.Status()).Render(orDash(execution.Status()))
			fmt.Println(header)
			for _, key := range []string{"conversation_duration", "cost", "conversation_time"} {
				if v, ok := execution[key]; ok {
					fmt.Printf("%s %v\n", theme.Subtitle.Render(key+":"), v)
				}
			}
			if transcript := execution.Transcript(); transcript != "" {
				rendered, err := glamour.Render("**Transcript**\n\n"+transcript, "auto")
				if err == nil {
					fmt.Println(rendered)
				} else {
					fmt.Println(theme.Subtitle.Render("Transcript"))
					fmt.Println(transcript)
				}
			} else {
				fmt.Println(theme.Muted.Render("(no transcript field on this execution — use --json for the raw response)"))
			}
			return nil
		},
	}
	return cmd
}
