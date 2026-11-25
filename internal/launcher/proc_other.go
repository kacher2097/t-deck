//go:build !windows

package launcher

import "os/exec"

func hideConsoleWindow(_ *exec.Cmd) {}
