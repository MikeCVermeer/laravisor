package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const ConfigFileName = ".laravisor.json"

// Load loads the configuration from the working directory
func Load(workingDir string) (*Config, error) {
	configPath := filepath.Join(workingDir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return NewDefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to the working directory
func Save(workingDir string, cfg *Config) error {
	configPath := filepath.Join(workingDir, ConfigFileName)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// Exists checks if a config file exists in the working directory
func Exists(workingDir string) bool {
	configPath := filepath.Join(workingDir, ConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}
