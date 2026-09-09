package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// `runs`'s three typed parameters past --since and --limit (§9, ADR-0013,
// issue #165). They are read where --since is read, so that the commands
// milestone 8 gives them to inherit one spelling and one refusal rather than
// writing several that agree until they do not.

// TestParseArgs_ReadsRunsTypedParameters. Each is read in both spellings, the
// spaced one and the `=`-joined one every other valued flag takes, so a caller
// who learnt one on --repo-dir has learnt it here.
func TestParseArgs_ReadsRunsTypedParameters(t *testing.T) {
	for name, args := range map[string][]string{
		"the spaced spelling": {
			"--procedure", "retire-preview-envs",
			"--target", "staging",
			"--outcome", "failed",
		},
		"the equals spelling, which every other valued flag takes": {
			"--procedure=retire-preview-envs",
			"--target=staging",
			"--outcome=failed",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			parsed, _, code := parseArgs(runsCommand, args, runsParameters, environment(nil), streams{stderr: &stderr})

			if code != 0 {
				t.Fatalf("parseArgs() = %d, want 0; stderr said %q", code, stderr.String())
			}
			if parsed.procedure != "retire-preview-envs" {
				t.Errorf("procedure = %q, want retire-preview-envs", parsed.procedure)
			}
			if parsed.target != "staging" {
				t.Errorf("target = %q, want staging", parsed.target)
			}
			if parsed.outcome != store.OutcomeFailed {
				t.Errorf("outcome = %q, want %q", parsed.outcome, store.OutcomeFailed)
			}
		})
	}
}

// TestParseArgs_RefusesAnOutcomeOutsideTheTriple. §12 closes the outcome at
// three values and `--outcome` filters that set, so a fourth name selects
// nothing and is refused where it is read rather than carried into a command
// that would answer an empty listing to a caller who typed a state hyper has.
//
// **`open` is the value this is really about.** It is a state and not a member
// of the triple, and a parameter that accepted it would relitigate the
// distinction the outcome cell exists to hold.
//
// Every one of these is a usage error at exit 2 carrying no error_code: an
// error_code names a check that declined an artefact, and a value typed at a
// command line is not one (ADR-0060).
func TestParseArgs_RefusesAnOutcomeOutsideTheTriple(t *testing.T) {
	for name, value := range map[string]string{
		"open, which is a state and not a member of the triple":     "open",
		"contested, which is an entry's account and not an outcome": "contested",
		"a value in the wrong case":                                 "Completed",
		"a value that is nothing at all":                            "",
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			_, _, code := parseArgs(runsCommand, []string{"--outcome", value}, runsParameters, environment(nil), streams{stderr: &stderr})

			if code != ExitUsage {
				t.Fatalf("parseArgs() = %d, want %d", code, ExitUsage)
			}
			said := stderr.String()
			if !strings.Contains(said, "--outcome") {
				t.Errorf("stderr said %q, want the flag named", said)
			}
			for _, member := range []string{"completed", "refused", "failed"} {
				if !strings.Contains(said, member) {
					t.Errorf("stderr said %q, want %s among the values it wanted", said, member)
				}
			}
			if strings.Contains(said, "error_code") {
				t.Errorf("stderr said %q, want no error_code: a value typed at a command line is not a check", said)
			}
		})
	}
}

// TestParseArgs_LeavesRunsParametersUnknownToACommandThatTakesNone. The
// parameters are named at each command's call site, so `hyper providers
// --outcome failed` is the unknown flag it is (§9).
func TestParseArgs_LeavesRunsParametersUnknownToACommandThatTakesNone(t *testing.T) {
	for _, flag := range []string{"--procedure", "--target", "--outcome"} {
		var stderr bytes.Buffer
		_, _, code := parseArgs("providers", []string{flag, "whatever"}, parameters{limit: defaultListLimit}, environment(nil), streams{stderr: &stderr})

		if code != ExitUsage {
			t.Errorf("parseArgs(%s) = %d, want %d", flag, code, ExitUsage)
		}
		if said := stderr.String(); !strings.Contains(said, "unknown flag "+flag) {
			t.Errorf("stderr said %q, want the unknown flag named", said)
		}
	}
}

// TestTruncationLine_NamesTheNarrowingRatherThanALargerLimit is the page's half
// of the truncation marker (§9, §12, ADR-0065).
//
// A command whose axis has parameters that narrow it offers the narrower
// question and nothing else — a larger `--limit` beside it would be the page
// offering a bigger answer where the wire's marker offers a smaller one, and
// the two surfaces may not disagree. A listing over a namespace with nothing to
// narrow keeps the larger cap it has always offered, which is the second form
// here.
//
// The corpus drives the named-cap form; the default is unreachable from a
// checked-in Journal without fifty-one entries in it, so it is asserted here.
func TestTruncationLine_NamesTheNarrowingRatherThanALargerLimit(t *testing.T) {
	for name, c := range map[string]struct {
		parsed    commandArgs
		narrowing render.Narrowing
		want      string
	}{
		"a cap the caller named, on an axis with parameters that narrow it": {
			parsed:    commandArgs{limit: 2, limitNamed: true},
			narrowing: runsNarrowing,
			want:      "returned 2 of 5 Runs; 3 dropped by --limit 2 — narrow with --since or --target",
		},
		"the default cap, on an axis with parameters that narrow it": {
			parsed:    commandArgs{limit: defaultListLimit},
			narrowing: runsNarrowing,
			want:      "returned 50 of 61 Runs; 11 dropped by the default limit of 50 — narrow with --since or --target",
		},
		"the default cap, on a namespace with nothing to narrow": {
			parsed: commandArgs{limit: defaultListLimit},
			want:   "returned 50 of 61 Runs; 11 dropped by the default limit of 50 — name a larger --limit for the rest",
		},
	} {
		t.Run(name, func(t *testing.T) {
			returned, found := 50, 61
			if c.parsed.limitNamed {
				returned, found = 2, 5
			}
			if got := truncationLine("Runs", returned, found, c.parsed, c.narrowing); got != c.want {
				t.Errorf("truncationLine() = %q, want %q", got, c.want)
			}
		})
	}
}

// aListedEntry is one Journal entry as `runs` receives it from the Store: the
// one fact these cases turn on and enough beside it for a row to be built.
func aListedEntry(rehearsal bool) store.Listed {
	return store.Listed{
		Entry: store.Entry{
			RunFile: store.RunFile{
				Run:       aRunID("01991f80-1a2b-7c3d-8e4f-5a6b7c8d9e10"),
				Procedure: "retire-preview-envs",
				DryRun:    rehearsal,
			},
			Owner: store.OutcomeFile{Outcome: store.OutcomeCompleted},
		},
		Targets: []string{"local"},
	}
}

// TestJournalRows_CarriesTheRehearsalMarkerOnEveryRow. The marker goes out on
// every row, the bare `false` included: §7 writes this one always because a
// reader that takes its absence for `false` gets a permanent wrong answer, and
// a row that dropped the `false` would be that absence built at the surface
// (§9, ADR-0160, issue #289).
//
// **There is no third state here**, where `records` has one. Every row on this
// surface is an entry, so there is no join to fail and no absence to spell.
func TestJournalRows_CarriesTheRehearsalMarkerOnEveryRow(t *testing.T) {
	for name, rehearsal := range map[string]bool{
		"a rehearsal":      true,
		"an effecting Run": false,
	} {
		t.Run(name, func(t *testing.T) {
			rows := journalRows([]store.Listed{aListedEntry(rehearsal)}, narrowings{})
			if len(rows) != 1 {
				t.Fatalf("journalRows() returned %d rows, want 1", len(rows))
			}
			row, ok := rows[0].(runRow)
			if !ok {
				t.Fatalf("journalRows() returned a %T, want a runRow", rows[0])
			}
			if row.DryRun != rehearsal {
				t.Errorf("the row carries dry_run: %t, want %t", row.DryRun, rehearsal)
			}
			wire, err := render.MarshalRow(row)
			if err != nil {
				t.Fatalf("encoding the row: %v", err)
			}
			if !strings.Contains(string(wire), `"dry_run":`) {
				t.Errorf("the row encodes as %s, carrying no dry_run — the marker is written always", wire)
			}
		})
	}
}

// TestJournalRows_RendersTheRehearsalAsAWordAndTheOrdinaryRunAsNothing is the
// page half of that exception: the word where there is something to say and a
// blank where there is not, which is the reading `show`'s own header and
// `records`' own column already take over the same marker (§9, ADR-0114,
// ADR-0160).
func TestJournalRows_RendersTheRehearsalAsAWordAndTheOrdinaryRunAsNothing(t *testing.T) {
	at := slices.Index(runsColumns, "REHEARSAL")
	if at < 0 {
		t.Fatal("the page has no REHEARSAL column; the marker has no place to render")
	}
	for name, one := range map[string]struct {
		rehearsal bool
		want      string
	}{
		"a rehearsal":      {rehearsal: true, want: "yes"},
		"an effecting Run": {rehearsal: false, want: ""},
	} {
		t.Run(name, func(t *testing.T) {
			cells := journalRows([]store.Listed{aListedEntry(one.rehearsal)}, narrowings{})[0].Cells()
			if len(cells) != len(runsColumns) {
				t.Fatalf("the row renders %d cells under %d columns", len(cells), len(runsColumns))
			}
			if cells[at] != one.want {
				t.Errorf("REHEARSAL renders %q, want %q", cells[at], one.want)
			}
		})
	}
}
