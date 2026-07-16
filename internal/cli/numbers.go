package cli

import (
	"github.com/spf13/cobra"
)

func newNumbersCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "numbers",
		Aliases: []string{"number"},
		Short:   "Phone numbers on the account",
	}
	cmd.AddCommand(newNumbersListCmd(a))
	return cmd
}

func newNumbersListCmd(a *appCtx) *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Phone numbers on the account and their linked agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			numbers, err := client.ListPhoneNumbers()
			if err != nil {
				return friendlyAPIErr(err, "")
			}
			headers := []string{"NUMBER", "PROVIDER", "AGENT ID", "RENTED", "PRICE", "RENEWAL"}
			rows := make([][]string, len(numbers))
			for i, n := range numbers {
				rented := "no"
				if n.Rented {
					rented = "yes"
				}
				rows[i] = []string{n.PhoneNumber, n.TelephonyProvider, orDash(n.AgentID), rented, n.Price, n.RenewalAt}
			}
			return a.renderList(headers, rows, 0, -1, numbers, quiet)
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print only phone numbers, one per line (for scripting)")
	return cmd
}
