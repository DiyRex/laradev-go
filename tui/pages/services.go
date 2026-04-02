package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/proxy"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type svcState int

const (
	svcStateMenu svcState = iota
	svcStateAction
	svcStateRunning
	svcStateResult
)

type ServicesPage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   svcState
	action  components.Menu // sub-menu: Restart/Stop/Cancel
	spinner components.Spinner
	result  *components.ResultBox
	selSvc  string // selected service name
	width   int
	height  int
}

func NewServicesPage(cfg *config.Config, mgr *process.Manager) *ServicesPage {
	return &ServicesPage{cfg: cfg, mgr: mgr}
}

func (p *ServicesPage) Enter() {
	p.state = svcStateMenu
	p.rebuildMenu()
}

func (p *ServicesPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *ServicesPage) rebuildMenu() {
	var items []components.MenuItem
	for _, def := range process.AllServices {
		var label string
		if p.mgr.IsRunning(def.Name) {
			pid := p.mgr.GetPID(def.Name)
			mem := p.mgr.GetMemory(pid)
			port := ""
			if def.Name == "php-server" {
				port = fmt.Sprintf(" (:%s)", p.cfg.PHPPort)
			} else if def.Name == "vite" {
				port = fmt.Sprintf(" (:%s)", p.cfg.VitePort)
			}
			label = fmt.Sprintf("[ON]  %s%s  --  PID:%d %s", def.Label, port, pid, mem)
		} else {
			port := ""
			if def.Name == "php-server" {
				port = fmt.Sprintf(" (:%s)", p.cfg.PHPPort)
			} else if def.Name == "vite" {
				port = fmt.Sprintf(" (:%s)", p.cfg.VitePort)
			}
			label = fmt.Sprintf("[--]  %s%s  --  stopped", def.Label, port)
		}
		items = append(items, components.MenuItem{Label: label, Type: components.MenuAction, ID: def.Name})
	}

	// HTTPS Proxy entry — managed outside the process.Manager
	proxyCfg := proxy.LoadProjectProxy(p.cfg.ProjectDir, p.cfg.PHPPort)
	var proxyLabel string
	if proxyCfg.IsConfigured() {
		if proxy.IsRunning(p.cfg.ProjectDir) {
			proxyLabel = fmt.Sprintf("[ON]  HTTPS Proxy (%s)  --  running", proxyCfg.Domain)
		} else {
			proxyLabel = fmt.Sprintf("[--]  HTTPS Proxy (%s)  --  stopped", proxyCfg.Domain)
		}
	} else {
		proxyLabel = "[--]  HTTPS Proxy  --  not configured"
	}
	items = append(items, components.MenuItem{Label: proxyLabel, Type: components.MenuAction, ID: "proxy"})

	items = append(items, components.MenuItem{Label: "Back", Type: components.MenuAction, ID: "back"})
	p.menu = components.NewMenu(items)
}

func (p *ServicesPage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case svcStateMenu:
		return p.updateMenu(msg)
	case svcStateAction:
		return p.updateAction(msg)
	case svcStateRunning:
		return p.updateRunning(msg)
	case svcStateResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *ServicesPage) updateMenu(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, shared.Keys.Up):
			p.menu.Up()
		case key.Matches(msg, shared.Keys.Down):
			p.menu.Down()
		case key.Matches(msg, shared.Keys.Enter):
			id := p.menu.SelectedID()
			if id == "back" {
				return backCmd()
			}
			// HTTPS Proxy is managed separately from process.Manager
			if id == "proxy" {
				return p.handleProxySelect()
			}
			p.selSvc = id
			if p.mgr.IsRunning(id) {
				// Show action sub-menu
				p.action = components.NewMenu([]components.MenuItem{
					{Label: "Restart", Type: components.MenuAction, ID: "restart"},
					{Label: "Stop", Type: components.MenuAction, ID: "stop"},
					{Label: "Cancel", Type: components.MenuAction, ID: "cancel"},
				})
				p.state = svcStateAction
			} else {
				// Start it
				p.state = svcStateRunning
				p.spinner = components.NewSpinner(fmt.Sprintf("Starting %s...", id))
				return tea.Batch(p.spinner.Init(), p.cmdStart(id))
			}
		case key.Matches(msg, shared.Keys.Back):
			return backCmd()
		}
	}
	return nil
}

func (p *ServicesPage) updateAction(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, shared.Keys.Up):
			p.action.Up()
		case key.Matches(msg, shared.Keys.Down):
			p.action.Down()
		case key.Matches(msg, shared.Keys.Enter):
			id := p.action.SelectedID()
			switch id {
			case "restart":
				p.state = svcStateRunning
				p.spinner = components.NewSpinner(fmt.Sprintf("Restarting %s...", p.selSvc))
				return tea.Batch(p.spinner.Init(), p.cmdRestart(p.selSvc))
			case "stop":
				p.state = svcStateRunning
				p.spinner = components.NewSpinner(fmt.Sprintf("Stopping %s...", p.selSvc))
				return tea.Batch(p.spinner.Init(), p.cmdStop(p.selSvc))
			case "proxy-stop":
				p.state = svcStateRunning
				p.spinner = components.NewSpinner("Stopping HTTPS proxy...")
				return tea.Batch(p.spinner.Init(), p.cmdStopProxy())
			case "cancel":
				p.state = svcStateMenu
			}
		case key.Matches(msg, shared.Keys.Back):
			p.state = svcStateMenu
		}
	}
	return nil
}

func (p *ServicesPage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = svcStateResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *ServicesPage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = svcStateMenu
			p.result = nil
			p.rebuildMenu()
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *ServicesPage) handleProxySelect() tea.Cmd {
	proxyCfg := proxy.LoadProjectProxy(p.cfg.ProjectDir, p.cfg.PHPPort)
	if !proxyCfg.IsConfigured() {
		rb := components.NewResultBox(
			"HTTPS proxy is not configured for this project.\n\n"+
				"Run the following command from your terminal to set it up:\n\n"+
				"  laradev proxy:setup\n\n"+
				"This will:\n"+
				"  • Generate a trusted TLS certificate (via mkcert)\n"+
				"  • Add your .test domain to /etc/hosts\n"+
				"  • Save proxy config to ~/.laradev/\n\n"+
				"After setup, the proxy starts automatically with 'laradev up'.",
			p.width, p.height-2)
		p.result = &rb
		p.state = svcStateResult
		return nil
	}
	if proxy.IsRunning(p.cfg.ProjectDir) {
		p.action = components.NewMenu([]components.MenuItem{
			{Label: fmt.Sprintf("Stop HTTPS Proxy (%s)", proxyCfg.Domain), Type: components.MenuAction, ID: "proxy-stop"},
			{Label: "Cancel", Type: components.MenuAction, ID: "cancel"},
		})
		p.state = svcStateAction
		return nil
	}
	// Start proxy
	p.state = svcStateRunning
	p.spinner = components.NewSpinner("Starting HTTPS proxy...")
	return tea.Batch(p.spinner.Init(), p.cmdStartProxy())
}

func (p *ServicesPage) cmdStartProxy() tea.Cmd {
	return func() tea.Msg {
		proxyCfg := proxy.LoadProjectProxy(p.cfg.ProjectDir, p.cfg.PHPPort)
		if err := proxy.StartDaemon(proxyCfg, p.cfg.ProjectDir); err != nil {
			return shared.CommandDoneMsg{Output: "ERR  Failed to start HTTPS proxy: " + err.Error()}
		}
		return shared.CommandDoneMsg{Output: "OK  HTTPS proxy started\n    → " + proxyCfg.AppURL()}
	}
}

func (p *ServicesPage) cmdStopProxy() tea.Cmd {
	return func() tea.Msg {
		if err := proxy.StopDaemon(p.cfg.ProjectDir); err != nil {
			return shared.CommandDoneMsg{Output: "WARN  " + err.Error()}
		}
		return shared.CommandDoneMsg{Output: "OK  HTTPS proxy stopped"}
	}
}

func (p *ServicesPage) cmdStart(name string) tea.Cmd {
	return func() tea.Msg {
		err := p.mgr.StartService(name)
		if err != nil {
			return shared.CommandDoneMsg{Output: fmt.Sprintf("ERR  Failed to start %s: %s", name, err)}
		}
		pid := p.mgr.GetPID(name)
		return shared.CommandDoneMsg{Output: fmt.Sprintf("OK  Started %s (PID:%d)", name, pid)}
	}
}

func (p *ServicesPage) cmdStop(name string) tea.Cmd {
	return func() tea.Msg {
		p.mgr.StopService(name)
		return shared.CommandDoneMsg{Output: fmt.Sprintf("OK  Stopped %s", name)}
	}
}

func (p *ServicesPage) cmdRestart(name string) tea.Cmd {
	return func() tea.Msg {
		p.mgr.StopService(name)
		err := p.mgr.StartService(name)
		if err != nil {
			return shared.CommandDoneMsg{Output: fmt.Sprintf("ERR  Failed to restart %s: %s", name, err)}
		}
		pid := p.mgr.GetPID(name)
		return shared.CommandDoneMsg{Output: fmt.Sprintf("OK  Restarted %s (PID:%d)", name, pid)}
	}
}

func (p *ServicesPage) View() string {
	switch p.state {
	case svcStateAction:
		return "\n" + shared.HintStyle.Render(fmt.Sprintf("  %s is running:", p.selSvc)) + "\n\n" + p.action.View()
	case svcStateRunning:
		return p.spinner.View()
	case svcStateResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
