package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/espinosajuanma/slingr/internal/api"
)

func TestResolveCommandSupportsNestedDispatch(t *testing.T) {
	leaf := &Command{
		Name:    "show",
		Summary: "show current workspace",
		Handler: func(*App, context.Context, []string) error { return nil },
	}
	group := &Command{Name: "workspace", Summary: "workspace commands", Children: []*Command{leaf}}
	commands := []*Command{group}

	resolved, args, path, ok := resolveCommand(commands, []string{"workspace", "show", "--verbose"})
	if !ok || resolved != leaf {
		t.Fatalf("resolved = %#v, ok = %v; want nested show command", resolved, ok)
	}
	if len(args) != 1 || args[0] != "--verbose" {
		t.Fatalf("command args = %#v", args)
	}
	if got, want := strings.Join(path, " "), "workspace show"; got != want {
		t.Fatalf("command path = %q, want %q", got, want)
	}
	groupCommand, groupArgs, groupPath, ok := resolveCommand(commands, []string{"workspace"})
	if !ok || groupCommand != group || len(groupArgs) != 0 || strings.Join(groupPath, " ") != "workspace" {
		t.Fatalf("resolving group returned %#v, %#v, %#v, %v", groupCommand, groupArgs, groupPath, ok)
	}
	if found, path, ok := findCommandPath(commands, []string{"workspace", "show"}); !ok || found != leaf || strings.Join(path, " ") != "workspace show" {
		t.Fatalf("nested command lookup = %#v, %v", found, ok)
	}
}

func TestResolveCommandRejectsUnknownPath(t *testing.T) {
	commands := []*Command{{Name: "workspace", Children: []*Command{{Name: "show"}}}}
	if _, _, _, ok := resolveCommand(commands, []string{"missing"}); ok {
		t.Fatal("unknown top-level command resolved")
	}
	command, args, _, ok := resolveCommand(commands, []string{"workspace", "missing"})
	if !ok || command.Name != "workspace" || len(args) != 1 || args[0] != "missing" {
		t.Fatalf("unknown nested command should return its group for usage: %#v, %#v, %v", command, args, ok)
	}
}

func TestRegisteredCommandsContainMetadataAndHandlers(t *testing.T) {
	application, err := New(Options{
		Name:  "slingr",
		Store: &memoryStore{values: map[string]string{}},
		API:   api.NewClient(nil, ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	seenNames := make(map[string]bool)
	for _, name := range commandNames(application.commands) {
		if seenNames[name] {
			t.Errorf("command name or alias %q is registered more than once", name)
		}
		seenNames[name] = true
	}
	for _, command := range application.commands {
		if command.Summary == "" || command.Description == "" || command.Handler == nil {
			t.Errorf("command %q is missing summary, description, or handler", command.Name)
		}
	}
}

func TestAliasesResolveToCanonicalCommands(t *testing.T) {
	application, err := New(Options{
		Name:  "solutions",
		Store: &memoryStore{values: map[string]string{}},
		API:   api.NewClient(nil, ""),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		alias      string
		canonical  string
		prefixArgs []string
	}{
		{alias: "app", canonical: "config", prefixArgs: []string{"app"}},
		{alias: "env", canonical: "config", prefixArgs: []string{"env"}},
		{alias: "email", canonical: "config", prefixArgs: []string{"email"}},
		{alias: "q", canonical: "query"},
		{alias: "rm", canonical: "remove"},
		{alias: "add", canonical: "new"},
	} {
		command, alias, ok := findCommand(application.commands, test.alias)
		if !ok || command.Name != test.canonical || alias == nil {
			t.Fatalf("alias %q resolved to %#v, %v", test.alias, command, ok)
		}
		if strings.Join(alias.PrefixArgs, " ") != strings.Join(test.prefixArgs, " ") {
			t.Errorf("alias %q args prefix = %#v, want %#v", test.alias, alias.PrefixArgs, test.prefixArgs)
		}
	}

	command, args, path, ok := resolveCommand(application.commands, []string{"app", "my-app"})
	if !ok || command.Name != "config" || strings.Join(args, " ") != "app my-app" || strings.Join(path, " ") != "config" {
		t.Fatalf("app alias dispatch = %#v, %#v, %#v, %v", command, args, path, ok)
	}
}

func TestNestedCommandsUseApplicationSliceForHelpAndCompletion(t *testing.T) {
	store := &memoryStore{values: map[string]string{}}
	application, output := newTestApp(t, store, api.NewClient(nil, ""), "solutions workspace cu")
	leaf := &Command{
		Name:        "current",
		Summary:     "show the current workspace",
		Description: "Prints the current workspace name.",
		Examples:    []string{"workspace current"},
		Handler:     func(*App, context.Context, []string) error { return nil },
	}
	group := &Command{
		Name:        "workspace",
		Summary:     "workspace operations",
		Description: "Manage the current workspace.",
		Children:    []*Command{leaf},
	}
	application.commands = append(application.commands, group)

	if err := application.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "current\n"; got != want {
		t.Fatalf("nested completion = %q, want %q", got, want)
	}

	application.completionLine = ""
	output.Reset()
	if err := application.Run(context.Background(), []string{"help", "workspace", "current"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "solutions workspace current - show the current workspace") {
		t.Fatalf("nested manual page missing command path: %q", output.String())
	}
}

func TestAppCommandsAreIndependentSlices(t *testing.T) {
	first, err := New(Options{Name: "one", Store: &memoryStore{values: map[string]string{}}, API: api.NewClient(nil, "")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(Options{Name: "two", Store: &memoryStore{values: map[string]string{}}, API: api.NewClient(nil, "")})
	if err != nil {
		t.Fatal(err)
	}
	first.commands = append(first.commands, &Command{Name: "custom"})
	if len(first.commands) == len(second.commands) {
		t.Fatal("adding a command to one app changed the other app")
	}
	first.commands[0].Children = append(first.commands[0].Children, &Command{Name: "private-child"})
	if len(second.commands[0].Children) != 0 {
		t.Fatal("mutating one app's command tree changed the other app")
	}
}
