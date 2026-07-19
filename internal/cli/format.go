package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/bolna-ai/cli/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// renderList is the single formatting path every `<resource> list` command
// goes through: --quiet prints bare IDs (for piping into xargs etc.),
// otherwise -o/--output picks table (default, human-friendly)/json/csv, all
// using only the standard library (no table/YAML dependency) to keep the
// binary lean.
func (a *appCtx) renderList(headers []string, rows [][]string, idCol, statusCol int, jsonData any, quiet bool) error {
	if quiet {
		for _, row := range rows {
			if idCol < len(row) {
				fmt.Println(row[idCol])
			}
		}
		return nil
	}

	switch a.Format() {
	case "json":
		return printJSON(jsonData)
	case "csv":
		return printCSV(headers, rows)
	default:
		theme := a.Theme()
		if len(rows) == 0 {
			fmt.Println(theme.Muted.Render("No results."))
			return nil
		}
		fmt.Println(renderTable(theme, headers, rows, statusCol))
		return nil
	}
}

func printCSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// renderTable draws a minimal Lip Gloss table for plain-terminal (non-TUI)
// command output: header row + separator + data rows, columns padded to the
// widest cell. statusCol, if >= 0, gets its cells colored via StatusColor.
func renderTable(theme styles.Theme, headers []string, rows [][]string, statusCol int) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && lipgloss.Width(cell) > widths[i] {
				widths[i] = lipgloss.Width(cell)
			}
		}
	}

	var b strings.Builder
	for i, h := range headers {
		b.WriteString(theme.TableHead.Render(pad(h, widths[i])))
		if i < len(headers)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	for i := range headers {
		b.WriteString(theme.Muted.Render(strings.Repeat("─", widths[i])))
		if i < len(headers)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	for _, row := range rows {
		for i, cell := range row {
			style := theme.TableRow
			if i == statusCol {
				style = theme.StatusColor(cell)
			}
			b.WriteString(style.Render(pad(cell, widths[i])))
			if i < len(row)-1 {
				b.WriteString("  ")
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func pad(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func fmtDuration(seconds *float64) string {
	if seconds == nil {
		return "—"
	}
	total := int(*seconds)
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}
