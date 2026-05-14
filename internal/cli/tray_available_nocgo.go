//go:build !cgo

package cli

// errNoTray surfaces the same message from `daemon start` as the
// `tray` command itself prints in CGO-disabled builds.
var errNoTray = errNoCGO

// trayAvailable is false in builds compiled with CGO_ENABLED=0.
func trayAvailable() bool { return false }
