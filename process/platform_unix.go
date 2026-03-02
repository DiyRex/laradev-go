//go:build !windows

package process

import (
	"os/exec"
	"syscall"
)

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
