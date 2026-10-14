package cli

import (
	"context"
	"errors"
	"fmt"
)

var currentCmd = &Command{
	Name:        "current",
	Summary:     "print the configured app and environment",
	Description: "Prints the configured app and environment as APP-ENV, for example solutions-dev or solutions-prod.",
	Examples:    []string{"current"},
	Handler:     (*App).current,
}

func (a *App) current(_ context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: current")
	}
	if err := a.requireConfig("app", "env"); err != nil {
		return err
	}
	_, err := fmt.Fprintf(a.output, "%s-%s\n", a.config["app"], a.config["env"])
	return err
}
