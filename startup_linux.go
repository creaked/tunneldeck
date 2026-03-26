//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func setStartOnBoot(enable bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	autostartDir := filepath.Join(home, ".config", "autostart")
	desktopPath := filepath.Join(autostartDir, "tunneldeck.desktop")
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(autostartDir, 0755); err != nil {
			return err
		}
		desktop := fmt.Sprintf(
			"[Desktop Entry]\nType=Application\nName=TunnelDeck\nExec=%s\nHidden=false\nNoDisplay=false\nX-GNOME-Autostart-enabled=true\n",
			exe,
		)
		return os.WriteFile(desktopPath, []byte(desktop), 0644)
	}
	err = os.Remove(desktopPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
