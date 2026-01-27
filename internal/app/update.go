package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikecvermeer/laravisor/internal/config"
	"github.com/mikecvermeer/laravisor/internal/log"
	"github.com/mikecvermeer/laravisor/internal/proc"
)

// ProcessAutoRestartMsg is sent when a process should auto-restart
type ProcessAutoRestartMsg struct {
	ID proc.ProcessID
}

// Update handles all incoming messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case TickMsg:
		// Check for new log entries
		if m.LogWatcher != nil {
			select {
			case entry := <-m.LogWatcher.Entries():
				m.AddLogLine(LogLine{
					Content: entry.Content,
					Level:   convertLogLevel(entry.Level),
					File:    entry.File,
				})
			default:
				// No new entries
			}
		}

		// Update process stats every 20 ticks (2 seconds at 100ms interval)
		m.statsTickCount++
		if m.statsTickCount >= 20 {
			m.statsTickCount = 0
			m.UpdateProcessStats()
		}

		return m, tickCmd()

	case proc.ProcessOutputMsg:
		// Add output to process buffer
		if p, ok := m.Processes[msg.ID]; ok {
			p.AddOutput(proc.OutputLine{
				Content:  msg.Line,
				IsStderr: msg.IsStderr,
				Time:     time.Now(),
			})
			// Continue listening for output only if process is still running
			if p.Status == proc.ProcessStatusRunning {
				if ch, ok := m.OutputChans[msg.ID]; ok {
					return m, listenForOutput(msg.ID, ch)
				}
			}
		}
		return m, nil

	case proc.ProcessExitedMsg:
		// Update process status
		if p, ok := m.Processes[msg.ID]; ok {
			if msg.ExitCode != nil && *msg.ExitCode != 0 {
				p.Status = proc.ProcessStatusFailed
			} else {
				p.Status = proc.ProcessStatusStopped
			}
			p.PID = nil

			// Check for auto-restart
			if m.Manager.ShouldRestart(msg.ID) {
				m.Manager.RecordFailure(msg.ID)
				delay := m.Manager.GetBackoffDelay(msg.ID)
				p.Status = proc.ProcessStatusRestarting
				return m, scheduleRestart(msg.ID, delay)
			}
		}
		return m, nil

	case ProcessAutoRestartMsg:
		// Auto-restart a failed process
		if p, ok := m.Processes[msg.ID]; ok && p.Status == proc.ProcessStatusRestarting {
			return m, m.Manager.Spawn(msg.ID, m.OutputChans[msg.ID])
		}
		return m, nil

	case proc.CommandOutputMsg:
		// Add output to the appropriate tab based on command source
		outputLine := proc.OutputLine{
			Content:  msg.Line,
			IsStderr: msg.IsStderr,
			Time:     time.Now(),
		}
		switch m.CommandSource {
		case CommandSourceArtisan:
			m.ArtisanTab.CommandOutput = append(m.ArtisanTab.CommandOutput, outputLine)
		case CommandSourceMake:
			m.MakeTab.CommandOutput = append(m.MakeTab.CommandOutput, outputLine)
		case CommandSourceQuality:
			m.QualityTab.CommandOutput = append(m.QualityTab.CommandOutput, outputLine)
		}
		// Continue listening for more output
		if m.CommandRunner != nil {
			return m, m.CommandRunner.ListenForMore()
		}
		return m, nil

	case proc.CommandExitedMsg:
		// Command finished
		switch m.CommandSource {
		case CommandSourceArtisan:
			m.ArtisanTab.RunningCommand = nil
		case CommandSourceMake:
			m.MakeTab.RunningCommand = nil
		case CommandSourceQuality:
			m.QualityTab.RunningCommand = nil
		}
		m.CommandSource = CommandSourceNone
		return m, nil
	}

	return m, nil
}

// listenForOutput returns a command to listen for process output
func listenForOutput(id proc.ProcessID, ch <-chan proc.ProcessOutputMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return proc.ProcessExitedMsg{ID: id}
		}
		return msg
	}
}

// scheduleRestart schedules an auto-restart after a delay
func scheduleRestart(id proc.ProcessID, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return ProcessAutoRestartMsg{ID: id}
	})
}

// convertLogLevel converts log.LogLevel to app.LogLevel
func convertLogLevel(level log.LogLevel) LogLevel {
	switch level {
	case log.LogLevelDebug:
		return LogLevelDebug
	case log.LogLevelInfo:
		return LogLevelInfo
	case log.LogLevelNotice:
		return LogLevelNotice
	case log.LogLevelWarning:
		return LogLevelWarning
	case log.LogLevelError:
		return LogLevelError
	case log.LogLevelCritical:
		return LogLevelCritical
	case log.LogLevelAlert:
		return LogLevelAlert
	case log.LogLevelEmergency:
		return LogLevelEmergency
	default:
		return LogLevelUnknown
	}
}

// handleKeyMsg processes keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global key bindings that work regardless of tab
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		// Shutdown all processes gracefully
		if m.Manager != nil {
			m.Manager.Shutdown()
		}
		return m, tea.Quit

	case "tab":
		m.ActiveTab = m.ActiveTab.Next()
		return m, nil

	case "shift+tab":
		m.ActiveTab = m.ActiveTab.Previous()
		return m, nil

	case "1":
		m.ActiveTab = TabProcesses
		return m, nil
	case "2":
		m.ActiveTab = TabLogs
		return m, nil
	case "3":
		m.ActiveTab = TabArtisan
		return m, nil
	case "4":
		m.ActiveTab = TabMake
		return m, nil
	case "5":
		m.ActiveTab = TabQuality
		return m, nil
	case "6":
		m.ActiveTab = TabConfig
		return m, nil
	case "?":
		m.ActiveTab = TabAbout
		return m, nil
	}

	// Tab-specific key bindings
	switch m.ActiveTab {
	case TabProcesses:
		return m.updateProcessesTab(msg)
	case TabLogs:
		return m.updateLogsTab(msg)
	case TabArtisan:
		return m.updateArtisanTab(msg)
	case TabMake:
		return m.updateMakeTab(msg)
	case TabQuality:
		return m.updateQualityTab(msg)
	case TabConfig:
		return m.updateConfigTab(msg)
	case TabAbout:
		return m.updateAboutTab(msg)
	}

	return m, nil
}

// updateProcessesTab handles Processes tab key bindings
func (m Model) updateProcessesTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.SelectPrevious()
	case "down", "j":
		m.SelectNext()
	case "enter":
		// Toggle between list and output view
		if m.ProcessesTab.View == ProcessesViewList {
			m.ProcessesTab.View = ProcessesViewOutput
		} else {
			m.ProcessesTab.View = ProcessesViewList
		}
	case "c":
		// Clear output for selected process
		if p := m.SelectedProcess(); p != nil {
			p.ClearOutput()
		}
	case "s":
		// Start selected process
		id := m.SelectedProcessID()
		if id != "" && !m.Manager.IsRunning(id) {
			return m, m.Manager.Spawn(id, m.OutputChans[id])
		}
	case "x":
		// Stop selected process
		id := m.SelectedProcessID()
		if id != "" && m.Manager.IsRunning(id) {
			m.Manager.Kill(id)
			if p := m.SelectedProcess(); p != nil {
				p.Status = proc.ProcessStatusStopped
			}
		}
	case "r":
		// Restart selected process
		id := m.SelectedProcessID()
		if id != "" {
			return m, m.Manager.Restart(id, m.OutputChans[id])
		}
	case "R":
		// Restart all processes
		return m, m.restartAllProcesses()
	case "S":
		// Start all processes
		return m, m.startAllProcesses()
	case "X":
		// Stop all processes
		m.Manager.KillAll()
		for _, p := range m.Processes {
			p.Status = proc.ProcessStatusStopped
		}
	}
	return m, nil
}

// startAllProcesses returns a command to start all processes
func (m *Model) startAllProcesses() tea.Cmd {
	var cmds []tea.Cmd
	for id := range m.Processes {
		if !m.Manager.IsRunning(id) {
			cmds = append(cmds, m.Manager.Spawn(id, m.OutputChans[id]))
		}
	}
	return tea.Batch(cmds...)
}

// restartAllProcesses returns a command to restart all processes
func (m *Model) restartAllProcesses() tea.Cmd {
	var cmds []tea.Cmd
	for id := range m.Processes {
		cmds = append(cmds, m.Manager.Restart(id, m.OutputChans[id]))
	}
	return tea.Batch(cmds...)
}

// updateLogsTab handles Logs tab key bindings
func (m Model) updateLogsTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Input mode for search
	if m.LogsTab.InputMode {
		switch msg.String() {
		case "enter", "esc":
			m.LogsTab.InputMode = false
		case "backspace":
			if len(m.LogsTab.SearchQuery) > 0 {
				m.LogsTab.SearchQuery = m.LogsTab.SearchQuery[:len(m.LogsTab.SearchQuery)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.LogsTab.SearchQuery += msg.String()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		m.LogsTab.ScrollOffset++
	case "down", "j":
		if m.LogsTab.ScrollOffset > 0 {
			m.LogsTab.ScrollOffset--
		}
	case "g":
		// Go to top
		m.LogsTab.ScrollOffset = len(m.LogLines) - 1
	case "G":
		// Go to bottom
		m.LogsTab.ScrollOffset = 0
	case "/":
		// Enter search mode
		m.LogsTab.InputMode = true
		m.LogsTab.SearchQuery = ""
	case "f":
		// Cycle filter level
		m.cycleLogFilter()
	case "F":
		// Cycle log file
		m.cycleLogFile()
	}
	return m, nil
}

// cycleLogFilter cycles through log filter levels
func (m *Model) cycleLogFilter() {
	if m.LogsTab.FilterLevel == nil {
		level := LogLevelDebug
		m.LogsTab.FilterLevel = &level
	} else {
		switch *m.LogsTab.FilterLevel {
		case LogLevelDebug:
			level := LogLevelInfo
			m.LogsTab.FilterLevel = &level
		case LogLevelInfo:
			level := LogLevelNotice
			m.LogsTab.FilterLevel = &level
		case LogLevelNotice:
			level := LogLevelWarning
			m.LogsTab.FilterLevel = &level
		case LogLevelWarning:
			level := LogLevelError
			m.LogsTab.FilterLevel = &level
		case LogLevelError:
			level := LogLevelCritical
			m.LogsTab.FilterLevel = &level
		default:
			m.LogsTab.FilterLevel = nil
		}
	}
}

// cycleLogFile cycles through available log files
func (m *Model) cycleLogFile() {
	if len(m.LogsTab.AvailableFiles) == 0 {
		return
	}

	if m.LogsTab.SelectedFile == nil {
		m.LogsTab.SelectedFile = &m.LogsTab.AvailableFiles[0]
		return
	}

	// Find current index
	currentIdx := -1
	for i, f := range m.LogsTab.AvailableFiles {
		if f == *m.LogsTab.SelectedFile {
			currentIdx = i
			break
		}
	}

	// Move to next or back to "All"
	if currentIdx == -1 || currentIdx >= len(m.LogsTab.AvailableFiles)-1 {
		m.LogsTab.SelectedFile = nil
	} else {
		m.LogsTab.SelectedFile = &m.LogsTab.AvailableFiles[currentIdx+1]
	}
}

// updateArtisanTab handles Artisan tab key bindings
func (m Model) updateArtisanTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input mode for search
	if m.ArtisanTab.SearchMode {
		switch msg.String() {
		case "enter", "esc":
			m.ArtisanTab.SearchMode = false
		case "backspace":
			if len(m.ArtisanTab.SearchQuery) > 0 {
				m.ArtisanTab.SearchQuery = m.ArtisanTab.SearchQuery[:len(m.ArtisanTab.SearchQuery)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.ArtisanTab.SearchQuery += msg.String()
			}
		}
		return m, nil
	}

	// Handle input mode for command args
	if m.ArtisanTab.InputMode {
		switch msg.String() {
		case "enter":
			m.ArtisanTab.InputMode = false
			return m.runArtisanCommand()
		case "esc":
			m.ArtisanTab.InputMode = false
			m.ArtisanTab.InputBuffer = ""
		case "backspace":
			if len(m.ArtisanTab.InputBuffer) > 0 {
				m.ArtisanTab.InputBuffer = m.ArtisanTab.InputBuffer[:len(m.ArtisanTab.InputBuffer)-1]
			}
		default:
			if len(msg.String()) == 1 || msg.String() == " " {
				m.ArtisanTab.InputBuffer += msg.String()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		filtered := m.filteredArtisanCommands()
		if m.ArtisanTab.SelectedCommand > 0 && m.ArtisanTab.SelectedCommand < len(filtered) {
			m.ArtisanTab.SelectedCommand--
			m.ArtisanTab.DetailsScrollOffset = 0 // Reset scroll on selection change
		}
	case "down", "j":
		filtered := m.filteredArtisanCommands()
		if m.ArtisanTab.SelectedCommand < len(filtered)-1 {
			m.ArtisanTab.SelectedCommand++
			m.ArtisanTab.DetailsScrollOffset = 0 // Reset scroll on selection change
		}
	case "ctrl+up", "ctrl+k":
		// Scroll details panel up
		if m.ArtisanTab.DetailsScrollOffset > 0 {
			m.ArtisanTab.DetailsScrollOffset--
		}
	case "ctrl+down", "ctrl+j":
		// Scroll details panel down
		m.ArtisanTab.DetailsScrollOffset++
	case "/":
		// Enter search mode
		m.ArtisanTab.SearchMode = true
		m.ArtisanTab.SearchQuery = ""
	case "i":
		// Enter input mode for command arguments
		m.ArtisanTab.InputMode = true
		m.ArtisanTab.InputBuffer = ""
	case "enter":
		// Run the selected command
		return m.runArtisanCommand()
	case "c":
		// Clear command output
		m.ArtisanTab.CommandOutput = nil
	case "x":
		// Stop running command
		if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
			m.CommandRunner.Stop()
			m.ArtisanTab.RunningCommand = nil
			m.CommandSource = CommandSourceNone
		}
	case "f":
		// Toggle favorite for selected command
		filtered := m.filteredArtisanCommands()
		if m.ArtisanTab.SelectedCommand < len(filtered) {
			cmdName := filtered[m.ArtisanTab.SelectedCommand].Name
			m.toggleArtisanFavorite(cmdName)
			if m.Config != nil {
				config.Save(m.WorkingDir, m.Config)
			}
		}
	case "esc":
		// First check if a command is running - cancel it
		if m.ArtisanTab.RunningCommand != nil {
			if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
				m.CommandRunner.Stop()
			}
			m.ArtisanTab.RunningCommand = nil
			m.CommandSource = CommandSourceNone
			return m, nil
		}
		// If viewing output, go back to list
		if len(m.ArtisanTab.CommandOutput) > 0 {
			m.ArtisanTab.CommandOutput = nil
		}
		// Clear search
		m.ArtisanTab.SearchQuery = ""
	}
	return m, nil
}

// toggleArtisanFavorite toggles a command as a favorite
func (m *Model) toggleArtisanFavorite(name string) {
	if m.Config == nil {
		m.Config = config.NewDefaultConfig()
	}
	// Check if already a favorite
	for i, fav := range m.Config.Artisan.Favorites {
		if fav == name {
			// Remove from favorites
			m.Config.Artisan.Favorites = append(m.Config.Artisan.Favorites[:i], m.Config.Artisan.Favorites[i+1:]...)
			return
		}
	}
	// Add to favorites
	m.Config.Artisan.Favorites = append(m.Config.Artisan.Favorites, name)
}

// isArtisanFavorite checks if a command is a favorite
func (m *Model) isArtisanFavorite(name string) bool {
	if m.Config == nil {
		return false
	}
	for _, fav := range m.Config.Artisan.Favorites {
		if fav == name {
			return true
		}
	}
	return false
}

// filteredArtisanCommands returns commands matching the search query, sorted with favorites first
func (m Model) filteredArtisanCommands() []ArtisanCommand {
	var commands []ArtisanCommand

	if m.ArtisanTab.SearchQuery == "" {
		commands = m.ArtisanTab.Commands
	} else {
		query := strings.ToLower(m.ArtisanTab.SearchQuery)
		for _, cmd := range m.ArtisanTab.Commands {
			if strings.Contains(strings.ToLower(cmd.Name), query) ||
				strings.Contains(strings.ToLower(cmd.Description), query) {
				commands = append(commands, cmd)
			}
		}
	}

	// Sort favorites first, then alphabetically
	var favorites []ArtisanCommand
	var regular []ArtisanCommand
	for _, cmd := range commands {
		if m.isArtisanFavorite(cmd.Name) {
			favorites = append(favorites, cmd)
		} else {
			regular = append(regular, cmd)
		}
	}

	return append(favorites, regular...)
}

// runArtisanCommand runs the selected artisan command
func (m Model) runArtisanCommand() (tea.Model, tea.Cmd) {
	filtered := m.filteredArtisanCommands()
	if len(filtered) == 0 || m.ArtisanTab.SelectedCommand >= len(filtered) {
		return m, nil
	}

	// Stop any running command
	if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
		m.CommandRunner.Stop()
	}

	cmd := filtered[m.ArtisanTab.SelectedCommand]
	args := []string{"artisan", cmd.Name}

	// Add input buffer args if any
	if m.ArtisanTab.InputBuffer != "" {
		args = append(args, strings.Fields(m.ArtisanTab.InputBuffer)...)
	}

	// Clear previous output
	m.ArtisanTab.CommandOutput = nil
	m.ArtisanTab.RunningCommand = &cmd.Name
	m.CommandSource = CommandSourceArtisan
	m.ArtisanTab.InputBuffer = ""

	return m, m.CommandRunner.Run("php", args)
}

// updateMakeTab handles Make tab key bindings
func (m Model) updateMakeTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input mode for search
	if m.MakeTab.SearchMode {
		switch msg.String() {
		case "enter", "esc":
			m.MakeTab.SearchMode = false
		case "backspace":
			if len(m.MakeTab.SearchQuery) > 0 {
				m.MakeTab.SearchQuery = m.MakeTab.SearchQuery[:len(m.MakeTab.SearchQuery)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.MakeTab.SearchQuery += msg.String()
			}
		}
		return m, nil
	}

	// Handle input mode for name/args
	if m.MakeTab.InputMode {
		switch msg.String() {
		case "enter":
			if m.MakeTab.InputBuffer == "" {
				return m, nil // Require a name
			}
			m.MakeTab.InputMode = false
			return m.runMakeCommand()
		case "esc":
			m.MakeTab.InputMode = false
			m.MakeTab.InputBuffer = ""
		case "backspace":
			if len(m.MakeTab.InputBuffer) > 0 {
				m.MakeTab.InputBuffer = m.MakeTab.InputBuffer[:len(m.MakeTab.InputBuffer)-1]
			}
		default:
			if len(msg.String()) == 1 || msg.String() == " " {
				m.MakeTab.InputBuffer += msg.String()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		filtered := m.filteredMakeCommands()
		if m.MakeTab.SelectedCommand > 0 && m.MakeTab.SelectedCommand < len(filtered) {
			m.MakeTab.SelectedCommand--
		}
	case "down", "j":
		filtered := m.filteredMakeCommands()
		if m.MakeTab.SelectedCommand < len(filtered)-1 {
			m.MakeTab.SelectedCommand++
		}
	case "/":
		// Enter search mode
		m.MakeTab.SearchMode = true
		m.MakeTab.SearchQuery = ""
	case "i", "enter":
		// Enter input mode for name (required for make commands)
		m.MakeTab.InputMode = true
		m.MakeTab.InputBuffer = ""
	case "c":
		// Clear command output
		m.MakeTab.CommandOutput = nil
	case "x":
		// Stop running command
		if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
			m.CommandRunner.Stop()
			m.MakeTab.RunningCommand = nil
			m.CommandSource = CommandSourceNone
		}
	case "f":
		// Toggle favorite for selected command
		filtered := m.filteredMakeCommands()
		if m.MakeTab.SelectedCommand < len(filtered) {
			cmdName := filtered[m.MakeTab.SelectedCommand].Name
			m.toggleMakeFavorite(cmdName)
			if m.Config != nil {
				config.Save(m.WorkingDir, m.Config)
			}
		}
	case "esc":
		// First check if a command is running - cancel it
		if m.MakeTab.RunningCommand != nil {
			if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
				m.CommandRunner.Stop()
			}
			m.MakeTab.RunningCommand = nil
			m.CommandSource = CommandSourceNone
			return m, nil
		}
		// If viewing output, go back to list
		if len(m.MakeTab.CommandOutput) > 0 {
			m.MakeTab.CommandOutput = nil
		}
		// Clear search
		m.MakeTab.SearchQuery = ""
	}
	return m, nil
}

// toggleMakeFavorite toggles a command as a favorite
func (m *Model) toggleMakeFavorite(name string) {
	if m.Config == nil {
		m.Config = config.NewDefaultConfig()
	}
	// Check if already a favorite
	for i, fav := range m.Config.Make.Favorites {
		if fav == name {
			// Remove from favorites
			m.Config.Make.Favorites = append(m.Config.Make.Favorites[:i], m.Config.Make.Favorites[i+1:]...)
			return
		}
	}
	// Add to favorites
	m.Config.Make.Favorites = append(m.Config.Make.Favorites, name)
}

// isMakeFavorite checks if a command is a favorite
func (m *Model) isMakeFavorite(name string) bool {
	if m.Config == nil {
		return false
	}
	for _, fav := range m.Config.Make.Favorites {
		if fav == name {
			return true
		}
	}
	return false
}

// filteredMakeCommands returns commands matching the search query, sorted with favorites first
func (m Model) filteredMakeCommands() []ArtisanCommand {
	var commands []ArtisanCommand

	if m.MakeTab.SearchQuery == "" {
		commands = m.MakeTab.Commands
	} else {
		query := strings.ToLower(m.MakeTab.SearchQuery)
		for _, cmd := range m.MakeTab.Commands {
			if strings.Contains(strings.ToLower(cmd.Name), query) ||
				strings.Contains(strings.ToLower(cmd.Description), query) {
				commands = append(commands, cmd)
			}
		}
	}

	// Sort favorites first, then alphabetically
	var favorites []ArtisanCommand
	var regular []ArtisanCommand
	for _, cmd := range commands {
		if m.isMakeFavorite(cmd.Name) {
			favorites = append(favorites, cmd)
		} else {
			regular = append(regular, cmd)
		}
	}

	return append(favorites, regular...)
}

// runMakeCommand runs the selected make command
func (m Model) runMakeCommand() (tea.Model, tea.Cmd) {
	filtered := m.filteredMakeCommands()
	if len(filtered) == 0 || m.MakeTab.SelectedCommand >= len(filtered) {
		return m, nil
	}

	// Stop any running command
	if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
		m.CommandRunner.Stop()
	}

	cmd := filtered[m.MakeTab.SelectedCommand]
	args := []string{"artisan", cmd.Name}

	// Add input buffer (name and any extra args)
	if m.MakeTab.InputBuffer != "" {
		args = append(args, strings.Fields(m.MakeTab.InputBuffer)...)
	}

	// Clear previous output
	m.MakeTab.CommandOutput = nil
	m.MakeTab.RunningCommand = &cmd.Name
	m.CommandSource = CommandSourceMake
	m.MakeTab.InputBuffer = ""

	return m, m.CommandRunner.Run("php", args)
}

// updateQualityTab handles Quality tab key bindings
func (m Model) updateQualityTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input mode for extra args
	if m.QualityTab.InputMode {
		switch msg.String() {
		case "enter":
			m.QualityTab.InputMode = false
			return m.runQualityTool()
		case "esc":
			m.QualityTab.InputMode = false
			m.QualityTab.InputBuffer = ""
		case "backspace":
			if len(m.QualityTab.InputBuffer) > 0 {
				m.QualityTab.InputBuffer = m.QualityTab.InputBuffer[:len(m.QualityTab.InputBuffer)-1]
			}
		default:
			if len(msg.String()) == 1 || msg.String() == " " {
				m.QualityTab.InputBuffer += msg.String()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "left", "h":
		if m.QualityTab.SelectedCategory == QualityCategoryTesting {
			m.QualityTab.SelectedCategory = QualityCategoryTools
			m.QualityTab.SelectedTool = 0
		}
	case "right", "l":
		if m.QualityTab.SelectedCategory == QualityCategoryTools {
			m.QualityTab.SelectedCategory = QualityCategoryTesting
			m.QualityTab.SelectedTool = 0
		}
	case "up", "k":
		if m.QualityTab.SelectedTool > 0 {
			m.QualityTab.SelectedTool--
		}
	case "down", "j":
		tools := m.currentQualityTools()
		if m.QualityTab.SelectedTool < len(tools)-1 {
			m.QualityTab.SelectedTool++
		}
	case "i":
		// Enter input mode for extra arguments
		m.QualityTab.InputMode = true
		m.QualityTab.InputBuffer = ""
	case "enter":
		// Run the selected tool
		return m.runQualityTool()
	case "c":
		// Clear command output
		m.QualityTab.CommandOutput = nil
	case "x":
		// Stop running command
		if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
			m.CommandRunner.Stop()
			m.QualityTab.RunningCommand = nil
			m.CommandSource = CommandSourceNone
		}
	case "esc":
		// First check if a command is running - cancel it
		if m.QualityTab.RunningCommand != nil {
			if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
				m.CommandRunner.Stop()
			}
			m.QualityTab.RunningCommand = nil
			m.CommandSource = CommandSourceNone
			return m, nil
		}
		// If viewing output, go back to list
		if len(m.QualityTab.CommandOutput) > 0 {
			m.QualityTab.CommandOutput = nil
		}
	}
	return m, nil
}

// runQualityTool runs the selected quality tool
func (m Model) runQualityTool() (tea.Model, tea.Cmd) {
	tools := m.currentQualityTools()
	if len(tools) == 0 || m.QualityTab.SelectedTool >= len(tools) {
		return m, nil
	}

	// Stop any running command
	if m.CommandRunner != nil && m.CommandRunner.IsRunning() {
		m.CommandRunner.Stop()
	}

	tool := tools[m.QualityTab.SelectedTool]
	args := make([]string, len(tool.Args))
	copy(args, tool.Args)

	// Add input buffer args if any
	if m.QualityTab.InputBuffer != "" {
		args = append(args, strings.Fields(m.QualityTab.InputBuffer)...)
	}

	// Clear previous output
	m.QualityTab.CommandOutput = nil
	m.QualityTab.RunningCommand = &tool.DisplayName
	m.CommandSource = CommandSourceQuality
	m.QualityTab.InputBuffer = ""

	return m, m.CommandRunner.Run(tool.Command, args)
}

// currentQualityTools returns the tools for the current category
func (m *Model) currentQualityTools() []QualityTool {
	if m.QualityTab.SelectedCategory == QualityCategoryTools {
		return m.QualityTab.QualityTools
	}
	return m.QualityTab.TestingTools
}

// updateConfigTab handles Config tab key bindings
func (m Model) updateConfigTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle edit mode for log settings
	if m.ConfigTab.EditMode {
		switch msg.String() {
		case "enter":
			m.ConfigTab.EditMode = false
			// Apply the edit
			if m.ConfigTab.SelectedSection == ConfigSectionLogs && m.ConfigTab.SelectedItem == 0 {
				// Parse max_lines
				var maxLines int
				if _, err := fmt.Sscanf(m.ConfigTab.EditBuffer, "%d", &maxLines); err == nil && maxLines > 0 {
					if m.Config == nil {
						m.Config = config.NewDefaultConfig()
					}
					m.Config.Logs.MaxLines = &maxLines
					m.MaxLogLines = maxLines
					m.ConfigTab.HasChanges = true
				}
			}
			m.ConfigTab.EditBuffer = ""
		case "esc":
			m.ConfigTab.EditMode = false
			m.ConfigTab.EditBuffer = ""
		case "backspace":
			if len(m.ConfigTab.EditBuffer) > 0 {
				m.ConfigTab.EditBuffer = m.ConfigTab.EditBuffer[:len(m.ConfigTab.EditBuffer)-1]
			}
		default:
			// Only accept digits for numeric fields
			if len(msg.String()) == 1 && msg.String() >= "0" && msg.String() <= "9" {
				m.ConfigTab.EditBuffer += msg.String()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "left", "h":
		if m.ConfigTab.SelectedSection > 0 {
			m.ConfigTab.SelectedSection--
			m.ConfigTab.SelectedItem = 0
		}
	case "right", "l":
		if m.ConfigTab.SelectedSection < ConfigSectionInfo {
			m.ConfigTab.SelectedSection++
			m.ConfigTab.SelectedItem = 0
		}
	case "up", "k":
		if m.ConfigTab.SelectedItem > 0 {
			m.ConfigTab.SelectedItem--
		}
	case "down", "j":
		maxItems := m.configSectionItemCount()
		if m.ConfigTab.SelectedItem < maxItems-1 {
			m.ConfigTab.SelectedItem++
		}
	case "enter", " ":
		// Toggle or edit the selected item
		return m.toggleConfigItem()
	case "s":
		// Save configuration
		return m.saveConfig()
	}
	return m, nil
}

// configSectionItemCount returns the number of items in the current config section
func (m Model) configSectionItemCount() int {
	switch m.ConfigTab.SelectedSection {
	case ConfigSectionDisabled:
		return 5 // serve, vite, queue, horizon, reverb
	case ConfigSectionLogs:
		return 1 // max_lines
	case ConfigSectionInfo:
		return 1 // config file path
	}
	return 0
}

// toggleConfigItem toggles or edits the selected config item
func (m Model) toggleConfigItem() (tea.Model, tea.Cmd) {
	if m.Config == nil {
		m.Config = config.NewDefaultConfig()
	}

	switch m.ConfigTab.SelectedSection {
	case ConfigSectionDisabled:
		switch m.ConfigTab.SelectedItem {
		case 0:
			m.Config.Disabled.Serve = !m.Config.Disabled.Serve
		case 1:
			m.Config.Disabled.Vite = !m.Config.Disabled.Vite
		case 2:
			m.Config.Disabled.Queue = !m.Config.Disabled.Queue
		case 3:
			m.Config.Disabled.Horizon = !m.Config.Disabled.Horizon
		case 4:
			m.Config.Disabled.Reverb = !m.Config.Disabled.Reverb
		}
		m.ConfigTab.HasChanges = true

	case ConfigSectionLogs:
		if m.ConfigTab.SelectedItem == 0 {
			// Edit max_lines
			m.ConfigTab.EditMode = true
			if m.Config.Logs.MaxLines != nil {
				m.ConfigTab.EditBuffer = fmt.Sprintf("%d", *m.Config.Logs.MaxLines)
			} else {
				m.ConfigTab.EditBuffer = fmt.Sprintf("%d", m.MaxLogLines)
			}
		}
	}

	return m, nil
}

// saveConfig saves the configuration to disk
func (m Model) saveConfig() (tea.Model, tea.Cmd) {
	if m.Config == nil {
		m.ConfigTab.StatusMessage = "No configuration to save"
		return m, nil
	}

	if err := config.Save(m.WorkingDir, m.Config); err != nil {
		m.ConfigTab.StatusMessage = fmt.Sprintf("Error: %v", err)
	} else {
		m.ConfigTab.StatusMessage = "Configuration saved"
		m.ConfigTab.HasChanges = false
	}

	return m, nil
}

// updateAboutTab handles About tab key bindings
func (m Model) updateAboutTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.AboutTab.ScrollOffset++
	case "down", "j":
		if m.AboutTab.ScrollOffset > 0 {
			m.AboutTab.ScrollOffset--
		}
	}
	return m, nil
}
