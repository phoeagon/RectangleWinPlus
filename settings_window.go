package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gonutz/w32/v2"
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
	if *debug {
		cmd = exec.Command(exe, "--debug")
	}
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
		"pushToLeft", "pushToRight", "pushToTop", "pushToBottom",
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
		MinSize:  Size{Width: 900, Height: 600},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text: "Export Config...",
						OnClicked: func() {
							exportConfig(sw)
						},
					},
					PushButton{
						Text: "Import Config...",
						OnClicked: func() {
							importConfig(sw)
						},
					},
					PushButton{
						Text: "Edit Config in Text Editor",
						OnClicked: func() {
							openConfigInEditor(sw)
						},
					},
					PushButton{
						Text: "Import from URL...",
						OnClicked: func() {
							importConfigFromURL(sw)
						},
					},
					HSpacer{},
				},
			},
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
	// Split rows into two columns
	midpoint := (len(sw.rows) + 1) / 2
	leftRows := sw.rows[:midpoint]
	rightRows := sw.rows[midpoint:]

	// Build left column widgets
	var leftWidgets []Widget
	for _, row := range leftRows {
		r := row // capture loop variable
		leftWidgets = append(leftWidgets, buildRowWidget(sw, r))
	}

	// Build right column widgets
	var rightWidgets []Widget
	for _, row := range rightRows {
		r := row // capture loop variable
		rightWidgets = append(rightWidgets, buildRowWidget(sw, r))
	}

	// Return a two-column layout
	return []Widget{
		Composite{
			Layout: HBox{Spacing: 20},
			Children: []Widget{
				Composite{
					Layout:   VBox{Spacing: 5},
					Children: leftWidgets,
				},
				Composite{
					Layout:   VBox{Spacing: 5},
					Children: rightWidgets,
				},
			},
		},
	}
}

func buildRowWidget(sw *SettingsWindowApp, r *HotkeyRow) Widget {
	return Composite{
		Layout: HBox{},
		Children: []Widget{
			Label{
				AssignTo: &r.Label,
				Text:     r.DisplayName,
				MinSize:  Size{Width: 150, Height: 0},
			},
			HSpacer{},
			PushButton{
				AssignTo: &r.Button,
				Text:     formatHotkey(r.Binding),
				MinSize:  Size{Width: 150, Height: 0},
				OnClicked: func() {
					startRecordingHotkey(sw, r)
				},
			},
			PushButton{
				AssignTo: &r.ClearBtn,
				Text:     "×",
				MaxSize:  Size{Width: 30, Height: 0},
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
	}
}

func startRecordingHotkey(sw *SettingsWindowApp, row *HotkeyRow) {
	sw.recording = row

	var dlg *walk.Dialog
	var lineEdit *walk.LineEdit

	// Create a dialog with a LineEdit to capture keyboard input
	if _, err := (Dialog{
		AssignTo: &dlg,
		Title:    "Record Hotkey - " + row.DisplayName,
		MinSize:  Size{Width: 300, Height: 100},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: "Press your desired key combination...",
			},
			LineEdit{
				AssignTo: &lineEdit,
				ReadOnly: true,
				Text:     "Waiting for input...",
				OnKeyDown: func(key walk.Key) {
					// Get modifiers
					mods := walk.ModifiersDown()

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
					// Check for Windows key (walk.ModifiersDown doesn't include it)
					// VK_LWIN = 0x5B, VK_RWIN = 0x5C
					// GetAsyncKeyState returns a short where the high-order bit indicates if key is down
					if (w32.GetAsyncKeyState(w32.VK_LWIN)&0x8000 != 0) || (w32.GetAsyncKeyState(w32.VK_RWIN)&0x8000 != 0) {
						modifiers = append(modifiers, "Win")
						modCodes = append(modCodes, MOD_WIN)
					}

					// Map key
					keyName := mapWalkKeyToName(key)
					keyCode := int32(key)

					// Ignore modifier-only presses
					if isModifierKey(key) {
						return
					}

					// Create temporary binding for validation
					tempBinding := KeyBinding{
						Key:          keyName,
						KeyCode:      keyCode,
						Modifier:     modifiers,
						ModifierCode: modCodes,
						CombinedMod:  bitwiseOr(modCodes),
					}

					// Check for duplicate hotkey
					if conflictRow := findHotkeyConflict(sw, sw.recording, tempBinding); conflictRow != nil {
						errorMsg := fmt.Sprintf("This hotkey is already assigned to '%s'.\\n\\nPlease choose a different key combination.", conflictRow.DisplayName)
						walk.MsgBox(dlg, "Duplicate Hotkey", errorMsg, walk.MsgBoxIconWarning)
						return
					}

					// Update binding
					sw.recording.Binding.Key = keyName
					sw.recording.Binding.KeyCode = keyCode
					sw.recording.Binding.Modifier = modifiers
					sw.recording.Binding.ModifierCode = modCodes
					sw.recording.Binding.CombinedMod = bitwiseOr(modCodes)

					// Update the row's button text
					sw.recording.UpdateText()

					// Close the dialog
					dlg.Accept()
					sw.recording = nil
				},
			},
		},
	}).Run(sw.MainWindow); err != nil {
		fmt.Printf("Failed to create recording dialog: %v\n", err)
		sw.recording = nil
		return
	}
}

func isModifierKey(key walk.Key) bool {
	return key == walk.KeyControl || key == walk.KeyAlt || key == walk.KeyShift || key == walk.Key(w32.VK_LWIN) || key == walk.Key(w32.VK_RWIN)
}

// findHotkeyConflict checks if the given hotkey conflicts with any other row's hotkey
// Returns the conflicting row if found, nil otherwise
func findHotkeyConflict(sw *SettingsWindowApp, currentRow *HotkeyRow, newBinding KeyBinding) *HotkeyRow {
	// Empty hotkey cannot conflict
	if newBinding.Key == "" {
		return nil
	}

	for _, row := range sw.rows {
		// Skip the current row being edited
		if row == currentRow {
			continue
		}

		// Skip empty bindings
		if row.Binding.Key == "" {
			continue
		}

		// Check if hotkeys match
		if hotkeyMatches(row.Binding, newBinding) {
			return row
		}
	}

	return nil
}

// hotkeyMatches checks if two key bindings represent the same hotkey
func hotkeyMatches(kb1, kb2 KeyBinding) bool {
	// Compare key codes
	if kb1.KeyCode != kb2.KeyCode {
		return false
	}

	// Compare combined modifiers
	if kb1.CombinedMod != kb2.CombinedMod {
		return false
	}

	return true
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
	return strings.ReplaceAll(keyNames[int(key)], " key", "")
}

// exportConfig allows the user to save the current configuration to a file
func exportConfig(sw *SettingsWindowApp) {
	dlg := new(walk.FileDialog)
	dlg.Title = "Export Configuration"
	dlg.Filter = "YAML Files (*.yaml)|*.yaml|All Files (*.*)|*.*"

	if ok, err := dlg.ShowSave(sw); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to show save dialog: %v", err), walk.MsgBoxIconError)
		return
	} else if !ok {
		return // User cancelled
	}

	// Get current config path
	configPath, err := getValidConfigPathOrCreate()
	if err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to get config path: %v", err), walk.MsgBoxIconError)
		return
	}

	// Read current config
	data, err := os.ReadFile(configPath)
	if err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to read config file: %v", err), walk.MsgBoxIconError)
		return
	}

	// Write to selected file
	if err := os.WriteFile(dlg.FilePath, data, 0644); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to export config: %v", err), walk.MsgBoxIconError)
		return
	}

	walk.MsgBox(sw, "Success", fmt.Sprintf("Configuration exported to:\n%s", dlg.FilePath), walk.MsgBoxIconInformation)
}

// importConfig allows the user to load a configuration from a file
func importConfig(sw *SettingsWindowApp) {
	dlg := new(walk.FileDialog)
	dlg.Title = "Import Configuration"
	dlg.Filter = "YAML Files (*.yaml)|*.yaml|All Files (*.*)|*.*"

	if ok, err := dlg.ShowOpen(sw); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to show open dialog: %v", err), walk.MsgBoxIconError)
		return
	} else if !ok {
		return // User cancelled
	}

	// Read selected file
	data, err := os.ReadFile(dlg.FilePath)
	if err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to read file: %v", err), walk.MsgBoxIconError)
		return
	}

	// Validate YAML format
	var testConfig Configuration
	if err := yaml.Unmarshal(data, &testConfig); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Invalid configuration file:\n%v", err), walk.MsgBoxIconError)
		return
	}

	// Get config path
	configPath, err := getValidConfigPathOrCreate()
	if err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to get config path: %v", err), walk.MsgBoxIconError)
		return
	}

	// Write to config location
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to import config: %v", err), walk.MsgBoxIconError)
		return
	}

	walk.MsgBox(sw, "Success", "Configuration imported successfully!\n\nPlease restart RectangleWin Plus to apply the changes.", walk.MsgBoxIconInformation)

	// Close settings window and restart main app
	sw.Close()
}

// openConfigInEditor opens the configuration file in notepad
func openConfigInEditor(sw *SettingsWindowApp) {
	configFilePath, err := getValidConfigPathOrCreate()
	if err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Can't locate config path:\n%v", err), walk.MsgBoxIconError)
		return
	}

	cmd := exec.Command("notepad.exe", configFilePath)
	if err := cmd.Start(); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to open config file in notepad:\n%v", err), walk.MsgBoxIconError)
		return
	}

	walk.MsgBox(sw, "Info", "Config file opened in Notepad.\n\nAfter making changes, save the file and restart RectangleWin Plus to apply them.", walk.MsgBoxIconInformation)
}

// importConfigFromURL allows the user to import configuration from a web URL
func importConfigFromURL(sw *SettingsWindowApp) {
	var dlg *walk.Dialog
	var urlEdit *walk.LineEdit
	var url string

	// Create URL input dialog
	if _, err := (Dialog{
		AssignTo: &dlg,
		Title:    "Import Configuration from URL",
		MinSize:  Size{Width: 500, Height: 150},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: "Enter the URL of the configuration file:",
			},
			Label{
				Text:      "Supports: GitHub files, GitHub Gist, Bitbucket, and direct raw URLs",
				TextColor: walk.RGB(100, 100, 100),
			},
			LineEdit{
				AssignTo: &urlEdit,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Import",
						OnClicked: func() {
							url = urlEdit.Text()
							dlg.Accept()
						},
					},
					PushButton{
						Text: "Cancel",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}).Run(sw); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to show URL dialog: %v", err), walk.MsgBoxIconError)
		return
	}

	// Clean up the URL
	url = strings.TrimSpace(url)
	if url == "" {
		return // User cancelled or entered empty URL
	}

	// Convert to raw URL if needed
	rawURL := convertToRawURL(url)
	if *debug {
		fmt.Printf("Original URL: %s\n", url)
		fmt.Printf("Downloading config from: %s\n", rawURL)
	}

	// Download the file
	resp, err := http.Get(rawURL)
	if err != nil {
		walk.MsgBox(sw, "Import Failed", fmt.Sprintf("Failed to download from URL:\n%v\n\nURL: %s", err, rawURL), walk.MsgBoxIconError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		walk.MsgBox(sw, "Import Failed", fmt.Sprintf("Failed to download config.\nHTTP Status: %d %s\n\nURL: %s\n\nPlease check the URL and try again.", resp.StatusCode, resp.Status, rawURL), walk.MsgBoxIconError)
		return
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		walk.MsgBox(sw, "Import Failed", fmt.Sprintf("Failed to read downloaded content:\n%v", err), walk.MsgBoxIconError)
		return
	}

	// Validate YAML format
	var testConfig Configuration
	if err := yaml.Unmarshal(data, &testConfig); err != nil {
		walk.MsgBox(sw, "Import Failed", fmt.Sprintf("Invalid configuration file (not valid YAML):\n%v\n\nPlease check the file format.", err), walk.MsgBoxIconError)
		return
	}

	// Show preview dialog with the downloaded content
	var previewDlg *walk.Dialog
	var textEdit *walk.TextEdit
	var accepted bool

	if _, err := (Dialog{
		AssignTo: &previewDlg,
		Title:    "Preview Configuration",
		MinSize:  Size{Width: 600, Height: 400},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: fmt.Sprintf("Downloaded from: %s\n\nPreview the configuration below and click 'Import' to proceed:", rawURL),
			},
			TextEdit{
				AssignTo: &textEdit,
				ReadOnly: true,
				VScroll:  true,
				Text:     string(data),
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Import",
						OnClicked: func() {
							accepted = true
							previewDlg.Accept()
						},
					},
					PushButton{
						Text: "Cancel",
						OnClicked: func() {
							accepted = false
							previewDlg.Cancel()
						},
					},
				},
			},
		},
	}).Run(sw); err != nil {
		walk.MsgBox(sw, "Error", fmt.Sprintf("Failed to show preview dialog: %v", err), walk.MsgBoxIconError)
		return
	}

	// If user cancelled the preview, don't import
	if !accepted {
		walk.MsgBox(sw, "Import Cancelled", "Configuration import was cancelled.", walk.MsgBoxIconInformation)
		return
	}

	// Get config path
	configPath, err := getValidConfigPathOrCreate()
	if err != nil {
		walk.MsgBox(sw, "Import Failed", fmt.Sprintf("Failed to get config path: %v", err), walk.MsgBoxIconError)
		return
	}

	// Write to config location
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		walk.MsgBox(sw, "Import Failed", fmt.Sprintf("Failed to write config file: %v", err), walk.MsgBoxIconError)
		return
	}

	// Success message
	walk.MsgBox(sw, "Import Successful", "Configuration imported successfully from URL!\n\nThe settings window will now close and RectangleWin Plus will restart to apply the changes.", walk.MsgBoxIconInformation)

	// Close settings window and restart main app
	sw.Close()
}
