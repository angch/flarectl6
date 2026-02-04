package cmd

import (
	"testing"
)

func TestUserCmd(t *testing.T) {
	if userCmd.Use != "user" {
		t.Errorf("expected Use 'user', got '%s'", userCmd.Use)
	}
	if len(userCmd.Aliases) != 1 || userCmd.Aliases[0] != "u" {
		t.Errorf("expected Alias 'u', got %v", userCmd.Aliases)
	}

	// Cobra adds help command by default if not disabled, but usually it's not in Commands() list unless explicitly added?
	// actually Commands() returns children.
	// userCmd adds userInfoCmd and userUpdateCmd in init().
	// But init() is called on package load.

	subCommands := userCmd.Commands()
	expectedCount := 2
	if len(subCommands) != expectedCount {
		t.Errorf("expected %d subcommands, got %d", expectedCount, len(subCommands))
	}

	foundInfo := false
	foundUpdate := false

	for _, cmd := range subCommands {
		switch cmd.Name() {
		case "info":
			foundInfo = true
			if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "i" {
				t.Errorf("info command should have alias 'i'")
			}
		case "update":
			foundUpdate = true
			if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "u" {
				t.Errorf("update command should have alias 'u'")
			}
		}
	}

	if !foundInfo {
		t.Error("info subcommand not found")
	}
	if !foundUpdate {
		t.Error("update subcommand not found")
	}
}

func TestUserInfoCmd(t *testing.T) {
	if userInfoCmd.Use != "info" {
		t.Errorf("expected Use 'info', got '%s'", userInfoCmd.Use)
	}
}

func TestUserUpdateCmd(t *testing.T) {
	if userUpdateCmd.Use != "update" {
		t.Errorf("expected Use 'update', got '%s'", userUpdateCmd.Use)
	}
}
