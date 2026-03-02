package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type dbState int

const (
	dbMenu dbState = iota
	dbConfirm
	dbInput
	dbRunning
	dbResult
)

type DatabasePage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   dbState
	confirm components.Confirm
	input   components.InputPrompt
	spinner components.Spinner
	result  *components.ResultBox
	pending string // which action is pending after confirm/input
	width   int
	height  int
}

func NewDatabasePage(cfg *config.Config, mgr *process.Manager) *DatabasePage {
	return &DatabasePage{cfg: cfg, mgr: mgr}
}

func (p *DatabasePage) Enter() {
	p.state = dbMenu
	p.menu = components.NewMenu([]components.MenuItem{
		{Label: "Run Migrations", Type: components.MenuAction, ID: "migrate"},
		{Label: "Fresh + Seed", Type: components.MenuAction, ID: "fresh"},
		{Label: "Seed Database", Type: components.MenuAction, ID: "seed"},
		{Label: "Rollback (last batch)", Type: components.MenuAction, ID: "rollback"},
		{Label: "Rollback (N steps)", Type: components.MenuAction, ID: "rollback_n"},
		{Label: "Reset All Migrations", Type: components.MenuAction, ID: "reset"},
		{Label: "Back", Type: components.MenuAction, ID: "back"},
	})
}

func (p *DatabasePage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *DatabasePage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case dbMenu:
		return p.updateMenu(msg)
	case dbConfirm:
		return p.updateConfirm(msg)
	case dbInput:
		return p.updateInput(msg)
	case dbRunning:
		return p.updateRunning(msg)
	case dbResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *DatabasePage) updateMenu(msg tea.Msg) tea.Cmd {
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

func (p *DatabasePage) handleSelect() tea.Cmd {
	id := p.menu.SelectedID()
	switch id {
	case "migrate":
		return p.runCmd("Migrate", "php", "artisan", "migrate")
	case "fresh":
		p.confirm = components.NewConfirm("Drop ALL tables and re-run migrations with seeders?")
		p.pending = "fresh"
		p.state = dbConfirm
	case "seed":
		return p.runCmd("Seed", "php", "artisan", "db:seed")
	case "rollback":
		return p.runCmd("Rollback", "php", "artisan", "migrate:rollback")
	case "rollback_n":
		p.input = components.NewInputPrompt("Steps:", "Number of steps...", "")
		p.pending = "rollback_n"
		p.state = dbInput
		return p.input.Input.Focus()
	case "reset":
		p.confirm = components.NewConfirm("Rollback ALL migrations? Database will be empty.")
		p.pending = "reset"
		p.state = dbConfirm
	case "back":
		return backCmd()
	}
	return nil
}

func (p *DatabasePage) updateConfirm(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, shared.Keys.Left):
			p.confirm.Left()
		case key.Matches(msg, shared.Keys.Right):
			p.confirm.Right()
		case key.Matches(msg, shared.Keys.Tab):
			p.confirm.Toggle()
		case key.Matches(msg, shared.Keys.Enter):
			p.confirm.Accept()
			if p.confirm.Result {
				switch p.pending {
				case "fresh":
					return p.runCmd("Fresh + Seed", "php", "artisan", "migrate:fresh", "--seed")
				case "reset":
					return p.runCmd("Reset", "php", "artisan", "migrate:reset")
				}
			}
			p.state = dbMenu
		case key.Matches(msg, shared.Keys.Back):
			p.state = dbMenu
		}
	}
	return nil
}

func (p *DatabasePage) updateInput(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg:
		cmd := p.input.Update(msg)
		if p.input.Done {
			val := p.input.Value()
			if val != "" {
				return p.runCmd("Rollback", "php", "artisan", "migrate:rollback", "--step="+val)
			}
			p.state = dbMenu
		}
		if p.input.Canceled {
			p.state = dbMenu
		}
		return cmd
	}
	return p.input.Update(msg)
}

func (p *DatabasePage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = dbResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *DatabasePage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = dbMenu
			p.result = nil
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *DatabasePage) runCmd(title string, name string, args ...string) tea.Cmd {
	p.state = dbRunning
	p.spinner = components.NewSpinner(title + "...")
	return tea.Batch(p.spinner.Init(), func() tea.Msg {
		output := runCapture(name, args...)
		return shared.CommandDoneMsg{Output: output, Title: title}
	})
}

func (p *DatabasePage) View() string {
	switch p.state {
	case dbConfirm:
		return p.confirm.View()
	case dbInput:
		return p.input.View()
	case dbRunning:
		return p.spinner.View()
	case dbResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
