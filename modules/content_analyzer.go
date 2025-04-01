package modules

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/go-nlp/tfidf"
)

// ContentAnalyzer provides functionality for analyzing file contents
// and determining importance based on text analysis.
type ContentAnalyzer struct {
	DB            *sql.DB
	Logger        *log.Logger
	MaxTotalSize  int64 // Maximum total size for priority files (e.g., 1GB)
	analyzer      *tfidf.TFIDF
	fileScores    map[string]float64
	priorityFiles []PriorityFile
	mutex         sync.RWMutex
}

// PriorityFile represents a file with its calculated importance score
type PriorityFile struct {
	Path  string
	Size  int64
	Score float64
}

// Keywords that indicate importance when found in documents
var importantKeywords = []string{
	"password", "pass", "creds", "redentials", "secret", "private", "important",
	"credit card", "bank account", "social security", "ssn", "address",
	"phone", "email", "tax", "financial", "statement",
	"contract", "agreement", "proposal", "thesis", "report",
	"project", "research", "personal", "identity",
}

// NewContentAnalyzer creates a new content analyzer with the specified DB connection
func NewContentAnalyzer(db *sql.DB, logger *log.Logger, maxSizeBytes int64) *ContentAnalyzer {
	return &ContentAnalyzer{
		DB:            db,
		Logger:        logger,
		MaxTotalSize:  maxSizeBytes,
		analyzer:      tfidf.New(),
		fileScores:    make(map[string]float64),
		priorityFiles: []PriorityFile{},
		mutex:         sync.RWMutex{},
	}
}

// preprocessText performs basic text preprocessing: lowercase, remove punctuation
func preprocessText(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	return words
}

// AnalyzeFile reads a file and calculates its importance score
func (ca *ContentAnalyzer) AnalyzeFile(path string, size int64) (float64, error) {
	// Skip very small or very large files
	if size < 100 || size > 50*1024*1024 {
		return 0.0, nil
	}

	// Skip files that are likely not text
	ext := strings.ToLower(filepath.Ext(path))
	if !isTextFile(ext) {
		return 0.0, nil
	}

	// Attempt to read the file
	file, err := os.Open(path)
	if err != nil {
		return 0.0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read up to 1MB for analysis
	buffer := make([]byte, min(size, 1024*1024))
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return 0.0, fmt.Errorf("failed to read file: %w", err)
	}
	buffer = buffer[:n]

	// Skip binary files
	if isBinary(buffer) {
		return 0.0, nil
	}

	// Process the text and add to analyzer
	text := string(buffer)
	words := preprocessText(text)
	docID := path

	// Add document to TF-IDF analyzer
	ca.analyzer.AddDocs(tfidf.NewDoc(docID, words, ""))

	// Calculate initial score based on keyword matching
	score := ca.calculateKeywordScore(text)

	// Store score
	ca.mutex.Lock()
	ca.fileScores[path] = score
	ca.mutex.Unlock()

	return score, nil
}

// calculateKeywordScore gives a score based on the presence of important keywords
func (ca *ContentAnalyzer) calculateKeywordScore(text string) float64 {
	text = strings.ToLower(text)
	var score float64 = 1.0

	for _, keyword := range importantKeywords {
		if strings.Contains(text, keyword) {
			score += 2.0
		}
	}

	// Add some weight for longer content that's not excessive
	textLen := len(text)
	if textLen > 500 && textLen < 50000 {
		score *= 1.2
	}

	return score
}

// isTextFile determines if a file extension is likely text-based
func isTextFile(ext string) bool {
	textExts := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".json": true,
		".xml": true, ".html": true, ".htm": true, ".log": true,
		".ini": true, ".cfg": true, ".config": true, ".yaml": true,
		".yml": true, ".toml": true, ".sql": true, ".c": true,
		".cpp": true, ".h": true, ".cs": true, ".java": true,
		".py": true, ".js": true, ".ts": true, ".php": true,
		".rb": true, ".go": true, ".sh": true, ".bat": true,
		".ps1": true, ".tex": true, ".rtf": true, ".doc": true,
		".docx": true, ".pdf": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true,
	}
	return textExts[ext]
}

// isBinary makes a simple check to determine if content appears to be binary
func isBinary(data []byte) bool {
	if len(data) > 0 {
		// Count null bytes and control characters
		nullCount := 0
		controlCount := 0

		// Only check a small sample
		sampleSize := min(int64(len(data)), 512)
		for i := 0; i < int(sampleSize); i++ {
			if data[i] == 0 {
				nullCount++
			} else if data[i] < 32 && data[i] != 9 && data[i] != 10 && data[i] != 13 {
				controlCount++
			}
		}

		// If more than 10% is null or control chars, likely binary
		threshold := sampleSize / 10
		return nullCount > int(threshold) || controlCount > int(threshold)
	}
	return false
}

// AnalyzeBatch analyzes a batch of files from the database
func (ca *ContentAnalyzer) AnalyzeBatch(batchSize int) error {
	ca.Logger.Printf("Starting content analysis with batch size: %d", batchSize)

	// Query files from the database
	rows, err := ca.DB.Query(`
		SELECT path, size FROM file_metadata 
		WHERE size > 100 AND size < 50000000 
		ORDER BY size ASC
	`)
	if err != nil {
		return fmt.Errorf("error querying files for analysis: %w", err)
	}
	defer rows.Close()

	// Process files in batch
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4) // Limit concurrent goroutines

	fileCount := 0
	for rows.Next() {
		var path string
		var size int64

		if err := rows.Scan(&path, &size); err != nil {
			ca.Logger.Printf("Error scanning row: %v", err)
			continue
		}

		// Skip already analyzed files
		ca.mutex.RLock()
		_, exists := ca.fileScores[path]
		ca.mutex.RUnlock()
		if exists {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(filePath string, fileSize int64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			score, err := ca.AnalyzeFile(filePath, fileSize)
			if err != nil {
				ca.Logger.Printf("Error analyzing %s: %v", filePath, err)
				return
			}

			// Only track non-zero scores
			if score > 0 {
				ca.Logger.Printf("Analyzed: %s (Score: %.2f)", filePath, score)
			}
		}(path, size)

		fileCount++
		if fileCount >= batchSize {
			break
		}
	}

	wg.Wait()

	// Apply TF-IDF after batch processing to improve scoring
	ca.updateTFIDFScores()

	return rows.Err()
}

// updateTFIDFScores enhances scores with TF-IDF weighting
func (ca *ContentAnalyzer) updateTFIDFScores() {
	ca.analyzer.CalculateIDF()

	ca.mutex.Lock()
	defer ca.mutex.Unlock()

	// Enhance scores with TF-IDF values
	for docID := range ca.fileScores {
		// Apply a multiplier based on TF-IDF score
		tfidfScore := ca.analyzer.GetDocAverage(docID)
		if tfidfScore > 0 {
			ca.fileScores[docID] = ca.fileScores[docID] * (1 + tfidfScore)
		}
	}
}

// SelectPriorityFiles selects the highest-scored files up to the maximum size
func (ca *ContentAnalyzer) SelectPriorityFiles() []PriorityFile {
	ca.mutex.RLock()
	defer ca.mutex.RUnlock()

	// Get all files with scores
	var allFiles []PriorityFile
	for path, score := range ca.fileScores {
		// Get current file size
		info, err := os.Stat(path)
		if err != nil {
			ca.Logger.Printf("Error accessing file %s: %v", path, err)
			continue
		}

		allFiles = append(allFiles, PriorityFile{
			Path:  path,
			Size:  info.Size(),
			Score: score,
		})
	}

	// Sort by score (highest first)
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Score > allFiles[j].Score
	})

	// Select files up to the maximum total size
	var selected []PriorityFile
	var totalSize int64

	for _, file := range allFiles {
		if totalSize+file.Size <= ca.MaxTotalSize {
			selected = append(selected, file)
			totalSize += file.Size
		} else if len(selected) == 0 {
			// If the first file is already larger than max size, include it anyway
			selected = append(selected, file)
			break
		}
	}

	ca.priorityFiles = selected
	ca.Logger.Printf("Selected %d priority files totaling %d bytes",
		len(selected), totalSize)

	return selected
}

// GetPriorityFiles returns the current list of priority files
func (ca *ContentAnalyzer) GetPriorityFiles() []PriorityFile {
	ca.mutex.RLock()
	defer ca.mutex.RUnlock()
	return ca.priorityFiles
}

// min returns the smaller of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
