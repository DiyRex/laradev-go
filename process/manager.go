package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/DiyRex/laradev-go/config"
)

type Manager struct {
	Config *config.Config
	PidDir string
}

type ServiceInfo struct {
	Name    string
	Label   string
	Running bool
	PID     int
	Memory  string
}

type ServiceResult struct {
	Name    string
	OK      bool
	Message string
}

func NewManager(cfg *config.Config) *Manager {
	pidDir := cfg.PidDir()
	os.MkdirAll(pidDir, 0755)
	return &Manager{Config: cfg, PidDir: pidDir}
}

func (m *Manager) IsRunning(name string) bool {
	pidFile := m.PidDir + "/" + name + ".pid"
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(pidFile)
		return false
	}
	if !processAlive(pid) {
		os.Remove(pidFile)
		return false
	}
	return true
}

func (m *Manager) GetPID(name string) int {
	pidFile := m.PidDir + "/" + name + ".pid"
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func (m *Manager) GetMemory(pid int) string {
	// Try /proc first (Linux)
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.Atoi(fields[1])
					if err == nil {
						return fmt.Sprintf("%dMB", kb/1024)
					}
				}
			}
		}
	}
	// Fallback to ps
	out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "?"
	}
	kb, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return "?"
	}
	return fmt.Sprintf("%dMB", kb/1024)
}

func (m *Manager) StartService(name string) error {
	if m.IsRunning(name) {
		return nil // already running
	}

	def := GetServiceDef(name)
	if def == nil {
		return fmt.Errorf("unknown service: %s", name)
	}

	if def.Check != nil && !def.Check() {
		return fmt.Errorf("%s is not available", name)
	}

	logFile := m.PidDir + "/" + name + ".log"
	lf, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("cannot create log file: %w", err)
	}

	args := def.Args(m.Config)
	cmd := exec.Command(def.Command, args...)
	cmd.Stdout = lf
	cmd.Stderr = lf
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		lf.Close()
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	pid := cmd.Process.Pid
	os.WriteFile(m.PidDir+"/"+name+".pid", []byte(strconv.Itoa(pid)), 0644)

	// Detach — don't wait for the process
	go func() {
		cmd.Wait()
		lf.Close()
	}()

	// Verify it's still alive after a moment
	time.Sleep(1 * time.Second)
	if !processAlive(pid) {
		os.Remove(m.PidDir + "/" + name + ".pid")
		return fmt.Errorf("%s exited immediately", name)
	}

	return nil
}

func (m *Manager) StopService(name string) error {
	if !m.IsRunning(name) {
		return nil
	}

	pid := m.GetPID(name)
	if pid == 0 {
		return nil
	}

	killProcessTree(pid)
	os.Remove(m.PidDir + "/" + name + ".pid")
	return nil
}

func (m *Manager) Status() []ServiceInfo {
	var infos []ServiceInfo
	for _, def := range AllServices {
		info := ServiceInfo{
			Name:  def.Name,
			Label: def.Label,
		}
		if m.IsRunning(def.Name) {
			info.Running = true
			info.PID = m.GetPID(def.Name)
			info.Memory = m.GetMemory(info.PID)
		}
		infos = append(infos, info)
	}
	return infos
}

func (m *Manager) StartAll() []ServiceResult {
	var results []ServiceResult
	for _, name := range StartableServices {
		def := GetServiceDef(name)
		label := name
		if def != nil {
			label = def.Label
		}
		if err := m.StartService(name); err != nil {
			results = append(results, ServiceResult{Name: label, OK: false, Message: err.Error()})
		} else {
			pid := m.GetPID(name)
			results = append(results, ServiceResult{Name: label, OK: true, Message: fmt.Sprintf("PID:%d", pid)})
		}
	}
	return results
}

func (m *Manager) StopAll() []ServiceResult {
	var results []ServiceResult
	for _, def := range AllServices {
		if m.IsRunning(def.Name) {
			m.StopService(def.Name)
			results = append(results, ServiceResult{Name: def.Label, OK: true, Message: "stopped"})
		} else {
			results = append(results, ServiceResult{Name: def.Label, OK: true, Message: "was not running"})
		}
	}
	return results
}

func (m *Manager) RestartAll() []ServiceResult {
	m.StopAll()
	time.Sleep(500 * time.Millisecond)
	return m.StartAll()
}

func (m *Manager) LogPath(name string) string {
	return m.PidDir + "/" + name + ".log"
}
