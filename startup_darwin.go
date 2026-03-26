//go:build darwin

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
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.tunneldeck.app.plist")
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
			return err
		}
		plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.tunneldeck.app</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>`, exe)
		return os.WriteFile(plistPath, []byte(plist), 0644)
	}
	err = os.Remove(plistPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
