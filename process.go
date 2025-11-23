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

package main

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	PROCESS_TERMINATE         = 0x0001
	PROCESS_QUERY_INFORMATION = 0x0400
	TH32CS_SNAPPROCESS        = 0x00000002
)

type PROCESSENTRY32 struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
)

// killAllRectangleWinPlusProcesses terminates all running instances of RectangleWinPlus.exe
func killAllRectangleWinPlusProcesses() error {
	currentPID := uint32(os.Getpid())

	// Create a snapshot of all processes
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(
		uintptr(TH32CS_SNAPPROCESS),
		0,
	)
	if snapshot == 0 {
		return fmt.Errorf("CreateToolhelp32Snapshot failed: %v", err)
	}
	defer windows.CloseHandle(windows.Handle(snapshot))

	var pe32 PROCESSENTRY32
	pe32.dwSize = uint32(unsafe.Sizeof(pe32))

	// Get the first process
	ret, _, err := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return fmt.Errorf("Process32First failed: %v", err)
	}

	killedCount := 0
	targetName := "RectangleWinPlus.exe"

	// Iterate through all processes
	for {
		// Convert the process name from UTF-16 to string
		exeName := windows.UTF16ToString(pe32.szExeFile[:])

		// Check if this is a RectangleWinPlus.exe process
		if strings.EqualFold(exeName, targetName) {
			// Don't kill the current process (the one running --killall)
			if pe32.th32ProcessID != currentPID {
				// Open the process with terminate rights
				hProcess, err := windows.OpenProcess(PROCESS_TERMINATE, false, pe32.th32ProcessID)
				if err != nil {
					fmt.Printf("Warning: Failed to open process %d (%s): %v\n", pe32.th32ProcessID, exeName, err)
				} else {
					// Terminate the process
					err = windows.TerminateProcess(hProcess, 0)
					windows.CloseHandle(hProcess)

					if err != nil {
						fmt.Printf("Warning: Failed to terminate process %d (%s): %v\n", pe32.th32ProcessID, exeName, err)
					} else {
						fmt.Printf("Terminated process %d (%s)\n", pe32.th32ProcessID, exeName)
						killedCount++
					}
				}
			}
		}

		// Get the next process
		ret, _, err = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	if killedCount == 0 {
		fmt.Println("No RectangleWinPlus.exe processes found to terminate")
	} else {
		fmt.Printf("Terminated %d process(es)\n", killedCount)
	}

	fmt.Println("All RectangleWinPlus.exe processes terminated")
	return nil
}
