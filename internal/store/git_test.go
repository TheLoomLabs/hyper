package store_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/git"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The environment every git subprocess this package starts is run with, where
// what it carries is a rule about the call site rather than about an answer.

// TestEnvironment_EveryObjectReadRunsWithLazyFetchingOff is ADR-0071's line
// held on the half of a review that reads the record.
//
// A review's baseline is two reads and not one: the Journal entry that supplies
// the range is read here, and the code-branch object it names is read in
// internal/revision. On a partial clone the first of those is as much a silent
// network reach as the second, and *a review resolves no credential, reaches no
// network* would be false on the half nobody was looking at (§8, §9).
//
// The three calls that reach a remote — `ls-remote`, `fetch` and `push` — are
// the other side of the line and are untouched by this switch: they are
// explicit, and the branch they carry is fetched whole rather than filtered
// (ADR-0074).
func TestEnvironment_EveryObjectReadRunsWithLazyFetchingOff(t *testing.T) {
	// An ambient value, because that is the case the enforcement has to
	// survive: a machine configured its way past the switch is where the
	// reads would silently start reaching the network.
	t.Setenv("GIT_NO_LAZY_FETCH", "0")

	env := store.Environment(time.Date(2026, time.April, 2, 9, 41, 14, 0, time.UTC))
	if !slices.Contains(env, git.NoLazyFetch) {
		t.Errorf("the environment is %v, want lazy fetching off on every read", env)
	}
	// os/exec keeps the final value of a repeated name, which is the rule
	// this package's own identity and dates already rest on.
	last := ""
	for _, entry := range env {
		if name, _, named := strings.Cut(entry, "="); named && name == "GIT_NO_LAZY_FETCH" {
			last = entry
		}
	}
	if last != git.NoLazyFetch {
		t.Errorf("git would read %q for the switch, want %q", last, git.NoLazyFetch)
	}
}
