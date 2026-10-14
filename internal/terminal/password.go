package terminal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func ReadPassword(input io.Reader, prompt io.Writer) (string, error) {
	if _, err := fmt.Fprint(prompt, "Password: "); err != nil {
		return "", err
	}
	if file, ok := input.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			disableEcho := exec.Command("stty", "-echo")
			disableEcho.Stdin = file
			if err := disableEcho.Run(); err != nil {
				return "", fmt.Errorf("disable terminal echo: %w", err)
			}
			defer func() {
				restoreEcho := exec.Command("stty", "echo")
				restoreEcho.Stdin = file
				_ = restoreEcho.Run()
				fmt.Fprintln(prompt)
			}()
		}
	}
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		return "", errors.New("password cannot be empty")
	}
	return password, nil
}
