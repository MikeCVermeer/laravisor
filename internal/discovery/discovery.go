package discovery

import (
	"github.com/mikecvermeer/laravisor/internal/config"
	"github.com/mikecvermeer/laravisor/internal/proc"
)

// DiscoveryResult contains all discovered services and tools
type DiscoveryResult struct {
	Processes       []proc.ProcessConfig
	ArtisanCommands []ArtisanCommand
	MakeCommands    []ArtisanCommand
	QualityTools    []QualityTool
	TestingTools    []QualityTool
}

// QualityTool represents a discovered quality or testing tool
type QualityTool struct {
	DisplayName string
	Command     string
	Args        []string
}

// ArtisanCommand represents a discovered artisan command
type ArtisanCommand struct {
	Name        string
	Description string
	Arguments   []CommandArgument
	Options     []CommandOption
}

// CommandArgument represents a command argument
type CommandArgument struct {
	Name        string
	IsRequired  bool
	Description string
}

// CommandOption represents a command option/flag
type CommandOption struct {
	Name        string
	Shortcut    string
	Description string
}

// Discover discovers all Laravel services in the working directory
func Discover(workingDir string, cfg *config.Config) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		Processes:       make([]proc.ProcessConfig, 0),
		ArtisanCommands: make([]ArtisanCommand, 0),
		MakeCommands:    make([]ArtisanCommand, 0),
		QualityTools:    make([]QualityTool, 0),
		TestingTools:    make([]QualityTool, 0),
	}

	// Parse composer.json
	composer, err := parseComposerJSON(workingDir)
	if err != nil {
		return nil, err
	}

	// Check for Laravel framework
	if !composer.HasDependency("laravel/framework") {
		return nil, &NotLaravelError{}
	}

	// Helper to check if a process is disabled
	isDisabled := func(name string) bool {
		if cfg == nil {
			return false
		}
		switch name {
		case "serve":
			return cfg.Disabled.Serve
		case "vite":
			return cfg.Disabled.Vite
		case "queue":
			return cfg.Disabled.Queue
		case "horizon":
			return cfg.Disabled.Horizon
		case "reverb":
			return cfg.Disabled.Reverb
		}
		return false
	}

	// Add serve (unless Herd is installed or disabled)
	if !isHerdInstalled() && !isDisabled("serve") {
		serveConfig := proc.ProcessConfig{
			ID:          proc.ProcessID("serve"),
			Kind:        proc.ProcessKindServe,
			DisplayName: "Serve",
			Command:     "php",
			Args:        []string{"artisan", "serve"},
			WorkingDir:  workingDir,
		}
		serveConfig = applyOverrides(serveConfig, "serve", cfg, workingDir)
		result.Processes = append(result.Processes, serveConfig)
	}

	// Check for Horizon (skip queue if Horizon is available)
	hasHorizon := composer.HasDependency("laravel/horizon")
	if hasHorizon && !isDisabled("horizon") {
		horizonConfig := proc.ProcessConfig{
			ID:          proc.ProcessID("horizon"),
			Kind:        proc.ProcessKindHorizon,
			DisplayName: "Horizon",
			Command:     "php",
			Args:        []string{"artisan", "horizon"},
			WorkingDir:  workingDir,
		}
		horizonConfig = applyOverrides(horizonConfig, "horizon", cfg, workingDir)
		result.Processes = append(result.Processes, horizonConfig)
	} else if !isDisabled("queue") {
		queueConfig := proc.ProcessConfig{
			ID:          proc.ProcessID("queue"),
			Kind:        proc.ProcessKindQueue,
			DisplayName: "Queue",
			Command:     "php",
			Args:        []string{"artisan", "queue:work", "--tries=3"},
			WorkingDir:  workingDir,
		}
		queueConfig = applyOverrides(queueConfig, "queue", cfg, workingDir)
		result.Processes = append(result.Processes, queueConfig)
	}

	// Check for Reverb
	if composer.HasDependency("laravel/reverb") && !isDisabled("reverb") {
		reverbConfig := proc.ProcessConfig{
			ID:          proc.ProcessID("reverb"),
			Kind:        proc.ProcessKindReverb,
			DisplayName: "Reverb",
			Command:     "php",
			Args:        []string{"artisan", "reverb:start"},
			WorkingDir:  workingDir,
		}
		reverbConfig = applyOverrides(reverbConfig, "reverb", cfg, workingDir)
		result.Processes = append(result.Processes, reverbConfig)
	}

	// Check for Vite
	if !isDisabled("vite") {
		viteConfig := discoverVite(workingDir)
		if viteConfig != nil {
			*viteConfig = applyOverrides(*viteConfig, "vite", cfg, workingDir)
			result.Processes = append(result.Processes, *viteConfig)
		}
	}

	// Add custom processes from config
	if cfg != nil {
		for _, custom := range cfg.Custom {
			if !custom.Enabled {
				continue
			}
			customWorkingDir := workingDir
			if custom.WorkingDir != nil {
				customWorkingDir = *custom.WorkingDir
			}

			var hotkey *rune
			if custom.Hotkey != nil && len(*custom.Hotkey) > 0 {
				r := []rune(*custom.Hotkey)[0]
				hotkey = &r
			}

			customConfig := proc.ProcessConfig{
				ID:          proc.ProcessID(custom.Name),
				Kind:        proc.ProcessKindCustom,
				DisplayName: custom.DisplayName,
				Command:     custom.Command,
				Args:        custom.Args,
				WorkingDir:  customWorkingDir,
				Env:         custom.Env,
				Hotkey:      hotkey,
			}
			result.Processes = append(result.Processes, customConfig)
		}
	}

	// Discover artisan commands
	result.ArtisanCommands = discoverArtisanCommands(workingDir)
	result.MakeCommands = discoverMakeCommands(workingDir)

	// Discover quality and testing tools
	packageJSON, _ := parsePackageJSON(workingDir)
	packageManager := detectPackageManager(workingDir)
	quality, testing := discoverDevTools(composer, packageJSON, packageManager)

	// Apply quality config (filter disabled, add custom, merge args)
	if cfg != nil {
		quality = applyQualityConfig(quality, cfg)
		testing = applyQualityConfig(testing, cfg)
	}

	result.QualityTools = quality
	result.TestingTools = testing

	// Always add artisan test to testing tools
	result.TestingTools = append(result.TestingTools, QualityTool{
		DisplayName: "Artisan Test",
		Command:     "php",
		Args:        []string{"artisan", "test", "--ansi"},
	})

	return result, nil
}

// applyOverrides applies config overrides to a process config
func applyOverrides(config proc.ProcessConfig, kind string, cfg *config.Config, workingDir string) proc.ProcessConfig {
	if cfg == nil {
		return config
	}

	override, exists := cfg.Overrides[kind]
	if !exists {
		return config
	}

	if override.Command != nil {
		config.Command = *override.Command
	}
	if len(override.Args) > 0 {
		config.Args = override.Args
	}
	if override.WorkingDir != nil {
		config.WorkingDir = *override.WorkingDir
	}
	if len(override.Env) > 0 {
		config.Env = override.Env
	}
	if override.RestartPolicy != nil {
		switch *override.RestartPolicy {
		case "never":
			config.RestartPolicy = proc.RestartPolicyNever
		case "on_failure":
			config.RestartPolicy = proc.RestartPolicyOnFailure
		case "always":
			config.RestartPolicy = proc.RestartPolicyAlways
		}
	}

	return config
}

// applyQualityConfig filters disabled tools and adds custom tools
func applyQualityConfig(tools []QualityTool, cfg *config.Config) []QualityTool {
	// Filter disabled tools
	filtered := make([]QualityTool, 0)
	for _, tool := range tools {
		disabled := false
		for _, name := range cfg.Quality.DisabledTools {
			if name == tool.DisplayName {
				disabled = true
				break
			}
		}
		if !disabled {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// NotLaravelError indicates the project is not a Laravel project
type NotLaravelError struct{}

func (e *NotLaravelError) Error() string {
	return "not a Laravel project (laravel/framework not found in composer.json)"
}
