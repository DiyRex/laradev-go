package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var inputLabel = lipgloss.NewStyle().
	Foreground(lipgloss.Color("33")).
	Bold(true).
	PaddingLeft(2)

type InputPrompt struct {
	Label    string
	Input    textinput.Model
	Done     bool
	Canceled bool
}

func NewInputPrompt(label, placeholder string, value string) InputPrompt {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40
	if value != "" {
		ti.SetValue(value)
	}
	return InputPrompt{
		Label: label,
		Input: ti,
	}
}

func (p *InputPrompt) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			p.Done = true
			return nil
		case "esc":
			p.Canceled = true
			return nil
		}
	}
	var cmd tea.Cmd
	p.Input, cmd = p.Input.Update(msg)
	return cmd
}

func (p *InputPrompt) Value() string {
	return p.Input.Value()
}

func (p *InputPrompt) View() string {
	return "\n" + inputLabel.Render(p.Label) + "\n\n  " + p.Input.View() + "\n"
}
