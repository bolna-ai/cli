package cli

import (
	"github.com/spf13/cobra"
)

func newBatchesCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "batches",
		Aliases: []string{"batch"},
		Short:   "Call batch campaigns",
	}
	cmd.AddCommand(newBatchesListCmd(a))
	return cmd
}

func newBatchesListCmd(a *appCtx) *cobra.Command {
	var page, pageSize int
	var quiet bool
	cmd := &cobra.Command{
		Use:     "list <agent-id>",
		Aliases: []string{"ls"},
		Short:   "Batch campaigns for one agent — status and schedule",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			batches, err := client.ListBatches(args[0], page, pageSize)
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}
			headers := []string{"BATCH ID", "STATUS", "SCHEDULED AT", "CREATED"}
			rows := make([][]string, len(batches))
			for i, b := range batches {
				scheduled := "—"
				if b.ScheduledAt != nil {
					scheduled = *b.ScheduledAt
				}
				rows[i] = []string{b.BatchID, b.Status, scheduled, b.CreatedAt}
			}
			return a.renderList(headers, rows, 0, 1, batches, quiet)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 50, "page size, max 50")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print only batch IDs, one per line (for scripting)")
	return cmd
}
