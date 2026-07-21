package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// renderCardToPNG shells out to the `freeze` CLI (charmbracelet/freeze is
// distributed as a standalone tool, not an importable package) to rasterize
// an already-styled ANSI card into a PNG. The card is piped to freeze on
// stdin — freeze reads and auto-detects ANSI there (its documented
// `cat art.ansi | freeze` usage). Piping avoids a temp file, the non-Windows
// `cat`, and the shell word-splitting that broke on paths containing spaces.
func renderCardToPNG(ansiCard string, outPath string) error {
	freezePath, err := exec.LookPath("freeze")
	if err != nil {
		return fmt.Errorf("freeze CLI not found on PATH — install it with `go install github.com/charmbracelet/freeze@latest`, then retry")
	}

	var stderr bytes.Buffer
	cmd := exec.Command(freezePath,
		"-o", outPath,
		"--window",
		"--padding", "2,4",
	)
	cmd.Stdin = strings.NewReader(ansiCard)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("freeze: %s", stderr.String())
		}
		return fmt.Errorf("running freeze: %w", err)
	}
	return nil
}
