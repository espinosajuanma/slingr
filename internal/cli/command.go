package cli

import "context"

type CommandHandler func(*App, context.Context, []string) error

type CommandAlias struct {
	Name       string
	PrefixArgs []string
}

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
	Aliases     []CommandAlias
	Children    []*Command
}

func cloneCommands(commands []*Command) []*Command {
	cloned := make([]*Command, 0, len(commands))
	for _, command := range commands {
		copy := *command
		copy.Arguments = append([]string(nil), command.Arguments...)
		copy.Examples = append([]string(nil), command.Examples...)
		copy.Endpoints = append([]string(nil), command.Endpoints...)
		copy.Aliases = append([]CommandAlias(nil), command.Aliases...)
		for index := range copy.Aliases {
			copy.Aliases[index].PrefixArgs = append([]string(nil), command.Aliases[index].PrefixArgs...)
		}
		copy.Children = cloneCommands(command.Children)
		cloned = append(cloned, &copy)
	}
	return cloned
}

func findCommand(commands []*Command, name string) (*Command, *CommandAlias, bool) {
	for _, command := range commands {
		if command.Name == name {
			return command, nil, true
		}
		for index := range command.Aliases {
			if command.Aliases[index].Name == name {
				return command, &command.Aliases[index], true
			}
		}
	}
	return nil, nil, false
}

func commandNames(commands []*Command) []string {
	var names []string
	for _, command := range commands {
		names = append(names, command.Name)
		for _, alias := range command.Aliases {
			names = append(names, alias.Name)
		}
	}
	return names
}

func findCommandPath(commands []*Command, path []string) (*Command, []string, bool) {
	if len(path) == 0 {
		return nil, nil, false
	}
	command, _, ok := findCommand(commands, path[0])
	if !ok {
		return nil, nil, false
	}
	canonicalPath := []string{command.Name}
	if len(path) == 1 {
		return command, canonicalPath, true
	}
	child, childPath, ok := findCommandPath(command.Children, path[1:])
	if !ok {
		return nil, nil, false
	}
	return child, append(canonicalPath, childPath...), true
}

func resolveCommand(commands []*Command, args []string) (*Command, []string, []string, bool) {
	available := commands
	var current *Command
	var path []string
	var prefixArgs []string
	for index, argument := range args {
		if current != nil && current.Handler != nil {
			return current, append(prefixArgs, args[index:]...), path, true
		}
		command, alias, ok := findCommand(available, argument)
		if !ok {
			if current != nil && current.Handler == nil && len(current.Children) > 0 {
				return current, args[index:], path, true
			}
			return nil, nil, nil, false
		}
		current = command
		path = append(path, command.Name)
		if alias != nil {
			prefixArgs = append(prefixArgs, alias.PrefixArgs...)
		}
		available = command.Children
	}
	if current == nil {
		return nil, nil, nil, false
	}
	if current.Handler != nil || len(current.Children) > 0 {
		return current, prefixArgs, path, true
	}
	return nil, nil, nil, false
}
