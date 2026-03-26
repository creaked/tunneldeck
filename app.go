package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type App struct {
	ctx      context.Context
	tunnels  []TunnelConfig
	manager  *TunnelManager
	store    *Store
	settings Settings
}

func NewApp() *App {
	store, _ := NewStore()
	tunnels, _ := store.Load()
	if tunnels == nil {
		tunnels = []TunnelConfig{}
	}
	settings, _ := store.LoadSettings()
	return &App{
		tunnels:  tunnels,
		manager:  NewTunnelManager(settings),
		store:    store,
		settings: settings,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	for _, t := range a.tunnels {
		if t.AutoStart {
			_ = a.manager.Start(t)
		}
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.manager.StopAll()
}

func (a *App) GetTunnels() []TunnelConfig {
	return a.tunnels
}

func (a *App) AddTunnel(cfg TunnelConfig) (TunnelConfig, error) {
	cfg.ID = uuid.New().String()
	a.tunnels = append(a.tunnels, cfg)
	return cfg, a.store.Save(a.tunnels)
}

func (a *App) UpdateTunnel(cfg TunnelConfig) error {
	for i, t := range a.tunnels {
		if t.ID == cfg.ID {
			a.manager.Stop(cfg.ID)
			a.tunnels[i] = cfg
			return a.store.Save(a.tunnels)
		}
	}
	return fmt.Errorf("tunnel %s not found", cfg.ID)
}

func (a *App) DeleteTunnel(id string) error {
	a.manager.Stop(id)
	for i, t := range a.tunnels {
		if t.ID == id {
			a.tunnels = append(a.tunnels[:i], a.tunnels[i+1:]...)
			return a.store.Save(a.tunnels)
		}
	}
	return fmt.Errorf("tunnel %s not found", id)
}

func (a *App) StartTunnel(id string) error {
	for _, t := range a.tunnels {
		if t.ID == id {
			return a.manager.Start(t)
		}
	}
	return fmt.Errorf("tunnel %s not found", id)
}

func (a *App) StopTunnel(id string) error {
	return a.manager.Stop(id)
}

func (a *App) GetStatuses() []TunnelStatus {
	ids := make([]string, len(a.tunnels))
	for i, t := range a.tunnels {
		ids[i] = t.ID
	}
	return a.manager.GetAllStatuses(ids)
}

func (a *App) GetSettings() Settings {
	return a.settings
}

func (a *App) SaveSettings(s Settings) error {
	if s.StartOnBoot != a.settings.StartOnBoot {
		// Best-effort — don't block the save if OS registration fails.
		_ = setStartOnBoot(s.StartOnBoot)
	}
	a.settings = s
	a.manager.UpdateSettings(s)
	return a.store.SaveSettings(s)
}
