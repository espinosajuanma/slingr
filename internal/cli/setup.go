package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

func (a *App) configurationComplete() bool {
	return strings.TrimSpace(a.config["app"]) != "" &&
		strings.TrimSpace(a.config["email"]) != "" &&
		(a.config["env"] == "dev" || a.config["env"] == "staging" || a.config["env"] == "prod")
}

func (a *App) setupWizard() error {
	if _, err := fmt.Fprintf(a.output, "Slingr setup for %s\n", a.name); err != nil {
		return err
	}
	app, err := a.promptValue("App", a.config["app"])
	if err != nil {
		return err
	}
	a.config["app"] = app
	if err := a.saveConfig(); err != nil {
		return err
	}

	env, err := a.promptValue("Environment (dev|staging|prod)", a.config["env"])
	if err != nil {
		return err
	}
	if env != "dev" && env != "staging" && env != "prod" {
		return fmt.Errorf("invalid environment %q: expected dev, staging, or prod", env)
	}
	a.config["env"] = env
	if err := a.saveConfig(); err != nil {
		return err
	}

	email, err := a.promptValue("Email", a.config["email"])
	if err != nil {
		return err
	}
	a.config["email"] = email
	if err := a.saveConfig(); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.output, "Configuration saved. Run '%s login' to authenticate.\n", a.name)
	return err
}

func (a *App) promptValue(label, current string) (string, error) {
	current = strings.TrimSpace(current)
	if current == "" {
		if _, err := fmt.Fprintf(a.output, "%s: ", label); err != nil {
			return "", err
		}
	} else if _, err := fmt.Fprintf(a.output, "%s [%s]: ", label, current); err != nil {
		return "", err
	}
	line, err := readLine(a.input)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = current
	}
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", strings.ToLower(label))
	}
	return value, nil
}

// readLine consumes only through the next newline, so a following password
// prompt can still read directly from a terminal without buffered input loss.
func readLine(input io.Reader) (string, error) {
	var line strings.Builder
	var oneByte [1]byte
	for {
		n, err := input.Read(oneByte[:])
		if n > 0 {
			if oneByte[0] == '\n' {
				return strings.TrimSuffix(line.String(), "\r"), nil
			}
			line.WriteByte(oneByte[0])
		}
		if err != nil {
			if errors.Is(err, io.EOF) && line.Len() > 0 {
				return strings.TrimSuffix(line.String(), "\r"), nil
			}
			return line.String(), err
		}
		if n == 0 {
			return line.String(), io.ErrNoProgress
		}
	}
}
