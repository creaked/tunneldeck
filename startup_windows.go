//go:build windows

package main

import (
	"os"
	"os/exec"
)

func setStartOnBoot(enable bool) error {
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
			"/v", "TunnelDeck", "/t", "REG_SZ", "/d", exe, "/f",
		).Run()
	}
	// /f suppresses "value not found" errors when already absent.
	exec.Command("reg", "delete",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "TunnelDeck", "/f",
	).Run()
	return nil
}
