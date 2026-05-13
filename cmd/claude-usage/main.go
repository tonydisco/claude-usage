// Command claude-usage shows how much of your Claude.ai plan you've used.
//
// See `claude-usage --help` or the project README for details.
package main

import (
	"fmt"
	"os"

	"github.com/tonydisco/claude-usage/internal/cli"
)

// Overridden at build time via goreleaser ldflags (see .goreleaser.yml).
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	v := version
	if commit != "" {
		v = fmt.Sprintf("%s (%s, %s)", version, commit, date)
	}
	if err := cli.Execute(v); err != nil {
		fmt.Fprintln(os.Stderr, "claude-usage:", err)
		os.Exit(1)
	}
}
