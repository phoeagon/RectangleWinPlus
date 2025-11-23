// Copyright 2025 Phoeagon
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/getlantern/systray"
	"github.com/gonutz/w32/v2"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadUrl string `json:"browser_download_url"`
	} `json:"assets"`
}

func checkForUpdates() {
	resp, err := http.Get("https://api.github.com/repos/phoeagon/RectangleWinPlus/releases/latest")
	if err != nil {
		showMessageBox(fmt.Sprintf("Failed to check for updates: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		showMessageBox(fmt.Sprintf("Failed to check for updates: HTTP %d", resp.StatusCode))
		return
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		showMessageBox(fmt.Sprintf("Failed to parse release info: %v", err))
		return
	}

	if release.TagName == currentVersion {
		showMessageBox("You are up to date.")
		return
	}

	// Simple version check (assuming vX.Y.Z format)
	// If release.TagName != currentVersion, we assume it's newer for now
	// or we can just ask the user.
	msg := fmt.Sprintf("New version %s is available (current: %s).\nDo you want to update?", release.TagName, currentVersion)
	if w32.MessageBox(w32.GetActiveWindow(), msg, "Update Available", w32.MB_ICONQUESTION|w32.MB_YESNO) == w32.IDYES {
		// Find asset
		var downloadUrl string
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.Name, ".exe") {
				downloadUrl = asset.BrowserDownloadUrl
				break
			}
		}

		if downloadUrl == "" {
			showMessageBox("No executable asset found in the release.")
			return
		}

		if err := downloadAndUpdate(downloadUrl); err != nil {
			showMessageBox(fmt.Sprintf("Update failed: %v", err))
		}
	}
}

func downloadAndUpdate(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	oldPath := exePath + ".old"
	if err := os.Rename(exePath, oldPath); err != nil {
		return fmt.Errorf("failed to rename current executable: %v", err)
	}

	out, err := os.Create(exePath)
	if err != nil {
		// Try to restore
		os.Rename(oldPath, exePath)
		return fmt.Errorf("failed to create new executable: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		// Try to restore
		out.Close()
		os.Remove(exePath)
		os.Rename(oldPath, exePath)
		return fmt.Errorf("failed to download update: %v", err)
	}
	out.Close()

	showMessageBox("Update downloaded successfully. Restarting...")
	shouldRestart = true
	systray.Quit()
	return nil
}
