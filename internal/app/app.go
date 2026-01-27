package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikecvermeer/laravisor/internal/config"
	"github.com/mikecvermeer/laravisor/internal/log"
	"github.com/mikecvermeer/laravisor/internal/proc"
)

// CommandSource identifies where a command is running from
type CommandSource int

const (
	CommandSourceNone CommandSource = iota
	CommandSourceArtisan
	CommandSourceMake
	CommandSourceQuality
)

// DefaultMaxLogLines is the default maximum number of log lines to keep
const DefaultMaxLogLines = 100

// TickInterval is the interval for periodic updates
const TickInterval = 100 * time.Millisecond

// Model is the main Bubble Tea model for the application
type Model struct {
	// Current active tab
	ActiveTab Tab

	// Tab states
	ProcessesTab ProcessesTabState
	LogsTab      LogsTabState
	ArtisanTab   ArtisanTabState
	MakeTab      MakeTabState
	QualityTab   QualityTabState
	ConfigTab    ConfigTabState
	AboutTab     AboutTabState

	// Process management
	Processes    map[proc.ProcessID]*proc.Process
	ProcessOrder []proc.ProcessID
	Manager      *proc.Manager
	OutputChans  map[proc.ProcessID]chan proc.ProcessOutputMsg

	// Log management
	LogLines    []LogLine
	MaxLogLines int
	LogWatcher  *log.Watcher

	// Command runner for one-off commands (artisan, make, quality)
	CommandRunner *proc.CommandRunner
	CommandSource CommandSource

	// Application state
	WorkingDir    string
	Config        *config.Config
	ConfigError   string
	StatusMessage string
	Width         int
	Height        int

	// Internal
	quitting bool
}

// New creates a new application model
func New(workingDir string) Model {
	return Model{
		ActiveTab:     TabProcesses,
		Processes:     make(map[proc.ProcessID]*proc.Process),
		ProcessOrder:  make([]proc.ProcessID, 0),
		Manager:       proc.NewManager(),
		OutputChans:   make(map[proc.ProcessID]chan proc.ProcessOutputMsg),
		LogLines:      make([]LogLine, 0),
		MaxLogLines:   DefaultMaxLogLines,
		WorkingDir:    workingDir,
		CommandRunner: proc.NewCommandRunner(workingDir),
		CommandSource: CommandSourceNone,
	}
}

// TickMsg is sent periodically for updates
type TickMsg time.Time

// tickCmd returns a command that sends a tick after the interval
func tickCmd() tea.Cmd {
	return tea.Tick(TickInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tickCmd()
}

// SetConfig sets the configuration
func (m *Model) SetConfig(cfg *config.Config) {
	m.Config = cfg
	if cfg != nil && cfg.Logs.MaxLines != nil {
		m.MaxLogLines = *cfg.Logs.MaxLines
	}
}

// RegisterProcess registers a process configuration
func (m *Model) RegisterProcess(cfg proc.ProcessConfig) {
	id := cfg.ID
	if _, exists := m.Processes[id]; !exists {
		m.ProcessOrder = append(m.ProcessOrder, id)
	}
	p := proc.NewProcess(cfg)
	m.Processes[id] = p
	m.Manager.Register(p)
	m.OutputChans[id] = make(chan proc.ProcessOutputMsg, 100)
}

// SelectedProcess returns the currently selected process
func (m *Model) SelectedProcess() *proc.Process {
	if m.ProcessesTab.SelectedIndex >= len(m.ProcessOrder) {
		return nil
	}
	id := m.ProcessOrder[m.ProcessesTab.SelectedIndex]
	return m.Processes[id]
}

// SelectedProcessID returns the ID of the currently selected process
func (m *Model) SelectedProcessID() proc.ProcessID {
	if m.ProcessesTab.SelectedIndex >= len(m.ProcessOrder) {
		return ""
	}
	return m.ProcessOrder[m.ProcessesTab.SelectedIndex]
}

// SelectPrevious moves selection up
func (m *Model) SelectPrevious() {
	if len(m.ProcessOrder) > 0 && m.ProcessesTab.SelectedIndex > 0 {
		m.ProcessesTab.SelectedIndex--
	}
}

// SelectNext moves selection down
func (m *Model) SelectNext() {
	if len(m.ProcessOrder) > 0 && m.ProcessesTab.SelectedIndex < len(m.ProcessOrder)-1 {
		m.ProcessesTab.SelectedIndex++
	}
}

// SetStatus sets a status message
func (m *Model) SetStatus(msg string) {
	m.StatusMessage = msg
}

// ClearStatus clears the status message
func (m *Model) ClearStatus() {
	m.StatusMessage = ""
}

// AddLogLine adds a log line
func (m *Model) AddLogLine(line LogLine) {
	if len(m.LogLines) >= m.MaxLogLines {
		m.LogLines = m.LogLines[1:]
	}
	m.LogLines = append(m.LogLines, line)

	// Track available files
	found := false
	for _, f := range m.LogsTab.AvailableFiles {
		if f == line.File {
			found = true
			break
		}
	}
	if !found {
		m.LogsTab.AvailableFiles = append(m.LogsTab.AvailableFiles, line.File)
	}
}
