package cli

import (
	"context"
	"fmt"
	"strings"
)

type manualPage struct {
	Summary     string
	Synopsis    string
	Description string
	Arguments   []string
	Examples    []string
	API         []string
}

var manualPages = map[string]manualPage{
	"config": {
		Summary:     "read or update per-alias configuration",
		Synopsis:    "[KEY [VALUE...]]",
		Description: "With no arguments, lists all settings. With KEY, prints its value. With KEY and VALUE, stores the value; multiple value arguments are joined with spaces. Keys are arbitrary strings. Use app, env, and email to configure an application.",
		Arguments:   []string{"KEY: setting name to read or write.", "VALUE: setting value; any remaining arguments are joined with spaces."},
		Examples:    []string{"config", "config app", "config app my-solutions-app", "config env staging", "config email juan@example.com"},
	},
	"help": {
		Summary:     "show the CLI manual or a command's manual page",
		Synopsis:    "[COMMAND]",
		Description: "Without COMMAND, prints the command index. With COMMAND, prints the detailed man-page-style help for that command.",
		Arguments:   []string{"COMMAND: one of the supported CLI commands."},
		Examples:    []string{"help", "help query", "help action"},
	},
	"login": {
		Summary:     "authenticate and save an app/environment token",
		Synopsis:    "[PASSWORD]",
		Description: "Authenticates with the configured email. If PASSWORD is omitted, reads it from the terminal without echo. The token is stored under the current app and environment for this executable alias.",
		Arguments:   []string{"PASSWORD: optional password; prefer the hidden interactive prompt."},
		Examples:    []string{"login"},
		API:         []string{"POST /auth/login"},
	},
	"query": {
		Summary:     "query records in an entity",
		Synopsis:    "ENTITY [--field VALUE ...]",
		Description: "Sends each field/value pair as a query parameter. Dotted field names stay dotted in the query string. Values that are valid JSON use their JSON representation; other values are sent as text.",
		Arguments:   []string{"ENTITY: entity name.", "--field VALUE: zero or more query criteria."},
		Examples:    []string{"query holidays --country Argentina", "query orders --status active --amount 12"},
		API:         []string{"GET /data/{entityName}?{key}={value}"},
	},
	"get": {
		Summary:     "fetch one record",
		Synopsis:    "ENTITY RECORD_ID",
		Description: "Returns the record identified by ENTITY and RECORD_ID.",
		Arguments:   []string{"ENTITY: entity name.", "RECORD_ID: record identifier."},
		Examples:    []string{"get holidays d987485b-488e-43c9-b5d4-77ba5dd6950b"},
		API:         []string{"GET /data/{entityName}/{recordId}"},
	},
	"new": {
		Summary:     "create a record",
		Synopsis:    "ENTITY [--field VALUE ...]",
		Description: "Builds a JSON object from the supplied fields and creates a record. Dotted field names create nested objects. A value that is valid JSON is decoded as JSON (including strings, numbers, booleans, null, arrays, or objects); other values remain text.",
		Arguments:   []string{"ENTITY: entity name.", "--field VALUE: zero or more record fields."},
		Examples:    []string{"new holidays --country Argentina --daySettings.date 2026-10-14", `new orders --items '[{"sku":"A1","quantity":2}]'`},
		API:         []string{"POST /data/{entityName}"},
	},
	"remove": {
		Summary:     "delete a record",
		Synopsis:    "ENTITY RECORD_ID",
		Description: "Deletes the record identified by ENTITY and RECORD_ID.",
		Arguments:   []string{"ENTITY: entity name.", "RECORD_ID: record identifier."},
		Examples:    []string{"remove holidays d987485b-488e-43c9-b5d4-77ba5dd6950b"},
		API:         []string{"DELETE /data/{entityName}/{recordId}"},
	},
	"edit": {
		Summary:     "merge changes into an existing record",
		Synopsis:    "ENTITY RECORD_ID [--field VALUE ...]",
		Description: "Fetches the existing record, recursively merges the supplied fields into its JSON object, then sends the updated record. Nested fields use dotted names.",
		Arguments:   []string{"ENTITY: entity name.", "RECORD_ID: record identifier.", "--field VALUE: one or more values to merge into the record."},
		Examples:    []string{"edit holidays d987485b-488e-43c9-b5d4-77ba5dd6950b --daySettings.date 2026-10-10"},
		API:         []string{"GET /data/{entityName}/{recordId}", "PUT /data/{entityName}/{recordId}"},
	},
	"action": {
		Summary:     "run an entity-level or record-level action",
		Synopsis:    "ENTITY ACTION_NAME [RECORD_ID] [--field VALUE ...]",
		Description: "ACTION_NAME is required. If RECORD_ID is supplied, runs that action for the record; otherwise, runs the entity-level action. Fields form the JSON request payload.",
		Arguments:   []string{"ENTITY: entity name.", "ACTION_NAME: action name to invoke.", "RECORD_ID: optional record identifier; omit for an entity-level action.", "--field VALUE: zero or more action payload fields."},
		Examples:    []string{"action holidays refresh --force true", "action holidays approve d987485b-488e-43c9-b5d4-77ba5dd6950b --country USA"},
		API:         []string{"PUT /data/{entityName}/{actionName}", "PUT /data/{entityName}/{recordId}/{actionName}"},
	},
	"logout": {
		Summary:     "end the current app/environment session",
		Synopsis:    "",
		Description: "Calls the logout endpoint with the saved token and removes the token and associated user name from this alias's configuration.",
		Examples:    []string{"logout"},
		API:         []string{"POST /auth/logout"},
	},
	"token": {
		Summary:     "print the current app/environment token",
		Synopsis:    "",
		Description: "Prints the token saved for the configured app and environment for this executable alias.",
		Examples:    []string{"token"},
	},
}

func (a *App) help(_ context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: %s help [COMMAND]", a.name)
	}
	if len(args) == 0 {
		return a.writeHelpIndex()
	}
	page, ok := manualPages[args[0]]
	if !ok {
		return fmt.Errorf("no help available for %q; run '%s help' to list commands", args[0], a.name)
	}
	return a.writeManual(args[0], page)
}

func (a *App) writeHelpIndex() error {
	if _, err := fmt.Fprintf(a.output, "%s - Slingr CLI manual\n\n", strings.ToUpper(a.name)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "USAGE\n  %s COMMAND [ARGUMENTS...]\n\nCOMMANDS\n", a.name); err != nil {
		return err
	}
	for _, command := range commandNames {
		page := manualPages[command]
		if _, err := fmt.Fprintf(a.output, "  %-8s %s\n", command, page.Summary); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(a.output, "\nRun '%s help COMMAND' for a command's full manual page.\n", a.name)
	return err
}

func (a *App) writeManual(command string, page manualPage) error {
	if _, err := fmt.Fprintf(a.output, "NAME\n  %s %s - %s\n\n", a.name, command, page.Summary); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "SYNOPSIS\n  %s %s", a.name, command); err != nil {
		return err
	}
	if page.Synopsis != "" {
		if _, err := fmt.Fprintf(a.output, " %s", page.Synopsis); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(a.output, "\n\nDESCRIPTION\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.output, "  %s\n", page.Description); err != nil {
		return err
	}
	if len(page.Arguments) > 0 {
		if _, err := fmt.Fprint(a.output, "\nARGUMENTS\n"); err != nil {
			return err
		}
		for _, argument := range page.Arguments {
			if _, err := fmt.Fprintf(a.output, "  %s\n", argument); err != nil {
				return err
			}
		}
	}
	if len(page.Examples) > 0 {
		if _, err := fmt.Fprint(a.output, "\nEXAMPLES\n"); err != nil {
			return err
		}
		for _, example := range page.Examples {
			if _, err := fmt.Fprintf(a.output, "  %s %s\n", a.name, example); err != nil {
				return err
			}
		}
	}
	if len(page.API) > 0 {
		if _, err := fmt.Fprint(a.output, "\nAPI\n"); err != nil {
			return err
		}
		for _, endpoint := range page.API {
			if _, err := fmt.Fprintf(a.output, "  %s\n", endpoint); err != nil {
				return err
			}
		}
	}
	return nil
}
