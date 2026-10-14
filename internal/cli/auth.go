package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var loginCmd = &Command{
	Name:        "login",
	Summary:     "authenticate and save an app/environment token",
	Synopsis:    "[PASSWORD]",
	Description: "Authenticates with the configured email. If PASSWORD is omitted, reads it from the terminal without echo. The token is stored under the current app and environment for this executable alias.",
	Arguments:   []string{"PASSWORD: optional password; prefer the hidden interactive prompt."},
	Examples:    []string{"login"},
	Endpoints:   []string{"POST /auth/login"},
	Handler:     (*App).login,
}

var logoutCmd = &Command{
	Name:        "logout",
	Summary:     "end the current app/environment session",
	Description: "Calls the logout endpoint with the saved token and removes the token and associated user name from this alias's configuration.",
	Examples:    []string{"logout"},
	Endpoints:   []string{"POST /auth/logout"},
	Handler:     (*App).logout,
}

var tokenCmd = &Command{
	Name:        "token",
	Summary:     "print the current app/environment token",
	Description: "Prints the token saved for the configured app and environment for this executable alias.",
	Examples:    []string{"token"},
	Handler:     (*App).printToken,
}

var whoamiCmd = &Command{
	Name:        "whoami",
	Summary:     "show the currently authenticated user",
	Description: "Fetches the current user associated with the saved app/environment token.",
	Examples:    []string{"whoami"},
	Endpoints:   []string{"GET /users/current"},
	Handler:     (*App).whoami,
}

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

func (a *App) whoami(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: whoami")
	}
	token, err := a.requiredToken()
	if err != nil {
		return err
	}
	response, err := a.requestWithOptions(ctx, http.MethodGet, "/users/current", nil, nil, "application/json", "application/json", token)
	if err != nil {
		return err
	}
	return a.writeResponse(response)
}
