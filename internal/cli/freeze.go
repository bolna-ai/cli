package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// renderCardToPNG shells out to the `freeze` CLI (charmbracelet/freeze is
// distributed as a standalone tool, not an importable package) to rasterize
// an already-styled ANSI card into a PNG. freeze's --execute mode runs a
// command inside a real pty and captures its colored output, which is the
// supported way to snapshot arbitrary ANSI text rather than source code.
func renderCardToPNG(ansiCard string, outPath string) error {
	freezePath, err := exec.LookPath("freeze")
	if err != nil {
		return fmt.Errorf("freeze CLI not found on PATH — install it with `go install github.com/charmbracelet/freeze@latest`, then retry")
	}

	tmp, err := os.CreateTemp("", "bolna-card-*.ansi")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(ansiCard); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	var stderr bytes.Buffer
	cmd := exec.Command(freezePath,
		"--execute", "cat "+tmp.Name(),
		"-o", outPath,
		"--window",
		"--padding", "2,4",
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("freeze: %s", stderr.String())
		}
		return fmt.Errorf("running freeze: %w", err)
	}
	return nil
}
