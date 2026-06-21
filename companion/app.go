package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Server e um servidor de voz salvo (multi-servidor; lista fica escondida na UI).
type Server struct {
	Name             string  `json:"name"`
	URL              string  `json:"url"`
	Password         string  `json:"password"`
	VoiceRangeMeters float64 `json:"voiceRangeMeters"`
}

// Config persistida. Lista de servidores + selecionado; audio e global.
type Config struct {
	Servers        []Server `json:"servers"`
	Selected       int      `json:"selected"`
	Volume         float64  `json:"volume"`
	MicDeviceID    string   `json:"micDeviceId"`
	OutputDeviceID string   `json:"outputDeviceId"`
	AutoConnect    bool     `json:"autoConnect"`
	AutoDetect     bool     `json:"autoDetect"`   // detectar IP do servidor do jogo (Direct Connect)
	AutoPort       int      `json:"autoPort"`     // porta da voz no IP detectado (padrao 8765)
	AutoPassword   string   `json:"autoPassword"` // senha do modo auto (vazio = sem senha)
}

// GameServerIP le o ultimo IP de Direct Connect do GameUserSettings.ini do Palworld
// (so funciona pra Direct Connect; lista do Steam nao atualiza esse campo).
func (a *App) GameServerIP() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return ""
	}
	for _, sub := range []string{"WinGDK", "Windows"} {
		p := filepath.Join(local, "Pal", "Saved", "Config", sub, "GameUserSettings.ini")
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if v, ok := strings.CutPrefix(line, "InputIPAddress="); ok {
				v = strings.TrimSpace(v)
				if v == "" {
					continue
				}
				if i := strings.LastIndex(v, ":"); i > 0 { // tira a porta, fica so o IP/host
					v = v[:i]
				}
				return v
			}
		}
	}
	return ""
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func defaultConfig() Config {
	return Config{Servers: []Server{}, Selected: 0, Volume: 1.0, AutoConnect: true, AutoDetect: true, AutoPort: 8765}
}

// configPath = %APPDATA%/PalProxVoice/config.json (config salva pelo usuario).
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "PalProxVoice", "config.json"), nil
}

// GetConfig: 1) config salva do usuario; 2) config.json padrao ao lado do .exe
// (o que voce distribui); 3) vazio (a UI pede pra configurar).
func (a *App) GetConfig() Config {
	if path, err := configPath(); err == nil {
		if data, rerr := os.ReadFile(path); rerr == nil {
			cfg := defaultConfig()
			if json.Unmarshal(data, &cfg) == nil && len(cfg.Servers) > 0 {
				return cfg
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		def := filepath.Join(filepath.Dir(exe), "config.json")
		if data, rerr := os.ReadFile(def); rerr == nil {
			cfg := defaultConfig()
			if json.Unmarshal(data, &cfg) == nil {
				return cfg
			}
		}
	}
	return defaultConfig()
}

// SaveConfig grava a config do usuario em %APPDATA%.
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
