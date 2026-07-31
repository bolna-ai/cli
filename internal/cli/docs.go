package cli

import (
	"fmt"
	"strings"

	"github.com/bolna-ai/cli/internal/docs"
	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

// newDocsCmd is a CLI-only convenience, not a mirror of a Bolna MCP tool
// (hence living in the Utility group alongside doctor/version): it searches
// and fetches Bolna's public documentation via the llms.txt index Bolna
// publishes for machine consumption, so answering "how do I configure X"
// doesn't require leaving the terminal either. No API key is needed — docs
// are public.
func newDocsCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Search and fetch Bolna's public documentation",
	}
	cmd.AddCommand(newDocsSearchCmd(a), newDocsFetchCmd(a))
	return cmd
}

func newDocsSearchCmd(a *appCtx) *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Bolna's documentation by keyword",
		Long: "Searches page titles, descriptions, and paths from Bolna's llms.txt doc\n" +
			"index. No API key required. Pipe a result's path into `bolna docs fetch`\n" +
			"to read the full page.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			entries, err := docs.FetchIndex()
			if err != nil {
				return err
			}
			results := docs.Search(entries, query)

			headers := []string{"TITLE", "PATH", "DESCRIPTION"}
			rows := make([][]string, len(results))
			for i, e := range results {
				rows[i] = []string{e.Title, e.Path, truncate(e.Description, 70)}
			}
			if err := a.renderList(headers, rows, 1, -1, results, quiet); err != nil {
				return err
			}
			if !quiet && a.Format() == "table" && len(results) == 0 {
				fmt.Println(a.Theme().Muted.Render("No matching docs. Try different keywords."))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print only doc paths, one per line (for piping into bolna docs fetch)")
	return cmd
}

func newDocsFetchCmd(a *appCtx) *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <page>",
		Short: "Fetch and render one documentation page",
		Long: "Fetches a single page from Bolna's docs and renders it as Markdown.\n" +
			"<page> can be a bare path (e.g. build-with-ai/mcp-tool-list), a path\n" +
			"ending in .md, or a full bolna.ai/docs URL — as printed by\n" +
			"`bolna docs search`.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := docs.FetchPage(args[0])
			if err != nil {
				return err
			}
			if a.Format() == "json" {
				return printJSON(map[string]string{"path": args[0], "content": content})
			}
			if rendered, err := glamour.Render(content, "auto"); err == nil {
				fmt.Print(rendered)
			} else {
				fmt.Println(content)
			}
			return nil
		},
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
