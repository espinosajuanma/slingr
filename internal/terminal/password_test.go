package terminal

import (
	"strings"
	"testing"
)

func TestReadPasswordFromPipedInput(t *testing.T) {
	var prompt strings.Builder
	password, err := ReadPassword(strings.NewReader("test-password\n"), &prompt)
	if err != nil {
		t.Fatal(err)
	}
	if password != "test-password" {
		t.Fatalf("password = %q", password)
	}
	if prompt.String() != "Password: " {
		t.Fatalf("prompt = %q", prompt.String())
	}
}

func TestReadPasswordRejectsEmptyInput(t *testing.T) {
	_, err := ReadPassword(strings.NewReader("\n"), &strings.Builder{})
	if err == nil {
		t.Fatal("expected empty password to be rejected")
	}
}
