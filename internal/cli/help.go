package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
)

var helpCmd = &Command{
	Name:        "help",
	Summary:     "show the CLI manual or a command's manual page",
	Synopsis:    "[COMMAND [SUBCOMMAND...]]",
	Description: "Without COMMAND, prints the command index. With COMMAND, prints the detailed man-page-style help for that command.",
	Arguments:   []string{"COMMAND: one of the supported CLI commands.", "SUBCOMMAND: optional nested command path, if the selected command is a group."},
	Examples:    []string{"help", "help query", "help action"},
	Handler:     (*App).help,
}

func (a *App) help(_ context.Context, args []string) error {
	if len(args) == 0 {
		return a.writeHelpIndex()
	}
	command, ok := findCommandPath(a.commands, args)
	if !ok {
		return fmt.Errorf("no help available for %q; run '%s help' to list commands", strings.Join(args, " "), a.name)
	}
	return a.writeManual(command, strings.Join(args, " "))
}

func (a *App) writeHelpIndex() error {
	if _, err := fmt.Fprintf(a.output, "%s - Slingr CLI manual\n\n", strings.ToUpper(a.name)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "USAGE\n  %s COMMAND [ARGUMENTS...]\n\nCOMMANDS\n", a.name); err != nil {
		return err
	}
	for _, command := range a.commands {
		if _, err := fmt.Fprintf(a.output, "  %-8s %s\n", command.Name, command.Summary); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(a.output, "\nRun '%s help COMMAND' for a command's full manual page.\n", a.name)
	return err
}

func (a *App) writeManual(command *Command, path string) error {
	if _, err := fmt.Fprintf(a.output, "NAME\n  %s %s - %s\n\n", a.name, path, command.Summary); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "SYNOPSIS\n  %s %s", a.name, path); err != nil {
		return err
	}
	if command.Synopsis != "" {
		if _, err := fmt.Fprintf(a.output, " %s", command.Synopsis); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(a.output, "\n\nDESCRIPTION\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "  %s\n", command.Description); err != nil {
		return err
	}
	if err := writeManualSection(a.output, "ARGUMENTS", command.Arguments); err != nil {
		return err
	}
	if len(command.Children) > 0 {
		if _, err := fmt.Fprint(a.output, "\nSUBCOMMANDS\n"); err != nil {
			return err
		}
		for _, child := range command.Children {
			if _, err := fmt.Fprintf(a.output, "  %-16s %s\n", child.Name, child.Summary); err != nil {
				return err
			}
		}
	}
	if err := writeExamples(a.output, a.name, command.Examples); err != nil {
		return err
	}
	return writeManualSection(a.output, "API", command.Endpoints)
}

func writeManualSection(output io.Writer, title string, entries []string) error {
	if len(entries) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(output, "\n%s\n", title); err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := fmt.Fprintf(output, "  %s\n", entry); err != nil {
			return err
		}
	}
	return nil
}

func writeExamples(output io.Writer, executable string, examples []string) error {
	if len(examples) == 0 {
		return nil
	}
	if _, err := fmt.Fprint(output, "\nEXAMPLES\n"); err != nil {
		return err
	}
	for _, example := range examples {
		if _, err := fmt.Fprintf(output, "  %s %s\n", executable, example); err != nil {
			return err
		}
	}
	return nil
}
