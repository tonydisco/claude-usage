//go:build windows

package cli

import (
	"os"
	"os/exec"
	"syscall"
)

// detach configures cmd to start without a console window so closing
// the launching terminal doesn't take it down with it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// terminate asks the process to exit. Windows has no SIGTERM, so kill
// is the only safe option here.
func terminate(p *os.Process) error {
	return p.Kill()
}

// alive on Windows: FindProcess always succeeds, so probe via Signal(0)
// which returns an error if the handle is no longer running.
func alive(p *os.Process) bool {
	return p.Signal(syscall.Signal(0)) == nil
}
