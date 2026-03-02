package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Confirm struct {
	Question string
	Focus    int  // 0=Yes, 1=No
	Decided  bool
	Result   bool
}

func NewConfirm(question string) Confirm {
	return Confirm{Question: question, Focus: 1} // default to No
}

func (c *Confirm) Left()  { c.Focus = 0 }
func (c *Confirm) Right() { c.Focus = 1 }
func (c *Confirm) Toggle() {
	if c.Focus == 0 {
		c.Focus = 1
	} else {
		c.Focus = 0
	}
}

func (c *Confirm) Accept() {
	c.Decided = true
	c.Result = c.Focus == 0
}

func (c *Confirm) Cancel() {
	c.Decided = true
	c.Result = false
}

var (
	confirmBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#DE3163")).
			Padding(1, 3).
			MarginLeft(2).
			MarginTop(1)

	confirmQuestion = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#e2e8f0")).
				Bold(true).
				MarginBottom(1)

	btnActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#DE3163")).
			Bold(true).
			Padding(0, 3)

	btnInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 3)
)

func (c *Confirm) View() string {
	var yes, no string
	if c.Focus == 0 {
		yes = btnActive.Render("Yes")
		no = btnInactive.Render("No")
	} else {
		yes = btnInactive.Render("Yes")
		no = btnActive.Render("No")
	}

	content := fmt.Sprintf("%s\n\n  %s    %s",
		confirmQuestion.Render(c.Question),
		yes, no)

	return confirmBox.Render(content)
}
