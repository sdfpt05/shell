package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	HistoryFile string `yaml:"history_file"`
	HomeDir     string `yaml:"home_dir"`
}

func Load(file string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.HomeDir == "" {
		cfg.HomeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}

	if cfg.HistoryFile == "" {
		cfg.HistoryFile = filepath.Join(cfg.HomeDir, ".myshell_history")
	}

	return cfg, nil
}
