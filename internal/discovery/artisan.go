package discovery

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
)

// artisanListOutput represents the JSON output from artisan list
type artisanListOutput struct {
	Commands []artisanListCommand `json:"commands"`
}

type artisanListCommand struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Definition  *artisanCommandDefinition `json:"definition"`
}

type artisanCommandDefinition struct {
	Arguments map[string]artisanArgument `json:"arguments"`
	Options   map[string]artisanOption   `json:"options"`
}

type artisanArgument struct {
	Name        string `json:"name"`
	IsRequired  bool   `json:"is_required"`
	Description string `json:"description"`
}

type artisanOption struct {
	Name        string `json:"name"`
	Shortcut    string `json:"shortcut"`
	Description string `json:"description"`
}

// discoverArtisanCommands discovers all artisan commands (excluding make:*)
func discoverArtisanCommands(workingDir string) []ArtisanCommand {
	cmd := exec.Command("php", "artisan", "list", "--format=json")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return defaultArtisanCommands()
	}

	var parsed artisanListOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return defaultArtisanCommands()
	}

	var commands []ArtisanCommand
	for _, c := range parsed.Commands {
		// Skip make:*, internal, help, and list commands
		if strings.HasPrefix(c.Name, "make:") ||
			strings.HasPrefix(c.Name, "_") ||
			c.Name == "help" ||
			c.Name == "list" ||
			c.Name == "completion" {
			continue
		}

		cmd := ArtisanCommand{
			Name:        c.Name,
			Description: c.Description,
			Arguments:   make([]CommandArgument, 0),
			Options:     make([]CommandOption, 0),
		}

		if c.Definition != nil {
			for _, arg := range c.Definition.Arguments {
				if arg.Name == "command" {
					continue
				}
				cmd.Arguments = append(cmd.Arguments, CommandArgument{
					Name:        arg.Name,
					IsRequired:  arg.IsRequired,
					Description: arg.Description,
				})
			}
			sort.Slice(cmd.Arguments, func(i, j int) bool {
				return cmd.Arguments[i].Name < cmd.Arguments[j].Name
			})

			for _, opt := range c.Definition.Options {
				// Skip common global options
				if isGlobalOption(opt.Name) {
					continue
				}
				cmd.Options = append(cmd.Options, CommandOption{
					Name:        opt.Name,
					Shortcut:    opt.Shortcut,
					Description: opt.Description,
				})
			}
			sort.Slice(cmd.Options, func(i, j int) bool {
				return cmd.Options[i].Name < cmd.Options[j].Name
			})
		}

		commands = append(commands, cmd)
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	if len(commands) == 0 {
		return defaultArtisanCommands()
	}

	return commands
}

// discoverMakeCommands discovers make:* artisan commands
func discoverMakeCommands(workingDir string) []ArtisanCommand {
	cmd := exec.Command("php", "artisan", "list", "make", "--format=json")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return defaultMakeCommands()
	}

	var parsed artisanListOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return defaultMakeCommands()
	}

	var commands []ArtisanCommand
	for _, c := range parsed.Commands {
		if !strings.HasPrefix(c.Name, "make:") {
			continue
		}

		cmd := ArtisanCommand{
			Name:        c.Name,
			Description: c.Description,
			Arguments:   make([]CommandArgument, 0),
			Options:     make([]CommandOption, 0),
		}

		if c.Definition != nil {
			for _, arg := range c.Definition.Arguments {
				if arg.Name == "command" {
					continue
				}
				cmd.Arguments = append(cmd.Arguments, CommandArgument{
					Name:        arg.Name,
					IsRequired:  arg.IsRequired,
					Description: arg.Description,
				})
			}
			sort.Slice(cmd.Arguments, func(i, j int) bool {
				return cmd.Arguments[i].Name < cmd.Arguments[j].Name
			})

			for _, opt := range c.Definition.Options {
				if isGlobalOption(opt.Name) {
					continue
				}
				cmd.Options = append(cmd.Options, CommandOption{
					Name:        opt.Name,
					Shortcut:    opt.Shortcut,
					Description: opt.Description,
				})
			}
			sort.Slice(cmd.Options, func(i, j int) bool {
				return cmd.Options[i].Name < cmd.Options[j].Name
			})
		}

		commands = append(commands, cmd)
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	if len(commands) == 0 {
		return defaultMakeCommands()
	}

	return commands
}

// isGlobalOption checks if an option is a global artisan option
func isGlobalOption(name string) bool {
	globalOptions := map[string]bool{
		"--help":           true,
		"--quiet":          true,
		"--silent":         true,
		"--verbose":        true,
		"--version":        true,
		"--ansi":           true,
		"--no-ansi":        true,
		"--no-interaction": true,
		"--env":            true,
	}
	return globalOptions[name]
}

// defaultArtisanCommands returns default commands if discovery fails
func defaultArtisanCommands() []ArtisanCommand {
	defaults := []struct {
		name string
		desc string
	}{
		{"about", "Display basic information about your application"},
		{"clear-compiled", "Remove the compiled class file"},
		{"db", "Start a new database CLI session"},
		{"down", "Put the application into maintenance mode"},
		{"env", "Display the current framework environment"},
		{"inspire", "Display an inspiring quote"},
		{"migrate", "Run the database migrations"},
		{"optimize", "Cache framework bootstrap, configuration, and metadata"},
		{"optimize:clear", "Remove the cached bootstrap files"},
		{"serve", "Serve the application on the PHP development server"},
		{"tinker", "Interact with your application"},
		{"up", "Bring the application out of maintenance mode"},
		{"cache:clear", "Flush the application cache"},
		{"config:cache", "Create a cache file for faster configuration loading"},
		{"config:clear", "Remove the configuration cache file"},
		{"db:seed", "Seed the database with records"},
		{"event:cache", "Discover and cache the application's events and listeners"},
		{"event:clear", "Clear all cached events and listeners"},
		{"key:generate", "Set the application key"},
		{"migrate:fresh", "Drop all tables and re-run all migrations"},
		{"migrate:refresh", "Reset and re-run all migrations"},
		{"migrate:reset", "Rollback all database migrations"},
		{"migrate:rollback", "Rollback the last database migration"},
		{"migrate:status", "Show the status of each migration"},
		{"queue:clear", "Delete all of the jobs from the specified queue"},
		{"queue:failed", "List all of the failed queue jobs"},
		{"queue:flush", "Flush all of the failed queue jobs"},
		{"queue:restart", "Restart queue worker daemons after their current job"},
		{"queue:retry", "Retry a failed queue job"},
		{"route:cache", "Create a route cache file for faster route registration"},
		{"route:clear", "Remove the route cache file"},
		{"route:list", "List all registered routes"},
		{"schedule:list", "List all scheduled tasks"},
		{"storage:link", "Create the symbolic links configured for the application"},
		{"view:cache", "Compile all of the application's Blade templates"},
		{"view:clear", "Clear all compiled view files"},
	}

	commands := make([]ArtisanCommand, len(defaults))
	for i, d := range defaults {
		commands[i] = ArtisanCommand{
			Name:        d.name,
			Description: d.desc,
			Arguments:   []CommandArgument{},
			Options:     []CommandOption{},
		}
	}
	return commands
}

// defaultMakeCommands returns default make commands if discovery fails
func defaultMakeCommands() []ArtisanCommand {
	defaults := []struct {
		name string
		desc string
	}{
		{"make:controller", "Create a new controller class"},
		{"make:model", "Create a new Eloquent model class"},
		{"make:migration", "Create a new migration file"},
		{"make:seeder", "Create a new seeder class"},
		{"make:factory", "Create a new model factory"},
		{"make:request", "Create a new form request class"},
		{"make:resource", "Create a new resource"},
		{"make:event", "Create a new event class"},
		{"make:listener", "Create a new event listener class"},
		{"make:job", "Create a new job class"},
		{"make:command", "Create a new Artisan command"},
		{"make:mail", "Create a new email class"},
		{"make:notification", "Create a new notification class"},
		{"make:policy", "Create a new policy class"},
		{"make:rule", "Create a new validation rule"},
	}

	commands := make([]ArtisanCommand, len(defaults))
	for i, d := range defaults {
		commands[i] = ArtisanCommand{
			Name:        d.name,
			Description: d.desc,
			Arguments: []CommandArgument{
				{Name: "name", IsRequired: true, Description: "The name of the class"},
			},
			Options: []CommandOption{},
		}
	}
	return commands
}
