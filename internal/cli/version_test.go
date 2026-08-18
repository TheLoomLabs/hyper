package cli_test

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// The version command's golden cases live under testdata/version/, a sibling of
// testdata/check/ rather than a directory inside it: a case's directory says
// which command it exercises (issue #101). They are driven like every other
// case, by the one harness in golden_test.go, from their own argv (issue #108).
// What is here is what a corpus cannot say.

// TestRunVersion_TakesNoArgumentAtAll pins the rule the two rejection cases
// above stand for, over the argument shapes a corpus cannot enumerate: every
// one of the three globals, a lone `--`, a repeated flag, a positional. Any
// argument at all is a usage error, exit 2, with stdout untouched (issue #103).
//
// `--repo-dir` is the sharpest of them: it is meaningless on a command that
// reads no repository, and accepting it quietly would be a promise that the
// directory was consulted.
func TestRunVersion_TakesNoArgumentAtAll(t *testing.T) {
	for _, args := range [][]string{
		{"--json"},
		{"--no-color"},
		{"--repo-dir", "/tmp"},
		{"--repo-dir=/tmp"},
		{"--"},
		{"-v"},
		{"latest"},
		{"1.4.0"},
		{"version"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := cli.RunVersion(args, &stdout, &stderr, version.Current())

			if exit != cli.ExitUsage {
				t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want it untouched", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Error("stderr is empty, want the usage error rendered on it")
			}
		})
	}
}

// TestRunVersion_TakesNoRepositoryToRead is the structural half of the three
// acceptance criteria about repositories: outside a git tree, inside one with
// no hyper.yaml, and inside one pinning something else entirely, `hyper
// version` answers the same. The signature is the proof — it takes neither a
// working directory nor an environment, so there is no repository reachable
// from it to gate on (§9's exemption, ADR-0020). What a signature states, a
// test can only restate; the behaviour those three criteria describe is
// exercised against the real dispatch in Main, and against the real process in
// cmd/hyper.
func TestRunVersion_TakesNoRepositoryToRead(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := cli.RunVersion(nil, &stdout, &stderr, version.Current())

	if exit != cli.ExitClean {
		t.Errorf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want silence on the success path", stderr.String())
	}
	if got, want := stdout.String(), version.Current().Page(); got != want {
		t.Errorf("stdout = %q, want the page %q", got, want)
	}
}

// TestRunVersion_ReachesNothingButItsFacts fences the command's own file: the
// half of `hyper version` that lives beside `check` may import nothing but its
// streams and the facts it was handed. internal/version carries the same fence
// over its transitive graph; between them, no path from this command reaches a
// network, a file, or the environment (ADR-0016, ADR-0019, issue #103).
func TestRunVersion_ReachesNothingButItsFacts(t *testing.T) {
	allowed := map[string]bool{
		`"fmt"`: true,
		`"io"`:  true,
		`"github.com/TheLoomLabs/hyper/internal/version"`: true,
	}

	file, err := parser.ParseFile(token.NewFileSet(), "version.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("internal/cli/version.go imports %s; the command reaches nothing but its streams and its facts", imp.Path.Value)
		}
	}
}
