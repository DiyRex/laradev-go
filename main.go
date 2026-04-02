package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/DiyRex/laradev-go/cli"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/proxy"
	"github.com/DiyRex/laradev-go/tui"
)

func main() {
	projectDir := findProjectDir()
	os.Chdir(projectDir)

	cfg := config.Load(projectDir)
	mgr := process.NewManager(cfg)

	// CLI mode
	if len(os.Args) > 1 {
		code := cli.Run(os.Args[1:], cfg, mgr)
		os.Exit(code)
	}

	// TUI mode — auto-start proxy if configured, stop it when TUI exits.
	proxyCfg := proxy.LoadProjectProxy(projectDir, cfg.PHPPort)
	if proxyCfg.IsConfigured() && !proxy.IsRunning(projectDir) {
		_ = proxy.StartDaemon(proxyCfg, projectDir)
	}

	app := tui.NewApp(cfg, mgr)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		_ = proxy.StopDaemon(projectDir)
		os.Exit(1)
	}

	// Stop proxy when the TUI exits normally (q, Ctrl+C, etc.)
	_ = proxy.StopDaemon(projectDir)
}

func findProjectDir() string {
	// Try directory of the executable first
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if isLaravelProject(dir) {
			return dir
		}
	}

	// Try current working directory
	cwd, err := os.Getwd()
	if err == nil {
		if isLaravelProject(cwd) {
			return cwd
		}
	}

	// Walk up from cwd
	if cwd != "" {
		dir := cwd
		for {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			if isLaravelProject(parent) {
				return parent
			}
			dir = parent
		}
	}

	// Fallback to cwd
	if cwd != "" {
		return cwd
	}
	return "."
}

func isLaravelProject(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "artisan"))
	return err == nil
}
