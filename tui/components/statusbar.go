package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/DiyRex/laradev-go/process"
)

var (
	sbOnDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true)
	sbOnLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("#86efac"))
	sbOffDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sbOffLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	sbStyle    = lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1)
	sbSep      = lipgloss.NewStyle().Foreground(lipgloss.Color("#880E4F"))
)

func RenderStatusBar(mgr *process.Manager, cfg_phpPort, cfg_vitePort string) string {
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

	return sbStyle.Render(strings.Join(parts, sbSep.Render("  │  ")))
}
