package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	_ "github.com/mattn/go-sqlite3"
)

// Set silentMode to true for a completely silent (no output) version,
// except for the indexing log which will be output in a separate terminal.
var silentMode bool = false

// List of directories to exclude from indexing
var excludedDirs = []string{
	`C:\Windows\System32`,
	`C:\Windows\SysWOW64`,
	`C:\Windows\WinSxS`,
	`C:\Windows\servicing`,
	`C:\Program Files\Common Files`,
	`C:\Program Files (x86)\Common Files`,
	`C:\ProgramData\Microsoft`,
	`C:\Windows\Microsoft.NET`,
	`C:\$Recycle.Bin`,
	`C:\System Volume Information`,
}

// List of directories to scan first (in order of priority)
var priorityDirs = []string{
	`C:\Users`,
	`C:\Documents and Settings`,
	`D:\Users`,
	`E:\Users`,
	`C:\Projects`,
	`C:\Work`,
}

// FileMeta holds basic metadata for a file.
type FileMeta struct {
	Path           string
	Size           int64
	ModTime        time.Time
	CreationTime   time.Time
	AccessTime     time.Time
	FileAttributes int64
}

// PriorityConfig holds directories and files to be prioritized.
type PriorityConfig struct {
	Directories []string `json:"directories"`
	Files       []string `json:"files"`
}

// BackupConfig represents the expected JSON configuration on the USB device.
type BackupConfig struct {
	USBSize        int64          `json:"usb_size"`        // Informational: USB drive size in bytes
	Mode           string         `json:"mode"`            // "indexer", "file", or "both"
	FileExtensions []string       `json:"file_extensions"` // Allowed file extensions
	MinSize        int64          `json:"min_size"`        // Minimum file size (bytes) to backup
	MaxSize        int64          `json:"max_size"`        // Maximum file size (bytes) to backup
	Priority       PriorityConfig `json:"priority"`        // Priority settings for directories and files
}

var db *sql.DB

// fileTimeToTime converts a Windows FILETIME (stored in syscall.Filetime)
// to a Go time.Time value.
func fileTimeToTime(ft syscall.Filetime) time.Time {
	// FILETIME is in 100-nanosecond intervals since January 1, 1601 (UTC)
	const ticksPerSecond = 10000000
	// Number of 100-nanosecond intervals between 1601 and Unix epoch (1970)
	const epochDiff = 116444736000000000
	ticks := (int64(ft.HighDateTime) << 32) | int64(ft.LowDateTime)
	// Convert FILETIME to Unix time (in seconds and nanoseconds)
	unixTicks := ticks - epochDiff
	seconds := unixTicks / ticksPerSecond
	nanoseconds := (unixTicks % ticksPerSecond) * 100
	return time.Unix(seconds, nanoseconds)
}

// initDB opens (or creates) the SQLite database and sets up the table.
func initDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Updated table schema to store all metadata.
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS file_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL,
		size INTEGER,
		mod_time DATETIME,
		creation_time DATETIME,
		access_time DATETIME,
		file_attributes INTEGER
	);
	`
	_, err = db.Exec(createTableSQL)
	return err
}

// isExcludedPath checks if the given path should be excluded from indexing
func isExcludedPath(path string) bool {
	path = strings.ToLower(path)
	for _, excluded := range excludedDirs {
		excluded = strings.ToLower(excluded)
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// scanAndStoreFiles walks the given rootPath and stores file metadata in the database.
// It uses the provided logger to output the indexing process.
func scanAndStoreFiles(rootPath string, logger *log.Logger) error {
	logger.Printf("Scanning files under: %s", rootPath)
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/directories that can’t be accessed.
			return nil
		}

		// Skip excluded directories
		if info.IsDir() {
			if isExcludedPath(path) {
				logger.Printf("Skipping excluded directory: %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files in excluded directories
		if isExcludedPath(path) {
			return nil
		}

		// Log each file being processed
		logger.Printf("Indexing: %s", path)

		// Default values if extra metadata is unavailable.
		creationTime := info.ModTime()
		accessTime := info.ModTime()
		var fileAttributes int64 = 0

		// Attempt to extract Windows-specific metadata.
		if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
			creationTime = fileTimeToTime(stat.CreationTime)
			accessTime = fileTimeToTime(stat.LastAccessTime)
			fileAttributes = int64(stat.FileAttributes)
		}

		_, err = db.Exec(
			"INSERT INTO file_metadata (path, size, mod_time, creation_time, access_time, file_attributes) VALUES (?, ?, ?, ?, ?, ?)",
			path,
			info.Size(),
			info.ModTime().Format(time.RFC3339),
			creationTime.Format(time.RFC3339),
			accessTime.Format(time.RFC3339),
			fileAttributes,
		)
		if err != nil {
			logger.Printf("Failed to insert metadata for %s: %v", path, err)
		}
		return nil
	})
}

// getRemovableDrives returns a slice of drive letters for removable drives (USB).
func getRemovableDrives() ([]string, error) {
	var drives []string
	driveBits, err := windows.GetLogicalDrives()
	if err != nil {
		return drives, err
	}
	for i := 0; i < 26; i++ {
		if driveBits&(1<<uint(i)) != 0 {
			driveLetter := string('A'+i) + ":\\"
			driveType := windows.GetDriveType(windows.StringToUTF16Ptr(driveLetter))
			if driveType == windows.DRIVE_REMOVABLE {
				drives = append(drives, driveLetter)
			}
		}
	}
	return drives, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// monitorUSBDevices continuously checks for new removable drives.
func monitorUSBDevices() {
	log.Println("Starting USB device monitoring...")
	processedDrives := make(map[string]bool)
	for {
		drives, err := getRemovableDrives()
		if err != nil {
			log.Printf("Error detecting removable drives: %v", err)
		}
		// Process newly detected drives.
		for _, drive := range drives {
			if !processedDrives[drive] {
				log.Printf("Detected new USB drive: %s", drive)
				go handleUSBDevice(drive)
				processedDrives[drive] = true
			}
		}
		// Remove drives that are no longer present.
		for drive := range processedDrives {
			if !contains(drives, drive) {
				log.Printf("USB drive removed: %s", drive)
				delete(processedDrives, drive)
			}
		}
		time.Sleep(10 * time.Second)
	}
}

// handleUSBDevice reads the backup configuration from the USB and processes the backup.
func handleUSBDevice(drive string) {
	configPath := filepath.Join(drive, "backup_config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("No backup config found on USB drive at %s", drive)
		return
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("Failed to read backup config on %s: %v", drive, err)
		return
	}

	var config BackupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Failed to parse backup config on %s: %v", drive, err)
		return
	}
	log.Printf("Loaded backup config from %s: %+v", drive, config)

	// Process based on the mode.
	switch strings.ToLower(config.Mode) {
	case "indexer":
		backupDatabase(drive)
	case "file":
		backupFiles(drive, config)
	case "both":
		// First copy the database, then copy files.
		backupDatabase(drive)
		backupFiles(drive, config)
	default:
		log.Printf("Unknown mode '%s'. Defaulting to file backup.", config.Mode)
		backupFiles(drive, config)
	}
}

// backupDatabase copies the SQLite database file to the USB drive.
func backupDatabase(drive string) {
	sourceDB := "files.db" // assumes the DB is in the current working directory
	destDir := filepath.Join(drive, "backup", "indexer")
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		log.Printf("Failed to create directory %s: %v", destDir, err)
		return
	}
	destFile := filepath.Join(destDir, "files.db")
	if err := copyFile(sourceDB, destFile); err != nil {
		log.Printf("Failed to backup database to %s: %v", destFile, err)
	} else {
		log.Printf("Database backed up to %s", destFile)
	}
}

// backupFiles queries the database for matching files, sorts them by priority, and copies them.
func backupFiles(drive string, config BackupConfig) {
	// Updated query to include all metadata columns.
	query := `SELECT path, size, mod_time, creation_time, access_time, file_attributes
	          FROM file_metadata WHERE size >= ? AND size <= ?`
	rows, err := db.Query(query, config.MinSize, config.MaxSize)
	if err != nil {
		log.Printf("Database query failed: %v", err)
		return
	}
	defer rows.Close()

	var filesToBackup []FileMeta
	for rows.Next() {
		var f FileMeta
		var modTimeStr, creationTimeStr, accessTimeStr string
		if err := rows.Scan(&f.Path, &f.Size, &modTimeStr, &creationTimeStr, &accessTimeStr, &f.FileAttributes); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		f.ModTime, err = time.Parse(time.RFC3339, modTimeStr)
		if err != nil {
			f.ModTime = time.Now()
		}
		f.CreationTime, err = time.Parse(time.RFC3339, creationTimeStr)
		if err != nil {
			f.CreationTime = f.ModTime
		}
		f.AccessTime, err = time.Parse(time.RFC3339, accessTimeStr)
		if err != nil {
			f.AccessTime = f.ModTime
		}

		// If file extensions are specified, check that the file matches one.
		if len(config.FileExtensions) > 0 {
			ext := strings.ToLower(filepath.Ext(f.Path))
			match := false
			for _, allowedExt := range config.FileExtensions {
				if ext == strings.ToLower(allowedExt) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		filesToBackup = append(filesToBackup, f)
	}

	// Sort files based on priority: explicit file matches (score 1), in a priority directory (score 2), otherwise score 3.
	sort.Slice(filesToBackup, func(i, j int) bool {
		pi := priorityScore(filesToBackup[i], config)
		pj := priorityScore(filesToBackup[j], config)
		if pi == pj {
			return filesToBackup[i].Path < filesToBackup[j].Path
		}
		return pi < pj
	})

	// Copy each matching file.
	for _, file := range filesToBackup {
		relPath := strings.TrimPrefix(file.Path, filepath.VolumeName(file.Path))
		destDir := filepath.Join(drive, "backup", "files", filepath.Dir(relPath))
		if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
			log.Printf("Failed to create directory %s: %v", destDir, err)
			continue
		}
		destFile := filepath.Join(destDir, filepath.Base(file.Path))
		if err := copyFile(file.Path, destFile); err != nil {
			log.Printf("Failed to copy %s to %s: %v", file.Path, destFile, err)
		} else {
			log.Printf("Backed up %s to %s", file.Path, destFile)
		}
	}
}

// priorityScore returns a numeric score for a file (lower means higher priority).
func priorityScore(file FileMeta, config BackupConfig) int {
	// Check if the file is explicitly prioritized.
	for _, pfile := range config.Priority.Files {
		if strings.EqualFold(file.Path, pfile) {
			return 1
		}
	}
	// Check if the file is in a prioritized directory.
	for _, pdir := range config.Priority.Directories {
		if strings.HasPrefix(strings.ToLower(file.Path), strings.ToLower(pdir)) {
			return 2
		}
	}
	return 3
}

// copyFile performs a simple file copy from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

// scanPriorityDirectories scans high-priority directories first
func scanPriorityDirectories(logger *log.Logger) {
	logger.Printf("Starting priority directory scan...")
	for _, dir := range priorityDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logger.Printf("Priority directory does not exist, skipping: %s", dir)
			continue
		}
		logger.Printf("Scanning priority directory: %s", dir)
		if err := scanAndStoreFiles(dir, logger); err != nil {
			logger.Printf("Error scanning priority directory %s: %v", dir, err)
		}
	}
	logger.Printf("Priority directory scan completed")
}

func main() {
	// If silentMode is enabled, redirect the default log output.
	if silentMode {
		log.SetOutput(io.Discard)
	}

	// Initialize (or create) the database.
	if err := initDB("files.db"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Set up a separate log file for the indexing process.
	indexLogFile, err := os.OpenFile("indexing.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open indexing log file: %v", err)
	}
	defer indexLogFile.Close()
	indexLogger := log.New(indexLogFile, "INDEXER: ", log.LstdFlags)

	// Spawn a new terminal window that tails the indexing log.
	// This uses PowerShell's Get-Content -Wait.
	cmd := exec.Command("cmd.exe", "/C", "start", "cmd.exe", "/K", "powershell", "-Command", "Get-Content", "indexing.log", "-Wait")
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to launch indexing log terminal: %v", err)
	}

	// Start scanning the file system in a separate goroutine,
	// using the indexLogger for output.
	go func() {
		// First scan priority directories
		scanPriorityDirectories(indexLogger)

		// Then scan the rest of the system
		rootDir := "C:\\" // Adjust as needed
		if err := scanAndStoreFiles(rootDir, indexLogger); err != nil {
			indexLogger.Printf("Error scanning filesystem: %v", err)
		}
	}()

	// Begin monitoring for USB devices.
	monitorUSBDevices()
}
