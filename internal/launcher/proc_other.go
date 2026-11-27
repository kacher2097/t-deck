//go:build !windows

package launcher

import "os/exec"

func hideConsoleWindow(_ *exec.Cmd) {}

func openNewConsoleWindow(_ *exec.Cmd) {}

func queryProcessParentMap() map[int]int { return nil }
