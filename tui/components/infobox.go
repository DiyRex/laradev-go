package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/proxy"
	"github.com/DiyRex/laradev-go/runner"
)

var (
	infoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#880E4F")).
			Padding(0, 2).
			MarginLeft(2).
			MarginTop(1)

	infoLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#DE3163")).Bold(true)
	infoValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	infoProjectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8BBD0")).Bold(true)
	infoPathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	infoURLStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F06292")).Underline(true)
	infoDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#880E4F"))
	infoTagStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

func RenderInfoBox(cfg *config.Config, width int) string {
	env := config.DetectProject(cfg.ProjectDir)

	// Line 1: Project name, env, path
	projectName := infoProjectStyle.Render(env.AppName)
	envTag := infoTagStyle.Render("[") + infoValueStyle.Render(env.AppEnv) + infoTagStyle.Render("]")
	shortPath := shortenPath(cfg.ProjectDir)
	line1 := projectName + " " + envTag + "  " + infoPathStyle.Render(shortPath)

	// Line 2: Versions and DB
	phpV := strings.TrimSpace(runner.RunCapture("php", "-r", "echo PHP_VERSION;").Output)
	if phpV == "" {
		phpV = "?"
	}
	nodeV := strings.TrimSpace(runner.RunCapture("node", "-v").Output)
	if nodeV == "" {
		nodeV = "?"
	}

	dbInfo := env.DBConnection
	if dbInfo == "" {
		dbInfo = "?"
	}
	if env.DBConnection == "sqlite" {
		sqlitePath := cfg.ProjectDir + "/database/database.sqlite"
		if env.DBDatabase != "" && !strings.HasPrefix(env.DBDatabase, "/") {
			sqlitePath = cfg.ProjectDir + "/" + env.DBDatabase
		} else if env.DBDatabase != "" {
			sqlitePath = env.DBDatabase
		}
		if fi, err := os.Stat(sqlitePath); err == nil {
			size := fi.Size()
			if size > 1024*1024 {
				dbInfo = fmt.Sprintf("sqlite (%.1fMB)", float64(size)/(1024*1024))
			} else {
				dbInfo = fmt.Sprintf("sqlite (%dK)", size/1024)
			}
		}
	}

	logInfo := "empty"
	logPath := cfg.LogDir() + "/laravel.log"
	if fi, err := os.Stat(logPath); err == nil {
		size := fi.Size()
		if size > 1024*1024 {
			logInfo = fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
		} else if size > 1024 {
			logInfo = fmt.Sprintf("%dK", size/1024)
		} else if size > 0 {
			logInfo = fmt.Sprintf("%dB", size)
		}
	}

	sep := infoDimStyle.Render(" │ ")
	line2 := infoLabelStyle.Render("PHP ") + infoValueStyle.Render(phpV) + sep +
		infoLabelStyle.Render("Node ") + infoValueStyle.Render(nodeV) + sep +
		infoLabelStyle.Render("DB ") + infoValueStyle.Render(dbInfo) + sep +
		infoLabelStyle.Render("Log ") + infoValueStyle.Render(logInfo)

	// Line 3: URLs — show HTTPS domain if proxy is configured, with a status dot.
	proxyCfg := proxy.LoadProjectProxy(cfg.ProjectDir, cfg.PHPPort)
	var appURL, httpsIndicator string
	if proxyCfg.IsConfigured() {
		appURL = proxyCfg.AppURL()
		if proxy.IsRunning(cfg.ProjectDir) {
			// Green filled dot — proxy running, HTTPS active
			httpsIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true).Render("● ")
		} else {
			// Red filled dot — configured but proxy is stopped
			httpsIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).Render("● ")
		}
	} else {
		// Dim empty dot — HTTPS not configured
		appURL = fmt.Sprintf("http://%s:%s", cfg.PHPHost, cfg.PHPPort)
		httpsIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○ ")
	}
	viteURL := fmt.Sprintf("http://localhost:%s", cfg.VitePort)
	line3 := httpsIndicator + infoLabelStyle.Render("App ") + infoURLStyle.Render(appURL) +
		infoDimStyle.Render("    ") +
		infoLabelStyle.Render("Vite ") + infoURLStyle.Render(viteURL)

	// Line 4: Detected tools tags
	var tags []string
	if env.HasPest {
		tags = append(tags, "Pest")
	} else if env.HasPHPUnit {
		tags = append(tags, "PHPUnit")
	}
	if env.HasVite {
		tags = append(tags, "Vite")
	} else if env.HasMix {
		tags = append(tags, "Mix")
	}
	if env.StarterKit != "" {
		tags = append(tags, env.StarterKit)
	}
	if env.QueueConn != "" && env.QueueConn != "sync" {
		tags = append(tags, "Queue:"+env.QueueConn)
	}

	content := line1 + "\n\n" + line2 + "\n\n" + line3
	if len(tags) > 0 {
		content += "\n\n" + infoTagStyle.Render(strings.Join(tags, " · "))
	}

	maxW := width - 6
	if maxW < 40 {
		maxW = 40
	}
	if maxW > 90 {
		maxW = 90
	}

	return infoBoxStyle.Width(maxW).Render(content)
}

func shortenPath(dir string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(dir, home) {
		return "~" + dir[len(home):]
	}
	// If path is long, show last 3 segments
	parts := strings.Split(filepath.Clean(dir), string(filepath.Separator))
	if len(parts) > 3 {
		return ".../" + strings.Join(parts[len(parts)-3:], "/")
	}
	return dir
}
