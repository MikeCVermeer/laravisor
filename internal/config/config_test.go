package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewDefaultConfigInitializesCollections(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	if cfg == nil {
		t.Fatal("NewDefaultConfig() returned nil")
	}

	if cfg.Overrides == nil {
		t.Error("Overrides is nil, want initialized map")
	}
	if cfg.Custom == nil {
		t.Error("Custom is nil, want initialized slice")
	}
	if cfg.Quality.DisabledTools == nil {
		t.Error("Quality.DisabledTools is nil, want initialized slice")
	}
	if cfg.Quality.CustomTools == nil {
		t.Error("Quality.CustomTools is nil, want initialized slice")
	}
	if cfg.Quality.DefaultArgs == nil {
		t.Error("Quality.DefaultArgs is nil, want initialized map")
	}
	if cfg.Artisan.Favorites == nil {
		t.Error("Artisan.Favorites is nil, want initialized slice")
	}
	if cfg.Make.Favorites == nil {
		t.Error("Make.Favorites is nil, want initialized slice")
	}
}

func TestLoadReturnsDefaultWhenConfigDoesNotExist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	want := NewDefaultConfig()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want default %#v", got, want)
	}
	if Exists(dir) {
		t.Fatal("Exists() = true after loading a missing config, want false")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	maxLines := 250
	defaultFilter := "error"
	workingDir := "services/api"
	restartPolicy := "on_failure"
	hotkey := "q"
	overrideCommand := "php"

	want := &Config{
		Disabled: DisabledConfig{Vite: true, Reverb: true},
		Overrides: map[string]OverrideConfig{
			"queue": {
				Command:       &overrideCommand,
				Args:          []string{"artisan", "queue:work", "--tries=3"},
				WorkingDir:    &workingDir,
				Env:           map[string]string{"QUEUE_CONNECTION": "redis"},
				RestartPolicy: &restartPolicy,
			},
		},
		Custom: []CustomProcess{
			{
				Name:          "scheduler",
				DisplayName:   "Scheduler",
				Command:       "php",
				Args:          []string{"artisan", "schedule:work"},
				Hotkey:        &hotkey,
				Enabled:       true,
				WorkingDir:    &workingDir,
				Env:           map[string]string{"APP_ENV": "local"},
				RestartPolicy: &restartPolicy,
			},
		},
		Quality: QualityConfig{
			DisabledTools: []string{"pint"},
			CustomTools: []CustomTool{
				{
					Name:        "architecture",
					DisplayName: "Architecture tests",
					Command:     "php",
					Args:        []string{"artisan", "test", "--testsuite=Architecture"},
					Category:    "testing",
				},
			},
			DefaultArgs: map[string]string{"phpstan": "--memory-limit=1G"},
		},
		Logs: LogConfig{
			MaxLines:      &maxLines,
			Files:         []string{"storage/logs/worker.log"},
			DefaultFilter: &defaultFilter,
		},
		Artisan: ArtisanConfig{Favorites: []string{"migrate", "queue:restart"}},
		Make:    MakeConfig{Favorites: []string{"test", "lint"}},
	}

	if err := Save(dir, want); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	if !Exists(dir) {
		t.Fatal("Exists() = false after Save(), want true")
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip config = %#v, want %#v", got, want)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(`{"disabled":`), 0o600); err != nil {
		t.Fatalf("write malformed config: %v", err)
	}

	if _, err := Load(dir); err == nil {
		t.Fatal("Load() returned nil error for malformed JSON")
	}
}
