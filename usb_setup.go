package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

type PriorityConfig struct {
	Directories []string `json:"directories"`
	Files       []string `json:"files"`
}

type BackupConfig struct {
	USBSize        int64          `json:"usb_size"`
	Mode           string         `json:"mode"`
	FileExtensions []string       `json:"file_extensions"`
	MinSize        int64          `json:"min_size"`
	MaxSize        int64          `json:"max_size"`
	Priority       PriorityConfig `json:"priority"`
}

func getRemovableDrives() ([]string, error) {
	var drives []string
	driveBits, err := windows.GetLogicalDrives()
	if err != nil {
		return drives, err
	}
	for i := 0; i < 26; i++ {
		if driveBits&(1<<uint(i)) != 0 {
			driveLetter := string('A'+i) + ":\\" // Drive letter
			driveType := windows.GetDriveType(windows.StringToUTF16Ptr(driveLetter))
			if driveType == windows.DRIVE_REMOVABLE || driveType == windows.DRIVE_FIXED {
				// For fixed drives, check if it's external
				if driveType == windows.DRIVE_FIXED {
					isExternal, err := isExternalDrive(driveLetter)
					if err != nil || !isExternal {
						continue
					}
				}
				drives = append(drives, driveLetter)
			}
		}
	}
	return drives, nil
}

func isExternalDrive(driveLetter string) (bool, error) {
	// Enhanced PowerShell command that specifically checks for external drive characteristics
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
        $isExternal = $false
        $drive = Get-WmiObject Win32_DiskDrive | 
            Where-Object { $_.MediaType -like "*External*" -or 
                         $_.PNPDeviceID -like "*USB*" -or 
                         $_.Caption -like "*USB*" -or
                         $_.Caption -like "*NVMe*" -or
                         $_.Model -like "*UGreen*" } |
            ForEach-Object {
                $partition = Get-WmiObject -Query "ASSOCIATORS OF {Win32_DiskDrive.DeviceID='$($_.DeviceID)'} WHERE AssocClass=Win32_DiskDriveToDiskPartition"
                $partition | ForEach-Object {
                    $logical = Get-WmiObject -Query "ASSOCIATORS OF {Win32_DiskPartition.DeviceID='$($_.DeviceID)'} WHERE AssocClass=Win32_LogicalDiskToPartition"
                    if ($logical.DeviceID -eq '%s:') { $isExternal = $true }
                }
            }
        if ($isExternal) { Write-Output "1" } else {
            # Secondary check using physical disk info
            $disk = Get-PhysicalDisk | 
                Where-Object { Get-Partition -DiskNumber $_.DeviceID | 
                    Where-Object { $_.DriveLetter -eq '%s' } } |
                Select-Object BusType, MediaType, FriendlyName
            if ($disk.BusType -in @("USB", "SATA", "NVMe") -and 
                ($disk.FriendlyName -like "*USB*" -or 
                 $disk.FriendlyName -like "*External*" -or
                 $disk.FriendlyName -like "*UGreen*" -or
                 $disk.MediaType -eq "External hard disk media")) {
                Write-Output "1"
            } else {
                Write-Output "0"
            }
        }
    `, driveLetter, driveLetter))

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == "1", nil
}

func getDriveSize(drive string) int64 {
	var free, total, avail uint64
	path := windows.StringToUTF16Ptr(drive)
	windows.GetDiskFreeSpaceEx(path, &free, &total, &avail)
	return int64(total)
}

var commonExtensions = []string{
	".doc", ".docx", ".xls", ".xlsx", ".pdf", ".txt",
	".jpg", ".jpeg", ".png", ".gif", ".mp4", ".mp3",
	".zip", ".rar", ".7z", ".ppt", ".pptx", ".csv",
	".odt", ".ods", ".odp", ".rtf", ".psd", ".xml",
}

func getUserChoice(prompt string, options []string) []string {
	var selected []string
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("%d: %s\n", i+1, opt)
	}
	fmt.Println("Enter numbers (separated by spaces) or 0 for custom input: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()

	if input == "0" {
		fmt.Println("Enter custom extensions (including dot, separated by spaces, e.g., .pdf .txt): ")
		scanner.Scan()
		input = scanner.Text()
		selected = strings.Split(input, " ")
	} else {
		nums := strings.Fields(input)
		for _, numStr := range nums {
			if num, err := strconv.Atoi(numStr); err == nil && num > 0 && num <= len(options) {
				selected = append(selected, options[num-1])
			}
		}
	}

	// Remove empty strings and validate extensions
	var validExtensions []string
	for _, ext := range selected {
		ext = strings.TrimSpace(ext)
		if ext != "" && strings.HasPrefix(ext, ".") {
			validExtensions = append(validExtensions, ext)
		}
	}
	return validExtensions
}

func getPriorityPaths() []string {
	var paths []string
	fmt.Println("Enter priority paths (one per line, empty line to finish):")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		scanner.Scan()
		path := scanner.Text()
		path = strings.TrimSpace(path)
		if path == "" {
			break
		}
		// Convert to absolute path and clean the format
		if absPath, err := filepath.Abs(path); err == nil {
			paths = append(paths, filepath.Clean(absPath))
		}
	}
	return paths
}

func main() {
	// Get available USB drives
	drives, err := getRemovableDrives()
	if err != nil {
		fmt.Printf("Error detecting USB drives: %v\n", err)
		return
	}

	if len(drives) == 0 {
		fmt.Println("No USB drives detected. Please insert a USB drive and try again.")
		return
	}

	// List available drives
	fmt.Println("\nAvailable USB drives:")
	for i, drive := range drives {
		fmt.Printf("%d: %s\n", i+1, drive)
	}

	// Get user selection
	var selection int
	fmt.Print("\nSelect USB drive (enter number): ")
	fmt.Scan(&selection)
	if selection < 1 || selection > len(drives) {
		fmt.Println("Invalid selection")
		return
	}

	selectedDrive := drives[selection-1]
	driveSize := getDriveSize(selectedDrive)

	// Clear the input buffer before getting extensions
	var tmp string
	fmt.Scanln(&tmp)

	// Get file extensions
	fmt.Println("\n=== File Extension Configuration ===")
	extensions := getUserChoice("Select file extensions to backup:", commonExtensions)
	if len(extensions) == 0 {
		fmt.Println("Warning: No file extensions selected. Adding default (.txt, .doc, .pdf)")
		extensions = []string{".txt", ".doc", ".pdf"}
	}

	// Get priority paths
	fmt.Println("\n=== Priority Paths Configuration ===")
	priorityPaths := getPriorityPaths()
	if len(priorityPaths) == 0 {
		fmt.Println("Warning: No priority paths added. Adding default user directory")
		userHome, _ := os.UserHomeDir()
		priorityPaths = []string{userHome}
	}

	// Create configuration
	config := BackupConfig{
		USBSize:        driveSize,
		Mode:           "both",
		FileExtensions: extensions,
		MinSize:        1024,      // 1KB minimum
		MaxSize:        104857600, // 100MB maximum
		Priority: PriorityConfig{
			Directories: priorityPaths,
			Files:       []string{},
		},
	}

	// Create JSON file
	configPath := filepath.Join(selectedDrive, "backup_config.json")
	jsonData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Printf("Error creating JSON: %v\n", err)
		return
	}

	err = os.WriteFile(configPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		return
	}

	fmt.Printf("Successfully created backup_config.json on %s\n", selectedDrive)
	fmt.Println("USB drive is now ready to use with the backup program.")
}
