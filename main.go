// Copyright 2022 Ahmet Alp Balkan
// Copyright 2025 Phoeagon
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO make it possible to "go generate" on Windows (https://github.com/josephspurrier/goversioninfo/issues/52).
//go:generate /bin/bash -c "go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest -arm -64 -icon=assets/icon.ico - <<< '{}'"

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"time"

	"github.com/getlantern/systray"
	"github.com/gonutz/w32/v2"

	"github.com/ahmetb/RectangleWin/w32ex"
	"github.com/apenwarr/fixconsole"
)

var lastResized w32.HWND
var lastActiveWindow w32.HWND
var hks []HotKey
var shouldRestart bool

const currentVersion = "v1.0.1"

type Feature struct {
	Name        string
	DisplayName string
	Callback    func()
	HotkeyDesc  string
}

var features []Feature

func main() {
	// Set custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "RectangleWin Plus - Window management utility for Windows\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFor more information, visit: https://github.com/phoeagon/RectangleWinPlus\n")
	}

	debug := flag.Bool("debug", false, "enable debug mode (show console output)")
	killAll := flag.Bool("killall", false, "kill all RectangleWinPlus instances and quit")
	help := flag.Bool("help", false, "show this help message")
	action := flag.String("action", "", "action to perform (moveToTop, moveToBottom, moveToLeft, moveToRight, moveToTopLeft, moveToTopRight, moveToBottomLeft, moveToBottomRight, maximize, almostMaximize, makeFullHeight, makeLarger, makeSmaller)")
	loadTray := flag.Bool("load_tray", true, "load tray icon")
	flag.Parse()

	// Handle help flag
	if *help {
		flag.Usage()
		return
	}

	if *killAll {
		if err := killAllRectangleWinPlusProcesses(); err != nil {
			fmt.Printf("Failed to kill processes: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All RectangleWinPlus.exe processes terminated successfully")
		return
	}

	if *debug {
		// FixConsole ensures that we can see stdout/stderr in the console
		// even if the app is built as a GUI app (windowsgui).
		if err := fixconsole.FixConsoleIfNeeded(); err != nil {
			fmt.Printf("warn: fixconsole: %v\n", err)
		}
	}

	runtime.LockOSThread() // since we bind hotkeys etc that need to dispatch their message here
	if !w32ex.SetProcessDPIAware() {
		panic("failed to set DPI aware")
	}

	autorun, err := AutoRunEnabled()
	if err != nil {
		panic(err)
	}
	fmt.Printf("autorun enabled=%v\n", autorun)
	printMonitors()

	go func() {
		for {
			time.Sleep(200 * time.Millisecond)
			hwnd := w32.GetForegroundWindow()
			if isZonableWindow(hwnd) {
				lastActiveWindow = hwnd
			}
		}
	}()

	edgeFuncs := [][]resizeFunc{
		{leftHalf, leftTwoThirds, leftOneThirds},
		{rightHalf, rightTwoThirds, rightOneThirds},
		{topHalf, topTwoThirds, topOneThirds},
		{bottomHalf, bottomTwoThirds, bottomOneThirds}}
	edgeFuncTurn := make([]int, len(edgeFuncs))
	cornerFuncs := [][]resizeFunc{
		{topLeftHalf, topLeftTwoThirds, topLeftOneThirds},
		{topRightHalf, topRightTwoThirds, topRightOneThirds},
		{bottomLeftHalf, bottomLeftTwoThirds, bottomLeftOneThirds},
		{bottomRightHalf, bottomRightTwoThirds, bottomRightOneThirds}}
	cornerFuncTurn := make([]int, len(cornerFuncs))

	cycleFuncs := func(funcs [][]resizeFunc, turns *[]int, i int) {
		hwnd := getTargetWindow()
		if hwnd == 0 {
			fmt.Println("foreground window is NULL")
			return
		}
		if lastResized != hwnd {
			*turns = make([]int, len(edgeFuncs)) // reset
		}
		if _, err := resize(hwnd, funcs[i][(*turns)[i]%len(funcs[i])]); err != nil {
			fmt.Printf("warn: resize: %v\n", err)
			return
		}
		(*turns)[i]++
		for j := 0; j < len(*turns); j++ {
			if j != i {
				(*turns)[j] = 0
			}
		}
	}

	cycleEdgeFuncs := func(i int) { cycleFuncs(edgeFuncs, &edgeFuncTurn, i) }
	cycleCornerFuncs := func(i int) { cycleFuncs(cornerFuncs, &cornerFuncTurn, i) }

	// Define all available features
	featureMap := map[string]struct {
		DisplayName string
		Callback    func()
	}{
		"moveToTop":         {"Top half", func() { cycleEdgeFuncs(2) }},
		"moveToBottom":      {"Bottom half", func() { cycleEdgeFuncs(3) }},
		"moveToLeft":        {"Left half", func() { cycleEdgeFuncs(0) }},
		"moveToRight":       {"Right half", func() { cycleEdgeFuncs(1) }},
		"moveToTopLeft":     {"Top-Left corner", func() { cycleCornerFuncs(0) }},
		"moveToTopRight":    {"Top-Right corner", func() { cycleCornerFuncs(1) }},
		"moveToBottomLeft":  {"Bottom-Left corner", func() { cycleCornerFuncs(2) }},
		"moveToBottomRight": {"Bottom-Right corner", func() { cycleCornerFuncs(3) }},

		"maximize": {"Maximize", func() {
			lastResized = 0
			if err := maximize(); err != nil {
				fmt.Printf("warn: maximize: %v\n", err)
			}
		}},
		"almostMaximize": {"Almost Maximize", func() {
			lastResized = 0
			if _, err := resize(getTargetWindow(), func(disp, cur w32.RECT) w32.RECT {
				return makeSmaller(disp, disp)
			}); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"makeFullHeight": {"Maximize Height", func() {
			if _, err := resize(getTargetWindow(), maxHeight); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"makeLarger": {"Larger", func() {
			if _, err := resize(getTargetWindow(), makeLarger); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"makeSmaller": {"Smaller", func() {
			if _, err := resize(getTargetWindow(), makeSmaller); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"moveToCenter": {"Center", func() {
			lastResized = 0
			if _, err := resize(getTargetWindow(), center); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"nextDisplay": {"Next Display", func() {
			lastResized = 0
			if _, err := resizeAcrossMonitor(getTargetWindow(), center, 1); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"prevDisplay": {"Previous Display", func() {
			lastResized = 0
			if _, err := resizeAcrossMonitor(getTargetWindow(), center, -1); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"toggleAlwaysOnTop": {"Toggle Always On Top", func() {
			hwnd := getTargetWindow()
			if err := toggleAlwaysOnTop(hwnd); err != nil {
				fmt.Printf("warn: toggleAlwaysOnTop: %v\n", err)
				return
			}
			fmt.Printf("> toggled always on top: %v\n", hwnd)
		}},
	}
	if *action != "" {
		if feature, ok := featureMap[*action]; ok {
			feature.Callback()
			fmt.Printf("%s Action completed successfully\n", *action)
			os.Exit(0)
		}
		fmt.Printf("warn: unknown action: %s\n", *action)
		os.Exit(1)
	}

	hks = []HotKey{}

	myConfig := fetchConfiguration()
	fmt.Println(myConfig)
	// start from id 200
	id := 200
	for _, keyBinding := range myConfig.Keybindings {
		if feature, ok := featureMap[keyBinding.BindFeature]; ok {
			if keyBinding.BindFeature == "previousDisplay" {
				keyBinding.BindFeature = "prevDisplay"
			}
			id += 1
			hk := HotKey{
				id:          id,
				mod:         int(keyBinding.CombinedMod) | MOD_NOREPEAT,
				vk:          int(keyBinding.KeyCode),
				callback:    feature.Callback,
				bindFeature: keyBinding.BindFeature,
			}
			hks = append(hks, hk)
		}
	}

	// Populate global features list with hotkey info
	// Order matters for the menu
	orderedKeys := []string{
		"leftHalf", "rightHalf", "topHalf", "bottomHalf", // These are not directly in map, they are part of cycle
		"moveToLeft", "moveToRight", "moveToTop", "moveToBottom",
		"moveToTopLeft", "moveToTopRight", "moveToBottomLeft", "moveToBottomRight",
		"moveToCenter", "maximize", "almostMaximize", "makeLarger", "makeSmaller", "makeFullHeight",
		"nextDisplay", "prevDisplay", "toggleAlwaysOnTop",
	}

	for _, key := range orderedKeys {
		if val, ok := featureMap[key]; ok {
			desc := ""
			// Find if there is a hotkey for this feature
			for _, hk := range hks {
				if hk.bindFeature == key {
					desc = hk.Describe()
					break
				}
			}
			features = append(features, Feature{
				Name:        key,
				DisplayName: val.DisplayName,
				Callback:    val.Callback,
				HotkeyDesc:  desc,
			})
		}
	}

	var failedHotKeys []HotKey
	for _, hk := range hks {
		if !RegisterHotKey(hk) {
			failedHotKeys = append(failedHotKeys, hk)
		}
	}
	if len(failedHotKeys) > 0 {
		msg := "The following hotkey(s) are in use by another process:\n\n"
		for _, hk := range failedHotKeys {
			msg += "  - " + hk.Describe() + "\n"
		}
		msg += "\nTo use these hotkeys in RectangleWin Plus, close the other process using the key combination(s)."
		showMessageBox(msg)
	}

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt)
	go func() {
		<-exitCh
		fmt.Println("exit signal received")
		systray.Quit() // causes WM_CLOSE, WM_QUIT, not sure if a side-effect
	}()

	// TODO systray/systray.go already locks the OS thread in init()
	// however it's not clear if GetMessage(0,0) will continue to work
	// as we run "go initTray()" and not pin the thread that initializes the
	// tray.
	if *loadTray {
		initTray()
	}
	if err := msgLoop(); err != nil {
		panic(err)
	}

	if shouldRestart {
		fmt.Println("restarting...")
		unregisterAllHotKeys()

		exe, err := os.Executable()
		if err != nil {
			fmt.Printf("failed to get executable path: %v\n", err)
			return
		}

		_, err = os.StartProcess(exe, os.Args, &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		})
		if err != nil {
			fmt.Printf("failed to start new process: %v\n", err)
			return
		}
	}
}

func showMessageBox(text string) {
	w32.MessageBox(w32.GetActiveWindow(), text, "RectangleWin Plus", w32.MB_ICONWARNING|w32.MB_OK)
}

type resizeFunc func(disp, cur w32.RECT) w32.RECT

func center(disp, cur w32.RECT) w32.RECT {
	// TODO find a way to round up divisions consistently as it causes multiple runs to shift by 1px
	w := (disp.Width() - cur.Width()) / 2
	h := (disp.Height() - cur.Height()) / 2
	return w32.RECT{
		Left:   disp.Left + w,
		Right:  disp.Left + w + cur.Width(),
		Top:    disp.Top + h,
		Bottom: disp.Top + h + cur.Height()}
}

func resize(hwnd w32.HWND, f resizeFunc) (bool, error) {
	return resizeAcrossMonitor(hwnd, f, 0)
}
func resizeAcrossMonitor(hwnd w32.HWND, f resizeFunc, monitorIndexDiff int) (bool, error) {
	if !isZonableWindow(hwnd) {
		fmt.Printf("warn: non-zonable window: %s\n", w32.GetWindowText(hwnd))
		return false, nil
	}
	rect := w32.GetWindowRect(hwnd)
	mon := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	if monitorIndexDiff != 0 {
		monitorCount := w32.GetSystemMetrics(80 /*SM_CMONITORS*/)
		fmt.Printf("original monitorIndexDiff: %d monitorCount: %d\n", monitorIndexDiff, monitorCount)
		if monitorIndexDiff < 0 {
			monitorIndexDiff = monitorIndexDiff % monitorCount
			// we need to blindly add monitorCount, because even if
			// this is zero, the loop below doesn't return the original
			// in the foundOriginal setting true path.
			monitorIndexDiff += monitorCount
		}
		fmt.Printf("canonicalized monitorIndexDiff: %d\n", monitorIndexDiff)
		foundOriginal := false
		foundTarget := false
		cnt := 0
		originalWindowMonitor := mon
		// iterate for max of 20 times
		for i := 0; i < 20 && !foundTarget; i++ {
			EnumMonitors(func(d w32.HMONITOR) bool {
				if d == originalWindowMonitor {
					foundOriginal = true
				} else if foundOriginal {
					cnt += 1
					if cnt == monitorIndexDiff {
						mon = d
						foundTarget = true
						return false // stop iterating
					}
				}
				// continue to iterate
				return true
			})
		}
		// after this iteration, we either have found a new target based on
		// monitorIndex, or we have given up.
		if !foundTarget {
			// if we gave up, fall back to the original window.
			mon = originalWindowMonitor
		}
	}
	hdc := w32.GetDC(hwnd)
	displayDPI := w32.GetDeviceCaps(hdc, w32.LOGPIXELSY)
	if !w32.ReleaseDC(hwnd, hdc) {
		return false, fmt.Errorf("failed to ReleaseDC:%d", w32.GetLastError())
	}
	var monInfo w32.MONITORINFO
	if !w32.GetMonitorInfo(mon, &monInfo) {
		return false, fmt.Errorf("failed to GetMonitorInfo:%d", w32.GetLastError())
	}

	ok, frame := w32.DwmGetWindowAttributeEXTENDED_FRAME_BOUNDS(hwnd)
	if !ok {
		return false, fmt.Errorf("failed to DwmGetWindowAttributeEXTENDED_FRAME_BOUNDS:%d", w32.GetLastError())
	}
	windowDPI := w32ex.GetDpiForWindow(hwnd)
	resizedFrame := resizeForDpi(frame, int32(windowDPI), int32(displayDPI))

	fmt.Printf("> window: 0x%x %#v (w:%d,h:%d) mon=0x%X(@ display DPI:%d)\n", hwnd, rect, rect.Width(), rect.Height(), mon, displayDPI)
	fmt.Printf("> DWM frame:        %#v (W:%d,H:%d) @ window DPI=%v\n", frame, frame.Width(), frame.Height(), windowDPI)
	fmt.Printf("> DPI-less frame:   %#v (W:%d,H:%d)\n", resizedFrame, resizedFrame.Width(), resizedFrame.Height())

	// calculate how many extra pixels go to win10 invisible borders
	lExtra := resizedFrame.Left - rect.Left
	rExtra := -resizedFrame.Right + rect.Right
	tExtra := resizedFrame.Top - rect.Top
	bExtra := -resizedFrame.Bottom + rect.Bottom

	newPos := f(monInfo.RcWork, resizedFrame)

	// adjust offsets based on invisible borders
	newPos.Left -= lExtra
	newPos.Top -= tExtra
	newPos.Right += rExtra
	newPos.Bottom += bExtra

	lastResized = hwnd
	if sameRect(rect, &newPos) {
		fmt.Println("no resize")
		return false, nil
	}

	fmt.Printf("> resizing to: %#v (W:%d,H:%d)\n", newPos, newPos.Width(), newPos.Height())
	if !w32.ShowWindow(hwnd, w32.SW_SHOWNORMAL) { // normalize window first if it's set to SW_SHOWMAXIMIZE (and therefore stays maximized)
		return false, fmt.Errorf("failed to normalize window ShowWindow:%d", w32.GetLastError())
	}
	if !w32.SetWindowPos(hwnd, 0, int(newPos.Left), int(newPos.Top), int(newPos.Width()), int(newPos.Height()), w32.SWP_NOZORDER|w32.SWP_NOACTIVATE) {
		return false, fmt.Errorf("failed to SetWindowPos:%d", w32.GetLastError())
	}
	rect = w32.GetWindowRect(hwnd)
	fmt.Printf("> post-resize: %#v(W:%d,H:%d)\n", rect, rect.Width(), rect.Height())
	return true, nil
}

func maximize() error {
	hwnd := getTargetWindow()
	if !isZonableWindow(hwnd) {
		return errors.New("foreground window is not zonable")
	}
	if !w32.ShowWindow(hwnd, w32.SW_MAXIMIZE) {
		return fmt.Errorf("failed to ShowWindow:%d", w32.GetLastError())
	}
	return nil
}

func getTargetWindow() w32.HWND {
	hwnd := w32.GetForegroundWindow()
	if isZonableWindow(hwnd) {
		return hwnd
	}
	if isZonableWindow(lastActiveWindow) {
		return lastActiveWindow
	}
	return 0
}

func toggleAlwaysOnTop(hwnd w32.HWND) error {
	if !isZonableWindow(hwnd) {
		return errors.New("foreground window is not zonable")
	}

	if w32.GetWindowLong(hwnd, w32.GWL_EXSTYLE)&w32.WS_EX_TOPMOST != 0 {
		if !w32.SetWindowPos(hwnd, w32.HWND_NOTOPMOST, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE) {
			return fmt.Errorf("failed to SetWindowPos(HWND_NOTOPMOST): %v", w32.GetLastError())
		}
	} else {
		if !w32.SetWindowPos(hwnd, w32.HWND_TOPMOST, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE) {
			return fmt.Errorf("failed to SetWindowPos(HWND_TOPMOST) :%v", w32.GetLastError())
		}
	}
	return nil
}

func resizeForDpi(src w32.RECT, from, to int32) w32.RECT {
	return w32.RECT{
		Left:   src.Left * to / from,
		Right:  src.Right * to / from,
		Top:    src.Top * to / from,
		Bottom: src.Bottom * to / from,
	}
}

func sameRect(a, b *w32.RECT) bool {
	return a != nil && b != nil && reflect.DeepEqual(*a, *b)
}
