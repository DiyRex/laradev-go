package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	PHPHost      string
	PHPPort      string
	VitePort     string
	QueueTries   string
	QueueTimeout string
	QueueSleep   string

	ProjectDir string
	FilePath   string
}

func Load(projectDir string) *Config {
	cfg := &Config{
		PHPHost:      DefaultPHPHost,
		PHPPort:      DefaultPHPPort,
		VitePort:     DefaultVitePort,
		QueueTries:   DefaultQueueTries,
		QueueTimeout: DefaultQueueTimeout,
		QueueSleep:   DefaultQueueSleep,
		ProjectDir:   projectDir,
		FilePath:     projectDir + "/" + ConfigFileName,
	}

	f, err := os.Open(cfg.FilePath)
	if err != nil {
		return cfg
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
		case "PHP_HOST":
			cfg.PHPHost = val
		case "PHP_PORT":
			cfg.PHPPort = val
		case "VITE_PORT":
			cfg.VitePort = val
		case "QUEUE_TRIES":
			cfg.QueueTries = val
		case "QUEUE_TIMEOUT":
			cfg.QueueTimeout = val
		case "QUEUE_SLEEP":
			cfg.QueueSleep = val
		}
	}
	return cfg
}

func (c *Config) Save() error {
	lines := []string{
		"# LaraDev Configuration",
		fmt.Sprintf("PHP_HOST=\"%s\"", c.PHPHost),
		fmt.Sprintf("PHP_PORT=\"%s\"", c.PHPPort),
		fmt.Sprintf("VITE_PORT=\"%s\"", c.VitePort),
		fmt.Sprintf("QUEUE_TRIES=\"%s\"", c.QueueTries),
		fmt.Sprintf("QUEUE_TIMEOUT=\"%s\"", c.QueueTimeout),
		fmt.Sprintf("QUEUE_SLEEP=\"%s\"", c.QueueSleep),
		"",
	}
	return os.WriteFile(c.FilePath, []byte(strings.Join(lines, "\n")), 0644)
}

func (c *Config) Set(key, value string) {
	switch key {
	case "PHP_HOST":
		c.PHPHost = value
	case "PHP_PORT":
		c.PHPPort = value
	case "VITE_PORT":
		c.VitePort = value
	case "QUEUE_TRIES":
		c.QueueTries = value
	case "QUEUE_TIMEOUT":
		c.QueueTimeout = value
	case "QUEUE_SLEEP":
		c.QueueSleep = value
	}
}

func (c *Config) Get(key string) string {
	switch key {
	case "PHP_HOST":
		return c.PHPHost
	case "PHP_PORT":
		return c.PHPPort
	case "VITE_PORT":
		return c.VitePort
	case "QUEUE_TRIES":
		return c.QueueTries
	case "QUEUE_TIMEOUT":
		return c.QueueTimeout
	case "QUEUE_SLEEP":
		return c.QueueSleep
	}
	return ""
}

func (c *Config) ResetDefaults() {
	c.PHPHost = DefaultPHPHost
	c.PHPPort = DefaultPHPPort
	c.VitePort = DefaultVitePort
	c.QueueTries = DefaultQueueTries
	c.QueueTimeout = DefaultQueueTimeout
	c.QueueSleep = DefaultQueueSleep
	os.Remove(c.FilePath)
}

func (c *Config) PidDir() string {
	return c.ProjectDir + "/" + PidDirName
}

func (c *Config) LogDir() string {
	return c.ProjectDir + "/" + LogDirName
}
