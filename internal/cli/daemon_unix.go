//go:build !windows

package cli

import (
	"os"
	"os/exec"
	"syscall"
)

// detach configures cmd to start in a new session so it survives the
// shell that launched it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// terminate asks the process to exit cleanly via SIGTERM.
func terminate(p *os.Process) error {
	return p.Signal(syscall.SIGTERM)
}

// alive checks whether p is still running. Signal 0 is the canonical
// liveness probe on Unix.
func alive(p *os.Process) bool {
	return p.Signal(syscall.Signal(0)) == nil
}
