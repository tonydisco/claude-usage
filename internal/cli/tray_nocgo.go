//go:build !cgo

package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// errNoCGO is returned by tray/daemon commands in builds that were
// compiled with CGO_ENABLED=0 (e.g. the precompiled release binaries).
// Re-install via `go install github.com/tonydisco/claude-usage/cmd/claude-usage@latest`
// on macOS/Linux with a C toolchain available to get the full feature.
var errNoCGO = errors.New("tray/daemon require a CGO-enabled build; install from source with `go install github.com/tonydisco/claude-usage/cmd/claude-usage@latest`")

func newTrayCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "tray",
		Short:  "System tray icon (requires CGO build — see help)",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errNoCGO },
	}
}
