package ui

import "github.com/charmbracelet/lipgloss"

// Theme provides consistent styling across the application
var Theme = struct {
	// Primary accent color - warm orange
	Accent lipgloss.Color
	// Secondary accent - slightly dimmer
	AccentDim lipgloss.Color
	// Border color - subtle gray
	Border lipgloss.Color
	// Border color for focused panels
	BorderFocused lipgloss.Color
	// Border color for inactive panels
	BorderInactive lipgloss.Color
	// Selection background
	SelectionBg lipgloss.Color
	// Text colors
	Text         lipgloss.Color
	TextDim      lipgloss.Color
	TextMuted    lipgloss.Color
	TextDisabled lipgloss.Color
	// Status colors
	Success lipgloss.Color
	Error   lipgloss.Color
	Warning lipgloss.Color
	Info    lipgloss.Color
	// Log level colors
	LogDebug    lipgloss.Color
	LogInfo     lipgloss.Color
	LogNotice   lipgloss.Color
	LogWarning  lipgloss.Color
	LogError    lipgloss.Color
	LogCritical lipgloss.Color
	// Status bar background
	StatusBarBg lipgloss.Color
}{
	Accent:         lipgloss.Color("#D97757"),
	AccentDim:      lipgloss.Color("#B4644B"),
	Border:         lipgloss.Color("#4B5563"),
	BorderFocused:  lipgloss.Color("#D97706"),
	BorderInactive: lipgloss.Color("#374151"),
	SelectionBg:    lipgloss.Color("#374151"),
	Text:           lipgloss.Color("#E5E7EB"),
	TextDim:        lipgloss.Color("#9CA3AF"),
	TextMuted:      lipgloss.Color("#6B7280"),
	TextDisabled:   lipgloss.Color("#4B5563"),
	Success:        lipgloss.Color("#22C55E"),
	Error:          lipgloss.Color("#EF4444"),
	Warning:        lipgloss.Color("#EAB308"),
	Info:           lipgloss.Color("#60A5FA"),
	LogDebug:       lipgloss.Color("#6B7280"),
	LogInfo:        lipgloss.Color("#22C55E"),
	LogNotice:      lipgloss.Color("#60A5FA"),
	LogWarning:     lipgloss.Color("#EAB308"),
	LogError:       lipgloss.Color("#EF4444"),
	LogCritical:    lipgloss.Color("#EF4444"),
	StatusBarBg:    lipgloss.Color("#1F2937"),
}

// Symbols for status indicators
var Symbols = struct {
	Running    string
	Stopped    string
	Restarting string
	Failed     string
	Selector   string
	Favorite   string
}{
	Running:    "●",
	Stopped:    "○",
	Restarting: "↻",
	Failed:     "✗",
	Selector:   "▶",
	Favorite:   "★",
}

// Styles provides pre-configured lipgloss styles
var Styles = struct {
	// Tab styles
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	TabBar      lipgloss.Style
	// Title style
	Title lipgloss.Style
	// Status bar
	StatusBar     lipgloss.Style
	StatusMessage lipgloss.Style
	// Help text
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
	// Selection
	Selected lipgloss.Style
	// Process status
	StatusRunning    lipgloss.Style
	StatusStopped    lipgloss.Style
	StatusRestarting lipgloss.Style
	StatusFailed     lipgloss.Style
	// Text styles
	TextMuted lipgloss.Style
}{
	TabActive: lipgloss.NewStyle().
		Foreground(Theme.Accent).
		Background(Theme.SelectionBg).
		Bold(true).
		Padding(0, 2),
	TabInactive: lipgloss.NewStyle().
		Foreground(Theme.TextDim).
		Padding(0, 2),
	TabBar: lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(Theme.Border),
	Title: lipgloss.NewStyle().
		Foreground(Theme.Accent).
		Bold(true),
	StatusBar: lipgloss.NewStyle().
		Background(Theme.StatusBarBg).
		Foreground(Theme.Text).
		Padding(0, 1),
	StatusMessage: lipgloss.NewStyle().
		Foreground(Theme.Info),
	HelpKey: lipgloss.NewStyle().
		Foreground(Theme.Accent).
		Bold(true),
	HelpDesc: lipgloss.NewStyle().
		Foreground(Theme.TextMuted),
	Selected: lipgloss.NewStyle().
		Background(Theme.SelectionBg).
		Foreground(Theme.Text),
	StatusRunning: lipgloss.NewStyle().
		Foreground(Theme.Success),
	StatusStopped: lipgloss.NewStyle().
		Foreground(Theme.TextMuted),
	StatusRestarting: lipgloss.NewStyle().
		Foreground(Theme.Warning),
	StatusFailed: lipgloss.NewStyle().
		Foreground(Theme.Error),
	TextMuted: lipgloss.NewStyle().
		Foreground(Theme.TextMuted),
}

// BorderedBox creates a box with rounded borders and title
func BorderedBox(title string, focused bool) lipgloss.Style {
	borderColor := Theme.Border
	if focused {
		borderColor = Theme.BorderFocused
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
}
