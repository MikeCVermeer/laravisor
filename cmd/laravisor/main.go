package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikecvermeer/laravisor/internal/app"
	"github.com/mikecvermeer/laravisor/internal/config"
	"github.com/mikecvermeer/laravisor/internal/discovery"
	"github.com/mikecvermeer/laravisor/internal/log"
)

func main() {
	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Create the application model
	m := app.New(workingDir)

	// Load configuration
	cfg, err := config.Load(workingDir)
	if err != nil {
		m.ConfigError = fmt.Sprintf("Config error: %v", err)
	} else {
		m.SetConfig(cfg)
	}

	// Discover services
	result, err := discovery.Discover(workingDir, cfg)
	if err != nil {
		// Not a fatal error - might not be in a Laravel project
		m.SetStatus(fmt.Sprintf("Discovery: %v", err))
	} else {
		// Register discovered processes
		for _, procCfg := range result.Processes {
			m.RegisterProcess(procCfg)
		}

		// Set discovered commands and tools
		m.ArtisanTab.Commands = convertArtisanCommands(result.ArtisanCommands)
		m.MakeTab.Commands = convertArtisanCommands(result.MakeCommands)
		m.QualityTab.QualityTools = convertQualityTools(result.QualityTools)
		m.QualityTab.TestingTools = convertQualityTools(result.TestingTools)
	}

	// Start log watcher if log directory exists
	logDir := log.FindLogDir(workingDir)
	if logDir != "" {
		watcher, err := log.NewWatcher(logDir)
		if err == nil {
			// Add additional files from config
			if cfg != nil && len(cfg.Logs.Files) > 0 {
				watcher.AddAdditionalFiles(cfg.Logs.Files)
			}
			if err := watcher.Start(); err == nil {
				m.LogWatcher = watcher
			}
		}
	}

	// Create and run the Bubble Tea program
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	if m.LogWatcher != nil {
		m.LogWatcher.Stop()
	}
}

// convertArtisanCommands converts discovery commands to app commands
func convertArtisanCommands(cmds []discovery.ArtisanCommand) []app.ArtisanCommand {
	result := make([]app.ArtisanCommand, len(cmds))
	for i, cmd := range cmds {
		args := make([]string, len(cmd.Arguments))
		for j, arg := range cmd.Arguments {
			args[j] = arg.Name
		}
		opts := make([]string, len(cmd.Options))
		for j, opt := range cmd.Options {
			opts[j] = opt.Name
		}
		result[i] = app.ArtisanCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
			Arguments:   args,
			Options:     opts,
		}
	}
	return result
}

// convertQualityTools converts discovery tools to app tools
func convertQualityTools(tools []discovery.QualityTool) []app.QualityTool {
	result := make([]app.QualityTool, len(tools))
	for i, tool := range tools {
		result[i] = app.QualityTool{
			DisplayName: tool.DisplayName,
			Command:     tool.Command,
			Args:        tool.Args,
		}
	}
	return result
}
