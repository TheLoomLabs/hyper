package revision_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/revision"
)

// The artefacts as one revision held them (§7, issue #154).
//
// Every case is a fact about a real git repository, for the reason the rest of
// this suite's are: what a commit holds is what git says it holds, and a
// reader that answered anything else would be reading the working tree under
// another name.

// wanted is the selection a caller supplies. These cases hand over a rule of
// their own rather than internal/repository's, so that what is under test is
// the read and never the predicate — which has its own cases one package over.
func wanted(path string) bool {
	return path == "hyper.yaml" || strings.HasPrefix(path, "definitions/") || strings.HasPrefix(path, "procedures/")
}

// pathsOf is a read answered as the one thing most of these cases are about.
func pathsOf(files []revision.File) []string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.Path
	}
	return paths
}

// TestArtefactsAt_ReadsWhatTheCommitHeldAndNotWhatTheTreeHolds is the whole
// claim: a reaper loads the dead Run's Procedure at the revision that Run
// named, so an artefact edited since is read as it was and one added since is
// not read at all.
func TestArtefactsAt_ReadsWhatTheCommitHeldAndNotWhatTheTreeHolds(t *testing.T) {
	r := newRepo(t)
	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\n")
	r.commit()
	then := r.text("rev-parse", "HEAD")

	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\n# edited since\n")
	r.write("procedures/later.yaml", "kind: procedure\nprocedure: later\n")
	r.commit()

	files, ok, err := revision.ArtefactsAt(r.root, then, wanted)
	if err != nil {
		t.Fatalf("ArtefactsAt: %v", err)
	}
	if !ok {
		t.Fatal("the commit the case just made does not resolve")
	}
	if got := pathsOf(files); !slices.Equal(got, []string{"hyper.yaml", "procedures/publish.yaml"}) {
		t.Errorf("the revision holds %v, want hyper.yaml and the one Procedure it had then", got)
	}
	if got := string(files[1].Bytes); got != "kind: procedure\nprocedure: publish\n" {
		t.Errorf("the Procedure reads %q, want the bytes the commit held rather than the working tree's", got)
	}
}

// TestArtefactsAt_ReadsNoFileTheCallerDidNotWant is the scope: a repository
// holds files that are not artefacts, and a read that brought them back would
// be one whose cost grew with the repository rather than with the artefacts.
func TestArtefactsAt_ReadsNoFileTheCallerDidNotWant(t *testing.T) {
	r := newRepo(t)
	r.write("definitions/uptime-check.yaml", "kind: definition\ndefinition: uptime-check\n")
	r.write("README.md", "# not an artefact\n")
	r.write("definitions/notes/deep.yaml", "kind: definition\n")
	r.commit()

	files, _, err := revision.ArtefactsAt(r.root, "HEAD", func(path string) bool {
		return path == "definitions/uptime-check.yaml"
	})
	if err != nil {
		t.Fatalf("ArtefactsAt: %v", err)
	}
	if got := pathsOf(files); !slices.Equal(got, []string{"definitions/uptime-check.yaml"}) {
		t.Errorf("the revision answers %v, want only the file the caller wanted", got)
	}
}

// TestArtefactsAt_AnswersNotHeldForARevisionThisCloneDoesNotHave is the case
// the reaper is really about: a Run recorded on another machine names a commit
// this clone has never fetched, and the reaper writes its closing write with no
// code facts at all rather than inventing any (§7).
//
// It is answered rather than errored because *this clone does not hold that
// commit* is a fact about the clone and not a fault: a shallow fetch, a runner
// that pushed the Store and not the code, and a branch never pulled all reach
// it, and none of them is the world resisting.
func TestArtefactsAt_AnswersNotHeldForARevisionThisCloneDoesNotHave(t *testing.T) {
	r := newRepo(t)

	for name, named := range map[string]string{
		"a commit this clone does not hold": "bcce2c10592d43450874312bb51af5ee2188ccc3",
		"a revision that is not a commit":   "not-a-revision",
		"no revision at all":                "",
	} {
		t.Run(name, func(t *testing.T) {
			files, ok, err := revision.ArtefactsAt(r.root, named, wanted)
			if err != nil {
				t.Fatalf("ArtefactsAt: %v", err)
			}
			if ok || files != nil {
				t.Errorf("ArtefactsAt answered %v held for %q, want nothing held", pathsOf(files), named)
			}
		})
	}
}

// TestArtefactsAt_OfARevisionHoldingNoArtefactIsEmptyAndHeld is the difference
// the answer above turns on: a commit that resolves and holds nothing the
// caller wanted is a revision that was read, where one that does not resolve is
// a revision that was not.
func TestArtefactsAt_OfARevisionHoldingNoArtefactIsEmptyAndHeld(t *testing.T) {
	r := newRepo(t)

	files, ok, err := revision.ArtefactsAt(r.root, "HEAD", func(string) bool { return false })
	if err != nil {
		t.Fatalf("ArtefactsAt: %v", err)
	}
	if !ok {
		t.Error("a commit that resolves answered not held")
	}
	if len(files) != 0 {
		t.Errorf("the revision answers %v, want nothing", pathsOf(files))
	}
}
