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
//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest -icon=assets/icon.ico -manifest=app.manifest

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
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

// Command-line flags
var debug *bool
var killAll *bool
var help *bool
var action *string
var loadTray *bool
var settingsWindow *bool

const currentVersion = "v1.0.2"

type Feature struct {
	Name        string
	DisplayName string
	Callback    func()
	HotkeyDesc  string
}

var features []Feature

type FeatureDefinition struct {
	DisplayName string
	Callback    func()
}

var featureDefinitions map[string]FeatureDefinition

// Static map of feature display names for settings UI
var featureDisplayNames = map[string]string{
	"moveToTop":         "Top half",
	"pushToTop":         "Push to Top",
	"moveToBottom":      "Bottom half",
	"pushToBottom":      "Push to Bottom",
	"moveToLeft":        "Left half",
	"pushToLeft":        "Push to Left",
	"moveToRight":       "Right half",
	"pushToRight":       "Push to Right",
	"moveToTopLeft":     "Top-Left corner",
	"moveToTopRight":    "Top-Right corner",
	"moveToBottomLeft":  "Bottom-Left corner",
	"moveToBottomRight": "Bottom-Right corner",
	"maximize":          "Maximize",
	"almostMaximize":    "Almost Maximize",
	"makeFullHeight":    "Maximize Height",
	"makeLarger":        "Larger",
	"makeSmaller":       "Smaller",
	"moveToCenter":      "Center",
	"nextDisplay":       "Next Display",
	"prevDisplay":       "Previous Display",
	"toggleAlwaysOnTop": "Toggle Always On Top",
}

func main() {
	// Initialize flags with ContinueOnError to handle parsing errors
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)

	// Set custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "RectangleWin Plus - Window management utility for Windows\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFor more information, visit: https://github.com/phoeagon/RectangleWinPlus\n")
	}

	debug = flag.Bool("debug", false, "enable debug mode (show console output)")
	killAll = flag.Bool("killall", false, "kill all RectangleWinPlus instances and quit")
	help = flag.Bool("help", false, "show this help message")
	action = flag.String("action", "", "action to perform (moveToTop, moveToBottom, moveToLeft, moveToRight, moveToTopLeft, moveToTopRight, moveToBottomLeft, moveToBottomRight, maximize, almostMaximize, makeFullHeight, makeLarger, makeSmaller)")
	loadTray = flag.Bool("load_tray", true, "load tray icon")
	version := flag.Bool("version", false, "show version information")
	helpfull := flag.Bool("helpfull", false, "show detailed help message")
	settingsWindow = flag.Bool("settings-window", false, "open settings window (internal use)")

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		if err != flag.ErrHelp {
			fmt.Printf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *debug {
		// FixConsole ensures that we can see stdout/stderr in the console
		// even if the app is built as a GUI app (windowsgui).
		if err := fixconsole.FixConsoleIfNeeded(); err != nil {
			fmt.Printf("warn: fixconsole: %v\n", err)
		}
	}

	// Handle help flag
	if *help {
		fixconsole.FixConsoleIfNeeded()
		flag.Usage()
		return
	}

	if *version {
		fixconsole.FixConsoleIfNeeded()
		fmt.Println("RectangleWin Plus - Window management utility for Windows")
		fmt.Println("Version: " + currentVersion)
		if !*debug {
			showMessageBox(fmt.Sprintf("RectangleWin Plus \n - Version: %s", currentVersion))
		}
		return
	}

	// Handle settings window flag
	if *settingsWindow {
		runtime.LockOSThread() // since we bind hotkeys etc that need to dispatch their message here

		runSettingsWindow()
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
		"moveToTop": {"Top half", func() { cycleEdgeFuncs(2) }},
		"pushToTop": {"Push to Top", func() {
			if _, err := resize(getTargetWindow(), pushTop); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"moveToBottom": {"Bottom half", func() { cycleEdgeFuncs(3) }},
		"pushToBottom": {"Push to Bottom", func() {
			if _, err := resize(getTargetWindow(), pushBottom); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"moveToLeft": {"Left half", func() { cycleEdgeFuncs(0) }},
		"pushToLeft": {"Push to Left", func() {
			if _, err := resize(getTargetWindow(), pushLeft); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
		"moveToRight": {"Right half", func() { cycleEdgeFuncs(1) }},
		"pushToRight": {"Push to Right", func() {
			if _, err := resize(getTargetWindow(), pushRight); err != nil {
				fmt.Printf("warn: resize: %v\n", err)
			}
		}},
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

	if *helpfull {
		fixconsole.FixConsoleIfNeeded()
		flag.Usage()
		fmt.Println("\nFeatures:")
		for feature, value := range featureMap {
			fmt.Printf("%s: %s\n", feature, value.DisplayName)
		}
		return
	}

	hks = []HotKey{}

	myConfig := fetchConfiguration()
	fmt.Println(myConfig)
	// start from id 200
	id := 200
	for _, keyBinding := range myConfig.Keybindings {
		if feature, ok := featureMap[keyBinding.BindFeature]; ok {
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
		// pushTo series happen last, because they are less used, as aligned in Rectangle.
		"pushToLeft", "pushToRight", "pushToTop", "pushToBottom",
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
