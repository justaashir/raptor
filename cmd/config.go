package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Server   string `json:"server"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".raptor.json")
}

func LoadConfig() (Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func SaveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}
