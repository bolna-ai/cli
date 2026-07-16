package main

import (
	"fmt"
	"os"

	"github.com/bolna-ai/bolna-cli/internal/cli"
)

// version/commit/date are injected via -ldflags at build time by goreleaser;
// they default to "dev"/"none"/"unknown" for `go run`/`go build` locally.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
