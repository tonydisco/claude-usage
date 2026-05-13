//go:build cgo

package cli

import "errors"

// errNoTray is returned when daemon start is invoked in a build that
// doesn't include the tray feature.
var errNoTray = errors.New("tray feature is missing from this build")

// trayAvailable reports whether the `tray` subcommand is wired up to a
// real systray implementation.
func trayAvailable() bool { return true }
