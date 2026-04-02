package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/proxy"
)

var (
	sbOnDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true)
	sbOnLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("#86efac"))
	sbOffDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sbOffLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	sbStyle    = lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1)
	sbSep      = lipgloss.NewStyle().Foreground(lipgloss.Color("#880E4F"))
)

func RenderStatusBar(mgr *process.Manager, cfg_phpPort, cfg_vitePort, projectDir string) string {
	type svc struct {
		name  string
		label string
	}
	services := []svc{
		{"php-server", fmt.Sprintf("PHP:%s", cfg_phpPort)},
		{"vite", fmt.Sprintf("Vite:%s", cfg_vitePort)},
		{"queue-worker", "Queue"},
		{"scheduler", "Sched"},
	}

	var parts []string
	for _, s := range services {
		if mgr.IsRunning(s.name) {
			parts = append(parts, sbOnDot.Render("●")+" "+sbOnLabel.Render(s.label))
		} else {
			parts = append(parts, sbOffDot.Render("○")+" "+sbOffLabel.Render(s.label))
		}
	}

	// Proxy indicator — shows HTTPS status.
	proxyCfg := proxy.LoadProjectProxy(projectDir, cfg_phpPort)
	if proxyCfg.IsConfigured() {
		if proxy.IsRunning(projectDir) {
			parts = append(parts, sbOnDot.Render("●")+" "+sbOnLabel.Render("HTTPS"))
		} else {
			// Red dot: configured but not running.
			redDot := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true)
			redLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#fca5a5"))
			parts = append(parts, redDot.Render("●")+" "+redLabel.Render("HTTPS"))
		}
	} else {
		parts = append(parts, sbOffDot.Render("○")+" "+sbOffLabel.Render("HTTPS"))
	}

	return sbStyle.Render(strings.Join(parts, sbSep.Render("  │  ")))
}
