// Command claude-usage shows how much of your Claude.ai plan you've used.
//
// See `claude-usage --help` or the project README for details.
package main

import (
	"fmt"
	"os"

	"github.com/tonydisco/claude-usage/internal/cli"
)

// Version is overridden at build time via -ldflags="-X main.Version=..."
var Version = "dev"

func main() {
	if err := cli.Execute(Version); err != nil {
		fmt.Fprintln(os.Stderr, "claude-usage:", err)
		os.Exit(1)
	}
}
