package w32ex

import (
	"testing"
)

func TestRegisterHotKeyZero(t *testing.T) {
	// Call with zero arguments; ensure it does not panic and returns a bool.
	_ = RegisterHotKey(0, 0, 0, 0)
}

func TestUnregisterHotKeyZero(t *testing.T) {
	// Call with zero arguments; ensure it does not panic and returns a bool.
	_ = UnregisterHotKey(0, 0)
}

func TestGetDpiForWindowZero(t *testing.T) {
	// Zero HWND should return a non-negative DPI.
	dpi := GetDpiForWindow(0)
	if dpi < 0 {
		t.Errorf("GetDpiForWindow returned negative DPI: %d", dpi)
	}
}

func TestGetWindowModuleFileNameZero(t *testing.T) {
	// Zero HWND should return empty string.
	name := GetWindowModuleFileName(0)
	if name != "" {
		t.Errorf("GetWindowModuleFileName(0) = %s, want empty string", name)
	}
}

func TestGetAncestorZero(t *testing.T) {
	// Zero HWND should return zero.
	anc := GetAncestor(0, GA_ROOT)
	if anc != 0 {
		t.Errorf("GetAncestor(0) = %v, want 0", anc)
	}
}

func TestGetShellWindow(t *testing.T) {
	// Should return a HWND; may be zero in non‑GUI environments.
	hwnd := GetShellWindow()
	if hwnd == 0 {
		t.Log("GetShellWindow returned 0; this may occur in non‑GUI environments")
	}
}

func TestSetProcessDPIAware(t *testing.T) {
	// Ensure the call does not panic.
	_ = SetProcessDPIAware()
}
