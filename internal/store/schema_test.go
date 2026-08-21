package store_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The five shapes, read back. Every one of them encodes to canonical bytes and
// decodes to a value equal to the one it was written from, and the bytes are
// re-encoded as part of the read — which is what makes the round trip an
// assertion about the encoding rather than about a pair of functions agreeing
// with one another (§7, issue #129).

// roundTrip is the whole claim for one shape: what came back equals what went
// in, and what came back writes the bytes it was read from.
func roundTrip[T any](t *testing.T, want T, encode func() []byte, decode func([]byte) (T, error)) {
	t.Helper()
	data := encode()

	got, err := decode(data)
	if err != nil {
		t.Fatalf("decode = %v, want it read", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decoded:\n%+v\nwant:\n%+v", got, want)
	}
}

func TestEveryShape_DecodesBackToTheValueItWasWrittenFrom(t *testing.T) {
	t.Run("a Record version", func(t *testing.T) {
		want := recordVersion(t)
		roundTrip(t, want, want.Encode, store.DecodeRecordVersion)
	})

	t.Run("a Tombstone", func(t *testing.T) {
		want := recordVersion(t)
		want.Tombstone, want.Fields = true, nil
		roundTrip(t, want, want.Encode, store.DecodeRecordVersion)
	})

	t.Run("run.json", func(t *testing.T) {
		want := store.RunFile{
			Run:       entryRun(t),
			Procedure: "retire-preview-dns",
			Trigger: store.Trigger{
				Cause:       store.CauseCron,
				Executor:    store.ExecutorGitHubActions,
				Actor:       "TheLoomLabs",
				ExecutorRun: "10925741883",
				Attempt:     2,
				JobURL:      "https://github.com/TheLoomLabs/hyper/actions/runs/10925741883/job/30319482771",
			},
			StartedAt:  theRunStart,
			DryRun:     true,
			Provenance: theProvenance.Run,
		}
		roundTrip(t, want, want.Encode, store.DecodeRunFile)
	})

	t.Run("a Step file", func(t *testing.T) {
		want := stepFile()
		want.Path = "retire.probe"
		want.Pattern = store.Pattern{Attempts: 5}
		want.Answered = store.HTTPAnswer{Host: "api.cloudflare.com", Status: store.Arrived(500)}
		want.ProjectionFailedPath = "$.result.records[0].id"
		roundTrip(t, want, want.Encode, store.DecodeStepFile)
	})

	t.Run("outcome.json", func(t *testing.T) {
		want := store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart}
		roundTrip(t, want, want.Encode, store.DecodeOutcomeFile)
	})

	t.Run("a closed-by file", func(t *testing.T) {
		want := store.ClosedBy{
			EndedAt:  theRunStart,
			Step:     4,
			StepCode: store.StepCode{ID: "retire", Definition: "preview-dns", Operation: "delete_dns_record", Provider: "cloudflare-dns", Target: "cloudflare-prod", Kind: store.KindDestroy},
		}
		roundTrip(t, want, want.Encode, store.DecodeClosedBy)
	})
}

func TestEveryShape_TellsAnAbsentMemberFromAnEmptyOne(t *testing.T) {
	// The two exceptions to the absence rule are the two places the
	// distinction is the whole point: a set that moved to empty and an
	// Expansion that resolved to nothing (§7).
	t.Run("an identity set that moved to empty", func(t *testing.T) {
		want := stepFile()
		want.Identities = store.Identities{Digest: store.IdentityDigest(nil), Members: []string{}}
		roundTrip(t, want, want.Encode, store.DecodeStepFile)
	})

	t.Run("a Step that concluded about nothing", func(t *testing.T) {
		want := stepFile()
		want.Identities = store.Identities{}
		want.Disposition = store.DispositionAttemptedWorldUntouched
		roundTrip(t, want, want.Encode, store.DecodeStepFile)
	})

	t.Run("a selector that expanded to nothing", func(t *testing.T) {
		want := stepFile()
		want.Selector = store.Selector{Declared: theSelector, ExpandedTo: []string{}}
		roundTrip(t, want, want.Encode, store.DecodeStepFile)
	})
}

// The five integers, and the ceiling each one is its own.

func TestSchemaVersions_EachStartAtOne(t *testing.T) {
	for name, version := range map[string]int{
		"a Record version's": store.RecordSchemaVersion,
		"run.json's":         store.RunSchemaVersion,
		"a Step file's":      store.StepSchemaVersion,
		"outcome.json's":     store.OutcomeSchemaVersion,
		"a closed-by file's": store.ClosedBySchemaVersion,
	} {
		if version != 1 {
			t.Errorf("%s schema version = %d, want 1", name, version)
		}
	}
}

func TestEveryShape_RefusesAFileWrittenAboveItsOwnCeiling(t *testing.T) {
	// Each decode reads its own integer and no other, which is what five
	// ceilings buy: an older binary reading a Record file it understands
	// perfectly while a Step file's shape has moved past it (ADR-0028).
	for name, tc := range map[string]struct {
		file    []byte
		version int
		decode  func([]byte) error
	}{
		"a Record version": {recordVersion(t).Encode(), store.RecordSchemaVersion, func(b []byte) error { _, err := store.DecodeRecordVersion(b); return err }},
		"run.json":         {store.RunFile{Run: entryRun(t), Procedure: "p", Trigger: store.Trigger{Cause: store.CauseManual, Executor: store.ExecutorLocal, Actor: "igor"}, StartedAt: theRunStart, Provenance: theProvenance.Run}.Encode(), store.RunSchemaVersion, func(b []byte) error { _, err := store.DecodeRunFile(b); return err }},
		"a Step file":      {stepFile().Encode(), store.StepSchemaVersion, func(b []byte) error { _, err := store.DecodeStepFile(b); return err }},
		"outcome.json":     {store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart}.Encode(), store.OutcomeSchemaVersion, func(b []byte) error { _, err := store.DecodeOutcomeFile(b); return err }},
		"a closed-by file": {store.ClosedBy{EndedAt: theRunStart, Step: 4}.Encode(), store.ClosedBySchemaVersion, func(b []byte) error { _, err := store.DecodeClosedBy(b); return err }},
	} {
		t.Run(name, func(t *testing.T) {
			if err := tc.decode(tc.file); err != nil {
				t.Fatalf("at version %d: %v, want the file read", tc.version, err)
			}

			above := strings.Replace(string(tc.file), `"schema_version": 1`, `"schema_version": 2`, 1)

			var unsupported store.SchemaUnsupported
			err := tc.decode([]byte(above))
			if !errors.As(err, &unsupported) {
				t.Fatalf("at version 2: %v, want the store-schema-unsupported condition", err)
			}
			if unsupported.Written != 2 || unsupported.Known != tc.version {
				t.Errorf("condition = %+v, want the file at 2 and this reader at %d", unsupported, tc.version)
			}
		})
	}
}

func TestSchemaUnsupported_NamesBothVersionsAndCarriesTheCode(t *testing.T) {
	// The decode answers the condition and the caller renders the Refusal:
	// this package holds no Run, renders no terminal line and knows no
	// path, so what it can say is which shape it met and which it reads
	// (§7, §12).
	condition := store.SchemaUnsupported{Written: 2, Known: 1}

	if store.SchemaUnsupportedCode != "store-schema-unsupported" {
		t.Errorf("code = %q, want store-schema-unsupported", store.SchemaUnsupportedCode)
	}
	for _, want := range []string{"2", "1"} {
		if !strings.Contains(condition.Error(), want) {
			t.Errorf("message = %q, want it to name %s", condition.Error(), want)
		}
	}
}

// What a decode will not read. The Store holds files this package wrote, in the
// encoding it wrote them in, so a file outside that is refused at the door
// rather than at the use.

func TestDecode_RefusesAFileTheCanonicalEncodingDidNotProduce(t *testing.T) {
	for name, data := range map[string]string{
		"a key out of code point order": `{
  "outcome": "completed",
  "ended_at": "2026-08-06T09:43:43.319Z",
  "schema_version": 1
}
`,
		"a four-space indent": `{
    "ended_at": "2026-08-06T09:43:43.319Z",
    "outcome": "completed",
    "schema_version": 1
}
`,
		"no trailing newline": `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed",
  "schema_version": 1
}`,
		"a duplicated key": `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed",
  "outcome": "failed",
  "schema_version": 1
}
`,
		"a member outside the shape": `{
  "duration_ms": 3319,
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed",
  "schema_version": 1
}
`,
		"an outcome outside the triple": `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "partially-completed",
  "schema_version": 1
}
`,
		"a null where a member would be": `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": null,
  "schema_version": 1
}
`,
		"a timestamp carrying an offset": `{
  "ended_at": "2026-08-06T11:43:43.319+02:00",
  "outcome": "completed",
  "schema_version": 1
}
`,
		"a timestamp without milliseconds": `{
  "ended_at": "2026-08-06T09:43:43Z",
  "outcome": "completed",
  "schema_version": 1
}
`,
		"no schema version at all": `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed"
}
`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.DecodeOutcomeFile([]byte(data)); err == nil {
				t.Errorf("DecodeOutcomeFile = no error, want the file refused")
			}
		})
	}
}

func TestDecodeClosedBy_RefusesADispositionThatIsNotTheOneItCanCarry(t *testing.T) {
	// Without attempted-outcome-unknown §6's rule has nowhere to land and
	// the crashed Step reads as never reached, which re-runs an effect
	// nobody vouched for (§7).
	data := strings.Replace(
		string(store.ClosedBy{EndedAt: theRunStart, Step: 4}.Encode()),
		`"disposition": "attempted-outcome-unknown"`,
		`"disposition": "ran"`, 1)

	if _, err := store.DecodeClosedBy([]byte(data)); err == nil {
		t.Errorf("DecodeClosedBy = no error, want the file refused")
	}
}

func TestDecodeClosedBy_RefusesAStartedAtItCouldNotKnow(t *testing.T) {
	// The reaper does not know when the Step began, so no closing write
	// carries the key — and one that does was written by something else.
	data := strings.Replace(
		string(store.ClosedBy{EndedAt: theRunStart, Step: 4}.Encode()),
		`  "schema_version": 1,`,
		`  "schema_version": 1,
  "started_at": "2026-08-06T09:41:12.508Z",`, 1)

	if _, err := store.DecodeClosedBy([]byte(data)); err == nil {
		t.Errorf("DecodeClosedBy = no error, want the file refused")
	}
}

// What the shapes will not write. Each of these is a rule §7 states absolutely,
// and none of them can arrive from the world — each is hyper's own account of a
// call it made, its own counter, or its own reading of what a Step became — so
// a violation is hyper's arithmetic being wrong, which paths.go answers the
// same way.

func TestEveryShape_RefusesToWriteWhatSectionSevenSaysCannotExist(t *testing.T) {
	for name, encode := range map[string]func(){
		"an answer naming neither a host nor a command": func() {
			file := stepFile()
			file.Answered = store.ShellAnswer{}
			file.Encode()
		},
		"an http answer naming no host": func() {
			file := stepFile()
			file.Answered = store.HTTPAnswer{Status: store.Arrived(500)}
			file.Encode()
		},
		"an identity set carrying no digest": func() {
			file := stepFile()
			file.Identities = store.Identities{Members: []string{"preview-42.example.com"}}
			file.Encode()
		},
		"a Step carrying no Provenance": func() {
			file := stepFile()
			file.Provenance = store.StepProvenance{}
			file.Encode()
		},
		"a Run carrying no Provenance": func() {
			file := runFile(t)
			file.Provenance = store.RunProvenance{}
			file.Encode()
		},
		"a Step file wearing the Disposition read from a silence": func() {
			file := stepFile()
			file.Disposition = store.DispositionNeverReached
			file.Encode()
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Encode() = no panic, want the file refused")
				}
			}()
			encode()
		})
	}
}

// TestEncode_AVersionCarryingNoFieldsIsWritten is the state a projection that
// resolved to nothing at every path leaves: the key is absent and the version
// is a version like any other (§6, §7, issue #142).
//
// A `shell` command that could not be started at all is where it arrives — the
// response object is `command` and nothing else, and the built-in Provider
// projects `exit_code`, `stdout` and `stderr` — and it is the ordinary field
// absence §6 states applied to all of a projection at once rather than a shape
// of its own. What keeps it apart from a Tombstone opening the series it ends
// is the marker beside it and never the absence.
func TestEncode_AVersionCarryingNoFieldsIsWritten(t *testing.T) {
	version := recordVersion(t)
	version.Fields = nil

	written := string(version.Encode())
	if strings.Contains(written, `"fields"`) {
		t.Errorf("a version carrying no projected content writes a fields key:\n%s", written)
	}

	read, err := store.DecodeRecordVersion([]byte(written))
	if err != nil {
		t.Fatalf("the version would not read back: %v", err)
	}
	if read.Fields != nil {
		t.Errorf("the version read back with fields %v, want none at all", read.Fields)
	}
	if read.Tombstone {
		t.Error("the version read back as a Tombstone, and what tells the two apart is the marker rather than the absence")
	}
}

func TestDecodeStepFile_RefusesAFileThisPackageCouldNotHaveWritten(t *testing.T) {
	written := string(stepFile().Encode())

	for name, data := range map[string]string{
		"members out of code point order": strings.Replace(written,
			`      "preview-17.example.com",
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "kind"`,
			`      "preview-42.example.com",
      "preview-17.example.com",
      "preview-8.example.com"
    ]
  },
  "kind"`, 1),
		"a Step at no position":           strings.Replace(written, `"step": 3`, `"step": -1`, 1),
		"a Disposition outside the seven": strings.Replace(written, `"disposition": "ran"`, `"disposition": "went-fine"`, 1),
		"a Kind outside the three":        strings.Replace(written, `"kind": "destroy"`, `"kind": "delete"`, 1),
		"a Run-wide provenance member": strings.Replace(written, `    "definition_revision"`, `    "hyper_version": "1.4.0",
    "definition_revision"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.DecodeStepFile([]byte(data)); err == nil {
				t.Errorf("DecodeStepFile = no error, want the file refused")
			}
		})
	}
}

func TestDecodeStepFile_RefusesTheDispositionThatIsReadFromASilence(t *testing.T) {
	// Six of §12's seven values are borne by a Step file. *never reached*
	// is the seventh, read from the absence of one inside a closed entry —
	// so a file wearing it is a file claiming to be its own absence (§7).
	data := strings.Replace(string(stepFile().Encode()), `"disposition": "ran"`, `"disposition": "never-reached"`, 1)

	if _, err := store.DecodeStepFile([]byte(data)); err == nil {
		t.Errorf("DecodeStepFile = no error, want the file refused")
	}
}

func TestOutcomeFile_DerivesItsHeadFromTheFirstMemberAndStoresItNowhere(t *testing.T) {
	// The order is the order check prints in, by file path and then by
	// line, and what the terminal line names is the first member's
	// error_code (§7, §8).
	file := store.OutcomeFile{
		Outcome: store.OutcomeRefused,
		EndedAt: theRunStart,
		Refusal: []store.RefusalMember{
			{ErrorCode: "bound-exceeded", File: "procedures/retire-preview-envs.yaml", Line: 33, Message: "expansion resolved 23 assets on staging"},
			{ErrorCode: "bound-missing", File: "procedures/retire-preview-envs.yaml", Line: 41, Message: "no bound on an effectful Step"},
		},
	}

	if got := file.Head(); got != "bound-exceeded" {
		t.Errorf("head = %q, want the first member's error_code", got)
	}
	if got := (store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart}); got.Head() != "" {
		t.Errorf("head of a Run that did not refuse = %q, want nothing", got.Head())
	}
}

func TestAnswer_ReadsBackTheCodeAndWhetherOneArrived(t *testing.T) {
	// The two come back together: reading the first without the second is
	// how a request that never left acquires a status of 0 (§7, ADR-0050).
	if code, arrived := store.Arrived(500).Code(); code != 500 || !arrived {
		t.Errorf("Arrived(500).Code() = %d, %v, want 500, true", code, arrived)
	}

	var never store.Answer
	if code, arrived := never.Code(); code != 0 || arrived {
		t.Errorf("the zero Answer = %d, %v, want 0, false", code, arrived)
	}
}

func TestDecode_RefusesAMemberOfTheWrongType(t *testing.T) {
	// The shapes are closed and so is what each key holds, so a member
	// spelling its value another way is a member this package did not
	// write — and reading it would give a file a meaning nothing gave it.
	written := string(stepFile().Encode())

	for name, data := range map[string]string{
		"a string where a number goes": strings.Replace(written, `"step": 3`, `"step": "3"`, 1),
		"a number where a string goes": strings.Replace(written, `"id": "retire"`, `"id": 3`, 1),
		"a number that is not whole":   strings.Replace(written, `"step": 3`, `"step": 3.5`, 1),
		"a mapping where a list goes":  strings.Replace(written, `"expanded_to": [`, `"expanded_to": {"0": [`, 1),
		"a number inside a list of names": strings.Replace(written,
			`      "preview-17.example.com",
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "kind"`,
			`      17,
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "kind"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if data == written {
				t.Fatalf("the case changed nothing")
			}
			if _, err := store.DecodeStepFile([]byte(data)); err == nil {
				t.Errorf("DecodeStepFile = no error, want the file refused")
			}
		})
	}
}

func TestDecodeRecordVersion_RefusesARunIdThatIsNotOne(t *testing.T) {
	// An id that reached a path unchecked would name a directory nothing
	// could ever find again by looking for the id it was told (§12).
	data := strings.Replace(string(recordVersion(t).Encode()),
		`"run_id": "`+theEntryRunID+`"`, `"run_id": "the-last-run"`, 1)

	if _, err := store.DecodeRecordVersion([]byte(data)); err == nil {
		t.Errorf("DecodeRecordVersion = no error, want the file refused")
	}
}

func TestDecodeOutcomeFile_RefusesARefusalMemberThatIsNotAMapping(t *testing.T) {
	const data = `{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "refused",
  "refusal": [
    "bound-exceeded"
  ],
  "schema_version": 1
}
`
	if _, err := store.DecodeOutcomeFile([]byte(data)); err == nil {
		t.Errorf("DecodeOutcomeFile = no error, want the file refused")
	}
}

func TestDecode_RefusesWhatIsNotOneJSONValue(t *testing.T) {
	for name, data := range map[string]string{
		"nothing at all": "",
		"a second value after the first": `{"schema_version": 1}
{"schema_version": 1}
`,
		"a list where a file goes": "[]\n",
		"bytes that are not JSON":  "not json\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.DecodeOutcomeFile([]byte(data)); err == nil {
				t.Errorf("DecodeOutcomeFile = no error, want the bytes refused")
			}
		})
	}
}
