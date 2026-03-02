package pages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type logState int

const (
	logMenu logState = iota
	logTailing
	logInput
	logRunning
	logResult
)

type LogsPage struct {
	cfg      *config.Config
	mgr      *process.Manager
	menu     components.Menu
	state    logState
	vp       viewport.Model
	input    components.InputPrompt
	spinner  components.Spinner
	result   *components.ResultBox
	tailCmd  *exec.Cmd
	tailDone chan struct{}
	lines    []string
	width    int
	height   int
}

func NewLogsPage(cfg *config.Config, mgr *process.Manager) *LogsPage {
	return &LogsPage{cfg: cfg, mgr: mgr}
}

func (p *LogsPage) Enter() {
	p.state = logMenu
	p.menu = components.NewMenu([]components.MenuItem{
		{Label: "Laravel App Log", Type: components.MenuAction, ID: "app"},
		{Label: "Laravel Pail", Type: components.MenuAction, ID: "pail"},
		{Label: "PHP Server Log", Type: components.MenuAction, ID: "php"},
		{Label: "Vite Log", Type: components.MenuAction, ID: "vite"},
		{Label: "Queue Worker Log", Type: components.MenuAction, ID: "queue"},
		{Label: "All Service Logs", Type: components.MenuAction, ID: "all"},
		{Label: "Search Logs", Type: components.MenuAction, ID: "search"},
		{Label: "Clear Laravel Log", Type: components.MenuAction, ID: "clear"},
		{Label: "Back", Type: components.MenuAction, ID: "back"},
	})
}

func (p *LogsPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	if p.state == logTailing {
		p.vp.Width = w - 4
		p.vp.Height = h - 2
	}
	if p.result != nil {
		p.result.SetSize(w, h-2)
	}
}

func (p *LogsPage) Update(msg tea.Msg) tea.Cmd {
	switch p.state {
	case logMenu:
		return p.updateMenu(msg)
	case logTailing:
		return p.updateTailing(msg)
	case logInput:
		return p.updateInput(msg)
	case logRunning:
		return p.updateRunning(msg)
	case logResult:
		return p.updateResult(msg)
	}
	return nil
}

func (p *LogsPage) updateMenu(msg tea.Msg) tea.Cmd {
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

func (p *LogsPage) handleSelect() tea.Cmd {
	id := p.menu.SelectedID()
	switch id {
	case "app":
		return p.startTail(p.cfg.LogDir() + "/laravel.log")
	case "pail":
		cmd := exec.Command("php", "artisan", "pail", "--timeout=0")
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			return shared.ExecDoneMsg{Err: err}
		})
	case "php":
		return p.startTail(p.mgr.LogPath("php-server"))
	case "vite":
		return p.startTail(p.mgr.LogPath("vite"))
	case "queue":
		return p.startTail(p.mgr.LogPath("queue-worker"))
	case "all":
		return p.startTailAll()
	case "search":
		p.input = components.NewInputPrompt("Search:", "Search pattern...", "")
		p.state = logInput
		return p.input.Input.Focus()
	case "clear":
		logPath := p.cfg.LogDir() + "/laravel.log"
		if fi, err := os.Stat(logPath); err == nil {
			size := fi.Size()
			os.Truncate(logPath, 0)
			return p.showResult(fmt.Sprintf("Cleared log file (%d bytes)", size))
		}
		return p.showResult("No log file to clear")
	case "back":
		return backCmd()
	}
	return nil
}

func (p *LogsPage) startTail(path string) tea.Cmd {
	p.lines = nil
	p.vp = viewport.New(p.width-4, p.height-2)
	p.state = logTailing
	p.tailDone = make(chan struct{})

	tailCmd := exec.Command("tail", "-f", "-n", "50", path)
	p.tailCmd = tailCmd

	stdout, err := tailCmd.StdoutPipe()
	if err != nil {
		return p.showResult("Error: " + err.Error())
	}
	tailCmd.Stderr = tailCmd.Stdout

	if err := tailCmd.Start(); err != nil {
		return p.showResult("Error: " + err.Error())
	}

	// Read lines in background, send as messages
	return func() tea.Msg {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-p.tailDone:
				return nil
			default:
				return shared.LogLineMsg(scanner.Text())
			}
		}
		return shared.LogErrorMsg("Log stream ended")
	}
}

func (p *LogsPage) startTailAll() tea.Cmd {
	var files []string
	for _, def := range process.AllServices {
		logPath := p.mgr.LogPath(def.Name)
		if _, err := os.Stat(logPath); err == nil {
			files = append(files, logPath)
		}
	}
	if len(files) == 0 {
		return p.showResult("No log files found")
	}

	p.lines = nil
	p.vp = viewport.New(p.width-4, p.height-2)
	p.state = logTailing
	p.tailDone = make(chan struct{})

	args := append([]string{"-f", "-n", "20"}, files...)
	tailCmd := exec.Command("tail", args...)
	p.tailCmd = tailCmd

	stdout, err := tailCmd.StdoutPipe()
	if err != nil {
		return p.showResult("Error: " + err.Error())
	}
	tailCmd.Stderr = tailCmd.Stdout

	if err := tailCmd.Start(); err != nil {
		return p.showResult("Error: " + err.Error())
	}

	return func() tea.Msg {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-p.tailDone:
				return nil
			default:
				return shared.LogLineMsg(scanner.Text())
			}
		}
		return shared.LogErrorMsg("Log stream ended")
	}
}

func (p *LogsPage) stopTail() {
	if p.tailCmd != nil && p.tailCmd.Process != nil {
		if p.tailDone != nil {
			close(p.tailDone)
		}
		p.tailCmd.Process.Kill()
		p.tailCmd.Wait()
		p.tailCmd = nil
	}
}

func (p *LogsPage) updateTailing(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "backspace", "q":
			p.stopTail()
			p.state = logMenu
			return nil
		}
		var cmd tea.Cmd
		p.vp, cmd = p.vp.Update(msg)
		return cmd

	case shared.LogLineMsg:
		p.lines = append(p.lines, string(msg))
		// Keep last 500 lines
		if len(p.lines) > 500 {
			p.lines = p.lines[len(p.lines)-500:]
		}
		p.vp.SetContent(strings.Join(p.lines, "\n"))
		p.vp.GotoBottom()
		// Continue reading
		return p.readNextLine()

	case shared.LogErrorMsg:
		p.lines = append(p.lines, string(msg))
		p.vp.SetContent(strings.Join(p.lines, "\n"))
		return nil
	}

	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return cmd
}

func (p *LogsPage) readNextLine() tea.Cmd {
	if p.tailCmd == nil {
		return nil
	}
	// The initial goroutine handles reading; after first line we need to continue
	// Actually, the tail command sends lines continuously via the initial goroutine
	// But since bubbletea cmds return only one message, we need to re-read
	// This is handled by returning a new read command
	return nil // The initial reader goroutine will keep sending messages
}

func (p *LogsPage) updateInput(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg:
		cmd := p.input.Update(msg)
		if p.input.Done {
			val := p.input.Value()
			if val != "" {
				return p.runSearch(val)
			}
			p.state = logMenu
		}
		if p.input.Canceled {
			p.state = logMenu
		}
		return cmd
	}
	return p.input.Update(msg)
}

func (p *LogsPage) runSearch(pattern string) tea.Cmd {
	p.state = logRunning
	p.spinner = components.NewSpinner("Searching...")
	return tea.Batch(p.spinner.Init(), func() tea.Msg {
		output := runCapture("grep", "--color=never", "-n", "-i", pattern, p.cfg.LogDir()+"/laravel.log")
		if output == "" {
			output = "No matches found."
		}
		return shared.CommandDoneMsg{Output: output, Title: "Search: " + pattern}
	})
}

func (p *LogsPage) updateRunning(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case shared.CommandDoneMsg:
		rb := components.NewResultBox(msg.Output, p.width, p.height-2)
		p.result = &rb
		p.state = logResult
		return nil
	default:
		return p.spinner.Update(msg)
	}
}

func (p *LogsPage) updateResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "backspace":
			p.state = logMenu
			p.result = nil
			return nil
		}
		return p.result.Update(msg)
	}
	return p.result.Update(msg)
}

func (p *LogsPage) showResult(text string) tea.Cmd {
	rb := components.NewResultBox(text, p.width, p.height-2)
	p.result = &rb
	p.state = logResult
	return nil
}

var logTailBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240")).
	Padding(0, 1).
	MarginLeft(1)

func (p *LogsPage) View() string {
	switch p.state {
	case logTailing:
		hint := shared.HintStyle.Render("esc/q to go back  ↑↓ scroll")
		content := logTailBorder.Width(p.vp.Width).Render(p.vp.View())
		return "\n" + content + "\n" + hint
	case logInput:
		return p.input.View()
	case logRunning:
		return p.spinner.View()
	case logResult:
		if p.result != nil {
			return p.result.View()
		}
	}
	return "\n" + p.menu.View()
}
