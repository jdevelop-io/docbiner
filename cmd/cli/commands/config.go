package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configDir = ".docbiner"
const configFile = "config.json"

// Config represents the CLI configuration stored on disk.
type Config struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, configDir, configFile)
}

func loadConfig() (*Config, error) {
	path := configPath()
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path := configPath()
	if path == "" {
		return os.ErrNotExist
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func deleteConfig() error {
	path := configPath()
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func loadConfigKey() string {
	cfg, err := loadConfig()
	if err != nil {
		return ""
	}
	return cfg.APIKey
}

func loadConfigBaseURL() string {
	cfg, err := loadConfig()
	if err != nil {
		return ""
	}
	return cfg.BaseURL
}
