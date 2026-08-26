package cli_test

import (
	"slices"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// sectionNineTree is §9's tree, transcribed from the specification's own table
// rather than read from the package, so that this is a check on the list and
// not a restatement of it. Sixteen commands, flat, one noun group, no aliases
// and no hidden commands.
var sectionNineTree = []string{
	"providers", "provider", "operation",
	"targets",
	"check", "review",
	"run", "probe",
	"runs", "show", "changes", "records",
	"install", "project", "store", "compact",
}

// TestTree_IsSectionNines holds the one compiled-in surface against the
// specification it transcribes: sixteen in the tree, three outside it,
// nineteen in all, each name appearing exactly once. This is the test that
// fails the day a milestone adds a command to the tree, which is the point —
// the same edit that adds it here reaches all three shells (issue #104).
//
// The third member outside the tree is `mcp`, and it is transcribed here for
// the same reason the sixteen are: §9 names it, and a name the specification
// fixes is one this list carries whether or not a milestone has built it yet
// (issue #193, ADR-0088).
func TestTree_IsSectionNines(t *testing.T) {
	if got, want := cli.Tree(), sectionNineTree; !slices.Equal(got, want) {
		t.Errorf("the tree is %q,\n            want §9's %q", got, want)
	}
	if got, want := cli.OutsideTree(), []string{"version", "completions", "mcp"}; !slices.Equal(got, want) {
		t.Errorf("outside the tree is %q, want %q", got, want)
	}
	if got, want := cli.Globals(), []string{"--json", "--repo-dir", "--no-color"}; !slices.Equal(got, want) {
		t.Errorf("the globals are %q, want §9's three %q", got, want)
	}
	if got, want := cli.StoreSubVerbs(), []string{"init"}; !slices.Equal(got, want) {
		t.Errorf("store's sub-verbs are %q, want %q", got, want)
	}
	if got, want := cli.Shells(), []string{"bash", "fish", "zsh"}; !slices.Equal(got, want) {
		t.Errorf("the known shells are %q, want %q, alphabetically", got, want)
	}

	commands := cli.Commands()
	if len(commands) != 19 {
		t.Errorf("the surface is %d commands, want the nineteen: %q", len(commands), commands)
	}
	seen := make(map[string]bool, len(commands))
	for _, name := range commands {
		if seen[name] {
			t.Errorf("%q is named twice in the surface", name)
		}
		seen[name] = true
	}
}

// TestTree_IsReadThroughCopies pins what makes the surface a constant in a
// language that has no constant slice: every reader is handed its own copy, so
// a caller that sorts or truncates what it was given has not edited what every
// other reader sees. The lists themselves are unexported, which is the other
// half — there is no second way to reach them.
func TestTree_IsReadThroughCopies(t *testing.T) {
	for name, read := range map[string]func() []string{
		"Tree":          cli.Tree,
		"OutsideTree":   cli.OutsideTree,
		"StoreSubVerbs": cli.StoreSubVerbs,
		"Globals":       cli.Globals,
		"Shells":        cli.Shells,
		"Commands":      cli.Commands,
	} {
		t.Run(name, func(t *testing.T) {
			first := read()
			if len(first) == 0 {
				t.Fatalf("cli.%s() is empty", name)
			}
			first[0] = "clobbered"

			if got := read()[0]; got == "clobbered" {
				t.Errorf("editing what cli.%s() returned edited the compiled-in surface", name)
			}
		})
	}
}
