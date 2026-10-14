package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

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
