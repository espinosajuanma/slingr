package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/espinosajuanma/slingr/internal/api"
	"github.com/espinosajuanma/slingr/internal/cli"
	"github.com/espinosajuanma/slingr/internal/config"
	"github.com/espinosajuanma/slingr/internal/terminal"
)

func main() {
	name := os.Getenv("EXE")
	if name == "" {
		name = filepath.Base(os.Args[0])
	}
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "slingr"
	}
	completionPoint, _ := strconv.Atoi(os.Getenv("COMP_POINT"))

	store := config.NewFileStore(config.PathFor(
		name,
		os.Getenv("HOME"),
		os.Getenv("XDG_CONFIG_HOME"),
		os.Getenv("CONFIG_DIR"),
	))
	apiClient := api.NewClient(&http.Client{Timeout: 60 * time.Second}, os.Getenv("SLINGR_API_URL"))
	application, err := cli.New(cli.Options{
		Name:            name,
		Store:           store,
		API:             apiClient,
		Input:           os.Stdin,
		Output:          os.Stdout,
		ErrorOutput:     os.Stderr,
		PasswordPrompt:  terminal.ReadPassword,
		CompletionLine:  os.Getenv("COMP_LINE"),
		CompletionPoint: completionPoint,
	})
	if err == nil {
		err = application.Run(context.Background(), os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
