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
	Version         string
	ConfigPath      string
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
	version         string
	configPath      string
	store           ConfigStore
	api             API
	input           io.Reader
	output          io.Writer
	errorOutput     io.Writer
	passwordPrompt  PasswordPrompt
	completionLine  string
	completionPoint int
	color           bool
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
		version:         options.Version,
		configPath:      options.ConfigPath,
		store:           options.Store,
		api:             options.API,
		input:           options.Input,
		output:          options.Output,
		errorOutput:     options.ErrorOutput,
		passwordPrompt:  options.PasswordPrompt,
		completionLine:  options.CompletionLine,
		completionPoint: options.CompletionPoint,
		color:           colorOutputEnabled(options.Output),
		commands: cloneCommands([]*Command{
			configCmd,
			currentCmd,
			versionCmd,
			pathCmd,
			helpCmd,
			loginCmd,
			whoamiCmd,
			queryCmd,
			aggregateCmd,
			uploadCmd,
			downloadCmd,
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
		return a.writeHelpIndex()
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
	if !isConfigurationCommand(path[0]) && !a.configurationComplete() {
		if err := a.setupWizard(); err != nil {
			return err
		}
	}
	return command.Handler(a, ctx, commandArgs)
}

func isConfigurationCommand(name string) bool {
	switch name {
	case "config", "help", "version", "path":
		return true
	default:
		return false
	}
}

func (a *App) usageText() string {
	names := commandNames(a.commands)
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
		child, _, ok := findCommand(available, word)
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
	for _, name := range commandNames(available) {
		if strings.HasPrefix(name, prefix) {
			fmt.Fprintln(a.output, name)
		}
	}
}
