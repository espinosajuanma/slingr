package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

var configCmd = &Command{
	Name:        "config",
	Summary:     "read or update per-alias configuration",
	Synopsis:    "[KEY [VALUE...]]",
	Description: "With no arguments, lists all settings. With KEY, prints its value. With KEY and VALUE, stores the value; multiple value arguments are joined with spaces. Keys are arbitrary strings. Use app, env, and email to configure an application.",
	Arguments:   []string{"KEY: setting name to read or write.", "VALUE: setting value; any remaining arguments are joined with spaces."},
	Examples:    []string{"config", "config app", "config app my-solutions-app", "config env staging", "config email juan@example.com"},
	Aliases: []CommandAlias{
		{Name: "app", PrefixArgs: []string{"app"}},
		{Name: "env", PrefixArgs: []string{"env"}},
		{Name: "email", PrefixArgs: []string{"email"}},
	},
	Handler: (*App).configCommand,
}

func (a *App) configCommand(_ context.Context, args []string) error {
	switch len(args) {
	case 0:
		keys := make([]string, 0, len(a.config))
		for key := range a.config {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if _, err := fmt.Fprintf(a.output, "%s=%s\n", key, a.config[key]); err != nil {
				return err
			}
		}
	case 1:
		if value, ok := a.config[args[0]]; ok {
			_, err := fmt.Fprintln(a.output, value)
			return err
		}
	case 2:
		a.config[args[0]] = args[1]
		return a.store.Save(a.config)
	default:
		a.config[args[0]] = strings.Join(args[1:], " ")
		return a.store.Save(a.config)
	}
	return nil
}
