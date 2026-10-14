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
}

var commandNames = []string{"config", "login", "query", "get", "new", "remove", "edit", "action", "logout", "token"}

type commandHandler func(context.Context, []string) error

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
	command, args := args[0], args[1:]
	handlers := map[string]commandHandler{
		"config": a.configCommand,
		"login":  a.login,
		"query":  a.query,
		"get":    a.get,
		"new":    a.create,
		"remove": a.remove,
		"edit":   a.edit,
		"action": a.action,
		"logout": a.logout,
		"token":  a.printToken,
	}
	handler, ok := handlers[command]
	if !ok {
		return fmt.Errorf("unknown command %q\n%s", command, a.usageText())
	}
	if command != "config" && !a.configurationComplete() {
		if err := a.setupWizard(); err != nil {
			return err
		}
	}
	return handler(ctx, args)
}

func (a *App) usageText() string {
	return fmt.Sprintf("Usage: %s {%s}", a.name, strings.Join(commandNames, "|"))
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
	fields := strings.Fields(line)
	if len(fields) == 0 || len(fields) > 2 || (len(fields) == 2 && strings.HasSuffix(line, " ")) {
		return
	}
	prefix := ""
	if len(fields) > 1 {
		prefix = fields[1]
	}
	for _, command := range commandNames {
		if strings.HasPrefix(command, prefix) {
			fmt.Fprintln(a.output, command)
		}
	}
}
