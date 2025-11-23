// Copyright 2022 Ahmet Alp Balkan
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

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"

	"github.com/getlantern/systray"
	"github.com/gonutz/w32/v2"
)

//go:embed assets/tray_icon.ico
var icon []byte

const repo = "https://github.com/phoeagon/RectangleWinPlus"
const releases = "https://github.com/phoeagon/RectangleWinPlus/releases"

func initTray() {
	systray.Register(onReady, onExit)
}

func onReady() {
	systray.SetIcon(icon)
	systray.SetTitle("RectangleWin Plus")
	systray.SetTooltip("RectangleWin Plus")

	autorun, err := AutoRunEnabled()
	if err != nil {
		panic(err)
	}

	mRepo := systray.AddMenuItem("Documentation", "")
	go func() {
		for range mRepo.ClickedCh {
			if err := w32.ShellExecute(0, "open", repo, "", "", w32.SW_SHOWNORMAL); err != nil {
				fmt.Printf("failed to launch browser: (%d), %v\n", w32.GetLastError(), err)
			}
		}
	}()

	updates := systray.AddMenuItem("Check updates", "")
	go func() {
		for range updates.ClickedCh {
			checkForUpdates()
		}
	}()

	systray.AddSeparator()

	mAutoRun := systray.AddMenuItemCheckbox("Run on startup", "", autorun)
	go func() {
		for range mAutoRun.ClickedCh {
			if mAutoRun.Checked() {
				if err := AutoRunDisable(); err != nil {
					mAutoRun.SetTitle(err.Error())
					fmt.Printf("warn: autorun disable: %v\n", err)
					continue
				}
				fmt.Println("disabled autorun")
				mAutoRun.Uncheck()
			} else {
				if err := AutoRunEnable(); err != nil {
					mAutoRun.SetTitle(err.Error())
					fmt.Printf("warn: autorun enable: %v\n", err)
					continue
				}
				fmt.Println("enabled autorun")
				mAutoRun.Check()
			}

		}
	}()

	systray.AddSeparator()

	mConfig := systray.AddMenuItem("Configuration", "")
	go func() {
		<-mConfig.ClickedCh
		fmt.Println("opening editor for default config")
		configFilePath, err := getValidConfigPathOrCreate()
		if err != nil {
			showMessageBox(fmt.Sprintf(
				"Can't locate config path under user home directory %s\n%v", configFilePath, err))
			return
		}
		cmd := exec.Command("notepad.exe", configFilePath)
		err = cmd.Start()
		if err != nil {
			showMessageBox(fmt.Sprintf("Failed to open config file %s\n%v", configFilePath, err))
		}
		// TODO add a better way to reload current program.
		// Reloading programmatically is non-trivial because this program registers
		// hotkeys, so it much synchronize to start the child process, but quit
		// parent before the child starts to register hotkeys
	}()
	showLoadedConfig := systray.AddMenuItem("Show loaded config", "")
	go func() {
		<-showLoadedConfig.ClickedCh
		msg := "Here are the availble hotkeys:\n\n"
		for _, hotkey := range hks {
			msg += fmt.Sprintf("%s: %v\n", hotkey.bindFeature, hotkey.Describe())
		}
		msg += "\nTo change this config, update the config file at %HOME/.config/RectangleWinPlus/config.json"
		showMessageBox(msg)
	}()
	resetToDefault := systray.AddMenuItem("Reset to default", "")
	go func() {
		<-resetToDefault.ClickedCh
		configFilePath, err := getValidConfigPathOrCreate()
		if err != nil {
			showMessageBox(fmt.Sprintf("Failed to locate config file: %v", err))
			return
		}
		if err := os.Remove(configFilePath); err != nil && !os.IsNotExist(err) {
			showMessageBox(fmt.Sprintf("Failed to remove config file: %v", err))
			return
		}
		maybeDropExampleConfigFile(configFilePath)
		showMessageBox("Configuration reset to default. Restarting RectangleWinPlus...")
		shouldRestart = true
		systray.Quit()
	}()

	systray.AddSeparator()
	menuHeader := systray.AddMenuItem("Features", "")
	menuHeader.Disable()

	for _, f := range features {
		title := f.DisplayName
		if f.HotkeyDesc != "" {
			title += fmt.Sprintf(" (%s)", f.HotkeyDesc)
		}
		mItem := systray.AddMenuItem(title, "")
		// Capture variable for closure
		callback := f.Callback
		go func() {
			for range mItem.ClickedCh {
				callback()
			}
		}()
	}
	systray.AddSeparator()
	mRestart := systray.AddMenuItem("Restart to apply config", "")
	go func() {
		<-mRestart.ClickedCh
		fmt.Println("clicked Restart")
		shouldRestart = true
		systray.Quit()
	}()
	mQuit := systray.AddMenuItem("Quit", "")
	go func() {
		<-mQuit.ClickedCh
		fmt.Println("clicked Quit")
		systray.Quit()
	}()

	fmt.Println("tray ready")
}

func onExit() {
	fmt.Println("onExit invoked")
}
