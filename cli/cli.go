package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/runner"
)

func Run(args []string, cfg *config.Config, mgr *process.Manager) int {
	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "up", "start":
		return cmdUp(cfg, mgr)
	case "down", "stop":
		return cmdDown(mgr)
	case "restart":
		return cmdRestart(cfg, mgr)
	case "status", "st":
		return cmdStatus(cfg, mgr)
	case "serve", "server":
		return cmdStartSingle(mgr, "php-server")
	case "vite":
		return cmdStartSingle(mgr, "vite")
	case "queue":
		return cmdStartSingle(mgr, "queue-worker")
	case "schedule":
		return cmdStartSingle(mgr, "scheduler")
	case "build":
		return cmdBuild()
	case "test":
		return cmdTest(rest)
	case "tinker":
		return cmdExec("php", "artisan", "tinker")
	case "routes":
		return cmdRun("php", "artisan", "route:list", "--except-vendor")
	case "artisan":
		args := append([]string{"artisan"}, rest...)
		return cmdRun("php", args...)
	case "migrate", "mg":
		return cmdRun("php", "artisan", "migrate")
	case "fresh":
		return cmdFresh()
	case "seed":
		return cmdRun("php", "artisan", "db:seed")
	case "rollback", "rb":
		return cmdRun("php", "artisan", "migrate:rollback")
	case "make":
		fmt.Println("  Use interactive mode: ./laradev")
		return 0
	case "cache", "clear":
		return cmdCacheClear()
	case "optimize":
		return cmdOptimize()
	case "logs", "log:app":
		return cmdTail(cfg.LogDir() + "/laravel.log")
	case "log:pail", "pail":
		return cmdExec("php", "artisan", "pail", "--timeout=0")
	case "log:server":
		return cmdTail(mgr.LogPath("php-server"))
	case "log:vite":
		return cmdTail(mgr.LogPath("vite"))
	case "log:queue":
		return cmdTail(mgr.LogPath("queue-worker"))
	case "log:all":
		return cmdTailAll(mgr)
	case "log:clear":
		return cmdLogClear(cfg)
	case "setup":
		return cmdSetup()
	case "nuke":
		return cmdNuke()
	case "about":
		return cmdRun("php", "artisan", "about")
	case "help", "-h", "--help":
		PrintHelp()
		return 0
	default:
		Error("Unknown: " + cmd)
		PrintHelp()
		return 1
	}
}

func cmdUp(cfg *config.Config, mgr *process.Manager) int {
	Banner()
	fmt.Println()
	Step("Starting development environment...")
	fmt.Println()

	results := mgr.StartAll()
	for _, r := range results {
		if r.OK {
			Success(fmt.Sprintf("%s started (%s)", r.Name, r.Message))
		} else {
			Error(fmt.Sprintf("%s failed: %s", r.Name, r.Message))
		}
	}
	fmt.Println()
	Success("All services are up!")
	fmt.Printf("  App:  %shttp://%s:%s%s\n", cyan, cfg.PHPHost, cfg.PHPPort, rst)
	fmt.Printf("  Vite: %shttp://localhost:%s%s\n\n", cyan, cfg.VitePort, rst)
	return 0
}

func cmdDown(mgr *process.Manager) int {
	Banner()
	fmt.Println()
	Step("Stopping all services...")

	results := mgr.StopAll()
	for _, r := range results {
		if r.Message == "stopped" {
			Success(r.Name + " stopped")
		} else {
			Dimmed(r.Name + " " + r.Message)
		}
	}
	fmt.Println()
	Success("All services stopped")
	fmt.Println()
	return 0
}

func cmdRestart(cfg *config.Config, mgr *process.Manager) int {
	Banner()
	fmt.Println()
	Step("Restarting...")

	results := mgr.RestartAll()
	for _, r := range results {
		if r.OK {
			Success(r.Name + " started")
		} else {
			Error(r.Name + " failed: " + r.Message)
		}
	}
	fmt.Println()
	Success("All services restarted!")
	fmt.Println()
	return 0
}

func cmdStatus(cfg *config.Config, mgr *process.Manager) int {
	Banner()
	fmt.Println()

	env := config.DetectProject(cfg.ProjectDir)
	fmt.Printf("  %s%s%s%s [%s]  %s%s%s\n", bold, white, env.AppName, rst, env.AppEnv, gray, cfg.ProjectDir, rst)
	fmt.Println()

	fmt.Printf("  %-18s %-10s %-10s %-10s\n", "SERVICE", "STATUS", "PID", "MEMORY")
	fmt.Printf("  %s\n", strings.Repeat("-", 55))

	infos := mgr.Status()
	for _, info := range infos {
		if info.Running {
			fmt.Printf("  %s*%s %-16s %srunning   %s %-10d %s\n",
				green, rst, info.Label, green, rst, info.PID, info.Memory)
		} else {
			fmt.Printf("  %s-%s %-16s %sstopped   %s ---\n",
				red, rst, info.Label, red, rst)
		}
	}

	fmt.Println()
	phpV := runner.RunCapture("php", "-r", "echo PHP_VERSION;")
	nodeV := runner.RunCapture("node", "-v")
	fmt.Printf("  %sPHP%s %s  %sNode%s %s  %sDB%s %s\n", gray, rst,
		strings.TrimSpace(phpV.Output), gray, rst, strings.TrimSpace(nodeV.Output),
		gray, rst, env.DBConnection)
	fmt.Printf("  %sApp%s  %shttp://%s:%s%s\n", gray, rst, cyan, cfg.PHPHost, cfg.PHPPort, rst)
	fmt.Printf("  %sVite%s %shttp://localhost:%s%s\n\n", gray, rst, cyan, cfg.VitePort, rst)
	return 0
}

func cmdStartSingle(mgr *process.Manager, name string) int {
	if err := mgr.StartService(name); err != nil {
		Error(fmt.Sprintf("%s failed: %s", name, err))
		return 1
	}
	Success(fmt.Sprintf("%s started (PID:%d)", name, mgr.GetPID(name)))
	return 0
}

func cmdBuild() int {
	Banner()
	fmt.Println()
	Step("Building...")
	fmt.Println()
	r := runner.RunCapture("npm", "run", "build")
	fmt.Print(r.Output)
	if r.Err != nil {
		Error("Build failed")
		return 1
	}
	Success("Done!")
	fmt.Println()
	return 0
}

func cmdTest(args []string) int {
	a := append([]string{"artisan", "test"}, args...)
	return cmdRun("php", a...)
}

func cmdRun(name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return 1
	}
	return 0
}

func cmdExec(name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Run()
	return 0
}

func cmdFresh() int {
	fmt.Printf("  %sDrop ALL tables?%s [y/N] ", red, rst)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" || answer == "yes" {
		return cmdRun("php", "artisan", "migrate:fresh", "--seed")
	}
	Dimmed("Cancelled")
	return 0
}

func cmdCacheClear() int {
	Banner()
	fmt.Println()
	cmds := []struct {
		args []string
		msg  string
	}{
		{[]string{"artisan", "config:clear"}, "Config cleared"},
		{[]string{"artisan", "route:clear"}, "Routes cleared"},
		{[]string{"artisan", "view:clear"}, "Views cleared"},
		{[]string{"artisan", "event:clear"}, "Events cleared"},
		{[]string{"artisan", "cache:clear"}, "Cache cleared"},
		{[]string{"artisan", "clear-compiled"}, "Compiled cleared"},
	}
	for _, c := range cmds {
		r := runner.RunCapture("php", c.args...)
		if r.Err == nil {
			Success(c.msg)
		}
	}
	fmt.Println()
	Success("All caches cleared!")
	fmt.Println()
	return 0
}

func cmdOptimize() int {
	Banner()
	fmt.Println()
	cmdRun("php", "artisan", "optimize")
	Success("Optimized")
	fmt.Println()
	return 0
}

func cmdTail(path string) int {
	Info("Tailing " + path + " (Ctrl+C)")
	cmd := exec.Command("tail", "-f", "-n", "50", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	return 0
}

func cmdTailAll(mgr *process.Manager) int {
	files := []string{"-f"}
	for _, def := range process.AllServices {
		logPath := mgr.LogPath(def.Name)
		if _, err := os.Stat(logPath); err == nil {
			files = append(files, logPath)
		}
	}
	if len(files) <= 1 {
		Warn("No log files found")
		return 0
	}
	cmd := exec.Command("tail", files...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	return 0
}

func cmdLogClear(cfg *config.Config) int {
	logFile := cfg.LogDir() + "/laravel.log"
	if err := os.Truncate(logFile, 0); err != nil {
		Warn("No log to clear")
		return 0
	}
	Success("Log cleared")
	return 0
}

func cmdSetup() int {
	Banner()
	fmt.Println()
	Step("Running first-time setup...")
	fmt.Println()

	steps := []struct {
		msg  string
		name string
		args []string
	}{
		{"Checking .env...", "bash", []string{"-c", `[ ! -f .env ] && cp .env.example .env && echo "Created .env" || echo ".env exists"`}},
		{"Composer install...", "composer", []string{"install"}},
		{"Key generate...", "php", []string{"artisan", "key:generate"}},
		{"NPM install...", "npm", []string{"install"}},
		{"Database...", "bash", []string{"-c", `[ ! -f database/database.sqlite ] && touch database/database.sqlite && echo "Created sqlite" || echo "DB exists"`}},
		{"Migrations...", "php", []string{"artisan", "migrate", "--force"}},
		{"Build assets...", "npm", []string{"run", "build"}},
		{"Storage link...", "php", []string{"artisan", "storage:link"}},
	}

	for _, s := range steps {
		Step(s.msg)
		r := runner.RunCapture(s.name, s.args...)
		if r.Output != "" {
			fmt.Print("  " + strings.TrimSpace(r.Output) + "\n")
		}
	}
	fmt.Println()
	Success("Setup complete!")
	fmt.Println()
	return 0
}

func cmdNuke() int {
	fmt.Printf("  %sDANGER: Remove vendor/, node_modules/, fresh migrate, rebuild?%s [y/N] ", red, rst)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(answer)) != "y" {
		Dimmed("Cancelled")
		return 0
	}

	fmt.Printf("  %sAre you REALLY sure?%s [y/N] ", red, rst)
	answer2, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(answer2)) != "y" {
		Dimmed("Cancelled")
		return 0
	}

	Banner()
	fmt.Println()
	Step("Nuking...")
	fmt.Println()

	Step("Removing vendor/...")
	os.RemoveAll("vendor")
	Step("Removing node_modules/...")
	os.RemoveAll("node_modules")

	Step("Composer install...")
	cmdRun("composer", "install")
	Step("NPM install...")
	cmdRun("npm", "install")
	Step("Fresh migrate + seed...")
	cmdRun("php", "artisan", "migrate:fresh", "--seed", "--force")
	Step("Build assets...")
	cmdRun("npm", "run", "build")

	fmt.Println()
	Success("Nuke complete!")
	fmt.Println()
	return 0
}

func PrintHelp() {
	Banner()
	fmt.Println()
	fmt.Printf("  %s%sUsage:%s %s./laradev%s %s[command]%s\n\n", white, bold, rst, cyan, rst, gray, rst)

	sections := []struct {
		title string
		cmds  []struct{ cmd, desc string }
	}{
		{"Services", []struct{ cmd, desc string }{
			{"up, start", "Start all services"},
			{"down, stop", "Stop all services"},
			{"restart", "Restart all"},
			{"status, st", "Status dashboard"},
			{"serve", "PHP server only"},
			{"vite", "Vite only"},
			{"queue", "Queue worker only"},
			{"schedule", "Scheduler only"},
		}},
		{"Development", []struct{ cmd, desc string }{
			{"build", "npm run build"},
			{"test [args]", "Run tests"},
			{"tinker", "Tinker REPL"},
			{"routes", "Route list"},
			{"artisan [cmd]", "Any artisan command"},
		}},
		{"Database", []struct{ cmd, desc string }{
			{"migrate, mg", "Run migrations"},
			{"fresh", "Fresh + seed"},
			{"seed", "Run seeders"},
			{"rollback, rb", "Rollback"},
		}},
		{"Logs", []struct{ cmd, desc string }{
			{"logs, log:app", "Tail laravel.log"},
			{"log:pail, pail", "Laravel Pail"},
			{"log:server", "PHP server log"},
			{"log:vite", "Vite log"},
			{"log:queue", "Queue log"},
			{"log:all", "All logs"},
			{"log:clear", "Clear log"},
		}},
		{"Tools", []struct{ cmd, desc string }{
			{"cache, clear", "Clear all caches"},
			{"optimize", "Optimize app"},
			{"setup", "First-time setup"},
			{"nuke", "Full reset"},
		}},
	}

	for _, sec := range sections {
		fmt.Printf("  %s%s%s%s\n", cyan, bold, sec.title, rst)
		for _, c := range sec.cmds {
			fmt.Printf("    %-18s %s\n", c.cmd, c.desc)
		}
		fmt.Println()
	}
	fmt.Println("  Run without args for interactive TUI.")
	fmt.Println()
}
