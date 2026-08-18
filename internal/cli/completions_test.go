package cli_test

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// completionsCorpus is the completions command's own slice of testdata/, a
// sibling of check/ and version/ rather than a directory inside either: a case
// belongs to exactly one harness, and no harness runs another's cases (issue
// #101). What is here is how a completions case is driven — everything else it
// inherits from golden_test.go.
const completionsCorpus = "testdata/completions"

// TestCompletionsGolden drives cli.RunCompletions end to end, one directory
// per case under testdata/completions/ (issue #104). Each case supplies argv —
// the complete command line as typed — and its golden files (stdout, stderr,
// exit) are compared byte-for-byte and regenerated behind -update.
//
// Three of the cases hold a whole script, which is what makes a change to any
// of the three shells a diff a reviewer reads rather than a behaviour only a
// shell could show them.
func TestCompletionsGolden(t *testing.T) {
	for _, name := range corpusCases(t, completionsCorpus) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(completionsCorpus, name)
			args := readArgv(t, filepath.Join(dir, "argv"), "completions")

			var stdout, stderr bytes.Buffer
			exit := cli.RunCompletions(args, &stdout, &stderr)

			compareGolden(t, dir, stdout.Bytes(), stderr.Bytes(), exit)
		})
	}
}

// TestCompletions_EveryScriptNamesTheWholeSurface is the acceptance criterion
// that the three scripts describe one surface: each names all eighteen
// commands and `store init`, because each is built from the same list. A shell
// that fell behind the other two would fail here rather than on the machine of
// the operator who uses that shell.
func TestCompletions_EveryScriptNamesTheWholeSurface(t *testing.T) {
	for _, shell := range cli.Shells() {
		t.Run(shell, func(t *testing.T) {
			script := scriptFor(t, shell)

			for _, name := range append(cli.Commands(), "init") {
				if !regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(script) {
					t.Errorf("the %s script never names %q", shell, name)
				}
			}
			if !strings.Contains(script, "store") {
				t.Errorf("the %s script never names store, so `store init` cannot be reached", shell)
			}
			for _, flag := range cli.Globals() {
				if written := globalAsWritten(shell, flag); !strings.Contains(script, written) {
					t.Errorf("the %s script never names the global %q (as %q)", shell, flag, written)
				}
			}
			for _, other := range cli.Shells() {
				if !regexp.MustCompile(`\b` + other + `\b`).MatchString(script) {
					t.Errorf("the %s script never names the shell %q, which it completes after `completions`", shell, other)
				}
			}
		})
	}
}

// globalAsWritten is a global in the spelling its shell declares it in. bash
// and zsh complete the word a caller types, dashes included; fish declares a
// long option by its name alone, after -l, and writes the dashes itself. The
// difference is fish's own and not a second spelling of the flag.
func globalAsWritten(shell, flag string) string {
	if shell == "fish" {
		return "-l " + strings.TrimPrefix(flag, "--")
	}
	return flag
}

// commandSubstitution finds every place a script would start a process: bash's
// and zsh's $(…), and the (…) fish evaluates inside a completion's argument
// list. Backticks are matched too, though nothing writes them, because a
// script that grew one would otherwise slip past.
var commandSubstitution = regexp.MustCompile("\\$\\(([^)]*)\\)|`([^`]*)`|'\\(([^)]*)\\)'")

// completionTimeCommands is what a script is allowed to run while the cursor
// waits: bash's compgen and shopt, zsh's _path_files and fish's directory
// helper, each shipped by the shell itself and none of them reading anything
// but the shell's own state and the directory the cursor is in. Nothing else
// may appear, and `hyper` least of all — a completion is a keypress, and
// running a gated command behind one can Refuse 77, read a repository the
// caller is not in, or block on a slow filesystem (issue #104).
var completionTimeCommands = []string{"compgen", "shopt", "_path_files", "__fish_complete_directories"}

// TestCompletions_NoScriptRunsAnythingButTheShellsOwnHelpers is that decision
// as a test. It reads every command substitution in every script rather than
// searching for the string `hyper`, which would be the weaker check twice
// over: the scripts name hyper constantly — in `complete -c hyper`, in
// `_hyper`, in every comment — and a script invoking `git` or `curl` behind
// TAB would be just as wrong.
func TestCompletions_NoScriptRunsAnythingButTheShellsOwnHelpers(t *testing.T) {
	for _, shell := range cli.Shells() {
		t.Run(shell, func(t *testing.T) {
			script := scriptFor(t, shell)

			for _, match := range commandSubstitution.FindAllStringSubmatch(script, -1) {
				inner := strings.TrimSpace(match[1] + match[2] + match[3])
				command, _, _ := strings.Cut(inner, " ")
				if !slices.Contains(completionTimeCommands, command) {
					t.Errorf("the %s script runs %q at completion time (in %q); only %q may run there",
						shell, command, match[0], completionTimeCommands)
				}
			}

			// The other half of the same rule, and the one a substitution
			// cannot express: a script that sourced a file, read one, or
			// dialled out would do it as a plain command.
			for _, line := range strings.Split(script, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "#") {
					continue
				}
				for _, forbidden := range []string{"curl", "wget", "nc ", "/dev/tcp", "cat ", "source ", "eval "} {
					if strings.Contains(line, forbidden) {
						t.Errorf("the %s script line %q reaches %q at completion time", shell, line, forbidden)
					}
				}
			}
		})
	}
}

// TestCompletions_AreByteIdenticalEveryInvocation is the criterion that the
// scripts are compiled-in constants: nothing about the machine, the moment or
// the caller reaches their assembly, so two invocations are the same bytes.
// The signature carries the other half — no environment, no working directory
// and no repository is reachable from the entry point at all — and the golden
// files carry the third, being the bytes one machine wrote and every other
// must reproduce.
func TestCompletions_AreByteIdenticalEveryInvocation(t *testing.T) {
	for _, shell := range cli.Shells() {
		t.Run(shell, func(t *testing.T) {
			if first, second := scriptFor(t, shell), scriptFor(t, shell); first != second {
				t.Errorf("two invocations wrote different scripts:\n first:  %q\n second: %q", first, second)
			}
		})
	}
}

// TestRunCompletions_TakesExactlyOneShell walks the argument shapes a corpus
// cannot enumerate. Zero is a usage error and so is two, the known set is
// matched byte-exact and case-sensitively like every other name in the tool,
// and a global is not an argument this command takes: all of it is exit 2 with
// stdout untouched and the rendering on stderr (§9, ADR-0060, issue #104).
func TestRunCompletions_TakesExactlyOneShell(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"bash", "zsh"},
		{"bash", "bash"},
		{"nushell"},
		{"BASH"},
		{"Bash"},
		{"bash "},
		{"sh"},
		{"powershell"},
		{"--json"},
		{"--repo-dir", "/tmp"},
		{"--"},
		{"bash", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := cli.RunCompletions(args, &stdout, &stderr)

			if exit != cli.ExitUsage {
				t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want it untouched", stdout.String())
			}
			if !strings.Contains(stderr.String(), "known shells: bash, fish, zsh") {
				t.Errorf("stderr = %q, want it to name the known set alphabetically", stderr.String())
			}
		})
	}
}

// TestRunCompletions_WritesTheScriptAndNothingElse is the success path's own
// half: stdout carries the script, stderr is silent, and the exit is 0 (§9's
// stream discipline).
func TestRunCompletions_WritesTheScriptAndNothingElse(t *testing.T) {
	for _, shell := range cli.Shells() {
		t.Run(shell, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := cli.RunCompletions([]string{shell}, &stdout, &stderr)

			if exit != cli.ExitClean {
				t.Errorf("exit = %d, want %d", exit, cli.ExitClean)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want silence on the success path", stderr.String())
			}
			if stdout.Len() == 0 {
				t.Error("stdout is empty, want the script")
			}
			if !strings.HasSuffix(stdout.String(), "\n") {
				t.Error("the script does not end in a newline")
			}
		})
	}
}

// TestRunCompletions_TakesNoRepositoryToRead is the structural half of the
// three acceptance criteria about repositories: outside a git tree, inside one
// with no hyper.yaml, and inside one pinning something else entirely, `hyper
// completions bash` answers the same. The signature is the proof — it takes
// neither a working directory nor an environment, so there is no repository
// reachable from it to gate on (§9's exemption, ADR-0020). What a signature
// states a test can only restate; the behaviour those criteria describe is
// exercised against the real dispatch in cmd/hyper.
func TestRunCompletions_TakesNoRepositoryToRead(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := cli.RunCompletions([]string{"bash"}, &stdout, &stderr); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr.String())
	}
}

// TestRunCompletions_ReachesNothingButItsStreams fences the command's own
// file. It may import what it needs to assemble a string and write it, and
// nothing that could read a file, reach a network, or consult the environment —
// which is what makes the pin-gate exemption safe to state in a signature
// rather than enforce in a branch (ADR-0016, ADR-0019, ADR-0020).
func TestRunCompletions_ReachesNothingButItsStreams(t *testing.T) {
	allowed := map[string]bool{
		`"fmt"`:     true,
		`"io"`:      true,
		`"slices"`:  true,
		`"strings"`: true,
	}

	for _, source := range []string{"completions.go", "tree.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			if !allowed[imp.Path.Value] {
				t.Errorf("internal/cli/%s imports %s; the command assembles a string and writes it, and reaches nothing else", source, imp.Path.Value)
			}
		}
	}
}

// TestCompletions_TheBashScriptCompletesWhatSectionNineFixes is the one place
// a script is judged by a shell rather than by reading it: bash sources the
// script and is asked what it would offer at a cursor, once per rule §9 fixes.
// Only bash is driven this way — it is the shell a Go test can rely on finding
// on a runner, and the three scripts are built from one list, so what this
// proves about the surface it proves about all three. What it cannot prove for
// zsh and fish is their syntax, which their golden files hold for a reviewer.
//
// Where bash is absent the test skips rather than passes vacuously: a skip is
// visible in a test run, and a criterion nothing asserts is not.
func TestCompletions_TheBashScriptCompletesWhatSectionNineFixes(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("no bash on this machine (%v); the script cannot be driven here", err)
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "hyper.bash")
	if err := os.WriteFile(script, []byte(scriptFor(t, "bash")), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(bash, "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("the bash script does not parse: %v\n%s", err, out)
	}

	// Directories and a file, for the one path rule: --repo-dir offers
	// directories, and a regular file is not one. Two of the names are the
	// shapes an unguarded reply splits or expands on — a space is what IFS
	// would cut a directory in half at, and a glob character is what the
	// shell would expand against the others before the cursor ever saw it.
	for _, name := range []string{"alpha", "beta", "my repo", "v*"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "gamma.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name string
		// line is the command line as typed, its last word being what the
		// cursor stands on — empty for a bare TAB.
		line []string
		want []string
	}{
		{"the eighteen at position one", []string{"hyper", ""}, cli.Commands()},
		{"a prefix filters them", []string{"hyper", "pro"}, []string{"providers", "provider", "probe", "project"}},
		{"init after store, beside the globals", []string{"hyper", "store", ""}, append([]string{"init"}, cli.Globals()...)},
		{"the globals after a command in the tree", []string{"hyper", "check", ""}, cli.Globals()},
		{"the globals after store init", []string{"hyper", "store", "init", ""}, cli.Globals()},
		{"the three shells after completions", []string{"hyper", "completions", ""}, cli.Shells()},
		{"nothing after the shell completions named", []string{"hyper", "completions", "bash", ""}, nil},
		{"nothing at all after version", []string{"hyper", "version", ""}, nil},
		{"nothing after a word naming no command", []string{"hyper", "deploy", ""}, nil},
		{"no artefact name is ever offered", []string{"hyper", "review", ""}, cli.Globals()},
		{"no Provider name is ever offered", []string{"hyper", "provider", ""}, cli.Globals()},
		{"no Run id is ever offered", []string{"hyper", "show", ""}, cli.Globals()},
		{"directories after --repo-dir", []string{"hyper", "check", "--repo-dir", ""}, []string{"alpha", "beta", "my repo", "v*"}},
		{"and only the ones that match", []string{"hyper", "check", "--repo-dir", "al"}, []string{"alpha"}},
		{"a directory whose name carries a space is one reply", []string{"hyper", "check", "--repo-dir", "my"}, []string{"my repo"}},
		{"a directory whose name is a glob is itself", []string{"hyper", "check", "--repo-dir", "v"}, []string{"v*"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := completeWithBash(t, bash, dir, script, c.line)

			slices.Sort(got)
			want := slices.Clone(c.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("`%s<TAB>` offers %q, want %q", strings.Join(c.line, " "), got, want)
			}
		})
	}
}

// completeWithBash asks bash what the sourced script would offer for one
// command line, the cursor standing on its last word. COMP_WORDS and
// COMP_CWORD are what bash itself hands a completion function, so the harness
// puts the script in the position it will really be called in rather than
// calling its internals another way.
func completeWithBash(t *testing.T, bash, wd, script string, line []string) []string {
	t.Helper()

	var driver strings.Builder
	driver.WriteString("source " + script + "\n")
	driver.WriteString("COMP_WORDS=(")
	for _, word := range line {
		driver.WriteString("'" + word + "' ")
	}
	driver.WriteString(")\n")
	driver.WriteString("COMP_CWORD=" + strconv.Itoa(len(line)-1) + "\n")
	driver.WriteString("_hyper\n")
	driver.WriteString(`printf '%s\n' "${COMPREPLY[@]}"` + "\n")

	cmd := exec.Command(bash, "-c", driver.String())
	cmd.Dir = wd
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("driving the script failed: %v\n%s", err, out)
	}

	var offered []string
	for _, word := range strings.Split(string(out), "\n") {
		if word != "" {
			offered = append(offered, word)
		}
	}
	return offered
}

// scriptFor runs the command for one shell and returns what it wrote,
// asserting the success path so that no test below reads an empty string as a
// script with nothing in it.
func scriptFor(t *testing.T, shell string) string {
	t.Helper()

	var stdout, stderr bytes.Buffer
	if exit := cli.RunCompletions([]string{shell}, &stdout, &stderr); exit != cli.ExitClean {
		t.Fatalf("hyper completions %s exit = %d; stderr=%q", shell, exit, stderr.String())
	}
	return stdout.String()
}
