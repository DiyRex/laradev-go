package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

func detectTestFramework(projectDir string) {
	env := config.DetectProject(projectDir)
	idx := len(generators) - 1
	if env.HasPest {
		generators[idx].label = "Test (Pest)"
		generators[idx].cmd = func(n string) (string, []string) {
			return "php", []string{"artisan", "make:test", n, "--pest"}
		}
	} else {
		generators[idx].label = "Test (PHPUnit)"
		generators[idx].cmd = func(n string) (string, []string) {
			return "php", []string{"artisan", "make:test", n}
		}
	}
}

type makeState int

const (
	makeMenu makeState = iota
	makeInput
	makeRunning
	makeResult
)

type makeGenerator struct {
	id    string
	label string
	cmd   func(name string) (string, []string)
}

var generators = []makeGenerator{
	{"model", "Model (-mfscR)", func(n string) (string, []string) { return "php", []string{"artisan", "make:model", n, "-mfscR"} }},
	{"controller", "Controller (resource)", func(n string) (string, []string) { return "php", []string{"artisan", "make:controller", n, "--resource"} }},
	{"migration", "Migration", func(n string) (string, []string) { return "php", []string{"artisan", "make:migration", n} }},
	{"middleware", "Middleware", func(n string) (string, []string) { return "php", []string{"artisan", "make:middleware", n} }},
	{"request", "Request", func(n string) (string, []string) { return "php", []string{"artisan", "make:request", n} }},
	{"resource", "Resource", func(n string) (string, []string) { return "php", []string{"artisan", "make:resource", n} }},
	{"seeder", "Seeder", func(n string) (string, []string) { return "php", []string{"artisan", "make:seeder", n} }},
	{"factory", "Factory", func(n string) (string, []string) { return "php", []string{"artisan", "make:factory", n} }},
	{"job", "Job", func(n string) (string, []string) { return "php", []string{"artisan", "make:job", n} }},
	{"event", "Event + Listener", func(n string) (string, []string) {
		return "bash", []string{"-c", "php artisan make:event '" + n + "' && php artisan make:listener '" + n + "Listener' --event='" + n + "'"}
	}},
	{"mail", "Mail", func(n string) (string, []string) { return "php", []string{"artisan", "make:mail", n} }},
	{"notification", "Notification", func(n string) (string, []string) { return "php", []string{"artisan", "make:notification", n} }},
	{"command", "Command", func(n string) (string, []string) { return "php", []string{"artisan", "make:command", n} }},
	{"policy", "Policy", func(n string) (string, []string) { return "php", []string{"artisan", "make:policy", n} }},
	{"test", "Test", nil},
}

func init() {
	// Replaced at runtime by Enter() based on project detection
	generators[len(generators)-1].cmd = func(n string) (string, []string) {
		return "php", []string{"artisan", "make:test", n, "--pest"}
	}
}

type MakePage struct {
	cfg     *config.Config
	mgr     *process.Manager
	menu    components.Menu
	state   makeState
	input   components.InputPrompt
	spinner components.Spinner
	result  *components.ResultBox
	selGen  *makeGenerator
	width   int
	height  int
}

func NewMakePage(cfg *config.Config, mgr *process.Manager) *MakePage {
	return &MakePage{cfg: cfg, mgr: mgr}
}

func (p *MakePage) Enter() {
	p.state = makeMenu
	detectTestFramework(p.cfg.ProjectDir)
	var items []components.MenuItem
	for _, g := range generators {
		items = append(items, components.MenuItem{Label: g.label, Type: components.MenuAction, ID: g.id})
	}
	items = append(items, components.MenuItem{Label: "Back", Type: components.MenuAction, ID: "back"})
	p.menu = components.NewMenu(items)
}

func (p *MakePage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *MakePage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case makeMenu:
		return p.updateMenu(msg)
	case makeInput:
		return p.updateInput(msg)
	case makeRunning:
		return p.updateRunning(msg)
	case makeResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *MakePage) updateMenu(msg tea.Msg) tea.Cmd {
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
			for i := range generators {
				if generators[i].id == id {
					p.selGen = &generators[i]
					break
				}
			}
			if p.selGen != nil {
				p.input = components.NewInputPrompt("Name:", "Name...", "")
				p.state = makeInput
				return p.input.Input.Focus()
			}
		case key.Matches(msg, shared.Keys.Back):
			return backCmd()
		}
	}
	return nil
}

func (p *MakePage) updateInput(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg:
		cmd := p.input.Update(msg)
		if p.input.Done {
			val := p.input.Value()
			if val != "" && p.selGen != nil {
				name, args := p.selGen.cmd(val)
				return p.runCmd(p.selGen.label+": "+val, name, args...)
			}
			p.state = makeMenu
		}
		if p.input.Canceled {
			p.state = makeMenu
		}
		return cmd
	}
	return p.input.Update(msg)
}

func (p *MakePage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = makeResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *MakePage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = makeMenu
			p.result = nil
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *MakePage) runCmd(title string, name string, args ...string) tea.Cmd {
	p.state = makeRunning
	p.spinner = components.NewSpinner(title + "...")
	return tea.Batch(p.spinner.Init(), func() tea.Msg {
		output := runCapture(name, args...)
		return shared.CommandDoneMsg{Output: output, Title: title}
	})
}

func (p *MakePage) View() string {
	switch p.state {
	case makeInput:
		return p.input.View()
	case makeRunning:
		return p.spinner.View()
	case makeResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
