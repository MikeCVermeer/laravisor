package config

// Config represents the .laravisor.json configuration file
type Config struct {
	Disabled  DisabledConfig            `json:"disabled"`
	Overrides map[string]OverrideConfig `json:"overrides"`
	Custom    []CustomProcess           `json:"custom"`
	Quality   QualityConfig             `json:"quality"`
	Logs      LogConfig                 `json:"logs"`
	Artisan   ArtisanConfig             `json:"artisan"`
	Make      MakeConfig                `json:"make"`
}

// DisabledConfig specifies which default processes to disable
type DisabledConfig struct {
	Serve   bool `json:"serve"`
	Vite    bool `json:"vite"`
	Queue   bool `json:"queue"`
	Horizon bool `json:"horizon"`
	Reverb  bool `json:"reverb"`
}

// OverrideConfig allows customizing default process settings
type OverrideConfig struct {
	Command       *string           `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	WorkingDir    *string           `json:"working_dir,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	RestartPolicy *string           `json:"restart_policy,omitempty"`
}

// CustomProcess defines a user-defined process
type CustomProcess struct {
	Name          string            `json:"name"`
	DisplayName   string            `json:"display_name"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Hotkey        *string           `json:"hotkey,omitempty"`
	Enabled       bool              `json:"enabled"`
	WorkingDir    *string           `json:"working_dir,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	RestartPolicy *string           `json:"restart_policy,omitempty"`
}

// QualityConfig configures quality tools
type QualityConfig struct {
	DisabledTools []string          `json:"disabled_tools"`
	CustomTools   []CustomTool      `json:"custom_tools"`
	DefaultArgs   map[string]string `json:"default_args"`
}

// CustomTool defines a user-defined quality/testing tool
type CustomTool struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Category    string   `json:"category"` // "quality" or "testing"
}

// LogConfig configures log watching
type LogConfig struct {
	MaxLines      *int     `json:"max_lines,omitempty"`
	Files         []string `json:"files,omitempty"`
	DefaultFilter *string  `json:"default_filter,omitempty"`
}

// ArtisanConfig configures artisan commands
type ArtisanConfig struct {
	Favorites []string `json:"favorites"`
}

// MakeConfig configures make commands
type MakeConfig struct {
	Favorites []string `json:"favorites"`
}

// NewDefaultConfig creates a config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Disabled:  DisabledConfig{},
		Overrides: make(map[string]OverrideConfig),
		Custom:    make([]CustomProcess, 0),
		Quality: QualityConfig{
			DisabledTools: make([]string, 0),
			CustomTools:   make([]CustomTool, 0),
			DefaultArgs:   make(map[string]string),
		},
		Logs: LogConfig{},
		Artisan: ArtisanConfig{
			Favorites: make([]string, 0),
		},
		Make: MakeConfig{
			Favorites: make([]string, 0),
		},
	}
}
