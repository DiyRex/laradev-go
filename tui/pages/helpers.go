package pages

import (
	"bytes"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/tui/shared"
)

func runCapture(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Run()
	return buf.String()
}

func navCmd(page shared.PageID) tea.Cmd {
	return func() tea.Msg { return shared.NavigateMsg{Page: page} }
}

func backCmd() tea.Cmd {
	return func() tea.Msg { return shared.NavigateBackMsg{} }
}
