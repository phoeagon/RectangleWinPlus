package main

import (
	"testing"
)

func TestIsSystemClassName(t *testing.T) {
	cases := []struct {
		name     string
		class    string
		expected bool
	}{
		{"SysListView32", "SysListView32", true},
		{"WorkerW", "WorkerW", true},
		{"Shell_TrayWnd", "Shell_TrayWnd", true},
		{"Shell_SecondaryTrayWnd", "Shell_SecondaryTrayWnd", true},
		{"Progman", "Progman", true},
		{"OtherClass", "NotASystemClass", false},
		{"CaseInsensitive", "syslistview32", true},
	}
	for _, c := range cases {
		got := isSystemClassName(c.class)
		if got != c.expected {
			t.Errorf("%s: isSystemClassName(%s) = %v, want %v", c.name, c.class, got, c.expected)
		}
	}
}

func TestIsZonableWindowZero(t *testing.T) {
	if isZonableWindow(0) {
		t.Errorf("isZonableWindow(0) should be false")
	}
}
