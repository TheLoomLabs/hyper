package cli_test

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// versionCorpus is the version command's own slice of testdata/, a sibling of
// testdata/check/ rather than a directory inside it: a case belongs to exactly
// one harness, and no harness runs another's cases (issue #101). How a case is
// found and how its golden files are compared are golden_test.go's, shared with
// every other corpus; what is here is how a version case is driven.
const versionCorpus = "testdata/version"

// TestVersionGolden drives cli.RunVersion end to end, one directory per case
// under testdata/version/ (issue #103). Each case supplies argv — the complete
// command line as typed, `hyper version` and everything after it — and
// facts.json, the build facts the entry point is handed. Its golden files
// (stdout, stderr, exit) are compared byte-for-byte and regenerated behind
// -update.
//
// The facts come from the case rather than from the running binary, which is
// the whole reason this corpus can exist: a page assembled from the real build
// carries the commit of the tree it was built from and changes with every
// commit made to it.
func TestVersionGolden(t *testing.T) {
	for _, name := range corpusCases(t, versionCorpus) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(versionCorpus, name)
			args := readArgv(t, filepath.Join(dir, "argv"))
			facts := readFacts(t, filepath.Join(dir, "facts.json"))

			var stdout, stderr bytes.Buffer
			exit := cli.RunVersion(args, &stdout, &stderr, facts)

			compareGolden(t, dir, stdout.Bytes(), stderr.Bytes(), exit)
		})
	}
}

// readArgv reads a case's complete argv — `hyper version` and whatever
// follows — and returns what the entry point receives, which is everything past
// the command name. Storing the whole line rather than only the tail is what
// makes a case directory readable as the invocation it stands for; the two
// tokens it always starts with are asserted rather than assumed, so a case that
// meant to test another command cannot be run as a version case.
//
// Tokens are whitespace-separated, so no case can express an argument that
// carries whitespace of its own. That costs this corpus nothing — `version`
// rejects every argument by length before looking at one — and the day a
// command needs such a case is the day the file becomes one token per line, as
// testdata/check/'s args already is.
func readArgv(t *testing.T, path string) []string {
	t.Helper()
	argv := strings.Fields(readFile(t, path))
	if len(argv) < 2 || argv[0] != "hyper" || argv[1] != "version" {
		t.Fatalf("argv is %q, want a complete argv beginning `hyper version`", argv)
	}
	return argv[2:]
}

// factsFixture is the on-disk shape of a case's facts.json, stated here rather
// than by hanging json tags off version.Facts: the fixture format is the
// harness's business, and a domain value should not carry a serialisation it
// has no other use for. The tags are what make the file's keys a contract
// instead of a coincidence of Go's case-insensitive field matching.
type factsFixture struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Built     string `json:"built"`
	Modified  bool   `json:"modified"`
	Toolchain string `json:"toolchain"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// readFacts reads the build facts a case hands the entry point. Unknown fields
// are an error: a fixture with a misspelt key would otherwise render `unknown`
// and be frozen into a golden file as the very rendering it meant to avoid.
func readFacts(t *testing.T, path string) version.Facts {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var fixture factsFixture
	if err := dec.Decode(&fixture); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return version.Facts{
		Version:   fixture.Version,
		Commit:    fixture.Commit,
		Built:     fixture.Built,
		Modified:  fixture.Modified,
		Toolchain: fixture.Toolchain,
		OS:        fixture.OS,
		Arch:      fixture.Arch,
	}
}

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
// exercised against the real dispatch in cmd/hyper.
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
