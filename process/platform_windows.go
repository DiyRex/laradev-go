//go:build windows

package process

import (
	"os"
	"os/exec"
)

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	p.Release()
	// On Windows FindProcess always succeeds; best-effort check
	return true
}

func setSysProcAttr(cmd *exec.Cmd) {
	// No process group setup needed on Windows
}
