package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mikecvermeer/laravisor/internal/proc"
	"github.com/mikecvermeer/laravisor/internal/ui"
)

// View renders the application UI
func (m Model) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	var b strings.Builder

	// Render tab bar
	b.WriteString(m.renderTabBar())
	b.WriteString("\n")

	// Calculate content height (total - tab bar - status bar)
	contentHeight := m.Height - 4 // 2 for tab bar, 2 for status bar

	// Render active tab content
	content := m.renderActiveTab(contentHeight)
	b.WriteString(content)

	// Render status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderTabBar renders the tab navigation bar
func (m Model) renderTabBar() string {
	var tabs []string

	for _, tab := range AllTabs() {
		label := fmt.Sprintf("%s %s", tab.Shortcut(), tab.Name())
		if tab == m.ActiveTab {
			tabs = append(tabs, ui.Styles.TabActive.Render(label))
		} else {
			tabs = append(tabs, ui.Styles.TabInactive.Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return ui.Styles.TabBar.Width(m.Width).Render(tabBar)
}

// renderActiveTab renders the content of the active tab
func (m Model) renderActiveTab(height int) string {
	contentWidth := m.Width - 2 // Account for padding

	switch m.ActiveTab {
	case TabProcesses:
		return m.renderProcessesTab(contentWidth, height)
	case TabLogs:
		return m.renderLogsTab(contentWidth, height)
	case TabArtisan:
		return m.renderArtisanTab(contentWidth, height)
	case TabMake:
		return m.renderMakeTab(contentWidth, height)
	case TabQuality:
		return m.renderQualityTab(contentWidth, height)
	case TabConfig:
		return m.renderConfigTab(contentWidth, height)
	case TabAbout:
		return m.renderAboutTab(contentWidth, height)
	default:
		return ""
	}
}

// renderProcessesTab renders the Processes tab
func (m Model) renderProcessesTab(width, height int) string {
	if len(m.ProcessOrder) == 0 {
		return centerText("No processes configured", width, height)
	}

	if m.ProcessesTab.View == ProcessesViewOutput {
		return m.renderProcessOutput(width, height)
	}

	return m.renderProcessList(width, height)
}

// renderProcessList renders the process list view
func (m Model) renderProcessList(width, height int) string {
	var lines []string
	lines = append(lines, ui.Styles.Title.Render("Processes"))
	lines = append(lines, "")

	for i, id := range m.ProcessOrder {
		p := m.Processes[id]
		if p == nil {
			continue
		}

		// Status indicator
		var statusStyle lipgloss.Style
		var symbol string
		var actionHint string
		switch p.Status {
		case proc.ProcessStatusRunning:
			statusStyle = ui.Styles.StatusRunning
			symbol = ui.Symbols.Running
			actionHint = "[x]stop [r]restart"
		case proc.ProcessStatusRestarting:
			statusStyle = ui.Styles.StatusRestarting
			symbol = ui.Symbols.Restarting
			actionHint = "restarting..."
		case proc.ProcessStatusFailed:
			statusStyle = ui.Styles.StatusFailed
			symbol = ui.Symbols.Failed
			actionHint = "[s]start"
		default: // Stopped
			statusStyle = ui.Styles.StatusStopped
			symbol = ui.Symbols.Stopped
			actionHint = "[s]start"
		}

		// Selection indicator
		selector := "  "
		if i == m.ProcessesTab.SelectedIndex {
			selector = ui.Styles.StatusRunning.Render(ui.Symbols.Selector + " ")
		}

		// Build the line
		nameStyle := lipgloss.NewStyle().Foreground(ui.Theme.TextDim)
		if i == m.ProcessesTab.SelectedIndex {
			nameStyle = lipgloss.NewStyle().Foreground(ui.Theme.Text).Bold(true)
		}

		name := nameStyle.Render(fmt.Sprintf("%-12s", p.Config.DisplayName))
		hint := ui.Styles.TextMuted.Render(actionHint)

		line := fmt.Sprintf("%s%s %s  %s",
			selector,
			statusStyle.Render(symbol),
			name,
			hint,
		)

		if i == m.ProcessesTab.SelectedIndex {
			line = ui.Styles.Selected.Width(width).Render(line)
		}

		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, ui.Styles.HelpDesc.Render("[R] Restart All  [S] Start All  [X] Stop All  [Enter] View Output"))

	return strings.Join(lines, "\n")
}

// renderProcessOutput renders the output view for the selected process
func (m Model) renderProcessOutput(width, height int) string {
	proc := m.SelectedProcess()
	if proc == nil {
		return "No process selected"
	}

	var lines []string
	title := fmt.Sprintf("Output: %s", proc.Config.DisplayName)
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	output := proc.GetOutput()
	if len(output) == 0 {
		lines = append(lines, ui.Styles.TextMuted.Render("No output yet"))
	} else {
		// Show last N lines based on height
		maxLines := height - 5
		start := len(output) - maxLines - proc.ScrollOffset
		if start < 0 {
			start = 0
		}
		end := len(output) - proc.ScrollOffset
		if end > len(output) {
			end = len(output)
		}

		for i := start; i < end; i++ {
			line := output[i]
			style := lipgloss.NewStyle().Foreground(ui.Theme.Text)
			if line.IsStderr {
				style = style.Foreground(ui.Theme.Error)
			}
			lines = append(lines, style.Render(line.Content))
		}
	}

	lines = append(lines, "")
	lines = append(lines, ui.Styles.HelpDesc.Render("Enter back • c clear • ↑↓ scroll"))

	return strings.Join(lines, "\n")
}

// renderLogsTab renders the Logs tab
func (m Model) renderLogsTab(width, height int) string {
	var lines []string

	// Title with filter info
	filterName := "All"
	if m.LogsTab.FilterLevel != nil {
		filterName = m.LogsTab.FilterLevel.Name() + "+"
	}
	fileName := "All files"
	if m.LogsTab.SelectedFile != nil {
		fileName = *m.LogsTab.SelectedFile
	}

	title := fmt.Sprintf("Logs [%s] [%s]", filterName, fileName)
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	if len(m.LogLines) == 0 {
		lines = append(lines, ui.Styles.TextMuted.Render("No log entries"))
	} else {
		// Filter logs
		filtered := m.filteredLogs()

		if len(filtered) == 0 {
			lines = append(lines, ui.Styles.TextMuted.Render("No matching entries"))
		} else {
			// Show logs based on scroll and height
			maxLines := height - 5
			start := len(filtered) - maxLines - m.LogsTab.ScrollOffset
			if start < 0 {
				start = 0
			}
			end := len(filtered) - m.LogsTab.ScrollOffset
			if end > len(filtered) {
				end = len(filtered)
			}

			for i := start; i < end; i++ {
				log := filtered[i]
				style := m.logLevelStyle(log.Level)
				lines = append(lines, style.Render(log.Content))
			}
		}
	}

	lines = append(lines, "")
	help := "f filter • F file • / search • g top • G bottom"
	if m.LogsTab.InputMode {
		help = fmt.Sprintf("Search: %s█", m.LogsTab.SearchQuery)
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// filteredLogs returns logs matching the current filters
func (m Model) filteredLogs() []LogLine {
	var result []LogLine

	for _, log := range m.LogLines {
		// Filter by file
		if m.LogsTab.SelectedFile != nil && log.File != *m.LogsTab.SelectedFile {
			continue
		}

		// Filter by level
		if m.LogsTab.FilterLevel != nil && log.Level < *m.LogsTab.FilterLevel {
			continue
		}

		// Filter by search query
		if m.LogsTab.SearchQuery != "" {
			if !strings.Contains(strings.ToLower(log.Content), strings.ToLower(m.LogsTab.SearchQuery)) {
				continue
			}
		}

		result = append(result, log)
	}

	return result
}

// logLevelStyle returns the style for a log level
func (m Model) logLevelStyle(level LogLevel) lipgloss.Style {
	var color lipgloss.Color
	switch level {
	case LogLevelDebug:
		color = ui.Theme.LogDebug
	case LogLevelInfo:
		color = ui.Theme.LogInfo
	case LogLevelNotice:
		color = ui.Theme.LogNotice
	case LogLevelWarning:
		color = ui.Theme.LogWarning
	case LogLevelError, LogLevelCritical, LogLevelAlert, LogLevelEmergency:
		color = ui.Theme.LogError
	default:
		color = ui.Theme.Text
	}
	return lipgloss.NewStyle().Foreground(color)
}

// renderArtisanTab renders the Artisan tab
func (m Model) renderArtisanTab(width, height int) string {
	var lines []string

	// Show output view if command is running or has output
	if m.ArtisanTab.RunningCommand != nil || len(m.ArtisanTab.CommandOutput) > 0 {
		return m.renderArtisanOutput(width, height)
	}

	// Build title with search info
	title := "Artisan Commands"
	if m.ArtisanTab.SearchQuery != "" {
		title = fmt.Sprintf("Artisan Commands [Search: %s]", m.ArtisanTab.SearchQuery)
	}
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	commands := m.filteredArtisanCommands()
	if len(commands) == 0 {
		if m.ArtisanTab.SearchQuery != "" {
			lines = append(lines, ui.Styles.TextMuted.Render("No matching commands"))
		} else {
			lines = append(lines, ui.Styles.TextMuted.Render("Discovering commands..."))
		}
	} else {
		maxLines := height - 5
		for i, cmd := range commands {
			if i >= maxLines {
				lines = append(lines, ui.Styles.TextMuted.Render(fmt.Sprintf("... and %d more", len(commands)-maxLines)))
				break
			}

			selector := "  "
			if i == m.ArtisanTab.SelectedCommand {
				selector = ui.Styles.StatusRunning.Render(ui.Symbols.Selector + " ")
			}

			line := fmt.Sprintf("%s%s - %s", selector, cmd.Name, cmd.Description)
			if i == m.ArtisanTab.SelectedCommand {
				line = ui.Styles.Selected.Width(width).Render(line)
			}
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	help := "↑↓ navigate • Enter run • i input args • / search"
	if m.ArtisanTab.SearchMode {
		help = fmt.Sprintf("Search: %s█ (Enter/Esc to close)", m.ArtisanTab.SearchQuery)
	} else if m.ArtisanTab.InputMode {
		help = fmt.Sprintf("Args: %s█ (Enter to run, Esc to cancel)", m.ArtisanTab.InputBuffer)
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderArtisanOutput renders the command output view
func (m Model) renderArtisanOutput(width, height int) string {
	var lines []string

	// Title with running command
	title := "Artisan Output"
	if m.ArtisanTab.RunningCommand != nil {
		title = fmt.Sprintf("Running: php artisan %s", *m.ArtisanTab.RunningCommand)
	}
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	if len(m.ArtisanTab.CommandOutput) == 0 {
		lines = append(lines, ui.Styles.TextMuted.Render("Waiting for output..."))
	} else {
		// Show output lines based on height
		maxLines := height - 5
		output := m.ArtisanTab.CommandOutput
		start := len(output) - maxLines - m.ArtisanTab.OutputScrollOffset
		if start < 0 {
			start = 0
		}
		end := len(output) - m.ArtisanTab.OutputScrollOffset
		if end > len(output) {
			end = len(output)
		}

		for i := start; i < end; i++ {
			line := output[i]
			style := lipgloss.NewStyle().Foreground(ui.Theme.Text)
			if line.IsStderr {
				style = style.Foreground(ui.Theme.Error)
			}
			lines = append(lines, style.Render(line.Content))
		}
	}

	lines = append(lines, "")
	var help string
	if m.ArtisanTab.RunningCommand != nil {
		help = "x stop • ↑↓ scroll"
	} else {
		help = "c clear • Esc back to list"
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderMakeTab renders the Make tab
func (m Model) renderMakeTab(width, height int) string {
	var lines []string

	// Show output view if command is running or has output
	if m.MakeTab.RunningCommand != nil || len(m.MakeTab.CommandOutput) > 0 {
		return m.renderMakeOutput(width, height)
	}

	// Build title with search info
	title := "Make Commands"
	if m.MakeTab.SearchQuery != "" {
		title = fmt.Sprintf("Make Commands [Search: %s]", m.MakeTab.SearchQuery)
	}
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	commands := m.filteredMakeCommands()
	if len(commands) == 0 {
		if m.MakeTab.SearchQuery != "" {
			lines = append(lines, ui.Styles.TextMuted.Render("No matching commands"))
		} else {
			lines = append(lines, ui.Styles.TextMuted.Render("Discovering commands..."))
		}
	} else {
		maxLines := height - 5
		for i, cmd := range commands {
			if i >= maxLines {
				lines = append(lines, ui.Styles.TextMuted.Render(fmt.Sprintf("... and %d more", len(commands)-maxLines)))
				break
			}

			selector := "  "
			if i == m.MakeTab.SelectedCommand {
				selector = ui.Styles.StatusRunning.Render(ui.Symbols.Selector + " ")
			}

			line := fmt.Sprintf("%s%s - %s", selector, cmd.Name, cmd.Description)
			if i == m.MakeTab.SelectedCommand {
				line = ui.Styles.Selected.Width(width).Render(line)
			}
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	help := "↑↓ navigate • Enter/i input name • / search"
	if m.MakeTab.SearchMode {
		help = fmt.Sprintf("Search: %s█ (Enter/Esc to close)", m.MakeTab.SearchQuery)
	} else if m.MakeTab.InputMode {
		help = fmt.Sprintf("Name: %s█ (Enter to create, Esc to cancel)", m.MakeTab.InputBuffer)
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderMakeOutput renders the make command output view
func (m Model) renderMakeOutput(width, height int) string {
	var lines []string

	// Title with running command
	title := "Make Output"
	if m.MakeTab.RunningCommand != nil {
		title = fmt.Sprintf("Running: php artisan %s", *m.MakeTab.RunningCommand)
	}
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	if len(m.MakeTab.CommandOutput) == 0 {
		lines = append(lines, ui.Styles.TextMuted.Render("Waiting for output..."))
	} else {
		// Show output lines based on height
		maxLines := height - 5
		output := m.MakeTab.CommandOutput
		start := len(output) - maxLines - m.MakeTab.OutputScrollOffset
		if start < 0 {
			start = 0
		}
		end := len(output) - m.MakeTab.OutputScrollOffset
		if end > len(output) {
			end = len(output)
		}

		for i := start; i < end; i++ {
			line := output[i]
			style := lipgloss.NewStyle().Foreground(ui.Theme.Text)
			if line.IsStderr {
				style = style.Foreground(ui.Theme.Error)
			}
			lines = append(lines, style.Render(line.Content))
		}
	}

	lines = append(lines, "")
	var help string
	if m.MakeTab.RunningCommand != nil {
		help = "x stop • ↑↓ scroll"
	} else {
		help = "c clear • Esc back to list"
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderQualityTab renders the Quality tab
func (m Model) renderQualityTab(width, height int) string {
	var lines []string

	// Show output view if command is running or has output
	if m.QualityTab.RunningCommand != nil || len(m.QualityTab.CommandOutput) > 0 {
		return m.renderQualityOutput(width, height)
	}

	// Category tabs
	catTabs := []string{}
	for _, cat := range []QualityCategory{QualityCategoryTools, QualityCategoryTesting} {
		label := cat.Name()
		if cat == m.QualityTab.SelectedCategory {
			catTabs = append(catTabs, ui.Styles.TabActive.Render(label))
		} else {
			catTabs = append(catTabs, ui.Styles.TabInactive.Render(label))
		}
	}
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, catTabs...))
	lines = append(lines, "")

	tools := m.currentQualityTools()
	if len(tools) == 0 {
		lines = append(lines, ui.Styles.TextMuted.Render("No tools discovered"))
	} else {
		for i, tool := range tools {
			selector := "  "
			if i == m.QualityTab.SelectedTool {
				selector = ui.Styles.StatusRunning.Render(ui.Symbols.Selector + " ")
			}

			line := fmt.Sprintf("%s%s", selector, tool.DisplayName)
			if i == m.QualityTab.SelectedTool {
				line = ui.Styles.Selected.Width(width).Render(line)
			}
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	help := "←→ category • ↑↓ navigate • Enter run • i input args"
	if m.QualityTab.InputMode {
		help = fmt.Sprintf("Args: %s█ (Enter to run, Esc to cancel)", m.QualityTab.InputBuffer)
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderQualityOutput renders the quality tool output view
func (m Model) renderQualityOutput(width, height int) string {
	var lines []string

	// Title with running command
	title := "Quality Tool Output"
	if m.QualityTab.RunningCommand != nil {
		title = fmt.Sprintf("Running: %s", *m.QualityTab.RunningCommand)
	}
	lines = append(lines, ui.Styles.Title.Render(title))
	lines = append(lines, "")

	if len(m.QualityTab.CommandOutput) == 0 {
		lines = append(lines, ui.Styles.TextMuted.Render("Waiting for output..."))
	} else {
		// Show output lines based on height
		maxLines := height - 5
		output := m.QualityTab.CommandOutput
		start := len(output) - maxLines - m.QualityTab.OutputScrollOffset
		if start < 0 {
			start = 0
		}
		end := len(output) - m.QualityTab.OutputScrollOffset
		if end > len(output) {
			end = len(output)
		}

		for i := start; i < end; i++ {
			line := output[i]
			style := lipgloss.NewStyle().Foreground(ui.Theme.Text)
			if line.IsStderr {
				style = style.Foreground(ui.Theme.Error)
			}
			lines = append(lines, style.Render(line.Content))
		}
	}

	lines = append(lines, "")
	var help string
	if m.QualityTab.RunningCommand != nil {
		help = "x stop • ↑↓ scroll"
	} else {
		help = "c clear • Esc back to list"
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderConfigTab renders the Config tab
func (m Model) renderConfigTab(width, height int) string {
	var lines []string

	// Section tabs
	sectionTabs := []string{}
	for _, section := range []ConfigSection{ConfigSectionDisabled, ConfigSectionLogs, ConfigSectionInfo} {
		label := section.Name()
		if section == m.ConfigTab.SelectedSection {
			sectionTabs = append(sectionTabs, ui.Styles.TabActive.Render(label))
		} else {
			sectionTabs = append(sectionTabs, ui.Styles.TabInactive.Render(label))
		}
	}
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, sectionTabs...))
	lines = append(lines, "")

	// Render section content
	switch m.ConfigTab.SelectedSection {
	case ConfigSectionDisabled:
		lines = append(lines, m.renderDisabledSection()...)
	case ConfigSectionLogs:
		lines = append(lines, m.renderLogsSection()...)
	case ConfigSectionInfo:
		lines = append(lines, m.renderInfoSection()...)
	}

	lines = append(lines, "")

	// Status message or help
	if m.ConfigTab.StatusMessage != "" {
		lines = append(lines, ui.Styles.StatusMessage.Render(m.ConfigTab.StatusMessage))
	}

	// Help line
	help := "←→ section • ↑↓ navigate • Enter/Space toggle • s save"
	if m.ConfigTab.EditMode {
		help = fmt.Sprintf("Value: %s█ (Enter to save, Esc to cancel)", m.ConfigTab.EditBuffer)
	} else if m.ConfigTab.HasChanges {
		help = "←→ section • ↑↓ navigate • Enter/Space toggle • s save (unsaved changes)"
	}
	lines = append(lines, ui.Styles.HelpDesc.Render(help))

	return strings.Join(lines, "\n")
}

// renderDisabledSection renders the disabled processes section
func (m Model) renderDisabledSection() []string {
	var lines []string

	processes := []struct {
		name     string
		disabled bool
	}{
		{"Serve", m.Config != nil && m.Config.Disabled.Serve},
		{"Vite", m.Config != nil && m.Config.Disabled.Vite},
		{"Queue", m.Config != nil && m.Config.Disabled.Queue},
		{"Horizon", m.Config != nil && m.Config.Disabled.Horizon},
		{"Reverb", m.Config != nil && m.Config.Disabled.Reverb},
	}

	for i, proc := range processes {
		selector := "  "
		if i == m.ConfigTab.SelectedItem {
			selector = ui.Styles.StatusRunning.Render(ui.Symbols.Selector + " ")
		}

		checkbox := "[ ]"
		if proc.disabled {
			checkbox = "[✓]"
		}

		status := "enabled"
		statusStyle := ui.Styles.StatusRunning
		if proc.disabled {
			status = "disabled"
			statusStyle = ui.Styles.StatusStopped
		}

		line := fmt.Sprintf("%s%s %s  %s", selector, checkbox, proc.name, statusStyle.Render(status))
		if i == m.ConfigTab.SelectedItem {
			line = ui.Styles.Selected.Width(40).Render(line)
		}
		lines = append(lines, line)
	}

	return lines
}

// renderLogsSection renders the log settings section
func (m Model) renderLogsSection() []string {
	var lines []string

	maxLines := m.MaxLogLines
	if m.Config != nil && m.Config.Logs.MaxLines != nil {
		maxLines = *m.Config.Logs.MaxLines
	}

	selector := "  "
	if m.ConfigTab.SelectedItem == 0 {
		selector = ui.Styles.StatusRunning.Render(ui.Symbols.Selector + " ")
	}

	var value string
	if m.ConfigTab.EditMode && m.ConfigTab.SelectedItem == 0 {
		value = fmt.Sprintf("%s█", m.ConfigTab.EditBuffer)
	} else {
		value = fmt.Sprintf("%d", maxLines)
	}

	line := fmt.Sprintf("%sMax Log Lines: %s", selector, value)
	if m.ConfigTab.SelectedItem == 0 {
		line = ui.Styles.Selected.Width(40).Render(line)
	}
	lines = append(lines, line)

	return lines
}

// renderInfoSection renders the config file info section
func (m Model) renderInfoSection() []string {
	var lines []string

	configPath := fmt.Sprintf("%s/.laravisor.json", m.WorkingDir)

	lines = append(lines, fmt.Sprintf("Config file: %s", configPath))
	lines = append(lines, "")

	if m.ConfigTab.HasChanges {
		lines = append(lines, ui.Styles.StatusRestarting.Render("• Unsaved changes"))
	} else {
		lines = append(lines, ui.Styles.TextMuted.Render("• No unsaved changes"))
	}

	return lines
}

// renderAboutTab renders the About tab
func (m Model) renderAboutTab(width, height int) string {
	var lines []string

	lines = append(lines, ui.Styles.Title.Render("Laravisor"))
	lines = append(lines, "")
	lines = append(lines, "A TUI for managing Laravel development processes")
	lines = append(lines, "")
	lines = append(lines, ui.Styles.Title.Render("Keyboard Shortcuts"))
	lines = append(lines, "")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("1-6")+"   Switch tabs")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("Tab")+"   Next tab")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("?")+"     About/Help")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("q")+"     Quit")
	lines = append(lines, "")
	lines = append(lines, ui.Styles.Title.Render("Processes Tab"))
	lines = append(lines, "")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("↑↓")+"    Navigate processes")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("Enter")+" View output")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("s")+"     Start process")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("x")+"     Stop process")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("r")+"     Restart process")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("R")+"     Restart all")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("c")+"     Clear output")
	lines = append(lines, "")
	lines = append(lines, ui.Styles.Title.Render("Logs Tab"))
	lines = append(lines, "")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("f")+"     Filter level")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("F")+"     Cycle file")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("/")+"     Search")
	lines = append(lines, "  "+ui.Styles.HelpKey.Render("g/G")+"   Top/Bottom")

	return strings.Join(lines, "\n")
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	left := m.WorkingDir
	right := "Ctrl+C quit"

	if m.StatusMessage != "" {
		left = ui.Styles.StatusMessage.Render(m.StatusMessage)
	}

	// Calculate padding
	padding := m.Width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 0 {
		padding = 0
	}

	statusBar := left + strings.Repeat(" ", padding) + right
	return ui.Styles.StatusBar.Width(m.Width).Render(statusBar)
}

// centerText centers text in the given dimensions
func centerText(text string, width, height int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}

// TextMuted style for external access
var TextMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
