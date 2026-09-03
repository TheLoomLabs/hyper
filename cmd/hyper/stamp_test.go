package main

import (
	"bytes"
	"debug/buildinfo"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// flagless is what build is handed where the case is about a build that passes
// no `-ldflags` — the invocation a plain `go install` or `go build` makes.
//
// It is not the same as *unversioned*, and has not been since issue #263: such
// a build names no version at the link step and still reports one, read out of
// what the toolchain recorded. What it never carries is a version somebody
// chose.
const flagless = ""

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
//
// **Since issue #263 it also holds which of two stampers won.** The same build
// carries a module version Go derived from this checkout, and it is a different
// string; the flag decides, which is what leaves every release archive
// unchanged (`scripts/release.sh`, and
// TestRelease_TheArtefactCarriesTheBinaryTheTagNames on the published bytes).
func TestStamp_TheLinkerWritesTheVersionTheBuildNames(t *testing.T) {
	binary := build(t, "1.4.0-test")

	stdout, _, exit := run(t, binary, "version")

	if exit != 0 {
		t.Fatalf("hyper version exit = %d, want 0", exit)
	}
	if want := "hyper 1.4.0-test\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("the page starts %q, want it to start %q", stdout, want)
	}
	if module := moduleVersionOf(t, binary); module == "" || module == "1.4.0-test" || module == "v1.4.0-test" {
		t.Errorf("this build carries module version %q, and the case cannot tell the two stampers apart unless it differs from the flag's %q", module, "1.4.0-test")
	}
}

// TestStamp_AFlaglessBuildAtATagReportsTheTag is the other half of the same
// fact, and the one issue #263 changed: a build invoked without the flag used
// to report `unknown` and Refuse the pin gate in every repository it touched,
// over a version the toolchain had already written into it. It now reports the
// tag its source sits at, and the repository pinning that version clears it.
//
// **The tag is the whole reason this builds somewhere else.** The checkout a
// case runs in sits wherever its author left it — at a commit after the last
// release, mid-change, or both, none of which is a tag — so the state this case
// is about is one no assertion against `root(t)` could reach.
func TestStamp_AFlaglessBuildAtATagReportsTheTag(t *testing.T) {
	binary := buildIn(t, taggedRepository(t, "v1.4.0-test"), flagless)

	stdout, _, exit := run(t, binary, "version")

	if exit != 0 {
		t.Fatalf("hyper version exit = %d, want 0", exit)
	}
	if want := "hyper 1.4.0-test\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("the page starts %q, want it to start %q — the tag carries the `v` and no version hyper states does (§11)", stdout, want)
	}

	repo := pinning(t, "1.4.0-test")

	if _, stderr, exit := run(t, binary, "check", "--repo-dir", repo); exit != 0 {
		t.Errorf("a flagless build at the tag exits %d against a repository pinning it, want 0; stderr=%q", exit, stderr)
	}
}

// TestStamp_AFlaglessBuildFromAnEditedTreeDoesNotClaimTheRelease is where the
// fallback stops. Go marks a module version `+dirty` from the same `git status`
// that decides `vcs.modified`, so a build from a tree carrying edits states a
// version no release published and Refuses the pin gate — which is the whole of
// what keeps the row above from being a binary claiming to be a release it is
// not (issue #263).
//
// The page then carries two dirt markers from two stampers — `1.4.0-test+dirty`
// on the first line and the commit's `-dirty` on the second. They are the same
// fact said twice and neither is wrong.
func TestStamp_AFlaglessBuildFromAnEditedTreeDoesNotClaimTheRelease(t *testing.T) {
	tree := taggedRepository(t, "v1.4.0-test")
	writeFile(t, filepath.Join(tree, "edited"), "a file the commit does not account for\n")

	binary := buildIn(t, tree, flagless)

	stdout, _, exit := run(t, binary, "version")

	if exit != 0 {
		t.Fatalf("hyper version exit = %d, want 0", exit)
	}
	if want := "hyper 1.4.0-test+dirty\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("the page starts %q, want it to start %q", stdout, want)
	}

	repo := pinning(t, "1.4.0-test")

	_, stderr, exit := run(t, binary, "check", "--repo-dir", repo)

	if exit != 77 {
		t.Fatalf("a build from an edited tree exits %d against a repository pinning the tag, want 77; stderr=%q", exit, stderr)
	}
	if want := "this binary is 1.4.0-test+dirty; the repository pins 1.4.0-test"; !strings.Contains(stderr, want) {
		t.Errorf("the Refusal is %q, want it to carry %q", stderr, want)
	}
}

// TestStamp_TheReleaseScriptNamesTheSymbolThatWorks is the fence issue #263
// moved rather than removed. `-X` names its symbol by import path, which is
// text the linker checks against nothing: a symbol it does not recognise is
// ignored without complaint, and until #263 such a typo published a binary
// reporting `unknown` that `TestRelease_TheArtefactCarriesTheBinaryTheTagNames`
// caught by running it.
//
// **It no longer catches it where it matters.** A release is cut from a clean
// checkout standing on the tag, and there the module version the fallback reads
// is the same string the flag would have written — so a dead flag publishes a
// correct binary and nothing fails. That is a good outcome and a lost check,
// and what is left to hold is the spelling itself.
//
// So the script's symbol is compared against this package's, which is not two
// hand-spellings agreeing about nothing:
// `TestStamp_TheLinkerWritesTheVersionTheBuildNames` builds a binary with
// `stampFlag` and reads the version back out of the process, so the constant
// above is a symbol demonstrated to work rather than one somebody typed.
func TestStamp_TheReleaseScriptNamesTheSymbolThatWorks(t *testing.T) {
	path := filepath.Join(root(t), "scripts", "release.sh")
	script, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the release script is what stamps a release, and it cannot be read: %v", err)
	}

	assignment := regexp.MustCompile(`(?m)^stamp="([^"]*)"$`).FindStringSubmatch(string(script))
	if assignment == nil {
		t.Fatalf("%s assigns no `stamp=\"…\"`; the symbol the release passes to -X is not where this case can read it", path)
	}

	symbol := strings.TrimSuffix(strings.TrimPrefix(stampFlag, "-X "), "=")
	if assignment[1] != symbol {
		t.Errorf("the release stamps %q and the symbol this package builds with is %q — a release built with the first would fall back to the module version, which is right at the tag and wrong everywhere else", assignment[1], symbol)
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

	repo := pinning(t, "1.4.0")

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

// pinning is a repository whose declaration names one version of `hyper` and
// nothing else a case here reads. The digest beside the pin is zeroes: it is a
// required key that nothing local ever reads — the gate compares the version
// (§11, ADR-0020) — so a real one would be a fact the case pretends to have.
func pinning(t *testing.T, version string) string {
	t.Helper()

	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "hyper.yaml"), "kind: repository-declaration\nversion: "+version+"\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")
	return repo
}

// build compiles cmd/hyper out of the checkout the case is running in, stamped
// with version where one is given and invoked bare where none is. It is the
// default every case but two takes, which is why it stands in front of buildIn
// rather than being inlined into each of them. What it shares with the release
// is the flag and not the invocation — scripts/release.sh cross-compiles and
// trims paths besides, and what these cases are about is the stamp.
func build(t *testing.T, version string) string {
	t.Helper()
	return buildIn(t, root(t), version)
}

// buildIn is the same build somewhere else: the module rooted at dir, which is
// the checkout for most cases and a repository of the case's own where what is
// being read is Go's own stamping of a version (issue #263).
func buildIn(t *testing.T, dir, version string) string {
	t.Helper()
	needTools(t, "go")

	binary := filepath.Join(t.TempDir(), "hyper")
	args := []string{"build", "-o", binary}
	if version != "" {
		args = append(args, "-ldflags", stampFlag+version)
	}

	command := exec.Command("go", append(args, "./cmd/hyper")...)
	command.Dir = dir
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return binary
}

// taggedRepository is this module's source in a git repository of its own, all
// of it in one commit, tagged. It answers the directory.
//
// **A case cannot get this state from the checkout it runs in.** Go derives a
// build's module version from the repository the source sits in, and what a
// contributor's checkout says is whatever their `git log` and `git status` say
// — a pseudo-version, usually, and `+dirty` mid-change. So the repository whose
// tag is the subject is built here rather than assumed, and the tag is one
// nothing publishes.
//
// **The tag must be a version this module path could carry.** Go declines to
// stamp a major it would need an import-path suffix for, so `v9.9.9` on
// `github.com/TheLoomLabs/hyper` silently degrades to a pseudo-version and
// `v1.4.0-test` does not. That is a property of the toolchain, and the case
// that hands this a tag and asserts on it is what would catch it changing.
//
// What is copied is every `.go` file plus `go.mod` and `go.sum`, tracked and
// untracked alike, which is the whole of what `go build ./cmd/hyper` reads —
// nothing in this repository is embedded, and copying the testdata under
// `internal/` would be copying tens of megabytes no build opens.
func taggedRepository(t *testing.T, tag string) string {
	t.Helper()
	needTools(t, "git", "go")

	dir := t.TempDir()
	for _, path := range sourceFiles(t) {
		source, err := os.ReadFile(filepath.Join(root(t), path))
		if errors.Is(err, os.ErrNotExist) {
			// Tracked and deleted in the working tree: the repository
			// being built is the tree as it stands, so it is absent here too.
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		at := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, at, string(source))
	}

	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "add", "-A")
	git(t, dir, "-c", "user.name=hyper suite", "-c", "user.email=suite@hyper.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "the source this case builds")
	git(t, dir, "tag", tag)
	return dir
}

// sourceFiles is what taggedRepository copies: the module's Go source and the
// two files that resolve its dependencies, listed by git so that a file
// somebody has written and not yet added is carried like any other — a case
// that built the last commit instead of the working tree would report on code
// the author is not looking at.
func sourceFiles(t *testing.T) []string {
	t.Helper()

	command := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.go", "go.mod", "go.sum")
	command.Dir = root(t)
	out, err := command.Output()
	if err != nil {
		unavailable(t, "git ls-files failed in %s (%v); the source a case would build cannot be listed", root(t), err)
	}

	var files []string
	for _, path := range strings.Split(string(out), "\x00") {
		if path != "" {
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		t.Fatalf("git listed no source files in %s; there is nothing here to build", root(t))
	}
	return files
}

// git runs one git command in a repository the case owns, and fails naming it.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
}

// moduleVersionOf reads `Main.Version` out of a built binary — the second of
// the two stampers that can name a version, read off a file so that a case can
// say which of them the page is quoting (issue #263).
func moduleVersionOf(t *testing.T, binary string) string {
	t.Helper()

	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatalf("no build information in %s: %v", binary, err)
	}
	return info.Main.Version
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
