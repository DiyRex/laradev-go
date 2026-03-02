package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type MenuItemType int

const (
	MenuAction MenuItemType = iota
	MenuHeader
)

type MenuItem struct {
	Label string
	Type  MenuItemType
	ID    string // identifier for matching
}

type Menu struct {
	Items    []MenuItem
	Cursor   int
	Width    int
	MaxShow  int // max visible items (0 = all)
}

func NewMenu(items []MenuItem) Menu {
	m := Menu{Items: items, Width: 60}
	// Move cursor to first selectable item
	for i, item := range m.Items {
		if item.Type == MenuAction {
			m.Cursor = i
			break
		}
	}
	return m
}

func (m *Menu) Up() {
	for i := m.Cursor - 1; i >= 0; i-- {
		if m.Items[i].Type == MenuAction {
			m.Cursor = i
			return
		}
	}
}

func (m *Menu) Down() {
	for i := m.Cursor + 1; i < len(m.Items); i++ {
		if m.Items[i].Type == MenuAction {
			m.Cursor = i
			return
		}
	}
}

func (m *Menu) Selected() *MenuItem {
	if m.Cursor >= 0 && m.Cursor < len(m.Items) {
		item := m.Items[m.Cursor]
		if item.Type == MenuAction {
			return &item
		}
	}
	return nil
}

func (m *Menu) SelectedID() string {
	item := m.Selected()
	if item != nil {
		return item.ID
	}
	return ""
}

func (m *Menu) Reset() {
	for i, item := range m.Items {
		if item.Type == MenuAction {
			m.Cursor = i
			break
		}
	}
}

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DE3163")).
			Bold(true).
			PaddingLeft(2)
	normalStyle = lipgloss.NewStyle().
			PaddingLeft(5).
			Foreground(lipgloss.Color("250"))
	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F06292")).
			Bold(true)
	cursorBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DE3163")).
			Bold(true)
)

func (m *Menu) View() string {
	var b strings.Builder

	for i, item := range m.Items {
		switch item.Type {
		case MenuHeader:
			if i > 0 {
				b.WriteString("\n")
			}
			// Render section header with a subtle style
			label := strings.TrimPrefix(item.Label, "--- ")
			label = strings.TrimSuffix(label, " ---")
			b.WriteString(headerStyle.Render("  " + label))
			b.WriteString("\n")
		case MenuAction:
			if i == m.Cursor {
				b.WriteString(fmt.Sprintf("  %s %s",
					cursorBarStyle.Render("▸"),
					cursorStyle.Render(item.Label)))
			} else {
				b.WriteString(normalStyle.Render(item.Label))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
