package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stampFlag is the whole of the stamp: one flag and one symbol, the same
// invocation docs/build/releasing.md states for a human and scripts/release.sh
// runs for a tag (§11, ADR-0020, issue #191).
//
// It is spelled here rather than imported because the linker takes a string:
// `-X` names the symbol by its import path, which is text the compiler never
// checks, and a case that derived the path from the package it is stamping
// could not tell a working flag from a typo the linker silently ignored.
const stampFlag = "-X github.com/TheLoomLabs/hyper/internal/version.Version="

// unstamped is what build is handed where the case is about a build that named
// no version — the invocation a `go install` or a bare `go build` makes.
const unstamped = ""

// TestStamp_TheLinkerWritesTheVersionTheBuildNames is the criterion the
// declaration was converted from a `const` for: a build that names a version
// produces a binary that reports it, and the reporting surface is the real
// one — a process started from the built bytes, not cli.Main called in this
// one (issue #191).
//
// Against a `const` this case fails rather than errors, which is the fault
// worth having a case for: `-X` against a constant is inlined away and the
// flag is ignored without complaint, so nothing but running the binary tells
// the two builds apart.
func TestStamp_TheLinkerWritesTheVersionTheBuildNames(t *testing.T) {
	binary := build(t, "1.4.0-test")

	stdout, _, exit := run(t, binary, "version")

	if exit != 0 {
		t.Fatalf("hyper version exit = %d, want 0", exit)
	}
	if want := "hyper 1.4.0-test\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("the page starts %q, want it to start %q", stdout, want)
	}
}

// TestStamp_AnUnstampedBuildReportsUnknown is the other half of the same fact,
// read from the binary rather than from the package: a build invoked without
// the flag reports the word every unstamped fact on the page reports, and
// never an empty first line (issue #191).
func TestStamp_AnUnstampedBuildReportsUnknown(t *testing.T) {
	binary := build(t, unstamped)

	stdout, _, exit := run(t, binary, "version")

	if exit != 0 {
		t.Fatalf("hyper version exit = %d, want 0", exit)
	}
	if want := "hyper unknown\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("the page starts %q, want it to start %q", stdout, want)
	}
}

// TestStamp_ThePinGateComparesTwoRealBinaries is the gate's two arms exercised
// the way §11 states them and not the way fixtures could: one repository, one
// pin, and two binaries that differ in nothing but the version their builds
// named. The pinned one is cleared to read the repository and the other Refuses
// `version-pin-mismatch` at exit `77`, quoting both versions (§9, §11,
// ADR-0020, issue #191).
//
// Until a build could stamp, the mismatch arm was reachable only by writing a
// pin no binary could carry, which tested the comparison against a value the
// world would never supply. Both arms here are versions a release could publish.
func TestStamp_ThePinGateComparesTwoRealBinaries(t *testing.T) {
	pinned, newer := build(t, "1.4.0"), build(t, "1.5.0")

	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "hyper.yaml"), "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")

	if _, stderr, exit := run(t, pinned, "check", "--repo-dir", repo); exit != 0 {
		t.Errorf("the pinned binary exits %d, want 0 — it is the version the repository names; stderr=%q", exit, stderr)
	}

	_, stderr, exit := run(t, newer, "check", "--repo-dir", repo)

	if exit != 77 {
		t.Fatalf("the newer binary exits %d, want 77; stderr=%q", exit, stderr)
	}
	if want := "refused: version-pin-mismatch\n  this binary is 1.5.0; the repository pins 1.4.0"; !strings.Contains(stderr, want) {
		t.Errorf("the Refusal is %q, want it to carry %q", stderr, want)
	}
}

// build compiles cmd/hyper into a temporary file, stamped with version where
// one is given and invoked bare where it is unstamped. What it shares with the
// release is the flag and not the invocation — scripts/release.sh cross-compiles
// and trims paths besides, and what these cases are about is the stamp.
func build(t *testing.T, version string) string {
	t.Helper()
	needTools(t, "go")

	binary := filepath.Join(t.TempDir(), "hyper")
	args := []string{"build", "-o", binary}
	if version != "" {
		args = append(args, "-ldflags", stampFlag+version)
	}

	command := exec.Command("go", append(args, "./cmd/hyper")...)
	command.Dir = root(t)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return binary
}

// run invokes a built binary and answers its two streams and its exit code. It
// runs from a directory of its own so that nothing a case asserts depends on
// the repository the test binary was compiled in — `hyper version` answers
// identically anywhere, and every other command here is told its repository by
// flag.
func run(t *testing.T, binary string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()

	var out, errs bytes.Buffer
	command := exec.Command(binary, args...)
	command.Dir = t.TempDir()
	command.Stdout, command.Stderr = &out, &errs

	err := command.Run()
	var exited *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exited):
		exit = exited.ExitCode()
	default:
		t.Fatalf("%s %s: %v", binary, strings.Join(args, " "), err)
	}
	return out.String(), errs.String(), exit
}
