//go:build !windows

package cloud

import "os/exec"

func hideCloudCmdWindow(_ *exec.Cmd) {}
