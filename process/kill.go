package process

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func collectDescendants(pid int) []int {
	var result []int
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		return result
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		child, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		result = append(result, child)
		result = append(result, collectDescendants(child)...)
	}
	return result
}

func killProcessTree(pid int) {
	children := collectDescendants(pid)

	// Kill children in reverse order (deepest first)
	for i := len(children) - 1; i >= 0; i-- {
		syscall.Kill(children[i], syscall.SIGTERM)
	}

	// Kill parent
	syscall.Kill(pid, syscall.SIGTERM)

	// Wait up to 3 seconds
	for i := 0; i < 30; i++ {
		if err := syscall.Kill(pid, 0); err != nil {
			return // process gone
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill
	for _, child := range children {
		syscall.Kill(child, syscall.SIGKILL)
	}
	syscall.Kill(pid, syscall.SIGKILL)
}
