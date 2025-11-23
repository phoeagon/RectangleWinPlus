package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"gopkg.in/yaml.v3"
)

func restartMainApp() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		return
	}

	// Launch the main app without the --settings-window flag
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to restart main app: %v\n", err)
		walk.MsgBox(nil, "Error", fmt.Sprintf("Failed to restart application: %v", err), walk.MsgBoxIconError)
	} else {
		fmt.Println("Main application restarted successfully")
	}
}

type SettingsWindowApp struct {
	*walk.MainWindow
	rows      []*HotkeyRow
	recording *HotkeyRow
	handlerID int
}

type HotkeyRow struct {
	Feature     string
	DisplayName string
	Binding     KeyBinding
	Button      *walk.PushButton
	ClearBtn    *walk.PushButton
	Label       *walk.Label
}

func (r *HotkeyRow) UpdateText() {
	if r.Button != nil {
		r.Button.SetText(formatHotkey(r.Binding))
	}
}

func runSettingsWindow() {
	sw := &SettingsWindowApp{}

	// Load current config
	config := fetchConfiguration()

	// Create a map of existing bindings for easy lookup
	bindingMap := make(map[string]KeyBinding)
	for _, kb := range config.Keybindings {
		bindingMap[kb.BindFeature] = kb
	}

	// Prepare ordered list of features
	orderedKeys := []string{
		"moveToLeft", "moveToRight", "moveToTop", "moveToBottom",
		"moveToTopLeft", "moveToTopRight", "moveToBottomLeft", "moveToBottomRight",
		"moveToCenter", "maximize", "almostMaximize", "makeLarger", "makeSmaller", "makeFullHeight",
		"nextDisplay", "prevDisplay", "toggleAlwaysOnTop",
	}

	// Build rows
	for _, key := range orderedKeys {
		if displayName, ok := featureDisplayNames[key]; ok {
			binding, exists := bindingMap[key]
			if !exists {
				binding = KeyBinding{BindFeature: key}
			}

			row := &HotkeyRow{
				Feature:     key,
				DisplayName: displayName,
				Binding:     binding,
			}
			sw.rows = append(sw.rows, row)
		}
	}

	if _, err := (MainWindow{
		AssignTo: &sw.MainWindow,
		Title:    "RectangleWin Plus Settings",
		MinSize:  Size{Width: 700, Height: 600},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: "Keyboard Shortcuts",
				Font: Font{PointSize: 12, Bold: true},
			},
			ScrollView{
				Layout: VBox{},
				Children: []Widget{
					Composite{
						Layout:   VBox{},
						Children: buildSettingsRows(sw),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Save",
						OnClicked: func() {
							saveSettings(sw)
						},
					},
					PushButton{
						Text: "Cancel",
						OnClicked: func() {
							restartMainApp()
							sw.Close()
						},
					},
				},
			},
		},
	}).Run(); err != nil {
		fmt.Printf("Failed to open settings UI: %v\n", err)
	}

	// Relaunch main app after settings window closes
	restartMainApp()
}

func buildSettingsRows(sw *SettingsWindowApp) []Widget {
	var widgets []Widget
	for _, row := range sw.rows {
		r := row // capture loop variable
		widgets = append(widgets, Composite{
			Layout: HBox{},
			Children: []Widget{
				Label{
					AssignTo: &r.Label,
					Text:     r.DisplayName,
					MinSize:  Size{Width: 200, Height: 0},
				},
				HSpacer{},
				PushButton{
					AssignTo: &r.Button,
					Text:     formatHotkey(r.Binding),
					MinSize:  Size{Width: 200, Height: 0},
					OnClicked: func() {
						startRecordingHotkey(sw, r)
					},
				},
				PushButton{
					AssignTo: &r.ClearBtn,
					Text:     "Clear",
					MaxSize:  Size{Width: 60, Height: 0},
					OnClicked: func() {
						r.Binding.Key = ""
						r.Binding.KeyCode = 0
						r.Binding.Modifier = nil
						r.Binding.ModifierCode = nil
						r.Binding.CombinedMod = 0
						r.UpdateText()
					},
				},
			},
		})
	}
	return widgets
}

func startRecordingHotkey(sw *SettingsWindowApp, row *HotkeyRow) {
	if sw.recording != nil {
		if sw.handlerID != 0 {
			sw.recording.Button.KeyDown().Detach(sw.handlerID)
			sw.handlerID = 0
		}
		sw.recording.UpdateText() // Reset previous
	}
	sw.recording = row
	row.Button.SetText("Press keys...")

	sw.handlerID = row.Button.KeyDown().Attach(func(key walk.Key) {
		if sw.recording == nil {
			return
		}

		// Get modifiers
		mods := walk.ModifiersDown()

		// Map walk modifiers to our app's modifiers
		var modifiers []string
		var modCodes []int32

		if mods&walk.ModControl != 0 {
			modifiers = append(modifiers, "Ctrl")
			modCodes = append(modCodes, MOD_CONTROL)
		}
		if mods&walk.ModAlt != 0 {
			modifiers = append(modifiers, "Alt")
			modCodes = append(modCodes, MOD_ALT)
		}
		if mods&walk.ModShift != 0 {
			modifiers = append(modifiers, "Shift")
			modCodes = append(modCodes, MOD_SHIFT)
		}

		// Map key
		keyName := mapWalkKeyToName(key)
		keyCode := int32(key)

		// Ignore modifier-only presses
		if isModifierKey(key) {
			return
		}

		// Update binding
		sw.recording.Binding.Key = keyName
		sw.recording.Binding.KeyCode = keyCode
		sw.recording.Binding.Modifier = modifiers
		sw.recording.Binding.ModifierCode = modCodes
		sw.recording.Binding.CombinedMod = bitwiseOr(modCodes)

		sw.recording.UpdateText()

		// Detach and cleanup
		if sw.handlerID != 0 {
			sw.recording.Button.KeyDown().Detach(sw.handlerID)
			sw.handlerID = 0
		}
		sw.recording = nil
	})
}

func isModifierKey(key walk.Key) bool {
	return key == walk.KeyControl || key == walk.KeyAlt || key == walk.KeyShift
}

func saveSettings(sw *SettingsWindowApp) {
	// Construct new configuration
	newConfig := Configuration{}
	for _, row := range sw.rows {
		if row.Binding.Key != "" {
			newConfig.Keybindings = append(newConfig.Keybindings, row.Binding)
		}
	}

	// Save to file
	data, err := yaml.Marshal(newConfig)
	if err != nil {
		walk.MsgBox(sw, "Error", "Failed to marshal config: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	configPath, err := getValidConfigPathOrCreate()
	if err != nil {
		walk.MsgBox(sw, "Error", "Failed to get config path: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		walk.MsgBox(sw, "Error", "Failed to write config file: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	walk.MsgBox(sw, "Success", "Settings saved! Restarting RectangleWin Plus...", walk.MsgBoxIconInformation)
	sw.Close()
	// Main app will be restarted when runSettingsWindow returns
}

func mapWalkKeyToName(key walk.Key) string {
	switch key {
	case walk.KeyUp:
		return "UP_ARROW"
	case walk.KeyDown:
		return "DOWN_ARROW"
	case walk.KeyLeft:
		return "LEFT_ARROW"
	case walk.KeyRight:
		return "RIGHT_ARROW"
	case walk.KeyReturn:
		return "ENTER"
	case walk.KeyEscape:
		return "ESCAPE"
	case walk.KeySpace:
		return "SPACE"
	case walk.KeyBack:
		return "BACKSPACE"
	case walk.KeyDelete:
		return "DELETE"
	case walk.KeyTab:
		return "TAB"
	}

	// For letters and numbers, walk.Key string is usually correct (e.g. "A", "1")
	s := key.String()
	if len(s) == 1 {
		return s
	}
	return strings.ToUpper(s)
}
