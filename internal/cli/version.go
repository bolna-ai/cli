package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(a *appCtx) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print bolna's version, commit, and build date",
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.Format() == "json" {
				return printJSON(a.build)
			}
			fmt.Printf("bolna %s (%s, built %s)\n", a.build.Version, a.build.Commit, a.build.Date)
			return nil
		},
	}
}
