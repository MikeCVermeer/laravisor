package log

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// LogUpdateMsg is sent when new log entries are available
type LogUpdateMsg struct {
	Entries []LogEntry
}

// Watcher watches Laravel log files for changes
type Watcher struct {
	logDir          string
	additionalFiles []string
	watcher         *fsnotify.Watcher
	positions       map[string]int64
	mu              sync.Mutex
	stopCh          chan struct{}
	entryCh         chan LogEntry
}

// NewWatcher creates a new log watcher
func NewWatcher(logDir string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		logDir:    logDir,
		watcher:   fsWatcher,
		positions: make(map[string]int64),
		stopCh:    make(chan struct{}),
		entryCh:   make(chan LogEntry, 100),
	}

	return w, nil
}

// AddAdditionalFiles adds additional log files to watch
func (w *Watcher) AddAdditionalFiles(files []string) {
	w.additionalFiles = files
}

// Start begins watching for log changes
func (w *Watcher) Start() error {
	// Watch the log directory
	if err := w.watcher.Add(w.logDir); err != nil {
		return err
	}

	// Watch parent directories of additional files
	for _, file := range w.additionalFiles {
		parent := filepath.Dir(file)
		if parent != w.logDir {
			w.watcher.Add(parent)
		}
	}

	// Initialize positions and read recent history
	w.initializeFiles()

	// Start watching goroutine
	go w.watch()

	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.stopCh)
	w.watcher.Close()
}

// Entries returns the channel for receiving log entries
func (w *Watcher) Entries() <-chan LogEntry {
	return w.entryCh
}

// initializeFiles reads initial log files and sets up positions
func (w *Watcher) initializeFiles() {
	// Find all .log files in the log directory
	entries, err := os.ReadDir(w.logDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		path := filepath.Join(w.logDir, entry.Name())
		w.initializeFile(path)
	}

	// Initialize additional files
	for _, path := range w.additionalFiles {
		w.initializeFile(path)
	}
}

// initializeFile reads the last few lines of a file and sets the position
func (w *Watcher) initializeFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	fileName := filepath.Base(path)

	// Read last 5 lines as history
	lines := readLastNLines(file, 5)
	for _, line := range lines {
		w.entryCh <- LogEntry{
			Content: line,
			Level:   DetectLogLevel(line),
			File:    fileName,
		}
	}

	// Set position to end of file
	info, err := file.Stat()
	if err == nil {
		w.mu.Lock()
		w.positions[path] = info.Size()
		w.mu.Unlock()
	}
}

// watch is the main watch loop
func (w *Watcher) watch() {
	for {
		select {
		case <-w.stopCh:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only process write and create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Check if this is a log file we should process
			if !w.shouldProcess(event.Name) {
				continue
			}

			// Read new lines
			w.readNewLines(event.Name)

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// shouldProcess checks if we should process events for this file
func (w *Watcher) shouldProcess(path string) bool {
	// Check if it's a .log file in the log directory
	if strings.HasSuffix(path, ".log") && filepath.Dir(path) == w.logDir {
		return true
	}

	// Check if it's an additional file
	for _, additional := range w.additionalFiles {
		if path == additional {
			return true
		}
	}

	return false
}

// readNewLines reads new lines from a file
func (w *Watcher) readNewLines(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return
	}

	currentSize := info.Size()
	fileName := filepath.Base(path)

	w.mu.Lock()
	lastPos, exists := w.positions[path]
	if !exists {
		lastPos = 0
	}

	// File was truncated (rotated), start from beginning
	if currentSize < lastPos {
		lastPos = 0
	}

	// No new content
	if currentSize == lastPos {
		w.mu.Unlock()
		return
	}

	w.positions[path] = currentSize
	w.mu.Unlock()

	// Seek to last position
	_, err = file.Seek(lastPos, io.SeekStart)
	if err != nil {
		return
	}

	// Read new lines
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		select {
		case w.entryCh <- LogEntry{
			Content: line,
			Level:   DetectLogLevel(line),
			File:    fileName,
		}:
		case <-w.stopCh:
			return
		}
	}
}

// readLastNLines reads the last N non-empty lines from a file
func readLastNLines(file *os.File, n int) []string {
	scanner := bufio.NewScanner(file)
	var lines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	// Return last N lines
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// FindLogDir finds the Laravel log directory in a project
func FindLogDir(workingDir string) string {
	logDir := filepath.Join(workingDir, "storage", "logs")
	if info, err := os.Stat(logDir); err == nil && info.IsDir() {
		return logDir
	}
	return ""
}
