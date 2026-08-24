package compare_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fence around the package (§8, issue #167). `internal/compare` opens no
// file, starts no subprocess and reads no clock: the git reads are
// `internal/revision`'s and the Store reads are `internal/store`'s, and both
// are handed in. That is a property of what the package may reach, so it is
// asserted over the source rather than left to a reader to re-establish each
// time a derivation lands here — and this milestone lands three more of them
// (#170, #171).
//
// It is `internal/cli/version.go`'s own fence widened from one file to a
// package, and for the same reason: a path this package must not have is one
// nothing in a run of the tests would otherwise notice.

// TestPackage_ReachesNoFileNoSubprocessAndNoClock fences the import graph of
// every file in the package.
func TestPackage_ReachesNoFileNoSubprocessAndNoClock(t *testing.T) {
	allowed := map[string]bool{
		// time is the type an instant is, and never a clock: the two
		// readings are told apart by the source scan below.
		`"time"`:          true,
		`"bytes"`:         true,
		`"cmp"`:           true,
		`"encoding/json"`: true,
		`"maps"`:          true,
		`"slices"`:        true,
		`"strconv"`:       true,
		`"strings"`:       true,
		`"unicode/utf8"`:  true,
		// internal/artefact and internal/cadence are the two readings
		// `THE CODE MOVED` is written from — the code-fact vocabulary
		// §12 fixes and §10's mandatory gloss — and both are pure in
		// this package's own sense: each takes what it is handed and
		// opens nothing (#171).
		`"github.com/TheLoomLabs/hyper/internal/artefact"`: true,
		`"github.com/TheLoomLabs/hyper/internal/cadence"`:  true,
		`"github.com/TheLoomLabs/hyper/internal/render"`:   true,
		`"github.com/TheLoomLabs/hyper/internal/store"`:    true,
	}

	for _, path := range sources(t) {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			if !allowed[imported.Path.Value] {
				t.Errorf("%s imports %s; the package opens no file, starts no subprocess and reads no clock", path, imported.Path.Value)
			}
		}
	}
}

// TestPackage_ReadsNoClock fences the one import that is allowed for its types
// and refused for its reads.
//
// An instant is a value this package compares and an instant it *asks for* is
// a fact about when the Comparison was rendered, which no rule in §8 turns on:
// the window's two ends come off the entries, and a derivation that could read
// the clock is one whose answer depends on when it was called.
func TestPackage_ReadsNoClock(t *testing.T) {
	for _, path := range sources(t) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, read := range []string{"time.Now(", "time.Since(", "time.Until("} {
			if strings.Contains(string(content), read) {
				t.Errorf("%s calls %s); the clock is the caller's and the window's ends come off the entries", path, read)
			}
		}
	}
}

// sources is every non-test file in the package.
func sources(t *testing.T) []string {
	t.Helper()

	found, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, path := range found {
		if !strings.HasSuffix(path, "_test.go") {
			kept = append(kept, path)
		}
	}
	if len(kept) == 0 {
		t.Fatal("the package holds no source file; the fence would pass having read nothing")
	}
	return kept
}
