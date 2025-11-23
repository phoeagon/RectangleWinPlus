package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func openSettingsUI() {
	// Launch settings UI in a separate process to avoid message loop conflicts
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		return
	}

	// Launch with a special flag to open settings window
	cmd := exec.Command(exe, "--settings-window")
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to launch settings window: %v\n", err)
		showMessageBox(fmt.Sprintf("Failed to open settings: %v", err))
	}
}

func formatHotkey(kb KeyBinding) string {
	if kb.Key == "" {
		return "Not set"
	}
	var parts []string
	for _, mod := range kb.Modifier {
		parts = append(parts, mod)
	}
	parts = append(parts, kb.Key)
	return strings.Join(parts, " + ")
}
