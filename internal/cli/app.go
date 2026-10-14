package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/espinosajuanma/slingr/internal/api"
)

type ConfigStore interface {
	Load() (map[string]string, error)
	Save(map[string]string) error
}

type API interface {
	Do(context.Context, api.Request) ([]byte, error)
}

type PasswordPrompt func(io.Reader, io.Writer) (string, error)

type Options struct {
	Name            string
	Store           ConfigStore
	API             API
	Input           io.Reader
	Output          io.Writer
	ErrorOutput     io.Writer
	PasswordPrompt  PasswordPrompt
	CompletionLine  string
	CompletionPoint int
}

type App struct {
	name            string
	store           ConfigStore
	api             API
	input           io.Reader
	output          io.Writer
	errorOutput     io.Writer
	passwordPrompt  PasswordPrompt
	completionLine  string
	completionPoint int
	config          map[string]string
	commands        []*Command
}

func New(options Options) (*App, error) {
	if options.Name == "" {
		return nil, errors.New("executable name is required")
	}
	if options.Store == nil {
		return nil, errors.New("config store is required")
	}
	if options.API == nil {
		return nil, errors.New("API client is required")
	}
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.ErrorOutput == nil {
		options.ErrorOutput = os.Stderr
	}
	return &App{
		name:            options.Name,
		store:           options.Store,
		api:             options.API,
		input:           options.Input,
		output:          options.Output,
		errorOutput:     options.ErrorOutput,
		passwordPrompt:  options.PasswordPrompt,
		completionLine:  options.CompletionLine,
		completionPoint: options.CompletionPoint,
		commands: cloneCommands([]*Command{
			configCmd,
			helpCmd,
			loginCmd,
			queryCmd,
			getCmd,
			newCmd,
			removeCmd,
			editCmd,
			actionCmd,
			logoutCmd,
			tokenCmd,
		}),
	}, nil
}

func (a *App) Run(ctx context.Context, args []string) error {
	values, err := a.store.Load()
	if err != nil {
		return err
	}
	a.config = values
	if a.config == nil {
		a.config = make(map[string]string)
	}
	if a.completionLine != "" {
		a.complete()
		return nil
	}
	if len(args) == 0 {
		if !a.configurationComplete() {
			return a.setupWizard()
		}
		return a.usage()
	}
	command, commandArgs, path, ok := resolveCommand(a.commands, args)
	if !ok {
		return fmt.Errorf("unknown command %q\n%s", args[0], a.usageText())
	}
	if command.Handler == nil {
		children := command.Children
		names := make([]string, 0, len(children))
		for _, child := range children {
			names = append(names, child.Name)
		}
		return fmt.Errorf("usage: %s %s {%s}", a.name, strings.Join(path, " "), strings.Join(names, "|"))
	}
	if path[0] != "config" && path[0] != "help" && !a.configurationComplete() {
		if err := a.setupWizard(); err != nil {
			return err
		}
	}
	return command.Handler(a, ctx, commandArgs)
}

func (a *App) usageText() string {
	names := make([]string, 0, len(a.commands))
	for _, command := range a.commands {
		names = append(names, command.Name)
	}
	return fmt.Sprintf("Usage: %s {%s}", a.name, strings.Join(names, "|"))
}

func (a *App) usage() error {
	_, err := fmt.Fprintln(a.output, a.usageText())
	return err
}

func (a *App) complete() {
	line := a.completionLine
	if a.completionPoint > 0 && a.completionPoint < len(line) {
		line = line[:a.completionPoint]
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return
	}
	prefix := ""
	commandWords := words[1:]
	if !strings.HasSuffix(line, " ") && len(commandWords) > 0 {
		prefix = commandWords[len(commandWords)-1]
		commandWords = commandWords[:len(commandWords)-1]
	}
	available := a.commands
	var current *Command
	for _, word := range commandWords {
		if current != nil && current.Name == "help" {
			available = a.commands
			current = nil
		}
		child, ok := findCommand(available, word)
		if !ok {
			return
		}
		current = child
		available = child.Children
	}
	// Help takes a command path as its argument, so its next completion level
	// mirrors the command tree itself.
	if current != nil && current.Name == "help" {
		available = a.commands
	}
	for _, command := range available {
		if strings.HasPrefix(command.Name, prefix) {
			fmt.Fprintln(a.output, command.Name)
		}
	}
}
