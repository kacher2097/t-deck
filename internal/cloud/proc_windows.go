//go:build windows

package cloud

import (
	"os/exec"
	"syscall"
)

func hideCloudCmdWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
