package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type cfgState int

const (
	cfgMenu cfgState = iota
	cfgInput
	cfgMessage
)

type configItem struct {
	label string
	key   string
}

var configItems = []configItem{
	{"PHP Host", "PHP_HOST"},
	{"PHP Port", "PHP_PORT"},
	{"Vite Port", "VITE_PORT"},
	{"Queue Tries", "QUEUE_TRIES"},
	{"Queue Timeout", "QUEUE_TIMEOUT"},
	{"Queue Sleep", "QUEUE_SLEEP"},
}

type ConfigPage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   cfgState
	input   components.InputPrompt
	selItem *configItem
	message string
	width   int
	height  int
}

func NewConfigPage(cfg *config.Config, mgr *process.Manager) *ConfigPage {
	return &ConfigPage{cfg: cfg, mgr: mgr}
}

func (p *ConfigPage) Enter() {
	p.state = cfgMenu
	p.rebuildMenu()
}

func (p *ConfigPage) rebuildMenu() {
	var items []components.MenuItem
	for _, ci := range configItems {
		items = append(items, components.MenuItem{
			Label: ci.label,
			Type:  components.MenuAction,
			ID:    ci.key,
		})
	}
	items = append(items,
		components.MenuItem{Label: "Reset to Defaults", Type: components.MenuAction, ID: "reset"},
		components.MenuItem{Label: "Back", Type: components.MenuAction, ID: "back"},
	)
	p.menu = components.NewMenu(items)
}

func (p *ConfigPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *ConfigPage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case cfgMenu:
		return p.updateMenu(msg)
	case cfgInput:
		return p.updateInput(msg)
	case cfgMessage:
		return p.updateMessage(msg)
	}
	return nil
}

func (p *ConfigPage) updateMenu(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, shared.Keys.Up):
			p.menu.Up()
		case key.Matches(msg, shared.Keys.Down):
			p.menu.Down()
		case key.Matches(msg, shared.Keys.Enter):
			return p.handleSelect()
		case key.Matches(msg, shared.Keys.Back):
			return backCmd()
		}
	}
	return nil
}

func (p *ConfigPage) handleSelect() tea.Cmd {
	id := p.menu.SelectedID()
	if id == "back" {
		return backCmd()
	}
	if id == "reset" {
		p.cfg.ResetDefaults()
		p.message = "Reset to defaults"
		p.state = cfgMessage
		return nil
	}
	for i := range configItems {
		if configItems[i].key == id {
			p.selItem = &configItems[i]
			break
		}
	}
	if p.selItem != nil {
		current := p.cfg.Get(p.selItem.key)
		p.input = components.NewInputPrompt(p.selItem.label+":", "", current)
		p.state = cfgInput
		return p.input.Input.Focus()
	}
	return nil
}

func (p *ConfigPage) updateInput(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg:
		cmd := p.input.Update(msg)
		if p.input.Done {
			val := p.input.Value()
			if val != "" && p.selItem != nil {
				p.cfg.Set(p.selItem.key, val)
				p.cfg.Save()
				p.message = fmt.Sprintf("%s = %s (restart services to apply)", p.selItem.key, val)
				p.state = cfgMessage
				return nil
			}
			p.state = cfgMenu
		}
		if p.input.Canceled {
			p.state = cfgMenu
		}
		return cmd
	}
	return p.input.Update(msg)
}

func (p *ConfigPage) updateMessage(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = cfgMenu
			return nil
		}
	}
	return nil
}

var cfgBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#880E4F")).
	Padding(0, 2).
	MarginLeft(2)

func (p *ConfigPage) View() string {
	// Config value display
	values := fmt.Sprintf(
		"PHP Host:      %s\nPHP Port:      %s\nVite Port:     %s\nQueue Tries:   %s\nQueue Timeout: %ss\nQueue Sleep:   %ss",
		p.cfg.PHPHost, p.cfg.PHPPort, p.cfg.VitePort,
		p.cfg.QueueTries, p.cfg.QueueTimeout, p.cfg.QueueSleep,
	)
	box := cfgBoxStyle.Render(values)

	switch p.state {
	case cfgInput:
		return "\n" + box + "\n" + p.input.View()
	case cfgMessage:
		return "\n" + box + "\n\n" + shared.SuccessStyle.Render("  "+p.message) + "\n" +
			shared.HintStyle.Render("Press enter to continue...")
	}
	return "\n" + box + "\n\n" + p.menu.View()
}
