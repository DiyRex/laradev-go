//go:build windows

package process

import (
	"os/exec"
	"strconv"
)

func killProcessTree(pid int) {
	exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
}
