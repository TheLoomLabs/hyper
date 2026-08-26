package version_test

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/version"
)

// fullFacts is a build that stamped everything — the shape every case below
// varies one fact of.
func fullFacts() version.Facts {
	return version.Facts{
		Version:   "1.4.0",
		Commit:    "9b29751b4a2c7e6f0d3a1b8c5e4f2a9d7c6b3e10",
		Built:     "2026-08-18T09:14:02Z",
		Toolchain: "go1.25.0",
		OS:        "linux",
		Arch:      "amd64",
	}
}

// TestPage_IsTheFiveLineRendering pins the whole page byte for byte (issue
// #103). The four labels are lowercase and fixed, and the values start at one
// column — `os/arch`, the longest label, plus a single space.
func TestPage_IsTheFiveLineRendering(t *testing.T) {
	want := "hyper 1.4.0\n" +
		"commit  9b29751b4a2c7e6f0d3a1b8c5e4f2a9d7c6b3e10\n" +
		"built   2026-08-18T09:14:02Z\n" +
		"go      go1.25.0\n" +
		"os/arch linux/amd64\n"

	if got := fullFacts().Page(); got != want {
		t.Errorf("Page() =\n%q\nwant\n%q", got, want)
	}
}

// TestPage_ValuesShareOneColumn reads the property the case above pins by
// example: every value begins at the same column, whatever its label. A page
// whose columns drift is one an operator reads by counting rather than by
// looking.
func TestPage_ValuesShareOneColumn(t *testing.T) {
	lines := strings.Split(strings.TrimRight(fullFacts().Page(), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("Page() has %d lines, want 5", len(lines))
	}

	column := -1
	for _, line := range lines[1:] {
		label, rest, ok := strings.Cut(line, " ")
		if !ok {
			t.Fatalf("line %q carries no value", line)
		}
		at := len(label) + 1 + (len(rest) - len(strings.TrimLeft(rest, " ")))
		if column == -1 {
			column = at
		}
		if at != column {
			t.Errorf("line %q starts its value at column %d, want %d", line, at, column)
		}
		if label != strings.ToLower(label) {
			t.Errorf("label %q is not lowercase", label)
		}
	}
}

// TestPage_AnUnstampedFactRendersUnknown pins the rule that keeps the five
// lines a fixed shape: a fact the build did not stamp is stated as unknown and
// never omitted (issue #103). A reader who takes an omission for a value is a
// reader the layout created.
func TestPage_AnUnstampedFactRendersUnknown(t *testing.T) {
	page := version.Facts{Version: "1.4.0", Toolchain: "go1.25.0", OS: "linux", Arch: "amd64"}.Page()
	want := "hyper 1.4.0\n" +
		"commit  unknown\n" +
		"built   unknown\n" +
		"go      go1.25.0\n" +
		"os/arch linux/amd64\n"
	if page != want {
		t.Errorf("Page() with no VCS stamp =\n%q\nwant\n%q", page, want)
	}

	for _, tc := range []struct {
		name  string
		facts version.Facts
		line  string
	}{
		{"no version", version.Facts{}, "hyper unknown"},
		{"no toolchain", version.Facts{}, "go      unknown"},
		{"no os", version.Facts{Arch: "amd64"}, "os/arch unknown"},
		{"no arch", version.Facts{OS: "linux"}, "os/arch unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.facts.Page()
			if !strings.Contains(got, tc.line+"\n") {
				t.Errorf("Page() =\n%q\nwant a line %q", got, tc.line)
			}
			if n := strings.Count(got, "\n"); n != 5 {
				t.Errorf("Page() has %d lines, want 5 whatever is unstamped", n)
			}
		})
	}
}

// TestPage_AModifiedTreeMarksTheCommit pins the suffix that keeps a hash from
// being a claim about bytes edited after it (issue #103).
func TestPage_AModifiedTreeMarksTheCommit(t *testing.T) {
	facts := fullFacts()
	facts.Modified = true

	want := "commit  9b29751b4a2c7e6f0d3a1b8c5e4f2a9d7c6b3e10-dirty\n"
	if got := facts.Page(); !strings.Contains(got, want) {
		t.Errorf("Page() =\n%q\nwant it to carry %q", got, want)
	}
}

// TestPage_AnUnstampedCommitIsNotMarkedDirty states where the suffix stops: it
// qualifies a hash, and there is no hash here to qualify. `unknown-dirty` would
// read as a commit named unknown.
func TestPage_AnUnstampedCommitIsNotMarkedDirty(t *testing.T) {
	facts := version.Facts{Version: "1.4.0", Modified: true}
	if got := facts.Page(); !strings.Contains(got, "commit  unknown\n") {
		t.Errorf("Page() =\n%q\nwant it to carry %q", got, "commit  unknown\n")
	}
}

// TestCurrent_CarriesTheOneVersion pins the first of the three readers of one
// string: the page, the pin gate, and the Refusal message that quotes it all
// say what the build stamped and never a second reading of it (§9, ADR-0020,
// issues #103 and #191).
func TestCurrent_CarriesTheOneVersion(t *testing.T) {
	got := version.Current()
	if got.Version != version.Version {
		t.Errorf("Current().Version = %q, want the package's own %q", got.Version, version.Version)
	}
	if want := "hyper " + version.Version + "\n"; !strings.HasPrefix(got.Page(), want) {
		t.Errorf("Current().Page() starts %q, want it to start %q", got.Page(), want)
	}
}

// TestCurrent_ReadsTheRuntimeItRunsOn pins the two facts that cannot be
// unstamped: GOOS/GOARCH and the toolchain are compiled in, so Current never
// renders them unknown even in a `go test` binary with no VCS stamping.
func TestCurrent_ReadsTheRuntimeItRunsOn(t *testing.T) {
	got := version.Current()
	if got.OS != runtime.GOOS {
		t.Errorf("Current().OS = %q, want %q", got.OS, runtime.GOOS)
	}
	if got.Arch != runtime.GOARCH {
		t.Errorf("Current().Arch = %q, want %q", got.Arch, runtime.GOARCH)
	}
	if got.Toolchain != runtime.Version() {
		t.Errorf("Current().Toolchain = %q, want %q", got.Toolchain, runtime.Version())
	}
}

// TestPackage_ReachesNoNetwork pins the two halves of ADR-0019 that this page
// is the most tempting place to break: `hyper version` reaches no network on
// any path, and nothing in it asks whether a newer release exists. The fence is
// the transitive import graph of the package that holds the facts — a package
// that cannot reach net/http cannot check for an update, whatever its code
// says.
func TestPackage_ReachesNoNetwork(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH; the import graph cannot be read")
	}

	out, err := exec.Command("go", "list", "-deps", "github.com/TheLoomLabs/hyper/internal/version").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if dep == "net" || strings.HasPrefix(dep, "net/") || strings.HasPrefix(dep, "crypto/tls") {
			t.Errorf("internal/version reaches %s; `hyper version` reaches no network on any path (ADR-0019)", dep)
		}
	}
}

// TestVersion_AnUnstampedBuildSaysUnknown is what the binary claims when
// nothing stamped it. `go test` links no `-X`, so the value read here is the
// declaration's own — and it is the same word every other unstamped fact on the
// page renders as, because *what version is this* has no better answer from a
// build that was never told (issue #191).
//
// The word is spelled here rather than read from the package, which an
// external test package could not do anyway: a case that checked one of the
// two declarations against the other would pass whatever they were changed to
// together.
func TestVersion_AnUnstampedBuildSaysUnknown(t *testing.T) {
	if got := version.Version; got != "unknown" {
		t.Errorf("Version = %q, want %q — an unstamped build names no version and says so", got, "unknown")
	}
	if got, want := version.Current().Page(), "hyper unknown\n"; !strings.HasPrefix(got, want) {
		t.Errorf("Page() starts %q, want it to start %q", got, want)
	}
}
