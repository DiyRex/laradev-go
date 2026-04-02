package pages

import (
	"os/exec"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type devState int

const (
	devMenu devState = iota
	devInput
	devRunning
	devResult
)

type DevelopPage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   devState
	input   components.InputPrompt
	spinner components.Spinner
	result  *components.ResultBox
	pending string
	width   int
	height  int
}

func NewDevelopPage(cfg *config.Config, mgr *process.Manager) *DevelopPage {
	return &DevelopPage{cfg: cfg, mgr: mgr}
}

func (p *DevelopPage) Enter() {
	p.state = devMenu
	p.menu = components.NewMenu([]components.MenuItem{
		{Label: "Install Dependencies  (composer + npm)", Type: components.MenuAction, ID: "install_deps"},
		{Label: "Build Assets", Type: components.MenuAction, ID: "build"},
		{Label: "Run All Tests", Type: components.MenuAction, ID: "test_all"},
		{Label: "Unit Tests", Type: components.MenuAction, ID: "test_unit"},
		{Label: "Feature Tests", Type: components.MenuAction, ID: "test_feature"},
		{Label: "Filter Tests", Type: components.MenuAction, ID: "test_filter"},
		{Label: "Route List", Type: components.MenuAction, ID: "routes"},
		{Label: "Tinker REPL", Type: components.MenuAction, ID: "tinker"},
		{Label: "Make (Generate)", Type: components.MenuAction, ID: "make"},
		{Label: "Artisan Command", Type: components.MenuAction, ID: "artisan"},
		{Label: "Back", Type: components.MenuAction, ID: "back"},
	})
}

func (p *DevelopPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *DevelopPage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case devMenu:
		return p.updateMenu(msg)
	case devInput:
		return p.updateInput(msg)
	case devRunning:
		return p.updateRunning(msg)
	case devResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *DevelopPage) updateMenu(msg tea.Msg) tea.Cmd {
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

func (p *DevelopPage) handleSelect() tea.Cmd {
	id := p.menu.SelectedID()
	switch id {
	case "install_deps":
		return p.runCmdShell("Install Dependencies", "composer install && npm install")
	case "build":
		return p.runCmd("Build", "npm", "run", "build")
	case "test_all":
		return p.runCmd("Tests", "php", "artisan", "test")
	case "test_unit":
		return p.runCmd("Unit Tests", "php", "artisan", "test", "--testsuite=Unit")
	case "test_feature":
		return p.runCmd("Feature Tests", "php", "artisan", "test", "--testsuite=Feature")
	case "test_filter":
		p.input = components.NewInputPrompt("Filter:", "Test name...", "")
		p.pending = "test_filter"
		p.state = devInput
		return p.input.Input.Focus()
	case "routes":
		return p.runCmd("Routes", "php", "artisan", "route:list", "--except-vendor")
	case "tinker":
		cmd := exec.Command("php", "artisan", "tinker")
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			return shared.ExecDoneMsg{Err: err}
		})
	case "make":
		return navCmd(shared.PageMake)
	case "artisan":
		p.input = components.NewInputPrompt("artisan", "e.g. about, queue:work --once", "")
		p.pending = "artisan"
		p.state = devInput
		return p.input.Input.Focus()
	case "back":
		return backCmd()
	}
	return nil
}

func (p *DevelopPage) updateInput(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg:
		cmd := p.input.Update(msg)
		if p.input.Done {
			val := p.input.Value()
			if val != "" {
				switch p.pending {
				case "test_filter":
					return p.runCmd("Tests: "+val, "php", "artisan", "test", "--filter="+val)
				case "artisan":
					// Split the value for artisan command
					return p.runCmd("artisan "+val, "bash", "-c", "php artisan "+val)
				}
			}
			p.state = devMenu
		}
		if p.input.Canceled {
			p.state = devMenu
		}
		return cmd
	}
	return p.input.Update(msg)
}

func (p *DevelopPage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = devResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *DevelopPage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = devMenu
			p.result = nil
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *DevelopPage) runCmd(title string, name string, args ...string) tea.Cmd {
	p.state = devRunning
	p.spinner = components.NewSpinner(title + "...")
	return tea.Batch(p.spinner.Init(), func() tea.Msg {
		output := runCapture(name, args...)
		return shared.CommandDoneMsg{Output: output, Title: title}
	})
}

func (p *DevelopPage) runCmdShell(title, shellCmd string) tea.Cmd {
	p.state = devRunning
	p.spinner = components.NewSpinner(title + "...")
	return tea.Batch(p.spinner.Init(), func() tea.Msg {
		output := runCapture("bash", "-c", shellCmd)
		return shared.CommandDoneMsg{Output: output, Title: title}
	})
}

func (p *DevelopPage) View() string {
	switch p.state {
	case devInput:
		return p.input.View()
	case devRunning:
		return p.spinner.View()
	case devResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
