package main

import (
	"bytes"
	"testing"
)

func TestRun_NoArgsIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run(nil, &stdout, &stderr)
	if got != 2 {
		t.Errorf("run(nil) exit = %d, want 2", got)
	}
	if stderr.Len() == 0 {
		t.Errorf("run(nil) wrote nothing to stderr, want a usage message")
	}
}

func TestRun_UnknownCommandIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"bogus"}, &stdout, &stderr)
	if got != 2 {
		t.Errorf(`run(["bogus"]) exit = %d, want 2`, got)
	}
}
