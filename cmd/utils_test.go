package cmd

import "testing"

func TestFormatBool(t *testing.T) {
	if got := formatBool(true); got != "true" {
		t.Errorf("formatBool(true) = %q; want \"true\"", got)
	}
	if got := formatBool(false); got != "false" {
		t.Errorf("formatBool(false) = %q; want \"false\"", got)
	}
}
