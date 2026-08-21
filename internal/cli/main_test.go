package cli_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
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

// process stands in for the six reads cli.Main is handed rather than makes, and
// records whether any of them was touched. Counting the calls is the only way
// the exemption can be asserted rather than assumed: `version` and
// `completions` are exempt from the pin gate because they resolve no
// repository, and a branch that never asks where it is standing cannot have
// reached one (§9, ADR-0020).
//
// It counts all six and not only the three a command reaches today, which is
// the point of the value being one type: the members the milestone threaded
// ahead of their callers are asserted untouched by exactly the cases that
// assert it of the older three, and the day one is called is the day these
// cases say which command called it (issue #134).
type process struct {
	wd        string
	wdErr     error
	getwd     int
	lookupenv int
	now       int
	mint      int
	dial      int
	exec      int
}

// value is the six as cli.Main takes them: one value, wired to the counting
// methods beneath it. It is what every case here hands the entry point, so that
// a case reads as the invocation it is about rather than as an assembly of
// stand-ins.
func (p *process) value() cli.Process {
	return cli.Process{
		LookupEnv: p.LookupEnv,
		Getwd:     p.Getwd,
		Now:       p.Now,
		Mint:      p.Mint,
		Dial:      p.Dial,
		Exec:      p.Exec,
	}
}

func (p *process) Getwd() (string, error) {
	p.getwd++
	return p.wd, p.wdErr
}

// LookupEnv answers an environment with nothing in it: every variable is
// absent, which is what the second return says. No case here needs one set —
// HYPER_REPO_DIR is resolveRepoRoot's, and check_test.go's — so the whole of
// what this stands for is the count beside it.
func (p *process) LookupEnv(string) (string, bool) {
	p.lookupenv++
	return "", false
}

// Now answers one fixed instant, and counts the reads of it beside the others.
// It is here rather than passed as a bare time.Now because the day a command
// reads a clock is the day these cases must be able to say which ones did.
func (p *process) Now() time.Time {
	p.now++
	return fixedInstant
}

// Mint answers a real id at whatever instant it is handed, and counts the mint.
// Nothing behind the dispatch mints one in this milestone — the read is
// threaded ahead of the Run that makes it (issue #134) — and where an id is
// never rendered here, a real one costs nothing and lies about nothing.
func (p *process) Mint(now time.Time) store.RunID {
	p.mint++
	return store.MintRunID(now)
}

// Dial reaches nothing: this stand-in is a process with no network under it, so
// every connection is refused and the count beside it is what a case asserts.
func (p *process) Dial(context.Context, string, string) (net.Conn, error) {
	p.dial++
	return nil, errors.New("this process dials nothing")
}

// Exec starts nothing, and answers nothing to wait on: a case that reached this
// has already failed its count, and there is no child here for it to go on to
// run.
func (p *process) Exec(context.Context, []string) *exec.Cmd {
	p.exec++
	return nil
}

// fixedInstant is the clock the stand-in answers with: one instant, so a case
// that ever compares a rendering of it compares a fixed string rather than the
// time the suite happened to run.
var fixedInstant = time.Date(2026, time.April, 2, 9, 41, 14, 221_000_000, time.UTC)

// untouched says the process was never read, which is the shape of the
// exemption: nothing of the six, and therefore no repository root and no gate.
func (p *process) untouched(t *testing.T) {
	t.Helper()
	for _, read := range []struct {
		what  string
		count int
	}{
		{"the working directory was resolved", p.getwd},
		{"the environment was read", p.lookupenv},
		{"the clock was read", p.now},
		{"a Run id was minted", p.mint},
		{"a host was dialled", p.dial},
		{"a child process was started", p.exec},
	} {
		if read.count != 0 {
			t.Errorf("%s %d times, want it left alone", read.what, read.count)
		}
	}
}

// TestMain_NoArgumentsIsUsageError is a bare `hyper`: nothing to dispatch on,
// so no command runs and none of the process is read.
func TestMain_NoArgumentsIsUsageError(t *testing.T) {
	p := &process{wd: t.TempDir()}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main(nil, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitUsage {
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

	if exit := cli.Main([]string{"bogus"}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitUsage {
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

			exit := cli.Main(argv, &stdout, &stderr, p.value(), testFacts)

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

	if exit := cli.Main([]string{"version"}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
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

	if exit := cli.Main([]string{"check"}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitRefused {
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

	if exit := cli.Main([]string{"check"}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d; stderr=%q", exit, cli.ExitProblems, stderr.String())
	}
	if got := stderr.String(); !strings.Contains(got, "getwd: no such file or directory") {
		t.Errorf("stderr = %q, want it to carry the error", got)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it untouched", stdout.String())
	}
}
