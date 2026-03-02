package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	resultBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#880E4F")).
			Padding(0, 1).
			MarginLeft(2)

	resultHint = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true).
			PaddingLeft(3)
)

type ResultBox struct {
	Viewport viewport.Model
	Title    string
	Ready    bool
}

func NewResultBox(content string, width, height int) ResultBox {
	vp := viewport.New(width-6, height)
	vp.SetContent(content)
	return ResultBox{
		Viewport: vp,
		Ready:    true,
	}
}

func (r *ResultBox) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	r.Viewport, cmd = r.Viewport.Update(msg)
	return cmd
}

func (r *ResultBox) SetSize(width, height int) {
	w := width - 6
	if w < 20 {
		w = 20
	}
	r.Viewport.Width = w
	r.Viewport.Height = height
}

func (r *ResultBox) View() string {
	content := resultBorder.Width(r.Viewport.Width).Render(r.Viewport.View())
	hint := resultHint.Render("↑↓ scroll  enter/esc back")
	return content + "\n" + hint
}
