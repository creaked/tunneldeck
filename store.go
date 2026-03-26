package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	path string
}

// diskTunnel is the on-disk representation of a tunnel.
// EncryptedPassword holds the credential encrypted by encryptSecret().
// Password is retained for reading legacy plaintext files only.
type diskTunnel struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	SSHHost           string `json:"sshHost"`
	SSHPort           int    `json:"sshPort"`
	User              string `json:"user"`
	AuthType          string `json:"authType"`
	Password          string `json:"password,omitempty"`          // legacy plaintext
	EncryptedPassword string `json:"encryptedPassword,omitempty"` // DPAPI / AES-GCM blob
	KeyPath           string `json:"keyPath,omitempty"`
	LocalPort         int    `json:"localPort"`
	RemoteHost        string `json:"remoteHost"`
	RemotePort        int    `json:"remotePort"`
}

type storeData struct {
	Tunnels []diskTunnel `json:"tunnels"`
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
	var stored storeData
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	tunnels := make([]TunnelConfig, len(stored.Tunnels))
	needsMigration := false
	for i, dt := range stored.Tunnels {
		tc := TunnelConfig{
			ID:         dt.ID,
			Name:       dt.Name,
			SSHHost:    dt.SSHHost,
			SSHPort:    dt.SSHPort,
			User:       dt.User,
			AuthType:   dt.AuthType,
			KeyPath:    dt.KeyPath,
			LocalPort:  dt.LocalPort,
			RemoteHost: dt.RemoteHost,
			RemotePort: dt.RemotePort,
		}
		switch {
		case dt.EncryptedPassword != "":
			pw, err := decryptSecret(dt.EncryptedPassword)
			if err != nil {
				return nil, err
			}
			tc.Password = pw
		case dt.Password != "":
			// Legacy plaintext — migrate to encrypted on next save.
			tc.Password = dt.Password
			needsMigration = true
		}
		tunnels[i] = tc
	}

	if needsMigration {
		_ = s.Save(tunnels) // best-effort; silently ignore errors during migration
	}

	return tunnels, nil
}

func (s *Store) Save(tunnels []TunnelConfig) error {
	dts := make([]diskTunnel, len(tunnels))
	for i, tc := range tunnels {
		dt := diskTunnel{
			ID:         tc.ID,
			Name:       tc.Name,
			SSHHost:    tc.SSHHost,
			SSHPort:    tc.SSHPort,
			User:       tc.User,
			AuthType:   tc.AuthType,
			KeyPath:    tc.KeyPath,
			LocalPort:  tc.LocalPort,
			RemoteHost: tc.RemoteHost,
			RemotePort: tc.RemotePort,
		}
		if tc.Password != "" {
			enc, err := encryptSecret(tc.Password)
			if err != nil {
				return err
			}
			dt.EncryptedPassword = enc
		}
		dts[i] = dt
	}
	data, err := json.MarshalIndent(storeData{Tunnels: dts}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}
