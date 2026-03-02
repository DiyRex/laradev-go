package config

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// EnvInfo holds data read from .env and detected from the project structure.
type EnvInfo struct {
	AppName      string
	AppEnv       string
	DBConnection string
	DBDatabase   string
	QueueConn    string
	HasPest      bool
	HasPHPUnit   bool
	HasVite      bool
	HasMix       bool
	StarterKit   string // "breeze", "jetstream", "filament", ""
}

// ReadEnv parses the Laravel .env file for key project values.
func ReadEnv(projectDir string) EnvInfo {
	info := EnvInfo{
		AppName: "Laravel",
		AppEnv:  "local",
	}

	f, err := os.Open(filepath.Join(projectDir, ".env"))
	if err != nil {
		return info
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		switch key {
		case "APP_NAME":
			if val != "" {
				info.AppName = val
			}
		case "APP_ENV":
			if val != "" {
				info.AppEnv = val
			}
		case "DB_CONNECTION":
			info.DBConnection = val
		case "DB_DATABASE":
			info.DBDatabase = val
		case "QUEUE_CONNECTION":
			info.QueueConn = val
		}
	}

	return info
}

// DetectProject inspects the project directory for tools and frameworks.
func DetectProject(projectDir string) EnvInfo {
	info := ReadEnv(projectDir)

	// Test framework detection
	info.HasPest = fileExists(filepath.Join(projectDir, "vendor/pestphp"))
	info.HasPHPUnit = fileExists(filepath.Join(projectDir, "phpunit.xml"))

	// Build tool detection
	info.HasVite = fileExists(filepath.Join(projectDir, "vite.config.js")) ||
		fileExists(filepath.Join(projectDir, "vite.config.ts"))
	info.HasMix = fileExists(filepath.Join(projectDir, "webpack.mix.js"))

	// Starter kit detection from composer.json
	info.StarterKit = detectStarterKit(projectDir)

	return info
}

func detectStarterKit(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "composer.json"))
	if err != nil {
		return ""
	}

	var composer struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal(data, &composer) != nil {
		return ""
	}

	// Check both require and require-dev
	all := make(map[string]bool)
	for k := range composer.Require {
		all[k] = true
	}
	for k := range composer.RequireDev {
		all[k] = true
	}

	if all["laravel/breeze"] {
		return "Breeze"
	}
	if all["laravel/jetstream"] {
		return "Jetstream"
	}
	if all["filament/filament"] {
		return "Filament"
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
