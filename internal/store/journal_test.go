package store_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Journal's four shapes, asserted against the bytes §7 publishes for three
// of them and against the members it enumerates for the fourth (issue #129).
//
// Every expected value here is the specification's own. §7 prints run.json, a
// Step file and outcome.json in full, so those three are compared byte for
// byte rather than through a decode: a version is minted where the bytes moved,
// and a shape that encodes to a plausible file is a shape with no signal in it.

// theRunStart is the clock every fixture in this file is threaded, and the
// instants around it are its own offsets, so a file's timestamps read as one
// Run rather than as five unrelated literals.
var theRunStart = time.Date(2026, 8, 6, 9, 41, 12, 508_000_000, time.UTC)

func TestOutcomeFile_IsTheFileSectionSevenPublishes(t *testing.T) {
	file := store.OutcomeFile{
		Outcome: store.OutcomeCompleted,
		EndedAt: time.Date(2026, 8, 6, 9, 43, 43, 319_000_000, time.UTC),
	}

	const want = `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed",
  "schema_version": 1
}
`
	if got := string(file.Encode()); got != want {
		t.Errorf("outcome.json:\n%s\nwant:\n%s", got, want)
	}
}

func TestOutcomeFile_CarriesTheRefusalSectionSevenPublishes(t *testing.T) {
	file := store.OutcomeFile{
		Outcome: store.OutcomeRefused,
		EndedAt: time.Date(2026, 8, 6, 9, 43, 43, 319_000_000, time.UTC),
		Refusal: []store.RefusalMember{{
			ErrorCode: "bound-exceeded",
			File:      "procedures/retire-preview-envs.yaml",
			Line:      33,
			Field:     "steps[2].bound",
			Message:   "expansion resolved 23 assets on staging",
			Step:      3,
			StepID:    "retire",
			Declared:  store.Int(5),
			Observed:  store.Int(23),
		}},
	}

	const want = `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "refused",
  "refusal": [
    {
      "declared": 5,
      "error_code": "bound-exceeded",
      "field": "steps[2].bound",
      "file": "procedures/retire-preview-envs.yaml",
      "line": 33,
      "message": "expansion resolved 23 assets on staging",
      "observed": 23,
      "step": 3,
      "step_id": "retire"
    }
  ],
  "schema_version": 1
}
`
	if got := string(file.Encode()); got != want {
		t.Errorf("outcome.json:\n%s\nwant:\n%s", got, want)
	}
}

func TestOutcomeFile_InventsNothingToFillAMemberThatDoesNotApply(t *testing.T) {
	// A check that cited no Step and compared no two values: the four keys
	// it has nothing for are absent, and nothing stands in their place.
	file := store.OutcomeFile{
		Outcome: store.OutcomeRefused,
		EndedAt: theRunStart,
		Refusal: []store.RefusalMember{{
			ErrorCode: "store-absent",
			File:      "hyper.yaml",
			Message:   "the Store branch is not here and not on origin",
		}},
	}

	const want = `{
  "ended_at": "2026-08-06T09:41:12.508Z",
  "outcome": "refused",
  "refusal": [
    {
      "error_code": "store-absent",
      "file": "hyper.yaml",
      "message": "the Store branch is not here and not on origin"
    }
  ],
  "schema_version": 1
}
`
	if got := string(file.Encode()); got != want {
		t.Errorf("outcome.json:\n%s\nwant:\n%s", got, want)
	}
}

// theEntryRunID is the Run id §7 writes across every file of the Journal entry
// it publishes, and it is that entry's own rather than the id paths_test.go
// builds its paths from: both are the specification's, and a published file is
// compared against the id the specification printed inside it.
const theEntryRunID = "01991e21-3c9f-7b04-9d18-5c7e2a94f083"

// entryRun is theEntryRunID as the shapes take one. It is well formed, so a
// failure here has found a broken fixture rather than a broken shape.
func entryRun(t *testing.T) store.RunID {
	t.Helper()
	id, err := store.ParseRunID(theEntryRunID)
	if err != nil {
		t.Fatalf("ParseRunID(%q) = %v, want the specification's own id accepted", theEntryRunID, err)
	}
	return id
}

// runFile is §7's own worked run.json, which every case in the corpus varies
// one member of rather than rebuilding whole.
func runFile(t *testing.T) store.RunFile {
	t.Helper()
	return store.RunFile{
		Run:       entryRun(t),
		Procedure: "retire-preview-dns",
		Trigger: store.Trigger{
			Cause:    store.CauseManual,
			Executor: store.ExecutorLocal,
			Actor:    "igor",
			Host:     "thinkpad",
		},
		StartedAt:  theRunStart,
		Provenance: theProvenance.Run,
	}
}

func TestRunFile_IsTheFileSectionSevenPublishes(t *testing.T) {
	file := runFile(t)

	const want = `{
  "dry_run": false,
  "procedure": "retire-preview-dns",
  "provenance": {
    "hyper_version": "1.4.0",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "run_id": "01991e21-3c9f-7b04-9d18-5c7e2a94f083",
  "schema_version": 1,
  "started_at": "2026-08-06T09:41:12.508Z",
  "trigger": {
    "actor": "igor",
    "cause": "manual",
    "executor": "local",
    "host": "thinkpad"
  }
}
`
	if got := string(file.Encode()); got != want {
		t.Errorf("run.json:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunFile_CarriesTheActionsOccasionOnActionsAndTheHostOnLocal(t *testing.T) {
	for name, tc := range map[string]struct {
		trigger store.Trigger
		want    string
	}{
		"on local, the host and no occasion": {
			trigger: store.Trigger{
				Cause:    store.CauseManual,
				Executor: store.ExecutorLocal,
				Actor:    "igor",
				Host:     "thinkpad",
			},
			want: `  "trigger": {
    "actor": "igor",
    "cause": "manual",
    "executor": "local",
    "host": "thinkpad"
  }`,
		},
		"on Actions, the occasion and no host": {
			trigger: store.Trigger{
				Cause:       store.CauseCron,
				Executor:    store.ExecutorGitHubActions,
				Actor:       "TheLoomLabs",
				ExecutorRun: "10925741883",
				Attempt:     2,
				JobURL:      "https://github.com/TheLoomLabs/hyper/actions/runs/10925741883/job/30319482771",
			},
			want: `  "trigger": {
    "actor": "TheLoomLabs",
    "cause": "cron",
    "executor": "github-actions",
    "job_url": "https://github.com/TheLoomLabs/hyper/actions/runs/10925741883/job/30319482771",
    "run_attempt": 2,
    "run_id": "10925741883"
  }`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			file := runFile(t)
			file.Trigger = tc.trigger

			if got := string(file.Encode()); !strings.Contains(got, tc.want) {
				t.Errorf("run.json:\n%s\nwant it to carry:\n%s", got, tc.want)
			}
		})
	}
}

func TestRunFile_WritesDryRunOnEveryEntryIncludingFalse(t *testing.T) {
	// The one marker in the Store exempt from the absence rule. A reader
	// taking its absence for false refuses every run-once Step in the
	// Procedure it rehearsed, permanently (§7, ADR-0001).
	for _, dryRun := range []bool{false, true} {
		file := runFile(t)
		file.DryRun = dryRun

		want := `"dry_run": ` + map[bool]string{false: "false", true: "true"}[dryRun]
		if got := string(file.Encode()); !strings.Contains(got, want) {
			t.Errorf("run.json:\n%s\nwant it to carry %s", got, want)
		}
	}
}

// The Step file. It is the Journal's largest shape and the one §8 reads
// Dispositions off, and §7 publishes it in full — the selector as authored, the
// identity set, the code facts held rather than resolved.

// theExpansion is the three names §7's worked Step both expanded to and
// concluded about, in the one order both are written in there.
var theExpansion = []string{"preview-17.example.com", "preview-42.example.com", "preview-8.example.com"}

// theSelector is the `assets:` predicate §7's Step declared, as authored.
var theSelector = store.Mapping{"assets": store.Array{
	store.Mapping{"field": store.String("name"), "starts_with": store.String("preview-")},
	store.Mapping{"field": store.String("created_on"), "older_than": store.String("14d")},
}}

func stepFile() store.StepFile {
	return store.StepFile{
		Step: 3,
		StepCode: store.StepCode{
			ID:         "retire",
			Definition: "preview-dns",
			Operation:  "delete_dns_record",
			Provider:   "cloudflare-dns",
			Target:     "cloudflare-prod",
			Kind:       store.KindDestroy,
		},
		Disposition: store.DispositionRan,
		StartedAt:   time.Date(2026, 8, 6, 9, 43, 35, 890_000_000, time.UTC),
		EndedAt:     time.Date(2026, 8, 6, 9, 43, 38, 105_000_000, time.UTC),
		Provenance: store.StepProvenance{
			DefinitionRevision: "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
			ManifestDigest:     "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
		},
		Identities: store.Identities{
			Digest:  "sha256:6f1c8d0a4b93e527f10c6ba8d34e79521f0badc6e84397b210f5cd6e0a4b7f38",
			Members: theExpansion,
		},
		Selector: store.Selector{Declared: theSelector, ExpandedTo: theExpansion, Bound: 5},
	}
}

func TestStepFile_IsTheFileSectionSevenPublishes(t *testing.T) {
	const want = `{
  "definition": "preview-dns",
  "disposition": "ran",
  "ended_at": "2026-08-06T09:43:38.105Z",
  "id": "retire",
  "identities": {
    "digest": "sha256:6f1c8d0a4b93e527f10c6ba8d34e79521f0badc6e84397b210f5cd6e0a4b7f38",
    "members": [
      "preview-17.example.com",
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "kind": "destroy",
  "operation": "delete_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1"
  },
  "provider": "cloudflare-dns",
  "schema_version": 1,
  "selector": {
    "bound": 5,
    "declared": {
      "assets": [
        {
          "field": "name",
          "starts_with": "preview-"
        },
        {
          "field": "created_on",
          "older_than": "14d"
        }
      ]
    },
    "expanded_to": [
      "preview-17.example.com",
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "started_at": "2026-08-06T09:43:35.890Z",
  "step": 3,
  "target": "cloudflare-prod"
}
`
	if got := string(stepFile().Encode()); got != want {
		t.Errorf("step file:\n%s\nwant:\n%s", got, want)
	}
}

func TestStepFile_CarriesADigestAloneWhereTheDigestDidNotMove(t *testing.T) {
	file := stepFile()
	file.Identities.Members = nil

	const want = `  "identities": {
    "digest": "sha256:6f1c8d0a4b93e527f10c6ba8d34e79521f0badc6e84397b210f5cd6e0a4b7f38"
  },`
	if got := string(file.Encode()); !strings.Contains(got, want) {
		t.Errorf("step file:\n%s\nwant it to carry:\n%s", got, want)
	}
}

func TestStepFile_WritesTheEmptySetInFullWhereTheDigestMovedToIt(t *testing.T) {
	// The exception the empty list earns: absence here already means *the
	// digest did not move*, so a reader would otherwise decode *we looked
	// and saw nothing* from recognising a constant (§7).
	file := stepFile()
	file.Identities = store.Identities{Digest: store.IdentityDigest(nil), Members: []string{}}

	const want = `  "identities": {
    "digest": "sha256:37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570",
    "members": []
  },`
	if got := string(file.Encode()); !strings.Contains(got, want) {
		t.Errorf("step file:\n%s\nwant it to carry:\n%s", got, want)
	}
}

func TestStepFile_WritesExpandedToWheneverASelectorExists(t *testing.T) {
	file := stepFile()
	file.Selector = store.Selector{Declared: store.Array{store.String("preview-42.example.com")}}

	const want = `  "selector": {
    "declared": [
      "preview-42.example.com"
    ],
    "expanded_to": []
  },`
	if got := string(file.Encode()); !strings.Contains(got, want) {
		t.Errorf("step file:\n%s\nwant it to carry:\n%s", got, want)
	}
}

func TestStepFile_CarriesNoSelectorWhereTheStepDeclaredNone(t *testing.T) {
	file := stepFile()
	file.Selector = store.Selector{}

	if got := string(file.Encode()); strings.Contains(got, "selector") || strings.Contains(got, "expanded_to") {
		t.Errorf("step file:\n%s\nwant no selector key at all", got)
	}
}

func TestStepFile_CarriesTheAnsweredSectionSevenPublishesForEachCapability(t *testing.T) {
	for name, tc := range map[string]struct {
		answered store.Answered
		want     string
	}{
		"an http status that came back": {
			answered: store.HTTPAnswer{Host: "api.cloudflare.com", Status: store.Arrived(500)},
			want: `  "answered": {
    "host": "api.cloudflare.com",
    "status": 500
  },`,
		},
		"a request that never left": {
			answered: store.HTTPAnswer{Host: "api.cloudflare.com"},
			want: `  "answered": {
    "host": "api.cloudflare.com"
  },`,
		},
		"a command that exited nonzero": {
			answered: store.ShellAnswer{Command: `["rm","-rf","/srv/app/releases/r41"]`, ExitCode: store.Arrived(1)},
			want: `  "answered": {
    "command": "[\"rm\",\"-rf\",\"/srv/app/releases/r41\"]",
    "exit_code": 1
  },`,
		},
		"a child that never started": {
			answered: store.ShellAnswer{Command: `["rm","-rf","/srv/app/releases/r41"]`},
			want: `  "answered": {
    "command": "[\"rm\",\"-rf\",\"/srv/app/releases/r41\"]"
  },`,
		},
		"a command that exited zero after all": {
			// 0 is an exit code a command really gives, so the
			// absence above is carried by the Answer and never by
			// the value.
			answered: store.ShellAnswer{Command: `["true"]`, ExitCode: store.Arrived(0)},
			want: `  "answered": {
    "command": "[\"true\"]",
    "exit_code": 0
  },`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			file := stepFile()
			file.Answered = tc.answered

			if got := string(file.Encode()); !strings.Contains(got, tc.want) {
				t.Errorf("step file:\n%s\nwant it to carry:\n%s", got, tc.want)
			}
		})
	}
}

func TestStepFile_IsNeverWrittenWithAnEmptyAnswered(t *testing.T) {
	// A Step that carries none writes no key, and one that carries one
	// always has a host or a command inside it — the two shapes have no
	// empty form between them.
	file := stepFile()

	if got := string(file.Encode()); strings.Contains(got, "answered") {
		t.Errorf("step file:\n%s\nwant no answered key at all", got)
	}
}

func TestStepFile_CarriesPathOnlyWhereItWasReachedThroughANestedInvocation(t *testing.T) {
	file := stepFile()
	if got := string(file.Encode()); strings.Contains(got, `"path"`) {
		t.Errorf("step file:\n%s\nwant a top-level Step to carry no path", got)
	}

	file.Path = "retire.probe"
	if got := string(file.Encode()); !strings.Contains(got, `"path": "retire.probe"`) {
		t.Errorf("step file:\n%s\nwant it to carry the invocation chain", got)
	}
}

func TestStepFile_CarriesTheProjectionPathThatFailedToResolve(t *testing.T) {
	file := stepFile()
	file.ProjectionFailedPath = "$.result.records[0].id"

	if got := string(file.Encode()); !strings.Contains(got, `"projection_failed_path": "$.result.records[0].id"`) {
		t.Errorf("step file:\n%s\nwant it to carry the path that failed", got)
	}
}

func TestStepFile_CarriesTheKindAndProviderAsStoredFacts(t *testing.T) {
	// Both are derivable — a Step names a Definition and a Definition names
	// its Provider and its Operation's Kind — and holding them is what
	// keeps a Disposition readable without fetching three artefacts at the
	// revision that Run names (§7).
	got := string(stepFile().Encode())

	for _, want := range []string{`"kind": "destroy"`, `"provider": "cloudflare-dns"`} {
		if !strings.Contains(got, want) {
			t.Errorf("step file:\n%s\nwant it to carry %s", got, want)
		}
	}
}

// The closing write. A Run that infers another died creates one path inside the
// dead Run's entry and edits nothing the dead Run wrote (§7, ADR-0011,
// ADR-0076). §7 does not publish its bytes, so what is asserted here is the
// member list it states: what a reaper knows, and what it declines to invent.

func TestClosedBy_CarriesWhatAReaperKnows(t *testing.T) {
	file := store.ClosedBy{
		EndedAt: time.Date(2026, 8, 6, 10, 12, 0, 0, time.UTC),
		Step:    4,
		StepCode: store.StepCode{
			ID:         "retire",
			Definition: "preview-dns",
			Operation:  "delete_dns_record",
			Provider:   "cloudflare-dns",
			Target:     "cloudflare-prod",
			Kind:       store.KindDestroy,
		},
	}

	const want = `{
  "definition": "preview-dns",
  "disposition": "attempted-outcome-unknown",
  "ended_at": "2026-08-06T10:12:00.000Z",
  "id": "retire",
  "kind": "destroy",
  "operation": "delete_dns_record",
  "provider": "cloudflare-dns",
  "schema_version": 1,
  "step": 4,
  "target": "cloudflare-prod"
}
`
	if got := string(file.Encode()); got != want {
		t.Errorf("closed-by file:\n%s\nwant:\n%s", got, want)
	}
}

func TestClosedBy_OmitsTheCodeFactsTheDeadRunsRevisionDoesNotResolve(t *testing.T) {
	// Every Run that recorded repo_dirty is this case: the revision names a
	// tree the reaper cannot reconstruct, so the four keys it would have
	// filled are absent and nothing stands in their place.
	file := store.ClosedBy{EndedAt: time.Date(2026, 8, 6, 10, 12, 0, 0, time.UTC), Step: 4}

	const want = `{
  "disposition": "attempted-outcome-unknown",
  "ended_at": "2026-08-06T10:12:00.000Z",
  "schema_version": 1,
  "step": 4
}
`
	if got := string(file.Encode()); got != want {
		t.Errorf("closed-by file:\n%s\nwant:\n%s", got, want)
	}
}

// The decode. Every shape reads back to the value it was written from, and the
// bytes it was read from are re-encoded and compared as part of the read — so
// a file this package did not write is refused at the door rather than at the
// use (§7, issue #129).

func TestDecodeOutcomeFile_ReadsBackWhatItWrote(t *testing.T) {
	want := store.OutcomeFile{
		Outcome: store.OutcomeRefused,
		EndedAt: time.Date(2026, 8, 6, 9, 43, 43, 319_000_000, time.UTC),
		Refusal: []store.RefusalMember{{
			ErrorCode: "bound-exceeded",
			File:      "procedures/retire-preview-envs.yaml",
			Line:      33,
			Field:     "steps[2].bound",
			Message:   "expansion resolved 23 assets on staging",
			Step:      3,
			StepID:    "retire",
			Declared:  store.Int(5),
			Observed:  store.Int(23),
		}},
	}

	got, err := store.DecodeOutcomeFile(want.Encode())
	if err != nil {
		t.Fatalf("DecodeOutcomeFile = %v, want it read", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decoded outcome.json = %+v, want %+v", got, want)
	}
}

func TestOutcomeFile_RefusesToWriteARefusalOfNoMembers(t *testing.T) {
	// An ordered array of at least one member, and a Refusal is what
	// declined this Run: an outcome of `refused` with no check to name is
	// hyper's own bookkeeping being wrong, not a file to write (§7).
	for name, file := range map[string]store.OutcomeFile{
		"refused, naming no check": {Outcome: store.OutcomeRefused, EndedAt: theRunStart},
		"completed, naming one":    {Outcome: store.OutcomeCompleted, EndedAt: theRunStart, Refusal: []store.RefusalMember{{ErrorCode: "store-absent", File: "hyper.yaml", Message: "no Store"}}},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Encode() = no panic, want the file refused")
				}
			}()
			file.Encode()
		})
	}
}

func TestDecodeOutcomeFile_RefusesAFileWhoseRefusalDoesNotMatchItsOutcome(t *testing.T) {
	const completedWithARefusal = `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed",
  "refusal": [
    {
      "error_code": "store-absent",
      "file": "hyper.yaml",
      "message": "no Store"
    }
  ],
  "schema_version": 1
}
`
	const refusedWithNone = `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "refused",
  "schema_version": 1
}
`
	for name, data := range map[string]string{
		"completed, carrying a refusal": completedWithARefusal,
		"refused, carrying none":        refusedWithNone,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.DecodeOutcomeFile([]byte(data)); err == nil {
				t.Errorf("DecodeOutcomeFile = no error, want the file refused")
			}
		})
	}
}

func TestStepFile_WritesExpandedToInExpansionOrderRatherThanSorted(t *testing.T) {
	// On a serial destroy it is the only place the halt point is legible,
	// and *which three of the five* is read off it by position (§7). The
	// identity set beside it is sorted, and the two differ wherever a
	// `values:` list is the selector.
	file := stepFile()
	order := []string{"preview-8.example.com", "preview-42.example.com", "preview-17.example.com"}
	file.Selector = store.Selector{Declared: theSelector, ExpandedTo: order}

	const want = `    "expanded_to": [
      "preview-8.example.com",
      "preview-42.example.com",
      "preview-17.example.com"
    ]`
	if got := string(file.Encode()); !strings.Contains(got, want) {
		t.Errorf("step file:\n%s\nwant it to carry:\n%s", got, want)
	}
}

func TestOutcomeFile_StoresNoExitCodeNoDurationAndNoAuthor(t *testing.T) {
	// All three derive — the exit code from the outcome by §12's mapping,
	// the duration from the instants the entry already holds, the author
	// from the <run-id> in the file's own path — and a stored one is a
	// second representation that can disagree (§7, ADR-0076).
	file := store.OutcomeFile{
		Outcome: store.OutcomeRefused,
		EndedAt: theRunStart,
		Refusal: []store.RefusalMember{{ErrorCode: "bound-exceeded", File: "procedures/retire-preview-envs.yaml", Message: "expansion resolved 23 assets on staging"}},
	}

	got := string(file.Encode())
	for _, absent := range []string{"exit_code", "duration", "run_id", "started_at"} {
		if strings.Contains(got, absent) {
			t.Errorf("outcome.json:\n%s\nwant no %s in it", got, absent)
		}
	}
	// The head of the array is derived and never stored: what a terminal
	// line names is the first member's error_code, read from inside the
	// array and from no key beside it.
	if strings.Count(got, "error_code") != 1 {
		t.Errorf("outcome.json:\n%s\nwant the error_code only inside the array", got)
	}
}
