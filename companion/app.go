package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the persisted companion configuration.
type Config struct {
	ServerURL        string  `json:"serverUrl"`
	Password         string  `json:"password"`
	VoiceRangeMeters float64 `json:"voiceRangeMeters"`
	Volume           float64 `json:"volume"`
}

// App holds the Wails runtime context and exposes bound methods.
type App struct {
	ctx context.Context
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// defaultConfig returns the baseline configuration used when no file exists or
// the stored file cannot be parsed.
func defaultConfig() Config {
	return Config{
		ServerURL:        "",
		Password:         "",
		VoiceRangeMeters: 50.0,
		Volume:           1.0,
	}
}

// configPath returns os.UserConfigDir()/PalProxVoice/config.json.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "PalProxVoice", "config.json"), nil
}

// GetConfig reads the Config from disk, returning defaults if it does not
// exist or is corrupt.
func (a *App) GetConfig() Config {
	path, err := configPath()
	if err != nil {
		return defaultConfig()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig() // missing -> defaults
	}

	cfg := defaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultConfig() // corrupt -> defaults
	}

	return cfg
}

// SaveConfig persists the Config to disk, creating the directory if needed.
func (a *App) SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
