package help

// CommandDescriptor defines the structure for a command and its help information
type CommandDescriptor struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Examples    []string
	SubCommands []*CommandDescriptor
	ParentCmd   *CommandDescriptor // Reference to parent for navigation
}

// Registry maintains a collection of all top-level commands
type Registry struct {
	Commands []*CommandDescriptor
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		Commands: make([]*CommandDescriptor, 0),
	}
}

// AddCommand adds a top-level command to the registry
func (r *Registry) AddCommand(cmd *CommandDescriptor) {
	r.Commands = append(r.Commands, cmd)
}

// FindCommand looks up a command by name or alias
func (r *Registry) FindCommand(name string) *CommandDescriptor {
	for _, cmd := range r.Commands {
		if cmd.Name == name {
			return cmd
		}

		for _, alias := range cmd.Aliases {
			if alias == name {
				return cmd
			}
		}
	}
	return nil
}

// AddSubCommand adds a subcommand to a command
func (cmd *CommandDescriptor) AddSubCommand(sub *CommandDescriptor) {
	sub.ParentCmd = cmd
	cmd.SubCommands = append(cmd.SubCommands, sub)
}

// FindSubCommand looks up a subcommand by name or alias
func (cmd *CommandDescriptor) FindSubCommand(name string) *CommandDescriptor {
	for _, sub := range cmd.SubCommands {
		if sub.Name == name {
			return sub
		}

		for _, alias := range sub.Aliases {
			if alias == name {
				return sub
			}
		}
	}
	return nil
}
