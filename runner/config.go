package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github-runner-manager/model"
)

// getConfigDir mengembalikan direktori konfigurasi default
func getConfigDir() string {
	var configDir string
	switch runtime.GOOS {
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "github-runner-manager")
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "github-runner-manager")
	default: // linux dan lainnya
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "github-runner-manager")
	}
	return configDir
}

// getDefaultConfigPath mengembalikan path file konfigurasi default
func getDefaultConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

// ensureConfigDir memastikan direktori konfigurasi ada
func ensureConfigDir() error {
	configDir := getConfigDir()
	return os.MkdirAll(configDir, 0755)
}

// SaveConfig menyimpan konfigurasi runner ke file JSON
func SaveConfig(runners []model.RunnerConfig, path string) error {
	if path == "" {
		path = getDefaultConfigPath()
	}

	if err := ensureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	config := model.ConfigFile{
		Runners: runners,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfig membaca konfigurasi runner dari file JSON
func LoadConfig(path string) ([]model.RunnerConfig, error) {
	if path == "" {
		path = getDefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.RunnerConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config model.ConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config.Runners, nil
}

// SaveRunnersToDefault menyimpan daftar runner ke lokasi default
func SaveRunnersToDefault(runners []model.RunnerConfig) error {
	return SaveConfig(runners, "")
}

// LoadRunnersFromDefault memuat daftar runner dari lokasi default
func LoadRunnersFromDefault() ([]model.RunnerConfig, error) {
	return LoadConfig("")
}
