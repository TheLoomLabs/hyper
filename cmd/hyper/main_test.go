package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// dispatch is the one line of main(), with the two streams redirected and
// nothing else changed: the same entry point, handed the same five reads of
// the same process. The cases below are about what the binary does when it is
// standing somewhere in particular, so they drive the real os.LookupEnv,
// os.Getwd and build facts rather than stand-ins — cli.Main's own behaviour
// against a fabricated process is internal/cli/main_test.go's (issue #107) —
// including the clock, which nothing behind the dispatch calls yet and which
// this file therefore drives as the real time.Now, exactly as main() does.
//
// Keeping this a single call is the point: anything it did that main() does
// not would be a second entry point, and these cases would stop testing the
// binary.
func dispatch(args []string, stdout, stderr io.Writer) int {
	return cli.Main(args, stdout, stderr, os.LookupEnv, os.Getwd, time.Now, version.Current())
}

func TestDispatch_NoArgsIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := dispatch(nil, &stdout, &stderr)
	if got != 2 {
		t.Errorf("dispatch(nil) exit = %d, want 2", got)
	}
	if stderr.Len() == 0 {
		t.Errorf("dispatch(nil) wrote nothing to stderr, want a usage message")
	}
}

func TestDispatch_UnknownCommandIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := dispatch([]string{"bogus"}, &stdout, &stderr)
	if got != 2 {
		t.Errorf(`dispatch(["bogus"]) exit = %d, want 2`, got)
	}
}

// TestDispatch_VersionAnswersFromAnyDirectory drives the real dispatch — the
// facts read from the running build, the working directory whatever the
// process is standing in — against the three repository contexts issue #103
// requires `hyper version` to answer identically in: outside a git tree, inside
// a repository with no hyper.yaml, and inside one whose pin is a different
// version entirely. The last is the one that matters: every one of the sixteen
// Refuses there with 77, and this command is exempt (§9, ADR-0020).
func TestDispatch_VersionAnswersFromAnyDirectory(t *testing.T) {
	outsideAnyRepo := t.TempDir()

	noDeclaration := t.TempDir()
	fakeGitRoot(t, noDeclaration)

	foreignPin := t.TempDir()
	fakeGitRoot(t, foreignPin)
	writeFile(t, filepath.Join(foreignPin, "hyper.yaml"), "kind: repository-declaration\nversion: 9.9.9\n")

	for name, dir := range map[string]string{
		"outside a git repository":   outsideAnyRepo,
		"a repository with no pin":   noDeclaration,
		"a repository pinning 9.9.9": foreignPin,
	} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(dir)

			var stdout, stderr bytes.Buffer
			exit := dispatch([]string{"version"}, &stdout, &stderr)

			if exit != 0 {
				t.Errorf("exit = %d, want 0; stderr=%q", exit, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want silence on the success path", stderr.String())
			}
			if got, want := stdout.String(), version.Current().Page(); got != want {
				t.Errorf("stdout = %q, want %q", got, want)
			}
			if lines := strings.Count(stdout.String(), "\n"); lines != 5 {
				t.Errorf("stdout has %d lines, want 5", lines)
			}
		})
	}
}

// TestDispatch_VersionAndTheRefusalQuoteOneConstant is the acceptance
// criterion that the two readings cannot drift: the version on `hyper
// version`'s first line is byte-identical to the one the Refusal calls *this
// binary* when the pin gate declines. One constant, read here through the two
// surfaces that quote it (§9, §11, ADR-0020, issue #103).
func TestDispatch_VersionAndTheRefusalQuoteOneConstant(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "hyper.yaml"), "kind: repository-declaration\nversion: 9.9.9\n")

	var page, pageErr bytes.Buffer
	if exit := dispatch([]string{"version"}, &page, &pageErr); exit != 0 {
		t.Fatalf("hyper version exit = %d; stderr=%q", exit, pageErr.String())
	}
	firstLine, _, _ := strings.Cut(page.String(), "\n")
	stated := strings.TrimPrefix(firstLine, "hyper ")

	var refusalOut, refusal bytes.Buffer
	if exit := dispatch([]string{"check", "--repo-dir", repo}, &refusalOut, &refusal); exit != 77 {
		t.Fatalf("hyper check exit = %d, want 77; stderr=%q", exit, refusal.String())
	}

	if want := "this binary is " + stated + ";"; !strings.Contains(refusal.String(), want) {
		t.Errorf("the Refusal is %q, want it to quote the version the page states: %q", refusal.String(), want)
	}
	if stated != version.Version {
		t.Errorf("the page states %q, want the one constant %q", stated, version.Version)
	}
}

// TestDispatch_VersionNeedsNoWorkingDirectory pins the dispatch's own half
// of the exemption: the working directory is resolved lazily, inside the
// commands that read a repository, so a `version` invocation never depends on
// there being one. The probe is a process standing in a directory that has
// been removed — the one state where reading the working directory genuinely
// fails, and where a `wd` resolved above the switch would take this command
// down with it.
//
// The probe needs a platform where a process can outlive its own directory,
// which is every one hyper targets. Where it cannot be built the test skips
// rather than passes vacuously — a skip is visible in a test run, and a
// criterion nothing asserts is not.
func TestDispatch_VersionNeedsNoWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	removeErr := os.Remove(dir)
	_, getwdErr := os.Getwd()
	if removeErr != nil || getwdErr == nil {
		t.Skipf("this platform keeps the working directory readable after removal (remove: %v, getwd: %v); the probe cannot be built here", removeErr, getwdErr)
	}

	var stdout, stderr bytes.Buffer
	if exit := dispatch([]string{"version"}, &stdout, &stderr); exit != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, stderr.String())
	}
	if got, want := stdout.String(), version.Current().Page(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

// TestDispatch_VersionRejectsAnArgument checks the rejection reaches the
// exit code through the real dispatch; the argument shapes themselves are
// the cli package's corpus and table.
func TestDispatch_VersionRejectsAnArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := dispatch([]string{"version", "--json"}, &stdout, &stderr); exit != 2 {
		t.Errorf("exit = %d, want 2", exit)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it untouched", stdout.String())
	}
}

// TestDispatch_CompletionsAnswersFromAnyDirectory drives the real dispatch
// against the three repository contexts issue #104 requires `hyper
// completions` to answer identically in: outside a git tree, inside a
// repository with no hyper.yaml, and inside one whose pin is a different
// version entirely. Every one of the sixteen Refuses with 77 in the last of
// them, and this command is exempt for the reason the whole criterion exists
// — shell setup in a dotfiles bootstrap runs before any repository does (§9,
// ADR-0020).
func TestDispatch_CompletionsAnswersFromAnyDirectory(t *testing.T) {
	outsideAnyRepo := t.TempDir()

	noDeclaration := t.TempDir()
	fakeGitRoot(t, noDeclaration)

	foreignPin := t.TempDir()
	fakeGitRoot(t, foreignPin)
	writeFile(t, filepath.Join(foreignPin, "hyper.yaml"), "kind: repository-declaration\nversion: 9.9.9\n")

	for name, dir := range map[string]string{
		"outside a git repository":   outsideAnyRepo,
		"a repository with no pin":   noDeclaration,
		"a repository pinning 9.9.9": foreignPin,
	} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(dir)

			for _, shell := range cli.Shells() {
				var stdout, stderr bytes.Buffer
				exit := dispatch([]string{"completions", shell}, &stdout, &stderr)

				if exit != 0 {
					t.Errorf("%s: exit = %d, want 0; stderr=%q", shell, exit, stderr.String())
				}
				if stderr.Len() != 0 {
					t.Errorf("%s: stderr = %q, want silence on the success path", shell, stderr.String())
				}
				if !strings.Contains(stdout.String(), "hyper") {
					t.Errorf("%s: stdout = %q, want the completion script", shell, stdout.String())
				}
			}
		})
	}
}

// TestDispatch_CompletionsRejectsWhatIsNotOneShell checks the rejection
// reaches the exit code through the real dispatch; the argument shapes
// themselves are the cli package's corpus and table.
func TestDispatch_CompletionsRejectsWhatIsNotOneShell(t *testing.T) {
	for _, args := range [][]string{
		{"completions"},
		{"completions", "nushell"},
		{"completions", "BASH"},
		{"completions", "bash", "zsh"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if exit := dispatch(args, &stdout, &stderr); exit != 2 {
				t.Errorf("exit = %d, want 2", exit)
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

// fakeGitRoot makes dir look like the top of a git working tree to the only
// thing that asks — repository.FindGitRoot, which walks up looking for a .git
// entry and reads nothing inside it. No git binary is run, and the directory is
// not a repository in any other sense.
func fakeGitRoot(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
