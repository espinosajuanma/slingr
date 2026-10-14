package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/espinosajuanma/slingr/internal/api"
)

func TestFormatHelpTitleUsesBoldGreenANSI(t *testing.T) {
	got := formatHelpTitle("SYNOPSIS", true)
	if want := "\x1b[1;32mSYNOPSIS\x1b[0m"; got != want {
		t.Fatalf("styled title = %q, want %q", got, want)
	}
	if got := formatHelpTitle("SYNOPSIS", false); got != "SYNOPSIS" {
		t.Fatalf("plain title = %q", got)
	}
}

func TestHelpTitlesAreStyledWhenEnabled(t *testing.T) {
	application, output := newTestApp(t, &memoryStore{values: map[string]string{}}, api.NewClient(nil, ""), "")
	application.color = true
	if err := application.Run(context.Background(), []string{"help", "query"}); err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"NAME", "SYNOPSIS", "DESCRIPTION", "ARGUMENTS", "EXAMPLES", "API"} {
		styled := "\x1b[1;32m" + title + "\x1b[0m"
		if !strings.Contains(output.String(), styled) {
			t.Errorf("help output missing styled title %q: %q", title, output.String())
		}
	}
}

func TestColorOutputDisabledForNonTerminalWriter(t *testing.T) {
	if colorOutputEnabled(&strings.Builder{}) {
		t.Fatal("expected a non-terminal writer to disable ANSI styling")
	}
}
