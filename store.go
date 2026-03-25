package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	path string
}

type StoreData struct {
	Tunnels []TunnelConfig `json:"tunnels"`
}

func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	appDir := filepath.Join(dir, "TunnelDeck")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(appDir, "tunnels.json")}, nil
}

func (s *Store) Load() ([]TunnelConfig, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []TunnelConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var stored StoreData
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	return stored.Tunnels, nil
}

func (s *Store) Save(tunnels []TunnelConfig) error {
	data, err := json.MarshalIndent(StoreData{Tunnels: tunnels}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}
