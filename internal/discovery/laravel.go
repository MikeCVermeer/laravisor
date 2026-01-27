package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ComposerJSON represents the composer.json file
type ComposerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

// HasDependency checks if a dependency exists in require or require-dev
func (c *ComposerJSON) HasDependency(name string) bool {
	if _, ok := c.Require[name]; ok {
		return true
	}
	if _, ok := c.RequireDev[name]; ok {
		return true
	}
	return false
}

// parseComposerJSON parses the composer.json file
func parseComposerJSON(workingDir string) (*ComposerJSON, error) {
	composerPath := filepath.Join(workingDir, "composer.json")
	data, err := os.ReadFile(composerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ComposerNotFoundError{}
		}
		return nil, err
	}

	var composer ComposerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return nil, err
	}

	return &composer, nil
}

// ComposerNotFoundError indicates composer.json was not found
type ComposerNotFoundError struct{}

func (e *ComposerNotFoundError) Error() string {
	return "composer.json not found. Is this a Laravel project?"
}

// isHerdInstalled checks if Laravel Herd is installed
func isHerdInstalled() bool {
	// Check for Herd application on macOS
	if _, err := os.Stat("/Applications/Herd.app"); err == nil {
		return true
	}

	// Check for Herd config directory
	home := os.Getenv("HOME")
	if home != "" {
		herdConfig := filepath.Join(home, "Library/Application Support/Herd")
		if _, err := os.Stat(herdConfig); err == nil {
			return true
		}
	}

	return false
}
