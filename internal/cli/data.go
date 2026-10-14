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
