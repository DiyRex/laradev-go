package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type cacheState int

const (
	cacheMenu cacheState = iota
	cacheRunning
	cacheResult
)

type CachePage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   cacheState
	spinner components.Spinner
	result  *components.ResultBox
	width   int
	height  int
}

func NewCachePage(cfg *config.Config, mgr *process.Manager) *CachePage {
	return &CachePage{cfg: cfg, mgr: mgr}
}

func (p *CachePage) Enter() {
	p.state = cacheMenu
	p.menu = components.NewMenu([]components.MenuItem{
		{Label: "Clear ALL Caches", Type: components.MenuAction, ID: "clear_all"},
		{Label: "Clear App Cache", Type: components.MenuAction, ID: "cache"},
		{Label: "Clear Config", Type: components.MenuAction, ID: "config"},
		{Label: "Clear Routes", Type: components.MenuAction, ID: "routes"},
		{Label: "Clear Views", Type: components.MenuAction, ID: "views"},
		{Label: "Clear Events", Type: components.MenuAction, ID: "events"},
		{Label: "Clear Compiled", Type: components.MenuAction, ID: "compiled"},
		{Label: "Optimize App", Type: components.MenuAction, ID: "optimize"},
		{Label: "Back", Type: components.MenuAction, ID: "back"},
	})
}

func (p *CachePage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *CachePage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case cacheMenu:
		return p.updateMenu(msg)
	case cacheRunning:
		return p.updateRunning(msg)
	case cacheResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *CachePage) updateMenu(msg tea.Msg) tea.Cmd {
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

func (p *CachePage) handleSelect() tea.Cmd {
	id := p.menu.SelectedID()
	switch id {
	case "clear_all":
		return p.runCmd("Clear All", "bash", "-c",
			"php artisan config:clear 2>&1; php artisan route:clear 2>&1; "+
				"php artisan view:clear 2>&1; php artisan event:clear 2>&1; "+
				"php artisan cache:clear 2>&1; php artisan clear-compiled 2>&1; "+
				`find bootstrap/cache -name "*.php" -not -name ".gitignore" -delete 2>/dev/null; `+
				`echo ""; echo "All caches cleared!"`)
	case "cache":
		return p.runCmd("App Cache", "php", "artisan", "cache:clear")
	case "config":
		return p.runCmd("Config", "php", "artisan", "config:clear")
	case "routes":
		return p.runCmd("Routes", "php", "artisan", "route:clear")
	case "views":
		return p.runCmd("Views", "php", "artisan", "view:clear")
	case "events":
		return p.runCmd("Events", "php", "artisan", "event:clear")
	case "compiled":
		return p.runCmd("Compiled", "php", "artisan", "clear-compiled")
	case "optimize":
		return p.runCmd("Optimize", "php", "artisan", "optimize")
	case "back":
		return backCmd()
	}
	return nil
}

func (p *CachePage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = cacheResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *CachePage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = cacheMenu
			p.result = nil
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *CachePage) runCmd(title string, name string, args ...string) tea.Cmd {
	p.state = cacheRunning
	p.spinner = components.NewSpinner(title + "...")
	return tea.Batch(p.spinner.Init(), func() tea.Msg {
		output := runCapture(name, args...)
		return shared.CommandDoneMsg{Output: output, Title: title}
	})
}

func (p *CachePage) View() string {
	switch p.state {
	case cacheRunning:
		return p.spinner.View()
	case cacheResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
