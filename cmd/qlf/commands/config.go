package commands

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Temporal struct {
		Address   string        `yaml:"address"`
		Namespace string        `yaml:"namespace"`
		Timeout   time.Duration `yaml:"timeout"`
		TaskQueue string        `yaml:"taskQueue"`
	} `yaml:"temporal"`
	OutputDir string `yaml:"outputDir"`
	Verbose   bool   `yaml:"verbose"`
}

func LoadConfig(explicit string) (*Config, error) {
	c := &Config{}
	// defaults
	c.Temporal.Address = "localhost:7233"
	c.Temporal.Namespace = "default"
	c.Temporal.Timeout = 5 * time.Minute
	c.Temporal.TaskQueue = "factory-task-queue"
	c.OutputDir = "./generated"
	c.Verbose = false

	path := explicit
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".qlf.yaml")
	}
	if b, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(b, c)
	}
	// env overrides
	if v := os.Getenv("QLF_TEMPORAL_ADDR"); v != "" {
		c.Temporal.Address = v
	}
	if v := os.Getenv("QLF_OUTPUT_DIR"); v != "" {
		c.OutputDir = v
	}
	return c, nil
}