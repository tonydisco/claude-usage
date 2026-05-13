//go:build !darwin || !cgo

package cli

// No-op stubs for platforms where we don't drive a Dock tile
// (Linux, Windows, and CGO-disabled builds on any OS).

func dockSupported() bool   { return false }
func dockSetIcon(_ []byte)  {}
func dockShow(_ bool)       {}
