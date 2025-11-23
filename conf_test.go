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

func TestBitwiseOr(t *testing.T) {
	if got := bitwiseOr([]int32{}); got != 0 {
		t.Errorf("bitwiseOr empty slice = %d, want 0", got)
	}
	if got := bitwiseOr([]int32{5}); got != 5 {
		t.Errorf("bitwiseOr single = %d, want 5", got)
	}
	if got := bitwiseOr([]int32{1, 2, 4}); got != 7 {
		t.Errorf("bitwiseOr multiple = %d, want 7", got)
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
