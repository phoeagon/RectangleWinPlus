package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertModifier(t *testing.T) {
	cases := []struct {
		input    string
		expected int32
		wantErr  bool
	}{
		{"Ctrl", MOD_CONTROL, false},
		{"alt", MOD_ALT, false},
		{"SHIFT", MOD_SHIFT, false},
		{"win", MOD_WIN, false},
		{"meta", MOD_WIN, false},
		{"super", MOD_WIN, false},
		{"unknown", 0, true},
	}
	for _, c := range cases {
		got, err := convertModifier(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("convertModifier(%s) expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("convertModifier(%s) unexpected error: %v", c.input, err)
			continue
		}
		if got != c.expected {
			t.Errorf("convertModifier(%s) = %d, want %d", c.input, got, c.expected)
		}
	}
}

func TestConvertKeyCode(t *testing.T) {
	cases := []struct {
		input    string
		expected int32
		wantErr  bool
	}{
		{"a", int32('A'), false},
		{"Z", int32('Z'), false},
		{"0", int32('0'), false},
		{"9", int32('9'), false},
		{"up_arrow", 0x26, false},    // w32.VK_UP is 0x26
		{"down_arrow", 0x28, false},  // w32.VK_DOWN
		{"left_arrow", 0x25, false},  // w32.VK_LEFT
		{"right_arrow", 0x27, false}, // w32.VK_RIGHT
		{"-", 189, false},
		{"=", 187, false},
		{"|", 124, false},
		{"\\", 124, false},
		{"invalidkey", 0, true},
	}
	for _, c := range cases {
		got, err := convertKeyCode(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("convertKeyCode(%s) expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("convertKeyCode(%s) unexpected error: %v", c.input, err)
			continue
		}
		if got != c.expected {
			t.Errorf("convertKeyCode(%s) = %d, want %d", c.input, got, c.expected)
		}
	}
}

func TestGetValidConfigPathOrCreate(t *testing.T) {
	// Create a temporary home directory
	tmpDir, err := os.MkdirTemp("", "conf_test_home")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	// Override HOME env var
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	path, err := getValidConfigPathOrCreate()
	if err != nil {
		t.Fatalf("getValidConfigPathOrCreate returned error: %v", err)
	}
	expectedSuffix := "config.yaml"
	if !strings.HasSuffix(path, expectedSuffix) {
		t.Errorf("config path %s does not end with %s", path, expectedSuffix)
	}
	// Ensure directory exists
	dir := filepath.Dir(path)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("config directory %s not created", dir)
	}
}

func TestParseConfiguration(t *testing.T) {
	input := Configuration{
		Keybindings: []KeyBinding{
			{
				Modifier:    []string{"Ctrl", "Alt"},
				Key:         "T",
				BindFeature: "previousDisplay",
			},
			{
				Modifier:    []string{"Win"},
				Key:         "UP_ARROW",
				BindFeature: "maximize",
			},
		},
	}

	parsed := parseConfiguration(input)

	// Check alias handling
	if parsed.Keybindings[0].BindFeature != "prevDisplay" {
		t.Errorf("expected BindFeature 'prevDisplay', got '%s'", parsed.Keybindings[0].BindFeature)
	}

	// Check modifier conversion
	expectedMod0 := int32(MOD_CONTROL | MOD_ALT)
	if parsed.Keybindings[0].CombinedMod != expectedMod0 {
		t.Errorf("expected CombinedMod %d, got %d", expectedMod0, parsed.Keybindings[0].CombinedMod)
	}

	expectedMod1 := int32(MOD_WIN)
	if parsed.Keybindings[1].CombinedMod != expectedMod1 {
		t.Errorf("expected CombinedMod %d, got %d", expectedMod1, parsed.Keybindings[1].CombinedMod)
	}

	// Check key code conversion
	// 'T' is 84, 't' is 116. convertKeyCode('T') -> 't' -> 't'-32 = 84 ('T')
	// Wait, convertKeyCode implementation:
	// k[0] >= 'a' && k[0] <= 'z' -> return int32(k[0]) - 32.
	// So 't' (116) - 32 = 84. Correct.
	expectedKey0 := int32('T')
	if parsed.Keybindings[0].KeyCode != expectedKey0 {
		t.Errorf("expected KeyCode %d, got %d", expectedKey0, parsed.Keybindings[0].KeyCode)
	}

	// UP_ARROW is 0x26 (38)
	expectedKey1 := int32(0x26)
	if parsed.Keybindings[1].KeyCode != expectedKey1 {
		t.Errorf("expected KeyCode %d, got %d", expectedKey1, parsed.Keybindings[1].KeyCode)
	}
}

func TestFetchConfiguration(t *testing.T) {
	// Create a temporary home directory
	tmpDir, err := os.MkdirTemp("", "conf_test_fetch")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override HOME env var
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)
	// Also override USERPROFILE for Windows compatibility in getValidConfigPathOrCreate
	oldUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("USERPROFILE", tmpDir)
	defer os.Setenv("USERPROFILE", oldUserProfile)

	// 1. Test with no config file (should return default or create example)
	// In the current implementation, fetchConfiguration calls getValidConfigPathOrCreate,
	// which creates the directory. Then it calls maybeDropExampleConfigFile.
	// Then it reads the file.
	// So it should return the parsed example config.

	// We can't easily verify the exact return value of "default" vs "example" without knowing the example content,
	// but we can check if it returns a valid config.

	config := fetchConfiguration()
	if len(config.Keybindings) == 0 {
		t.Error("fetchConfiguration returned empty keybindings when file missing (should load example)")
	}

	// 2. Test with a custom config file
	configDir := filepath.Join(tmpDir, ".config", "RectangleWinPlus")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")

	customYaml := `
keybindings:
  - modifier: ["Alt"]
    key: "Z"
    bindfeature: "makeSmaller"
`
	err = os.WriteFile(configPath, []byte(customYaml), 0644)
	if err != nil {
		t.Fatalf("failed to write custom config: %v", err)
	}

	config = fetchConfiguration()
	if len(config.Keybindings) != 1 {
		t.Errorf("expected 1 keybinding, got %d", len(config.Keybindings))
	}
	if config.Keybindings[0].BindFeature != "makeSmaller" {
		t.Errorf("expected BindFeature 'makeSmaller', got '%s'", config.Keybindings[0].BindFeature)
	}
	if config.Keybindings[0].KeyCode != int32('Z') {
		t.Errorf("expected KeyCode %d, got %d", int32('Z'), config.Keybindings[0].KeyCode)
	}
}
