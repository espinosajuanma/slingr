package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/espinosajuanma/slingr/internal/api"
)

func (a *App) saveConfig() error {
	return a.store.Save(a.config)
}

func (a *App) request(ctx context.Context, method, path string, payload any, token string) ([]byte, error) {
	return a.requestWithOptions(ctx, method, path, payload, nil, "", "", token)
}

func (a *App) requestWithOptions(ctx context.Context, method, path string, payload any, body io.Reader, contentType, accept, token string) ([]byte, error) {
	return a.api.Do(ctx, api.Request{
		App:         a.config["app"],
		Env:         a.config["env"],
		Method:      method,
		Path:        path,
		Payload:     payload,
		Body:        body,
		ContentType: contentType,
		Accept:      accept,
		Token:       token,
	})
}

func (a *App) requireConfig(keys ...string) error {
	for _, key := range keys {
		if strings.TrimSpace(a.config[key]) == "" {
			return fmt.Errorf("missing %s; set it with '%s config %s VALUE'", key, a.name, key)
		}
	}
	if env := a.config["env"]; env != "" && env != "dev" && env != "staging" && env != "prod" {
		return fmt.Errorf("invalid env %q: expected dev, staging, or prod", env)
	}
	return nil
}

func (a *App) tokenKey() string {
	return a.config["app"] + "." + a.config["env"] + ".token"
}

func (a *App) personKey() string {
	return a.config["app"] + "." + a.config["env"] + ".person"
}

func (a *App) requiredToken() (string, error) {
	if err := a.requireConfig("app", "env"); err != nil {
		return "", err
	}
	token := a.config[a.tokenKey()]
	if token == "" {
		return "", fmt.Errorf("no token for %s-%s; run '%s login'", a.config["app"], a.config["env"], a.name)
	}
	return token, nil
}

func (a *App) dataRequest(ctx context.Context, method, path string, payload any) ([]byte, error) {
	token, err := a.requiredToken()
	if err != nil {
		return nil, err
	}
	return a.request(ctx, method, path, payload, token)
}
