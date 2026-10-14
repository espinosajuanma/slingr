package cli

import (
	"context"
	"fmt"
	"io"
	"os"
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
	command, path, ok := findCommandPath(a.commands, args)
	if !ok {
		return fmt.Errorf("no help available for %q; run '%s help' to list commands", strings.Join(args, " "), a.name)
	}
	return a.writeManual(command, strings.Join(path, " "))
}

func (a *App) writeHelpIndex() error {
	if _, err := fmt.Fprintln(a.output, formatHelpTitle(strings.ToUpper(a.name)+" - Slingr CLI manual", a.color)); err != nil {
		return err
	}
	if err := a.writeTitle("USAGE"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "  %s COMMAND [ARGUMENTS...]\n\n", a.name); err != nil {
		return err
	}
	if err := a.writeTitle("COMMANDS"); err != nil {
		return err
	}
	for _, command := range a.commands {
		name := command.Name
		if len(command.Aliases) > 0 {
			aliases := make([]string, 0, len(command.Aliases))
			for _, alias := range command.Aliases {
				aliases = append(aliases, alias.Name)
			}
			name += " (" + strings.Join(aliases, ", ") + ")"
		}
		if _, err := fmt.Fprintf(a.output, "  %-18s %s\n", name, command.Summary); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(a.output, "\nRun '%s help COMMAND' for a command's full manual page.\n", a.name)
	return err
}

func (a *App) writeManual(command *Command, path string) error {
	if err := a.writeTitle("NAME"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "  %s %s - %s\n\n", a.name, path, command.Summary); err != nil {
		return err
	}
	if err := a.writeTitle("SYNOPSIS"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "  %s %s", a.name, path); err != nil {
		return err
	}
	if command.Synopsis != "" {
		if _, err := fmt.Fprintf(a.output, " %s", command.Synopsis); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(a.output, "\n\n"); err != nil {
		return err
	}
	if err := a.writeTitle("DESCRIPTION"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "  %s\n", command.Description); err != nil {
		return err
	}
	if err := writeManualSection(a.output, "ARGUMENTS", command.Arguments, a.color); err != nil {
		return err
	}
	if len(command.Aliases) > 0 {
		aliases := make([]string, 0, len(command.Aliases))
		for _, alias := range command.Aliases {
			target := command.Name
			if len(alias.PrefixArgs) > 0 {
				target += " " + strings.Join(alias.PrefixArgs, " ")
			}
			aliases = append(aliases, alias.Name+" -> "+target)
		}
		if err := writeManualSection(a.output, "ALIASES", aliases, a.color); err != nil {
			return err
		}
	}
	if len(command.Children) > 0 {
		if _, err := fmt.Fprint(a.output, "\n"); err != nil {
			return err
		}
		if err := a.writeTitle("SUBCOMMANDS"); err != nil {
			return err
		}
		for _, child := range command.Children {
			if _, err := fmt.Fprintf(a.output, "  %-16s %s\n", child.Name, child.Summary); err != nil {
				return err
			}
		}
	}
	if err := writeExamples(a.output, a.name, command.Examples, a.color); err != nil {
		return err
	}
	return writeManualSection(a.output, "API", command.Endpoints, a.color)
}

func (a *App) writeTitle(title string) error {
	_, err := fmt.Fprintln(a.output, formatHelpTitle(title, a.color))
	return err
}

func formatHelpTitle(title string, color bool) string {
	if !color {
		return title
	}
	return "\x1b[1;32m" + title + "\x1b[0m"
}

func colorOutputEnabled(output io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := output.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func writeManualSection(output io.Writer, title string, entries []string, color bool) error {
	if len(entries) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(output, "\n%s\n", formatHelpTitle(title, color)); err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := fmt.Fprintf(output, "  %s\n", entry); err != nil {
			return err
		}
	}
	return nil
}

func writeExamples(output io.Writer, executable string, examples []string, color bool) error {
	if len(examples) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(output, "\n%s\n", formatHelpTitle("EXAMPLES", color)); err != nil {
		return err
	}
	for _, example := range examples {
		if _, err := fmt.Fprintf(output, "  %s %s\n", executable, example); err != nil {
			return err
		}
	}
	return nil
}
