package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// `THE CODE MOVED`, where a golden cannot hold it (§8, §12, issue #171).
//
// The eight artefact-authored classes and the catch-all's whole count read
// **bytes at two revisions**, and a `repo_revision` is a commit — a function of
// the tree, the message, the identity and the dates, which no case directory
// can name. So the cases here materialise a repository holding two code commits
// — the `code-baseline/` input, a directory committed **below** the case's
// `repo/` — read both revisions, seed the Journal against them and drive
// `hyper changes` at the result.
//
// What the checked-in [`changes`](testdata/changes) corpus holds instead is the
// other half: every case there seeds a `repo_revision` this fixture never
// committed, so the catch-all renders `not-in-clone` and `the digests` — the
// one class read off the two Journal entries (ADR-0086) — renders beside it.
// Both halves are the specification, and neither can be asserted where the
// other is.

// codeWindow is the two-commit fixture with both of its revisions in hand.
type codeWindow struct {
	rangedRepository
	baseline, subject string
}

// absentRevision is a commit this fixture never made: forty hex digits that
// resolve to nothing, which is what a Run recorded on a runner whose code
// branch was never fetched leaves behind (ADR-0071).
const absentRevision = "1f0a3d78c2e5b91467af03d28b5c9e610473fa8d"

// codeWindowed materialises it. The case directory carries a `repo/`, a
// `code-baseline/` committed below it and a `git` marker, and no argv at all:
// what is driven is decided here, one Comparison at a time.
func codeWindowed(t *testing.T) codeWindow {
	t.Helper()

	held := rangedIn(t, "changes/a-code-window-seeded-at-run-time")
	return codeWindow{
		rangedRepository: held,
		baseline:         held.run.fixture.text(t, held.run.fixture.root, "rev-parse", "HEAD~1"),
		subject:          held.head,
	}
}

// object is the blob id one revision holds at a path, which is what a
// `procedure_revision` and a `definition_revision` are (§7).
func (w codeWindow) object(t *testing.T, revision, path string) string {
	t.Helper()
	return w.run.fixture.text(t, w.run.fixture.root, "rev-parse", revision+":"+path)
}

// digest is SHA-256 over one revision's bytes at a path, which is what a
// `manifest_digest` is: over the bytes rather than over a canonical form of
// what they parse to, so that a reader checks it with `sha256sum` (§7).
func (w codeWindow) digest(t *testing.T, revision, path string) string {
	t.Helper()

	sum := sha256.Sum256(w.run.fixture.run(t, w.run.fixture.root, "show", revision+":"+path))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// seeded is one end of the window as a case asks for it: the revision the entry
// recorded, the Target its retiring Step bound, and whether it recorded
// `repo_dirty`.
//
// The revision is separate from the one whose bytes the entry's other members
// are read at, which is what lets a case seed an entry naming a commit this
// clone does not hold while still carrying a coherent Provenance.
type seeded struct {
	revision, reads, retiresOn string
	dirty                      bool
}

// at is one end read at the revision it names.
func at(revision, retiresOn string) seeded {
	return seeded{revision: revision, reads: revision, retiresOn: retiresOn}
}

// withDirtyTree is that end with `repo_dirty` recorded: the bytes that Run read
// are nowhere in git (§7).
func (s seeded) withDirtyTree() seeded { s.dirty = true; return s }

// naming is that end recording a revision the clone does not hold, with its
// other members still read at the revision it actually stands on.
func (s seeded) naming(revision string) seeded { s.revision = revision; return s }

// seed writes the two Journal entries this window is over: one Run of
// `sync-ci-keys` at each end, each holding the two Steps the Procedure
// declares.
func (w codeWindow) seed(t *testing.T, baseline, subject seeded) {
	t.Helper()

	files := map[string]string{}
	for path, content := range w.entry(t, "01991c3a-7d40-7a11-9c2e-4f0b8d61a3e7", "2026-08-04T09:12:03.000Z", "2026-08-04T09:13:51.000Z", baseline) {
		files[path] = content
	}
	for path, content := range w.entry(t, "01991ea6-b118-7c93-8d41-6b2f7ae05c19", "2026-08-06T11:03:18.000Z", "2026-08-06T11:05:49.000Z", subject) {
		files[path] = content
	}
	w.journal(t, files)
}

// entry is one closed, non-rehearsal Journal entry of `sync-ci-keys`, in the
// shape §7 writes one.
func (w codeWindow) entry(t *testing.T, id, started, ended string, end seeded) map[string]string {
	t.Helper()

	at := "journal/" + started[:4] + "/" + started[5:7] + "/" + started[8:10] + "/" + id + "/"
	dirty := ""
	if end.dirty {
		dirty = "\n    \"repo_dirty\": true,"
	}
	step := func(number, id, operation, target string) string {
		return `{
  "definition": "ci-keys",
  "disposition": "ran",
  "ended_at": "` + ended + `",
  "id": "` + id + `",
  "kind": "mutate",
  "operation": "` + operation + `",
  "provenance": {
    "definition_revision": "` + w.object(t, end.reads, "definitions/ci-keys.yaml") + `",
    "manifest_digest": "` + w.digest(t, end.reads, "providers/tailscale.yaml") + `"
  },
  "provider": "tailscale",
  "schema_version": 1,
  "started_at": "` + started + `",
  "step": ` + number + `,
  "target": "` + target + `"
}
`
	}
	return map[string]string{
		at + "run.json": `{
  "dry_run": false,
  "procedure": "sync-ci-keys",
  "provenance": {
    "hyper_version": "1.4.0",
    "procedure_revision": "` + w.object(t, end.reads, "procedures/sync-ci-keys.yaml") + `",` + dirty + `
    "repo_revision": "` + end.revision + `"
  },
  "run_id": "` + id + `",
  "schema_version": 1,
  "started_at": "` + started + `",
  "trigger": {
    "actor": "igor",
    "cause": "cron",
    "executor": "local",
    "host": "thinkpad"
  }
}
`,
		at + "outcome.json": `{
  "ended_at": "` + ended + `",
  "outcome": "completed",
  "schema_version": 1
}
`,
		at + "steps/0001.json": step("1", "issue-runner-keys", "create_key", "staging"),
		at + "steps/0002.json": step("2", "retire-expired", "delete_key", end.retiresOn),
	}
}

// changes drives one Comparison of the materialised repository and answers its
// page.
func (w codeWindow) changes(t *testing.T, extra ...string) (stdout, stderr string, exit int) {
	t.Helper()

	var out, errs bytes.Buffer
	args := append([]string{"changes", "--repo-dir", w.run.fixture.root, "sync-ci-keys"}, extra...)
	exit = cli.Main(args, &out, &errs, w.c.process(t, w.run), w.c.facts(t))
	if exit != 0 || errs.Len() != 0 {
		t.Fatalf("changes exited %d with stderr %q", exit, errs.String())
	}
	return out.String(), errs.String(), exit
}

// codeTable is `THE CODE MOVED` read back off a page: the head line and
// everything beneath it, up to the blank line `TOTALS` stands after.
func codeTable(t *testing.T, page string) string {
	t.Helper()

	lines := strings.Split(page, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "THE CODE MOVED") {
			continue
		}
		for end := i + 1; end < len(lines); end++ {
			if strings.TrimSpace(lines[end]) == "" {
				return strings.Join(lines[i:end], "\n")
			}
		}
	}
	t.Fatalf("the page renders no THE CODE MOVED block:\n%s", page)
	return ""
}

// totalsOf is the `TOTALS` line of a page.
func totalsOf(t *testing.T, page string) string {
	t.Helper()

	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "TOTALS") {
			return strings.TrimSpace(line)
		}
	}
	t.Fatalf("the page renders no TOTALS line:\n%s", page)
	return ""
}

// short is a git object name as the page draws it: the algorithm kept where
// `hyper` named one, and seven hex digits behind it (§8, ADR-0047).
func short(name string) string {
	algorithm, hex, named := strings.Cut(name, ":")
	if !named {
		return name[:7]
	}
	return algorithm + ":" + hex[:7]
}

// TestCodeMoved_EveryClassEmitsOneRowPerSubjectAndFact is the whole table over
// one window.
//
// Every one of §12's nine classes moves in it: the declared Kinds of a
// Definition and of a Target declaration, the Procedure's envelope and the
// Definition's bindable Targets and the `target:` a Step binds, a Bound
// disappearing, the Cadence, a Manifest's required Capabilities, a credential
// source, a Manifest's exposed Operations, and four members of `the digests`.
// The subjects are kind-qualified, the rows sort by `(SUBJECT, FACT)` with the
// `—` subject last of the named ones, and the catch-all terminates the table.
func TestCodeMoved_EveryClassEmitsOneRowPerSubjectAndFact(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging"), at(window.subject, "production"))

	page, _, _ := window.changes(t)
	want := strings.Join([]string{
		"  THE CODE MOVED   17 facts",
		"  SUBJECT                 FACT                           FROM                                 TO",
		"  definition ci-keys      definition revision            " + short(window.object(t, window.baseline, "definitions/ci-keys.yaml")) + "                              " + short(window.object(t, window.subject, "definitions/ci-keys.yaml")),
		"  definition ci-keys      kinds                          mutate                               destroy · mutate",
		"  definition ci-keys      targets                        staging                              production · staging",
		"  manifest tailscale      capabilities                   http                                 http · shell",
		"  manifest tailscale      manifest digest                " + short(window.digest(t, window.baseline, "providers/tailscale.yaml")) + "                       " + short(window.digest(t, window.subject, "providers/tailscale.yaml")),
		"  manifest tailscale      operations                     create_key · delete_key · list_keys  create_key · delete_key · get_key · list_keys",
		"  procedure sync-ci-keys  cadence                        0 0 1 * *                            */5 * * * *",
		"                                                         00:00 UTC on the 1st of the month    every 5 minutes",
		"                                                         1 run/month                          ≈8800 runs/month",
		"  procedure sync-ci-keys  procedure revision             " + short(window.object(t, window.baseline, "procedures/sync-ci-keys.yaml")) + "                              " + short(window.object(t, window.subject, "procedures/sync-ci-keys.yaml")),
		"  procedure sync-ci-keys  step issue-runner-keys · over  values                               values",
		"                                                         ci-arm64 · ci-x86 · ci-macos         ci-arm64 · ci-x86 · ci-macos · ci-arm64-2",
		"  procedure sync-ci-keys  step retire-expired · bound    5                                    –",
		"  procedure sync-ci-keys  step retire-expired · over     assets                               assets",
		"                                                         expires older_than 0s                created older_than 30d",
		"                                                                                              expires older_than 0s",
		"  procedure sync-ci-keys  step retire-expired · target   staging                              production",
		"  procedure sync-ci-keys  targets                        staging                              production · staging",
		"  target production       credential token               –                                    TAILSCALE_PROD",
		"  target production       kinds                          –                                    destroy · mutate · read",
		"  target staging          credential token               TAILSCALE_STAGING                    TAILSCALE_PROD",
		"  —                       repository revision            " + short(window.baseline) + "                              " + short(window.subject),
		"  24 other lines changed · git diff " + short(window.baseline) + " " + short(window.subject),
	}, "\n")
	if got := codeTable(t, page); got != want {
		t.Errorf("the table is\n%s\n\nwant\n%s", got, want)
	}
	if got, want := totalsOf(t, page), "TOTALS  0 changes · 0 asset · 0 observation · 0 tombstone · the code moved"; got != want {
		t.Errorf("TOTALS is %q, want %q", got, want)
	}
}

// TestCodeMoved_TheManifestClassesAreComputedFromTheStepFilesProvider is which
// Manifests a Comparison reads.
//
// **Which Manifests a Run read is the Step files' `provider`** (§7), and the
// two Manifest classes cannot be computed without it: `manifest_digest` names
// the bytes that ran and never which Provider they were, so enumerating the
// Manifests any other way would mean resolving each Step's `definition` to its
// `provider:` at that Step's own revision — a git object per Step, on a surface
// that already holds both revisions of the artefacts (§8).
//
// The fixture holds a second Manifest, `unread`, whose `capabilities:` moves
// across the window and which no Step file names. It draws no row, and its two
// lines fall to the catch-all's count — which is the word *other* holding.
func TestCodeMoved_TheManifestClassesAreComputedFromTheStepFilesProvider(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging"), at(window.subject, "production"))

	table := codeTable(t, mustPage(t, window))
	if strings.Contains(table, "manifest unread") {
		t.Errorf("a Manifest no Step file named draws a row; the table is\n%s", table)
	}
	if !strings.Contains(table, "manifest tailscale      capabilities") {
		t.Errorf("the Manifest the Step files name draws no capabilities row; the table is\n%s", table)
	}
}

// TestCodeMoved_TheCountIsGitsOwnLessTheLinesTheRowsAboveReport is the
// catch-all's arithmetic stated against `git diff` itself.
//
// **`N` is in `git diff` lines** as git counts them, a modified line being two,
// **minus the lines a classed row above already reports** — and what a row
// subtracts is the lines its own value occupies and nothing else. The Manifest
// gaining `get_key` is the case §8 names: the key line is subtracted and the
// fifteen lines of request, projection and declared Kind beneath it are not.
func TestCodeMoved_TheCountIsGitsOwnLessTheLinesTheRowsAboveReport(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging"), at(window.subject, "production"))

	// git's own total over the reviewed five, which is what the row's unit is
	// defined as and what a reader running the command sees.
	total := 0
	for _, line := range strings.Split(string(window.run.fixture.run(t, window.run.fixture.root,
		"diff", "--numstat", "--no-renames", window.baseline, window.subject)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		total += atoi(t, fields[0]) + atoi(t, fields[1])
	}
	if total != 46 {
		t.Fatalf("git counts %d moved lines over the fixture, want 46; the fixture moved and the arithmetic below is stale", total)
	}

	page, _, _ := window.changes(t)
	if want := "  24 other lines changed"; !strings.Contains(page, want) {
		t.Errorf("the catch-all is not %q, and %d − 22 = 24 is the lines the classed values occupy at their two revisions:\n%s", want, total, codeTable(t, page))
	}
}

// atoi is one `--numstat` field as a number.
func atoi(t *testing.T, field string) int {
	t.Helper()

	held := 0
	for _, digit := range field {
		if digit < '0' || digit > '9' {
			t.Fatalf("git wrote %q where a count was expected", field)
		}
		held = held*10 + int(digit-'0')
	}
	return held
}

// TestCodeMoved_ADirtySideSuppressesTheCommandAndKeepsTheCount is the first of
// the two suppressions.
//
// **Where either side recorded `repo_dirty` the command is suppressed** and the
// row renders `N other lines changed` alone: `git diff <rev> <rev>` does not
// reproduce what moved, and printing a command that does not reproduce is worse
// than printing none. `N` and the classed rows are still computed between the
// two committed revisions (§7, §8).
func TestCodeMoved_ADirtySideSuppressesTheCommandAndKeepsTheCount(t *testing.T) {
	for _, side := range []string{"baseline", "subject"} {
		t.Run(side, func(t *testing.T) {
			window := codeWindowed(t)
			was, is := at(window.baseline, "staging"), at(window.subject, "production")
			if side == "baseline" {
				was = was.withDirtyTree()
			} else {
				is = is.withDirtyTree()
			}
			window.seed(t, was, is)

			page, _, _ := window.changes(t)
			table := codeTable(t, page)
			if !strings.HasSuffix(table, "\n  24 other lines changed") {
				t.Errorf("the table ends\n%s\nwant `24 other lines changed` alone, with no command behind it", table)
			}
			if !strings.Contains(table, "step retire-expired · bound") {
				t.Errorf("the classed rows are still computed between the two committed revisions; the table is\n%s", table)
			}
		})
	}
}

// TestCodeMoved_ARevisionAbsentFromTheCloneReplacesTheCatchAll is the second.
//
// **Where the clone does not contain a revision the window names**, the classes
// that read bytes have none to read and the catch-all row is **replaced** by
// one line naming what could not be read — keeping the command, the reader of a
// job summary rarely being in the clone that came up short. The classes read
// off the two Journal entries are unaffected (§8, §12, ADR-0071).
func TestCodeMoved_ARevisionAbsentFromTheCloneReplacesTheCatchAll(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging").naming(absentRevision), at(window.subject, "production"))

	page, _, _ := window.changes(t)
	table := codeTable(t, page)
	if want := "  other lines could not be counted · git diff " + short(absentRevision) + " " + short(window.subject); !strings.HasSuffix(table, "\n"+want) {
		t.Errorf("the table ends\n%s\nwant %q", table, want)
	}
	if strings.Contains(table, "other lines changed") {
		t.Errorf("the absence **replaces** the count rather than joining it; the table is\n%s", table)
	}
	for _, unreadable := range []string{"kinds", "cadence", "capabilities", "operations", "credential token"} {
		if strings.Contains(table, unreadable) {
			t.Errorf("the %s class reads bytes and the clone holds none; the table is\n%s", unreadable, table)
		}
	}
	for _, read := range []string{"definition revision", "manifest digest", "procedure revision", "repository revision"} {
		if !strings.Contains(table, read) {
			t.Errorf("`the digests` reads off two Journal entries and is unaffected; the table is\n%s", table)
		}
	}
}

// TestCodeMoved_TheTwoSuppressionsStack is the third form: a dirty side and a
// revision the clone does not hold, which reach this row for different reasons
// and are about different halves of it.
func TestCodeMoved_TheTwoSuppressionsStack(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging").naming(absentRevision).withDirtyTree(), at(window.subject, "production"))

	table := codeTable(t, mustPage(t, window))
	if !strings.HasSuffix(table, "\n  other lines could not be counted") {
		t.Errorf("the table ends\n%s\nwant the absence line with neither a count nor a command", table)
	}
}

// mustPage drives the Comparison and answers its page.
func mustPage(t *testing.T, window codeWindow) string {
	t.Helper()

	page, _, _ := window.changes(t)
	return page
}

// TestCodeMoved_TotalsHasThreeFormsTestedInOrder is `TOTALS`' last segment.
//
// **It has three forms and they are tested in order**: any classed row rendered
// → *the code moved*; otherwise the absence line rendered → *the code could not
// be fully read*; otherwise → *the code did not move*.
//
// **The order is what makes the line honest rather than merely careful.** A
// surviving classed row is positive proof where the absence line is proof of
// nothing either way, so a window in which a fact moved *and* a revision could
// not be read is reported by the fact — the line above having already named
// what went uncounted. What the ordering removes is the one reading this table
// may never produce: the negative asserted over bytes nobody read (§8).
func TestCodeMoved_TotalsHasThreeFormsTestedInOrder(t *testing.T) {
	for _, form := range []struct {
		name string
		ends func(w codeWindow) (baseline, subject seeded)
		want string
	}{
		{
			name: "a classed row rendered",
			ends: func(w codeWindow) (seeded, seeded) {
				return at(w.baseline, "staging"), at(w.subject, "production")
			},
			want: "the code moved",
		},
		{
			// The ordering itself: a Procedure revision that moved
			// stands beside a revision the clone does not hold, and
			// the fact wins over the absence.
			name: "a classed row surviving beside the absence line",
			ends: func(w codeWindow) (seeded, seeded) {
				return at(w.baseline, "staging").naming(absentRevision), at(w.subject, "production")
			},
			want: "the code moved",
		},
		{
			// Both ends read the same revision and both name one
			// the clone does not hold, so no classed fact moved and
			// nothing could be counted.
			name: "the absence line alone",
			ends: func(w codeWindow) (seeded, seeded) {
				return at(w.subject, "production").naming(absentRevision), at(w.subject, "production").naming(absentRevision)
			},
			want: "the code could not be fully read",
		},
		{
			name: "neither",
			ends: func(w codeWindow) (seeded, seeded) {
				return at(w.subject, "production"), at(w.subject, "production")
			},
			want: "the code did not move",
		},
	} {
		t.Run(form.name, func(t *testing.T) {
			held := codeWindowed(t)
			baseline, subject := form.ends(held)
			held.seed(t, baseline, subject)

			page, _, _ := held.changes(t)
			if got := totalsOf(t, page); !strings.HasSuffix(got, " · "+form.want) {
				t.Errorf("TOTALS is %q, want it to end in %q; the table is\n%s", got, form.want, codeTable(t, page))
			}
		})
	}
}

// TestCodeMoved_AWindowThatMovedNothingStillRendersItsTableAndItsCatchAll is
// the empty state.
//
// **All three tables render on every Comparison, header and count, whether or
// not they hold a row**, and the catch-all terminates this one whatever it
// counts: an absent block is ambiguous between *nothing to report* and *the
// renderer had nothing to say* (§8, §12).
func TestCodeMoved_AWindowThatMovedNothingStillRendersItsTableAndItsCatchAll(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.subject, "production"), at(window.subject, "production"))

	want := strings.Join([]string{
		"  THE CODE MOVED   0 facts",
		"  0 other lines changed · git diff " + short(window.subject) + " " + short(window.subject),
	}, "\n")
	if got := codeTable(t, mustPage(t, window)); got != want {
		t.Errorf("the table is\n%s\n\nwant\n%s", got, want)
	}
}

// TestCodeMoved_TheWireCarriesTheValueWholeAndInItsAuthoredShape is the row
// stream (§8).
//
// **No id and no digest is abbreviated anywhere on the wire**, and a `from` or
// `to` that is not a scalar carries **the artefact's own parsed shape** — a
// Bound as a number, a set as an array of names, a selector as the form it was
// written in with its members beneath. The one string that keeps the page's
// abbreviation is the catch-all's `command`, which is a command a reader runs
// rather than an id the row reports, and which git resolves short.
func TestCodeMoved_TheWireCarriesTheValueWholeAndInItsAuthoredShape(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging"), at(window.subject, "production"))

	stream, _, _ := window.changes(t, "--json")
	for _, want := range []string{
		`{"type":"code","subject_kind":"definition","subject":"ci-keys","fact":"kinds","from":["mutate"],"to":["destroy","mutate"]}`,
		`{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"step retire-expired · bound","from":5}`,
		`{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"step issue-runner-keys · over",` +
			`"from":{"values":["ci-arm64","ci-x86","ci-macos"]},"to":{"values":["ci-arm64","ci-x86","ci-macos","ci-arm64-2"]}}`,
		`{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"step retire-expired · over",` +
			`"from":{"assets":[{"field":"expires","older_than":"0s"}]},` +
			`"to":{"assets":[{"field":"created","older_than":"30d"},{"field":"expires","older_than":"0s"}]}}`,
		`{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"cadence","from":"0 0 1 * *","to":"*/5 * * * *",` +
			`"from_phrase":"00:00 UTC on the 1st of the month","to_phrase":"every 5 minutes","from_rate":1,"to_rate":8800}`,
		`{"type":"code","subject_kind":"target","subject":"production","fact":"kinds","to":["destroy","mutate","read"]}`,
		`{"type":"code","fact":"repository revision","from":"` + window.baseline + `","to":"` + window.subject + `"}`,
		`{"type":"code","fact":"other lines changed","count":24,"command":"git diff ` + short(window.baseline) + ` ` + short(window.subject) + `"}`,
	} {
		if !strings.Contains(stream, want+"\n") {
			t.Errorf("the stream carries no line\n%s\nit is\n%s", want, stream)
		}
	}
}

// TestCodeMoved_TheAbsenceCarriesNotInCloneInPlaceOfTheCount is that row on the
// wire: **one name in two positions**, carried beside the `command` it keeps
// and dropping `count` — rather than changing `fact` or nulling the count, an
// omitted `count` being exactly the key merely missing the absence rule
// refuses (§8, §12).
func TestCodeMoved_TheAbsenceCarriesNotInCloneInPlaceOfTheCount(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging").naming(absentRevision), at(window.subject, "production"))

	stream, _, _ := window.changes(t, "--json")
	want := `{"type":"code","fact":"other lines changed","baseline_absent":"not-in-clone","command":"git diff ` +
		short(absentRevision) + ` ` + short(window.subject) + `"}`
	if !strings.Contains(stream, want+"\n") {
		t.Errorf("the stream carries no line\n%s\nit is\n%s", want, stream)
	}
}

// TestCodeMoved_TheTableIsNarrowedByNothing is the three parameters that narrow
// the two Record tables reaching no code fact.
//
// All three range over the identity axis, which is the axis a code fact has no
// coordinate on: narrowing what a reader looked at may not narrow what they are
// told changed (§8, §9).
func TestCodeMoved_TheTableIsNarrowedByNothing(t *testing.T) {
	window := codeWindowed(t)
	window.seed(t, at(window.baseline, "staging"), at(window.subject, "production"))

	whole := codeTable(t, mustPage(t, window))
	for _, narrowed := range [][]string{{"--target", "staging"}, {"--kind", "asset"}, {"--limit", "1"}} {
		page, _, _ := window.changes(t, narrowed...)
		if got := codeTable(t, page); got != whole {
			t.Errorf("hyper changes %s renders\n%s\n\nwant the table the unnarrowed Comparison renders\n%s", strings.Join(narrowed, " "), got, whole)
		}
	}
}
