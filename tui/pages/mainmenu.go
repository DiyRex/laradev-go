package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type mainState int

const (
	mainStateMenu mainState = iota
	mainStateRunning
	mainStateResult
)

type MainMenuPage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   mainState
	spinner components.Spinner
	result  *components.ResultBox
	width   int
	height  int
}

func NewMainMenuPage(cfg *config.Config, mgr *process.Manager) *MainMenuPage {
	items := []components.MenuItem{
		{Label: "SERVICES", Type: components.MenuHeader},
		{Label: "Start All Services", Type: components.MenuAction, ID: "start_all"},
		{Label: "Stop All Services", Type: components.MenuAction, ID: "stop_all"},
		{Label: "Restart All Services", Type: components.MenuAction, ID: "restart_all"},
		{Label: "Manage Services", Type: components.MenuAction, ID: "services"},
		{Label: "DEVELOP", Type: components.MenuHeader},
		{Label: "Database", Type: components.MenuAction, ID: "database"},
		{Label: "Development", Type: components.MenuAction, ID: "develop"},
		{Label: "Cache & Optimize", Type: components.MenuAction, ID: "cache"},
		{Label: "MONITOR", Type: components.MenuHeader},
		{Label: "Logs", Type: components.MenuAction, ID: "logs"},
		{Label: "SYSTEM", Type: components.MenuHeader},
		{Label: "Config", Type: components.MenuAction, ID: "config"},
		{Label: "Exit", Type: components.MenuAction, ID: "exit"},
	}

	return &MainMenuPage{
		cfg:  cfg,
		mgr:  mgr,
		menu: components.NewMenu(items),
	}
}

func (p *MainMenuPage) Init() tea.Cmd { return nil }

func (p *MainMenuPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *MainMenuPage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case mainStateMenu:
		return p.updateMenu(msg)
	case mainStateRunning:
		return p.updateRunning(msg)
	case mainStateResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *MainMenuPage) updateMenu(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, shared.Keys.Up):
			p.menu.Up()
		case key.Matches(msg, shared.Keys.Down):
			p.menu.Down()
		case key.Matches(msg, shared.Keys.Enter):
			return p.handleSelect()
		case key.Matches(msg, shared.Keys.Quit):
			return tea.Quit
		}
	}
	return nil
}

func (p *MainMenuPage) handleSelect() tea.Cmd {
	id := p.menu.SelectedID()
	switch id {
	case "start_all":
		p.state = mainStateRunning
		p.spinner = components.NewSpinner("Starting all services...")
		return tea.Batch(p.spinner.Init(), p.cmdStartAll())
	case "stop_all":
		p.state = mainStateRunning
		p.spinner = components.NewSpinner("Stopping all services...")
		return tea.Batch(p.spinner.Init(), p.cmdStopAll())
	case "restart_all":
		p.state = mainStateRunning
		p.spinner = components.NewSpinner("Restarting all services...")
		return tea.Batch(p.spinner.Init(), p.cmdRestartAll())
	case "services":
		return navCmd(shared.PageServices)
	case "database":
		return navCmd(shared.PageDatabase)
	case "develop":
		return navCmd(shared.PageDevelop)
	case "logs":
		return navCmd(shared.PageLogs)
	case "cache":
		return navCmd(shared.PageCache)
	case "config":
		return navCmd(shared.PageConfig)
	case "exit":
		return tea.Quit
	}
	return nil
}

func (p *MainMenuPage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.ServiceActionDoneMsg:
		output := strings.Join(msg.Lines, "\n")
		rb := components.NewResultBox(output, p.width, p.height-2)
		p.result = &rb
		p.state = mainStateResult
		return nil
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = mainStateResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *MainMenuPage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace", "q":
			p.state = mainStateMenu
			p.result = nil
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *MainMenuPage) cmdStartAll() tea.Cmd {
	return func() tea.Msg {
		results := p.mgr.StartAll()
		var lines []string
		for _, r := range results {
			if r.OK {
				lines = append(lines, fmt.Sprintf("OK  %s (%s)", r.Name, r.Message))
			} else {
				lines = append(lines, fmt.Sprintf("ERR %s: %s", r.Name, r.Message))
			}
		}
		lines = append(lines, "", fmt.Sprintf("App:  http://%s:%s", p.cfg.PHPHost, p.cfg.PHPPort),
			fmt.Sprintf("Vite: http://localhost:%s", p.cfg.VitePort))
		return shared.ServiceActionDoneMsg{Lines: lines}
	}
}

func (p *MainMenuPage) cmdStopAll() tea.Cmd {
	return func() tea.Msg {
		results := p.mgr.StopAll()
		var lines []string
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("OK  %s %s", r.Name, r.Message))
		}
		lines = append(lines, "", "All services stopped.")
		return shared.ServiceActionDoneMsg{Lines: lines}
	}
}

func (p *MainMenuPage) cmdRestartAll() tea.Cmd {
	return func() tea.Msg {
		results := p.mgr.RestartAll()
		var lines []string
		for _, r := range results {
			if r.OK {
				lines = append(lines, fmt.Sprintf("OK  %s (%s)", r.Name, r.Message))
			} else {
				lines = append(lines, fmt.Sprintf("ERR %s: %s", r.Name, r.Message))
			}
		}
		lines = append(lines, "", "All services restarted.")
		return shared.ServiceActionDoneMsg{Lines: lines}
	}
}

func (p *MainMenuPage) View() string {
	switch p.state {
	case mainStateRunning:
		return p.spinner.View()
	case mainStateResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
