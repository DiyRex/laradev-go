package runner

import (
	"bytes"
	"os/exec"
)

type Result struct {
	Output string
	Err    error
}

// RunCapture runs a command and captures combined output.
func RunCapture(name string, args ...string) Result {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return Result{Output: buf.String(), Err: err}
}

// MakeCmd creates an exec.Cmd for interactive use (tinker, pail).
func MakeCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
