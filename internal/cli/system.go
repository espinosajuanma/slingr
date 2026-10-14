package cli

import (
	"context"
	"errors"
	"fmt"
)

var versionCmd = &Command{
	Name:        "version",
	Summary:     "display the CLI version",
	Description: "Prints the version of this Slingr CLI executable.",
	Examples:    []string{"version"},
	Handler:     (*App).versionCommand,
}

var pathCmd = &Command{
	Name:        "path",
	Summary:     "display the configuration file path",
	Description: "Prints the path to this executable alias's configuration file.",
	Examples:    []string{"path"},
	Handler:     (*App).configPathCommand,
}

func (a *App) versionCommand(_ context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: version")
	}
	version := a.version
	if version == "" {
		version = "dev"
	}
	_, err := fmt.Fprintln(a.output, version)
	return err
}

func (a *App) configPathCommand(_ context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: path")
	}
	path := a.configPath
	if path == "" {
		if provider, ok := a.store.(interface{ Path() string }); ok {
			path = provider.Path()
		}
	}
	if path == "" {
		return errors.New("configuration path is not available")
	}
	_, err := fmt.Fprintln(a.output, path)
	return err
}
