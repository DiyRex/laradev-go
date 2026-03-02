package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Spinner struct {
	Model   spinner.Model
	Message string
}

func NewSpinner(message string) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#DE3163"))
	return Spinner{Model: s, Message: message}
}

func (s *Spinner) Init() tea.Cmd {
	return s.Model.Tick
}

func (s *Spinner) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.Model, cmd = s.Model.Update(msg)
	return cmd
}

func (s *Spinner) View() string {
	return "\n  " + s.Model.View() + " " + s.Message + "\n"
}
