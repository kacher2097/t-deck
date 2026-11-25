//go:build windows

package launcher

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideConsoleWindow prevents Windows from flashing a transient black cmd.exe window
// for background helpers (tasklist, netstat, taskkill, git, shell dispatchers).
func hideConsoleWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
