package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mikecvermeer/laravisor/internal/proc"
)

// PackageJSON represents the package.json file
type PackageJSON struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// HasDependency checks if a dependency exists
func (p *PackageJSON) HasDependency(name string) bool {
	if p == nil {
		return false
	}
	if _, ok := p.Dependencies[name]; ok {
		return true
	}
	if _, ok := p.DevDependencies[name]; ok {
		return true
	}
	return false
}

// HasScript checks if a script exists
func (p *PackageJSON) HasScript(name string) bool {
	if p == nil {
		return false
	}
	_, ok := p.Scripts[name]
	return ok
}

// parsePackageJSON parses the package.json file
func parsePackageJSON(workingDir string) (*PackageJSON, error) {
	packagePath := filepath.Join(workingDir, "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

// detectPackageManager detects which package manager is used
func detectPackageManager(workingDir string) string {
	// Check for lock files in order of preference
	lockFiles := []struct {
		file    string
		manager string
	}{
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}

	for _, lf := range lockFiles {
		lockPath := filepath.Join(workingDir, lf.file)
		if _, err := os.Stat(lockPath); err == nil {
			return lf.manager
		}
	}

	// Default to npm
	return "npm"
}

// discoverVite checks for Vite and returns a process config if found
func discoverVite(workingDir string) *proc.ProcessConfig {
	pkg, err := parsePackageJSON(workingDir)
	if err != nil {
		return nil
	}

	// Check for vite or laravel-vite-plugin
	hasVite := pkg.HasDependency("vite") || pkg.HasDependency("laravel-vite-plugin")
	hasDevScript := pkg.HasScript("dev")

	if !hasVite || !hasDevScript {
		return nil
	}

	// Detect package manager and build command
	manager := detectPackageManager(workingDir)
	var args []string

	switch manager {
	case "yarn":
		args = []string{"dev"}
	case "bun":
		args = []string{"run", "dev"}
	case "pnpm":
		args = []string{"run", "dev"}
	default: // npm
		args = []string{"run", "dev"}
	}

	return &proc.ProcessConfig{
		ID:          proc.ProcessID("vite"),
		Kind:        proc.ProcessKindVite,
		DisplayName: "Vite",
		Command:     manager,
		Args:        args,
		WorkingDir:  workingDir,
	}
}

// discoverDevTools discovers quality and testing tools from dependencies
func discoverDevTools(composer *ComposerJSON, pkg *PackageJSON, packageManager string) ([]QualityTool, []QualityTool) {
	var qualityTools []QualityTool
	var testingTools []QualityTool

	if composer == nil {
		return qualityTools, testingTools
	}

	// PHP Quality tools
	if composer.HasDependency("phpstan/phpstan") || composer.HasDependency("larastan/larastan") {
		qualityTools = append(qualityTools, QualityTool{
			DisplayName: "PHPStan",
			Command:     "./vendor/bin/phpstan",
			Args:        []string{"analyse", "--ansi"},
		})
	}

	if composer.HasDependency("laravel/pint") {
		qualityTools = append(qualityTools, QualityTool{
			DisplayName: "Pint",
			Command:     "./vendor/bin/pint",
			Args:        []string{"--ansi"},
		})
	}

	if composer.HasDependency("friendsofphp/php-cs-fixer") {
		qualityTools = append(qualityTools, QualityTool{
			DisplayName: "PHP CS Fixer",
			Command:     "./vendor/bin/php-cs-fixer",
			Args:        []string{"fix", "--ansi"},
		})
	}

	if composer.HasDependency("rector/rector") || composer.HasDependency("rectorphp/rector") {
		qualityTools = append(qualityTools, QualityTool{
			DisplayName: "Rector",
			Command:     "./vendor/bin/rector",
			Args:        []string{"--ansi"},
		})
	}

	if composer.HasDependency("squizlabs/php_codesniffer") {
		qualityTools = append(qualityTools, QualityTool{
			DisplayName: "PHP_CodeSniffer",
			Command:     "./vendor/bin/phpcs",
			Args:        []string{"--colors"},
		})
	}

	if composer.HasDependency("vimeo/psalm") {
		qualityTools = append(qualityTools, QualityTool{
			DisplayName: "Psalm",
			Command:     "./vendor/bin/psalm",
			Args:        []string{"--output-format=console"},
		})
	}

	// NPM/Yarn/PNPM/Bun scripts from package.json
	if pkg != nil {
		// Quality scripts
		qualityScripts := []struct {
			script  string
			display string
		}{
			{"lint", "Lint"},
			{"lint:fix", "Lint Fix"},
			{"format", "Format"},
			{"format:check", "Format Check"},
			{"types", "Type Check"},
			{"typecheck", "Type Check"},
			{"type-check", "Type Check"},
			{"check", "Check"},
		}

		for _, qs := range qualityScripts {
			if pkg.HasScript(qs.script) {
				args := buildScriptArgs(packageManager, qs.script)
				qualityTools = append(qualityTools, QualityTool{
					DisplayName: qs.display,
					Command:     packageManager,
					Args:        args,
				})
			}
		}

		// Testing scripts
		testScripts := []struct {
			script  string
			display string
		}{
			{"test", "JS Test"},
			{"test:unit", "JS Unit Test"},
			{"test:e2e", "JS E2E Test"},
			{"test:coverage", "JS Test Coverage"},
		}

		for _, ts := range testScripts {
			if pkg.HasScript(ts.script) {
				args := buildScriptArgs(packageManager, ts.script)
				testingTools = append(testingTools, QualityTool{
					DisplayName: ts.display,
					Command:     packageManager,
					Args:        args,
				})
			}
		}
	}

	// PHP Testing tools
	if composer.HasDependency("pestphp/pest") {
		testingTools = append(testingTools, QualityTool{
			DisplayName: "Pest",
			Command:     "./vendor/bin/pest",
			Args:        []string{"--colors=always"},
		})
		testingTools = append(testingTools, QualityTool{
			DisplayName: "Pest Coverage",
			Command:     "./vendor/bin/pest",
			Args:        []string{"--coverage", "--colors=always"},
		})
	} else if composer.HasDependency("phpunit/phpunit") {
		testingTools = append(testingTools, QualityTool{
			DisplayName: "PHPUnit",
			Command:     "./vendor/bin/phpunit",
			Args:        []string{"--colors=always"},
		})
	}

	if composer.HasDependency("brianium/paratest") {
		testingTools = append(testingTools, QualityTool{
			DisplayName: "Paratest",
			Command:     "./vendor/bin/paratest",
			Args:        []string{"--colors"},
		})
	}

	return qualityTools, testingTools
}

// buildScriptArgs builds the args for running a package.json script
func buildScriptArgs(packageManager, script string) []string {
	if packageManager == "yarn" {
		return []string{script}
	}
	return []string{"run", script}
}
