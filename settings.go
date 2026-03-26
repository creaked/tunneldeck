package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	AutoReconnect    bool   `json:"autoReconnect"`
	KeepaliveSeconds int    `json:"keepaliveSeconds"`
	StartOnBoot      bool   `json:"startOnBoot"`
	Theme            string `json:"theme"` // "dark" | "light" | "system"
	DefaultSSHPort   int    `json:"defaultSshPort"`
	DefaultSSHUser   string `json:"defaultSshUser"`
	DefaultKeyPath   string `json:"defaultKeyPath"`
}

func DefaultSettings() Settings {
	return Settings{
		AutoReconnect:    true,
		KeepaliveSeconds: 15,
		StartOnBoot:      false,
		Theme:            "dark",
		DefaultSSHPort:   22,
	}
}

func (s *Store) LoadSettings() (Settings, error) {
	def := DefaultSettings()
	data, err := os.ReadFile(s.settingsPath())
	if os.IsNotExist(err) {
		return def, nil
	}
	if err != nil {
		return def, err
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		return def, err
	}
	// Fill zero values with defaults so old files get sensible values.
	if out.KeepaliveSeconds == 0 {
		out.KeepaliveSeconds = def.KeepaliveSeconds
	}
	if out.DefaultSSHPort == 0 {
		out.DefaultSSHPort = def.DefaultSSHPort
	}
	if out.Theme == "" {
		out.Theme = def.Theme
	}
	return out, nil
}

func (s *Store) SaveSettings(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.settingsPath(), data, 0600)
}

func (s *Store) settingsPath() string {
	return filepath.Join(filepath.Dir(s.path), "settings.json")
}
