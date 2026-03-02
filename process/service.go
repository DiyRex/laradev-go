package process

import (
	"os/exec"
	"strings"

	"github.com/DiyRex/laradev-go/config"
)

type ServiceDef struct {
	Name    string
	Label   string
	Command string
	Args    func(cfg *config.Config) []string
	Check   func() bool // optional: check if service is available
}

var AllServices = []ServiceDef{
	{
		Name:    "php-server",
		Label:   "PHP Server",
		Command: "php",
		Args: func(cfg *config.Config) []string {
			return []string{"artisan", "serve", "--host=" + cfg.PHPHost, "--port=" + cfg.PHPPort}
		},
	},
	{
		Name:    "vite",
		Label:   "Vite HMR",
		Command: "npx",
		Args: func(cfg *config.Config) []string {
			return []string{"vite", "--port", cfg.VitePort}
		},
	},
	{
		Name:    "queue-worker",
		Label:   "Queue Worker",
		Command: "php",
		Args: func(cfg *config.Config) []string {
			return []string{"artisan", "queue:listen",
				"--tries=" + cfg.QueueTries,
				"--timeout=" + cfg.QueueTimeout,
				"--sleep=" + cfg.QueueSleep,
			}
		},
	},
	{
		Name:    "scheduler",
		Label:   "Scheduler",
		Command: "php",
		Args: func(cfg *config.Config) []string {
			return []string{"artisan", "schedule:work"}
		},
	},
	{
		Name:    "reverb",
		Label:   "Reverb WebSocket",
		Command: "php",
		Args: func(cfg *config.Config) []string {
			return []string{"artisan", "reverb:start"}
		},
		Check: func() bool {
			out, err := exec.Command("php", "artisan", "list").CombinedOutput()
			if err != nil {
				return false
			}
			return strings.Contains(string(out), "reverb:start")
		},
	},
}

// StartableServices are started with "up" command (php, vite, queue)
var StartableServices = []string{"php-server", "vite", "queue-worker"}

func GetServiceDef(name string) *ServiceDef {
	for i := range AllServices {
		if AllServices[i].Name == name {
			return &AllServices[i]
		}
	}
	return nil
}
