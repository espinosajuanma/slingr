package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var queryCmd = &Command{
	Name:        "query",
	Summary:     "query records in an entity",
	Synopsis:    "ENTITY [--field VALUE ...]",
	Description: "Sends each field/value pair as a query parameter. Dotted field names stay dotted in the query string. Values that are valid JSON use their JSON representation; other values are sent as text.",
	Arguments:   []string{"ENTITY: entity name.", "--field VALUE: zero or more query criteria."},
	Examples:    []string{"query holidays --country Argentina", "query orders --status active --amount 12"},
	Endpoints:   []string{"GET /data/{entityName}?{key}={value}"},
	Handler:     (*App).query,
}

var getCmd = &Command{
	Name:        "get",
	Summary:     "fetch one record",
	Synopsis:    "ENTITY RECORD_ID",
	Description: "Returns the record identified by ENTITY and RECORD_ID.",
	Arguments:   []string{"ENTITY: entity name.", "RECORD_ID: record identifier."},
	Examples:    []string{"get holidays d987485b-488e-43c9-b5d4-77ba5dd6950b"},
	Endpoints:   []string{"GET /data/{entityName}/{recordId}"},
	Handler:     (*App).get,
}

var newCmd = &Command{
	Name:        "new",
	Summary:     "create a record",
	Synopsis:    "ENTITY [--field VALUE ...]",
	Description: "Builds a JSON object from the supplied fields and creates a record. Dotted field names create nested objects. A value that is valid JSON is decoded as JSON (including strings, numbers, booleans, null, arrays, or objects); other values remain text.",
	Arguments:   []string{"ENTITY: entity name.", "--field VALUE: zero or more record fields."},
	Examples:    []string{"new holidays --country Argentina --daySettings.date 2026-10-14", `new orders --items '[{"sku":"A1","quantity":2}]'`},
	Endpoints:   []string{"POST /data/{entityName}"},
	Handler:     (*App).create,
}

var removeCmd = &Command{
	Name:        "remove",
	Summary:     "delete a record",
	Synopsis:    "ENTITY RECORD_ID",
	Description: "Deletes the record identified by ENTITY and RECORD_ID.",
	Arguments:   []string{"ENTITY: entity name.", "RECORD_ID: record identifier."},
	Examples:    []string{"remove holidays d987485b-488e-43c9-b5d4-77ba5dd6950b"},
	Endpoints:   []string{"DELETE /data/{entityName}/{recordId}"},
	Handler:     (*App).remove,
}

var editCmd = &Command{
	Name:        "edit",
	Summary:     "merge changes into an existing record",
	Synopsis:    "ENTITY RECORD_ID [--field VALUE ...]",
	Description: "Fetches the existing record, recursively merges the supplied fields into its JSON object, then sends the updated record. Nested fields use dotted names.",
	Arguments:   []string{"ENTITY: entity name.", "RECORD_ID: record identifier.", "--field VALUE: one or more values to merge into the record."},
	Examples:    []string{"edit holidays d987485b-488e-43c9-b5d4-77ba5dd6950b --daySettings.date 2026-10-10"},
	Endpoints:   []string{"GET /data/{entityName}/{recordId}", "PUT /data/{entityName}/{recordId}"},
	Handler:     (*App).edit,
}

var actionCmd = &Command{
	Name:        "action",
	Summary:     "run an entity-level or record-level action",
	Synopsis:    "ENTITY ACTION_NAME [RECORD_ID] [--field VALUE ...]",
	Description: "ACTION_NAME is required. If RECORD_ID is supplied, runs that action for the record; otherwise, runs the entity-level action. Fields form the JSON request payload.",
	Arguments:   []string{"ENTITY: entity name.", "ACTION_NAME: action name to invoke.", "RECORD_ID: optional record identifier; omit for an entity-level action.", "--field VALUE: zero or more action payload fields."},
	Examples:    []string{"action holidays refresh --force true", "action holidays approve d987485b-488e-43c9-b5d4-77ba5dd6950b --country USA"},
	Endpoints:   []string{"PUT /data/{entityName}/{actionName}", "PUT /data/{entityName}/{recordId}/{actionName}"},
	Handler:     (*App).action,
}

func (a *App) query(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: query ENTITY [--field VALUE ...]")
	}
	token, err := a.requiredToken()
	if err != nil {
		return err
	}
	params, err := parseFlagValues(args[1:])
	if err != nil {
		return err
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, queryValue(value))
	}
	path := "/data/" + url.PathEscape(args[0])
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	response, err := a.request(ctx, http.MethodGet, path, nil, token)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) get(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: get ENTITY RECORD_ID")
	}
	response, err := a.dataRequest(ctx, http.MethodGet, "/data/"+url.PathEscape(args[0])+"/"+url.PathEscape(args[1]), nil)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) create(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: new ENTITY [--field VALUE ...]")
	}
	payload, err := parseFlags(args[1:])
	if err != nil {
		return err
	}
	response, err := a.dataRequest(ctx, http.MethodPost, "/data/"+url.PathEscape(args[0]), payload)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) remove(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: remove ENTITY RECORD_ID")
	}
	response, err := a.dataRequest(ctx, http.MethodDelete, "/data/"+url.PathEscape(args[0])+"/"+url.PathEscape(args[1]), nil)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) edit(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: edit ENTITY RECORD_ID [--field VALUE ...]")
	}
	changes, err := parseFlags(args[2:])
	if err != nil {
		return err
	}
	path := "/data/" + url.PathEscape(args[0]) + "/" + url.PathEscape(args[1])
	currentBody, err := a.dataRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	var current map[string]any
	if err := json.Unmarshal(currentBody, &current); err != nil {
		return fmt.Errorf("decode existing record: %w", err)
	}
	if current == nil {
		return errors.New("existing record response must be a JSON object")
	}
	mergeMaps(current, changes)
	response, err := a.dataRequest(ctx, http.MethodPut, path, current)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) action(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: action ENTITY ACTION_NAME [RECORD_ID] [--field VALUE ...]")
	}
	entity, actionName := args[0], args[1]
	index := 2
	recordID := ""
	if index < len(args) && !strings.HasPrefix(args[index], "--") {
		recordID = args[index]
		index++
	}
	path := "/data/" + url.PathEscape(entity) + "/" + url.PathEscape(actionName)
	if recordID != "" {
		path = "/data/" + url.PathEscape(entity) + "/" + url.PathEscape(recordID) + "/" + url.PathEscape(actionName)
	}
	payload, err := parseFlags(args[index:])
	if err != nil {
		return err
	}
	response, err := a.dataRequest(ctx, http.MethodPut, path, payload)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) writeResponse(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, trimmed, "", "  ") == nil {
		trimmed = pretty.Bytes()
	}
	_, err := fmt.Fprintln(a.output, string(trimmed))
	return err
}
