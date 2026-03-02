package shared

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// Page IDs
type PageID int

const (
	PageMainMenu PageID = iota
	PageServices
	PageDatabase
	PageDevelop
	PageMake
	PageLogs
	PageCache
	PageConfig
)

// Messages
type NavigateMsg struct{ Page PageID }
type NavigateBackMsg struct{}

type CommandDoneMsg struct {
	Output string
	Err    error
	Title  string
}

type ServiceActionDoneMsg struct {
	Lines []string
}

type LogLineMsg string
type LogErrorMsg string
type StopTailMsg struct{}
type TickMsg time.Time
type ExecDoneMsg struct{ Err error }

// Key bindings
type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
	Tab   key.Binding
	Left  key.Binding
	Right key.Binding
}

var Keys = KeyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k")),
	Down:  key.NewBinding(key.WithKeys("down", "j")),
	Enter: key.NewBinding(key.WithKeys("enter")),
	Back:  key.NewBinding(key.WithKeys("esc", "backspace")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Tab:   key.NewBinding(key.WithKeys("tab")),
	Left:  key.NewBinding(key.WithKeys("left", "h")),
	Right: key.NewBinding(key.WithKeys("right", "l")),
}

// Cerise color palette
const (
	Cerise     = lipgloss.Color("#DE3163") // primary accent
	CeriseLt   = lipgloss.Color("#F06292") // light — selected/hover
	CerisePale = lipgloss.Color("#F8BBD0") // pale — subtle text
	CeriseDk   = lipgloss.Color("#880E4F") // dark — borders, separators
)

// Styles
var (
	TitleBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(Cerise).
			Bold(true).
			Padding(0, 1)

	BrandStyle = lipgloss.NewStyle().
			Foreground(CerisePale).
			Background(Cerise)

	FooterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	FooterKeyStyle   = lipgloss.NewStyle().Foreground(CeriseLt).Bold(true)
	FooterBrandStyle = lipgloss.NewStyle().Foreground(Cerise).Bold(true)

	SuccessStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	ErrorMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	DimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	HintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true).PaddingLeft(2)
	AccentStyle   = lipgloss.NewStyle().Foreground(Cerise)
)
