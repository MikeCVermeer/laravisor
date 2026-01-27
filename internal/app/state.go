package app

import "github.com/mikecvermeer/laravisor/internal/proc"

// Tab represents the available tabs in the application
type Tab int

const (
	TabProcesses Tab = iota
	TabLogs
	TabArtisan
	TabMake
	TabQuality
	TabConfig
	TabAbout
)

// AllTabs returns all tabs in display order
func AllTabs() []Tab {
	return []Tab{
		TabProcesses,
		TabLogs,
		TabArtisan,
		TabMake,
		TabQuality,
		TabConfig,
		TabAbout,
	}
}

// Name returns the display name for the tab
func (t Tab) Name() string {
	switch t {
	case TabProcesses:
		return "Processes"
	case TabLogs:
		return "Logs"
	case TabArtisan:
		return "Artisan"
	case TabMake:
		return "Make"
	case TabQuality:
		return "Quality"
	case TabConfig:
		return "Config"
	case TabAbout:
		return "About"
	default:
		return "Unknown"
	}
}

// Shortcut returns the keyboard shortcut for the tab
func (t Tab) Shortcut() string {
	switch t {
	case TabProcesses:
		return "1"
	case TabLogs:
		return "2"
	case TabArtisan:
		return "3"
	case TabMake:
		return "4"
	case TabQuality:
		return "5"
	case TabConfig:
		return "6"
	case TabAbout:
		return "?"
	default:
		return ""
	}
}

// Next returns the next tab (wraps around)
func (t Tab) Next() Tab {
	if t == TabAbout {
		return TabProcesses
	}
	return t + 1
}

// Previous returns the previous tab (wraps around)
func (t Tab) Previous() Tab {
	if t == TabProcesses {
		return TabAbout
	}
	return t - 1
}

// ProcessesView represents the view mode in the Processes tab
type ProcessesView int

const (
	ProcessesViewList ProcessesView = iota
	ProcessesViewOutput
)

// ProcessesTabState holds state for the Processes tab
type ProcessesTabState struct {
	View               ProcessesView
	SelectedIndex      int
	OutputScrollOffset int
}

// LogLevel represents Laravel log levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelNotice
	LogLevelWarning
	LogLevelError
	LogLevelCritical
	LogLevelAlert
	LogLevelEmergency
	LogLevelUnknown
)

// Name returns the display name for the log level
func (l LogLevel) Name() string {
	switch l {
	case LogLevelDebug:
		return "Debug"
	case LogLevelInfo:
		return "Info"
	case LogLevelNotice:
		return "Notice"
	case LogLevelWarning:
		return "Warning"
	case LogLevelError:
		return "Error"
	case LogLevelCritical:
		return "Critical"
	case LogLevelAlert:
		return "Alert"
	case LogLevelEmergency:
		return "Emergency"
	default:
		return "Unknown"
	}
}

// AllLogLevels returns all log levels for filtering
func AllLogLevels() []LogLevel {
	return []LogLevel{
		LogLevelDebug,
		LogLevelInfo,
		LogLevelNotice,
		LogLevelWarning,
		LogLevelError,
		LogLevelCritical,
		LogLevelAlert,
		LogLevelEmergency,
	}
}

// LogLine represents a parsed log entry
type LogLine struct {
	Content string
	Level   LogLevel
	File    string
}

// LogsTabState holds state for the Logs tab
type LogsTabState struct {
	SearchQuery    string
	FilterLevel    *LogLevel
	ScrollOffset   int
	InputMode      bool
	SelectedFile   *string
	AvailableFiles []string
}

// ArtisanCommand represents a discovered artisan command
type ArtisanCommand struct {
	Name        string
	Description string
	Arguments   []string
	Options     []string
}

// ArtisanTabState holds state for the Artisan tab
type ArtisanTabState struct {
	SelectedCommand     int
	InputBuffer         string
	InputMode           bool
	CommandOutput       []proc.OutputLine
	OutputScrollOffset  int
	RunningCommand      *string
	Commands            []ArtisanCommand
	SearchQuery         string
	SearchMode          bool
	DetailsScrollOffset int
}

// MakeTabState holds state for the Make tab
type MakeTabState struct {
	SelectedCommand    int
	InputBuffer        string
	InputMode          bool
	CommandOutput      []proc.OutputLine
	OutputScrollOffset int
	RunningCommand     *string
	Commands           []ArtisanCommand
	SearchQuery        string
	SearchMode         bool
}

// QualityCategory represents categories in the Quality tab
type QualityCategory int

const (
	QualityCategoryTools QualityCategory = iota
	QualityCategoryTesting
)

// Name returns the display name for the category
func (c QualityCategory) Name() string {
	switch c {
	case QualityCategoryTools:
		return "Quality Tools"
	case QualityCategoryTesting:
		return "Testing"
	default:
		return "Unknown"
	}
}

// QualityTool represents a quality or testing tool
type QualityTool struct {
	Name        string
	DisplayName string
	Command     string
	Args        []string
	Category    QualityCategory
}

// QualityTabState holds state for the Quality tab
type QualityTabState struct {
	SelectedCategory   QualityCategory
	SelectedTool       int
	InputBuffer        string
	InputMode          bool
	CommandOutput      []proc.OutputLine
	OutputScrollOffset int
	RunningCommand     *string
	QualityTools       []QualityTool
	TestingTools       []QualityTool
}

// ConfigSection represents a section in the Config tab
type ConfigSection int

const (
	ConfigSectionDisabled ConfigSection = iota
	ConfigSectionLogs
	ConfigSectionInfo
)

// Name returns the display name for the section
func (s ConfigSection) Name() string {
	switch s {
	case ConfigSectionDisabled:
		return "Disabled Processes"
	case ConfigSectionLogs:
		return "Log Settings"
	case ConfigSectionInfo:
		return "Config File"
	default:
		return "Unknown"
	}
}

// ConfigTabState holds state for the Config tab
type ConfigTabState struct {
	SelectedSection ConfigSection
	SelectedItem    int
	EditMode        bool
	EditBuffer      string
	HasChanges      bool
	StatusMessage   string
}

// AboutTabState holds state for the About tab (minimal state needed)
type AboutTabState struct {
	ScrollOffset int
}
