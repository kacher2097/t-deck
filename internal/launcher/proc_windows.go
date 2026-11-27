//go:build windows

package launcher

import (
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	createNoWindow   = 0x08000000
	createNewConsole = 0x00000010
)

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

// openNewConsoleWindow allocates a dedicated visible console for long-running scripts
// so T-Deck tracks the real cmd.exe PID instead of an ephemeral `start` wrapper.
func openNewConsoleWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewConsole,
	}
}

// queryProcessParentMap snapshots active Windows PID -> ParentPID pairs without spawning a subprocess.
func queryProcessParentMap() map[int]int {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snap)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := syscall.Process32First(snap, &entry); err != nil {
		return nil
	}

	parents := make(map[int]int, 128)
	for {
		parents[int(entry.ProcessID)] = int(entry.ParentProcessID)
		if err := syscall.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return parents
}
