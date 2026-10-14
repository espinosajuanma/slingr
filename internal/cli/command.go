package cli

import "context"

type CommandHandler func(*App, context.Context, []string) error

// Command keeps its handler and user-facing manual metadata together. Children
// allow commands to be grouped into deeper command trees when needed.
type Command struct {
	Name        string
	Summary     string
	Synopsis    string
	Description string
	Arguments   []string
	Examples    []string
	Endpoints   []string
	Handler     CommandHandler
	Children    []*Command
}

func cloneCommands(commands []*Command) []*Command {
	cloned := make([]*Command, 0, len(commands))
	for _, command := range commands {
		copy := *command
		copy.Arguments = append([]string(nil), command.Arguments...)
		copy.Examples = append([]string(nil), command.Examples...)
		copy.Endpoints = append([]string(nil), command.Endpoints...)
		copy.Children = cloneCommands(command.Children)
		cloned = append(cloned, &copy)
	}
	return cloned
}

func findCommand(commands []*Command, name string) (*Command, bool) {
	for _, command := range commands {
		if command.Name == name {
			return command, true
		}
	}
	return nil, false
}

func findCommandPath(commands []*Command, path []string) (*Command, bool) {
	if len(path) == 0 {
		return nil, false
	}
	command, ok := findCommand(commands, path[0])
	if !ok || len(path) == 1 {
		return command, ok
	}
	return findCommandPath(command.Children, path[1:])
}

func resolveCommand(commands []*Command, args []string) (*Command, []string, []string, bool) {
	available := commands
	var current *Command
	var path []string
	for index, argument := range args {
		if current != nil && current.Handler != nil {
			return current, args[index:], path, true
		}
		command, ok := findCommand(available, argument)
		if !ok {
			if current != nil && current.Handler == nil && len(current.Children) > 0 {
				return current, args[index:], path, true
			}
			return nil, nil, nil, false
		}
		current = command
		path = append(path, command.Name)
		available = command.Children
	}
	if current == nil {
		return nil, nil, nil, false
	}
	if current.Handler != nil || len(current.Children) > 0 {
		return current, nil, path, true
	}
	return nil, nil, nil, false
}
