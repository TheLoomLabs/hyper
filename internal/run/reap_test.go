package run

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **What a closing write says about the Step, and what it declines to say**
// (§7, ADR-0076, issue #154).
//
// The corpus drives the two ends of it — a revision that resolves the Step, and
// a Run that recorded `repo_dirty` — through real Runs against real fixtures.
// What is here is the rest of the table: the readings where a revision resolves
// and the **Step** does not, which a case would need a repository written to be
// broken to reach, and which are the readings §7's *omits what it cannot
// establish* is entirely about.
//
// Each is a pure function of a load and an entry, so none of them reaches git:
// the cache the reap holds is seeded with the answer a revision gave, which is
// exactly what it holds after one read.

// theDeadRunsRevision is the commit the entries below name. Nothing resolves it
// — the loads are seeded against it — and it is a constant so that a case
// naming *another* revision says so by naming a different one.
const theDeadRunsRevision = "0068d7c040147dc36822c72d1bf2e80ad7f9265b"

// theReapersRepository is a repository holding one Procedure of two Steps and
// everything they bind, in the shapes the load itself reads. It is what one
// revision resolved to, seeded into the cache a reap holds.
func theReapersRepository() repository.Loaded {
	return repository.LoadFrom([]repository.Source{
		{Path: "definitions/preview-dns.yaml", Bytes: []byte(
			"kind: definition\ndefinition: preview-dns\nprovider: cloudflare-dns\nkinds: [mutate]\ntargets: [cloudflare-prod]\n")},
		{Path: "procedures/publish-two.yaml", Bytes: []byte(
			"kind: procedure\nprocedure: publish-two\ntargets: [cloudflare-prod]\nsteps:\n" +
				"  - id: publish\n    definition: preview-dns\n    operation: create_dns_record\n    target: cloudflare-prod\n" +
				"  - id: publish-again\n    definition: preview-dns\n    operation: create_dns_record\n    target: cloudflare-prod\n")},
		// One Step the walk reaches, then an invocation naming nothing.
		// The ordinal a reaper derives is **in range** of the sequence
		// this builds, so what stops it is the walk not being whole and
		// not the arithmetic running off the end.
		{Path: "procedures/publish-elsewhere.yaml", Bytes: []byte(
			"kind: procedure\nprocedure: publish-elsewhere\ntargets: [cloudflare-prod]\nsteps:\n" +
				"  - id: publish\n    definition: preview-dns\n    operation: create_dns_record\n    target: cloudflare-prod\n" +
				"  - id: inner\n    procedure: a-procedure-nothing-declares\n")},
		{Path: "providers/cloudflare-dns.yaml", Bytes: []byte(
			"kind: provider\nprovider: cloudflare-dns\nschema-version: 1\nclass: cloudflare\ncapabilities: [http]\n" +
				"operations:\n  create_dns_record:\n    kind: mutate\n    repeatability: repeatable\n" +
				"    http:\n      method: POST\n      host: \"{from-target}\"\n      path: /dns\n")},
		{Path: "targets/cloudflare-prod.yaml", Bytes: []byte(
			"kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [read, mutate]\n" +
				"capabilities: [http]\nhosts: [api.cloudflare.com]\n")},
	})
}

// readingAt is the cache a reap holds after one revision answered: the load
// where the clone held that commit, and the answer *not held* where it did not.
func readingAt(commit string, loaded repository.Loaded, held bool) revisions {
	read := reading("")
	read.read[commit] = revisionRead{loaded: loaded, held: held}
	return read
}

// anOpenEntry is one entry a reap found holding no account: the Procedure the
// dead Run named, the revision to load it at, and the highest Step ordinal the
// entry holds.
func anOpenEntry(procedure string, last int) store.OpenEntry {
	return store.OpenEntry{
		RunFile: store.RunFile{
			Procedure:  procedure,
			Provenance: store.RunProvenance{RepoRevision: theDeadRunsRevision},
		},
		Last: last,
	}
}

// TestWentQuietOn_ResolvesTheStepAfterTheHighestOrdinalPresent is the ordinary
// reading, and the arithmetic §7 fixes: the highest `<nnnn>` present is the last
// Step that finished, so the Step is the one after it — read out of the sequence
// the dead Run's own revision builds rather than guessed from a number.
func TestWentQuietOn_ResolvesTheStepAfterTheHighestOrdinalPresent(t *testing.T) {
	read := readingAt(theDeadRunsRevision, theReapersRepository(), true)

	for last, want := range map[int]string{0: "publish", 1: "publish-again"} {
		code, err := wentQuietOn(anOpenEntry("publish-two", last), read)
		if err != nil {
			t.Fatalf("wentQuietOn: %v", err)
		}
		if code.ID != want {
			t.Errorf("an entry holding %d Step files names %q, want %q", last, code.ID, want)
		}
		if code.Definition != "preview-dns" || code.Operation != "create_dns_record" ||
			code.Provider != "cloudflare-dns" || code.Target != "cloudflare-prod" || code.Kind != store.KindMutate {
			t.Errorf("the code facts are %+v, want the whole of what that revision resolves", code)
		}
	}
}

// TestWentQuietOn_OmitsWhatTheDeadRunsRevisionDoesNotResolve is the rest of the
// table, and every row of it is one absence: the file carries `step` and
// `disposition` and no `id` and no code facts.
//
// **The id and the code facts are one group.** §7 states them as one — *the
// Step's `id` and its code facts where the dead Run's revision resolves them,
// and absent where it does not* — so there is no row here in which some of them
// stand and the rest do not.
func TestWentQuietOn_OmitsWhatTheDeadRunsRevisionDoesNotResolve(t *testing.T) {
	loaded := theReapersRepository()

	for name, held := range map[string]struct {
		entry store.OpenEntry
		read  revisions
	}{
		// §7 names this one outright: the commit the Run recorded is
		// not the code it ran, so nothing that commit resolves is a
		// fact about the Step that went quiet.
		"a Run that recorded repo_dirty": {
			entry: func() store.OpenEntry {
				entry := anOpenEntry("publish-two", 1)
				entry.Provenance.RepoDirty = true
				return entry
			}(),
			read: readingAt(theDeadRunsRevision, loaded, true),
		},
		// Every Run recorded on a runner whose code branch this clone
		// never fetched.
		"a revision this clone does not hold": {
			entry: anOpenEntry("publish-two", 1),
			read:  readingAt(theDeadRunsRevision, repository.Loaded{}, false),
		},
		// The revision resolves and the Procedure does not: a Run of a
		// Procedure that revision never held.
		"a Procedure the revision does not hold": {
			entry: anOpenEntry("a-procedure-nobody-wrote", 0),
			read:  readingAt(theDeadRunsRevision, loaded, true),
		},
		// An invocation naming nothing contributes no Steps and every
		// Step after it moves up, so the ordinal is a number about a
		// different Procedure.
		"a sequence the walk could not reach in full": {
			entry: anOpenEntry("publish-elsewhere", 0),
			read:  readingAt(theDeadRunsRevision, loaded, true),
		},
		// The window between the last Step file and `outcome.json`:
		// the ordinal is past the end of the sequence and names no Step
		// at all.
		"an ordinal past the end of the sequence": {
			entry: anOpenEntry("publish-two", 2),
			read:  readingAt(theDeadRunsRevision, loaded, true),
		},
	} {
		t.Run(name, func(t *testing.T) {
			code, err := wentQuietOn(held.entry, held.read)
			if err != nil {
				t.Fatalf("wentQuietOn: %v", err)
			}
			if code != (store.StepCode{}) {
				t.Errorf("the closing write carries %+v, want none of it — the reaper omits what it cannot establish", code)
			}
		})
	}
}

// TestWentQuietOn_OmitsTheIdOfAStepWhoseBindingDoesNotResolve is the same rule
// where it is least obvious: the Procedure that revision holds authors the id,
// and the Definition it binds is not there.
//
// The id alone would be a shape §7 does not name, written off a repository
// `check` refuses — so it is omitted with the rest, and what stands is the
// `step` and the Disposition, which is what the reaper actually knows.
func TestWentQuietOn_OmitsTheIdOfAStepWhoseBindingDoesNotResolve(t *testing.T) {
	unbound := repository.LoadFrom([]repository.Source{
		{Path: "procedures/publish-two.yaml", Bytes: []byte(
			"kind: procedure\nprocedure: publish-two\ntargets: [cloudflare-prod]\nsteps:\n" +
				"  - id: publish\n    definition: a-definition-nothing-declares\n    operation: create_dns_record\n    target: cloudflare-prod\n")},
	})

	code, err := wentQuietOn(anOpenEntry("publish-two", 0), readingAt(theDeadRunsRevision, unbound, true))
	if err != nil {
		t.Fatalf("wentQuietOn: %v", err)
	}
	if code != (store.StepCode{}) {
		t.Errorf("the closing write carries %+v, want none of it: the id and the code facts are one group", code)
	}
}
