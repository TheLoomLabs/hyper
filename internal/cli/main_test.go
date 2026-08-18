package cli_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// testFacts is a build's facts with every member stamped, so that a case
// comparing stdout compares a fixed page rather than one that changes with the
// commit under test. Its version is not the running binary's, which is what
// makes it usable as the pin gate's other side (issue #107).
var testFacts = version.Facts{
	Version:   "1.4.0",
	Commit:    "0123456789abcdef0123456789abcdef01234567",
	Built:     "2026-01-01T00:00:00Z",
	Toolchain: "go1.24.0",
	OS:        "linux",
	Arch:      "amd64",
}

// process stands in for the two reads cli.Main is handed rather than makes —
// the environment and the working directory — and records whether either was
// touched. Counting the calls is the only way the exemption can be asserted
// rather than assumed: `version` and `completions` are exempt from the pin gate
// because they resolve no repository, and a branch that never asks where it is
// standing cannot have reached one (§9, ADR-0020).
type process struct {
	wd     string
	wdErr  error
	getwd  int
	getenv int
}

func (p *process) Getwd() (string, error) {
	p.getwd++
	return p.wd, p.wdErr
}

// Getenv answers an environment with nothing in it. No case here needs a
// variable set — HYPER_REPO_DIR is resolveRepoRoot's, and check_test.go's — so
// the whole of what this stands for is the count beside it.
func (p *process) Getenv(string) string {
	p.getenv++
	return ""
}

// untouched says the process was never read, which is the shape of the
// exemption: no working directory, no environment, and therefore no repository
// root and no gate.
func (p *process) untouched(t *testing.T) {
	t.Helper()
	if p.getwd != 0 {
		t.Errorf("the working directory was resolved %d times, want it left alone", p.getwd)
	}
	if p.getenv != 0 {
		t.Errorf("the environment was read %d times, want it left alone", p.getenv)
	}
}

// TestMain_NoArgumentsIsUsageError is a bare `hyper`: nothing to dispatch on,
// so no command runs and none of the process is read.
func TestMain_NoArgumentsIsUsageError(t *testing.T) {
	p := &process{wd: t.TempDir()}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main(nil, &stdout, &stderr, p.Getenv, p.Getwd, testFacts); exit != cli.ExitUsage {
		t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if !strings.HasPrefix(stderr.String(), "usage: hyper ") {
		t.Errorf("stderr = %q, want a usage line", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it untouched", stdout.String())
	}
	p.untouched(t)
}

// TestMain_UnknownCommandNamesWhatWasTyped is a word that is not one of the
// eighteen. It exits 2 quoting what was typed, and it never reaches the gate:
// the gate compares a repository's pin, a repository root is resolved from the
// environment or the working directory, and neither was read.
func TestMain_UnknownCommandNamesWhatWasTyped(t *testing.T) {
	p := &process{wd: t.TempDir()}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main([]string{"bogus"}, &stdout, &stderr, p.Getenv, p.Getwd, testFacts); exit != cli.ExitUsage {
		t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if got := stderr.String(); !strings.Contains(got, `"bogus"`) {
		t.Errorf("stderr = %q, want it to name what was typed", got)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it untouched", stdout.String())
	}
	p.untouched(t)
}

// TestMain_ExemptCommandsReadNothingOfTheProcess is the exemption as a property
// of the dispatch itself (§9, ADR-0020). The working directory is handed over
// as a function rather than a value, so the two branches that call no gate call
// nothing to resolve one either — and the probe makes that unmissable by
// handing over a working directory that cannot be read at all. A `wd` resolved
// above the switch would take both these commands down with it.
func TestMain_ExemptCommandsReadNothingOfTheProcess(t *testing.T) {
	for name, argv := range map[string][]string{
		"version":     {"version"},
		"completions": {"completions", "bash"},
	} {
		t.Run(name, func(t *testing.T) {
			p := &process{wdErr: errors.New("getwd: no such file or directory")}
			var stdout, stderr bytes.Buffer

			exit := cli.Main(argv, &stdout, &stderr, p.Getenv, p.Getwd, testFacts)

			if exit != cli.ExitClean {
				t.Errorf("exit = %d, want 0; stderr=%q", exit, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want silence on the success path", stderr.String())
			}
			if stdout.Len() == 0 {
				t.Error("stdout is empty, want the command's output on it")
			}
			p.untouched(t)
		})
	}
}

// TestMain_VersionStatesTheFactsItWasHanded pins the threading the signature
// exists for: the whole of version.Facts reaches RunVersion, so the page under
// test is the one the value describes rather than the one the running build
// would assemble (issue #103, issue #107).
func TestMain_VersionStatesTheFactsItWasHanded(t *testing.T) {
	p := &process{wd: t.TempDir()}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main([]string{"version"}, &stdout, &stderr, p.Getenv, p.Getwd, testFacts); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, stderr.String())
	}
	if got, want := stdout.String(), testFacts.Page(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

// TestMain_CheckIsGatedOnTheVersionInThoseFacts is the other half of the same
// threading, and the one that matters: the gate compares hyper.yaml's pin
// against the version out of the facts, so a Refusal quotes the same constant
// the page states. The repository pins 9.9.9 and the facts say 1.4.0 (§11,
// ADR-0020).
func TestMain_CheckIsGatedOnTheVersionInThoseFacts(t *testing.T) {
	repo := t.TempDir()
	// A .git entry, because repository-root resolution walks up from the
	// working directory looking for one, and hyper.yaml pinning a version the
	// facts below are not.
	writeFile(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(repo, "hyper.yaml"), "kind: repository-declaration\nversion: 9.9.9\n")

	p := &process{wd: repo}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main([]string{"check"}, &stdout, &stderr, p.Getenv, p.Getwd, testFacts); exit != cli.ExitRefused {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitRefused, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want silence: a Refusal is not a row", stdout.String())
	}
	for _, want := range []string{testFacts.Version, "9.9.9"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("the Refusal is %q, want it to name %q", stderr.String(), want)
		}
	}
	if p.getwd == 0 {
		t.Error("the working directory was never resolved, want a gated command to ask where it is standing")
	}
}

// TestMain_CheckReportsAWorkingDirectoryItCannotRead is the failure the lazy
// resolution moved rather than removed: a gated command does ask where it is
// standing, and where the answer is an error it says so and stops.
func TestMain_CheckReportsAWorkingDirectoryItCannotRead(t *testing.T) {
	p := &process{wdErr: errors.New("getwd: no such file or directory")}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main([]string{"check"}, &stdout, &stderr, p.Getenv, p.Getwd, testFacts); exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d; stderr=%q", exit, cli.ExitProblems, stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, "getwd: no such file or directory") {
		t.Errorf("stderr = %q, want it to carry the error", got)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it untouched", stdout.String())
	}
}
