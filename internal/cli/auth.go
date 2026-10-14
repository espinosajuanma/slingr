package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (a *App) login(ctx context.Context, args []string) error {
	if err := a.requireConfig("app", "env", "email"); err != nil {
		return err
	}
	password := strings.Join(args, " ")
	if password == "" {
		if a.passwordPrompt == nil {
			return errors.New("password prompt is not configured")
		}
		var err error
		password, err = a.passwordPrompt(a.input, a.errorOutput)
		if err != nil {
			return err
		}
	}
	payload := map[string]any{"email": a.config["email"], "password": password}
	response, err := a.request(ctx, http.MethodPost, "/auth/login", payload, "")
	if err != nil {
		return err
	}
	var auth struct {
		Token    string `json:"token"`
		UserName string `json:"userName"`
	}
	if err := json.Unmarshal(response, &auth); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}
	if auth.Token == "" {
		return errors.New("login response did not contain a token")
	}
	a.config[a.tokenKey()] = auth.Token
	if auth.UserName != "" {
		a.config[a.personKey()] = auth.UserName
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	return a.writeResponse([]byte(`{"success":true}`))
}

func (a *App) logout(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: logout")
	}
	token, err := a.requiredToken()
	if err != nil {
		return err
	}
	response, err := a.request(ctx, http.MethodPost, "/auth/logout", nil, token)
	if err != nil {
		return err
	}
	delete(a.config, a.tokenKey())
	delete(a.config, a.personKey())
	if err := a.saveConfig(); err != nil {
		return err
	}
	return a.writeResponse(response)
}

func (a *App) printToken(_ context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: token")
	}
	token, err := a.requiredToken()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.output, token)
	return err
}
