package compare_test

import (
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/compare"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The two Record tables (§8, issue #170): `YOU DID THIS` and `THE WORLD MOVED`.

// window is the shape every case here starts from: two Runs of one Procedure,
// the baseline ending at 09:02 and the subject at 11:02, each with the Step
// records the case hands it.
func window(baseline, subject []store.StepFile) compare.Window {
	return compare.Window{
		Procedure: "watch",
		Baseline:  compare.Side{Present: true, Entry: run("11", "watch", at(9, 0), at(9, 2)), Steps: baseline},
		Subject:   compare.Side{Present: true, Entry: run("13", "watch", at(11, 0), at(11, 2)), Steps: subject},
	}
}

// concluded is one Step record carrying an identity set in full.
func concluded(id string, target, definition string, names ...string) store.StepFile {
	return store.StepFile{
		Step:        1,
		StepCode:    store.StepCode{ID: id, Target: target, Definition: definition, Kind: store.KindRead},
		Disposition: store.DispositionRan,
		EndedAt:     at(9, 1),
		Identities:  store.Concluded(names, ""),
	}
}

// version is one version of a series as a listing answers it.
func version(id store.Identity, kind store.RecordType, ordinal int, written time.Time) store.Version {
	return store.Version{
		Metadata: store.Metadata{Identity: id, RecordType: kind, WrittenAt: written},
		File:     store.RecordPath(id, runID("0"+string(rune('0'+ordinal))), ordinal),
		Ordinal:  ordinal,
	}
}

// series is a Record's versions, oldest first, in the order the Head comes off.
func series(id store.Identity, kind store.RecordType, written ...time.Time) store.Series {
	held := store.Series{Identity: id}
	for i, at := range written {
		held.Versions = append(held.Versions, version(id, kind, i+1, at))
	}
	return held
}

// fields is a projected content mapping as a case writes one.
func fields(pairs ...string) store.Mapping {
	mapping := store.Mapping{}
	for i := 0; i < len(pairs); i += 2 {
		mapping[pairs[i]] = store.String(pairs[i+1])
	}
	return mapping
}

// changeRows is the rows of the two Record tables, the `window` row dropped.
func changeRows(t *testing.T, window compare.Window, records []compare.Record) []compare.ChangeRow {
	t.Helper()

	var held []compare.ChangeRow
	for _, row := range compare.Rows(window, records, compare.Code{}) {
		if change, is := row.(compare.ChangeRow); is {
			held = append(held, change)
		}
	}
	return held
}

// onlyRow is the one row a case expects, and a failure where it drew some other
// number.
func onlyRow(t *testing.T, window compare.Window, records []compare.Record) compare.ChangeRow {
	t.Helper()

	rows := changeRows(t, window, records)
	if len(rows) != 1 {
		t.Fatalf("the window drew %d rows, want exactly one: %v", len(rows), rows)
	}
	return rows[0]
}

var preview = store.Identity{Target: "staging", Definition: "hetzner-staging", Name: "preview-8801"}

func TestEligible_IsTheIdentitySetsUnionAndNotTheStores(t *testing.T) {
	mine := store.Identity{Target: "staging", Definition: "hetzner-staging", Name: "preview-1"}
	theirs := store.Identity{Target: "staging", Definition: "hetzner-staging", Name: "preview-2"}
	eligible := compare.Eligible(window(
		[]store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-2")},
		[]store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-1")},
	))
	if len(eligible) != 2 || eligible[0] != mine || eligible[1] != theirs {
		t.Errorf("Eligible() = %v, want the union of the two sides' sets in identity order", eligible)
	}
}

func TestEligible_ReadsTheStepsTargetAndDefinition(t *testing.T) {
	// A set holds the names a Step concluded about; the Step it sits in is
	// what says which series each of them belongs to (§7).
	eligible := compare.Eligible(window(nil, []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")}))
	want := store.Identity{Target: "local", Definition: "uptime", Name: "cert.hyper.dev"}
	if len(eligible) != 1 || eligible[0] != want {
		t.Errorf("Eligible() = %v, want %v", eligible, want)
	}
}

func TestEligible_ADispositionCarryingNoSetContributesNothing(t *testing.T) {
	skipped := store.StepFile{
		StepCode:    store.StepCode{ID: "label", Target: "staging", Definition: "hetzner-staging"},
		Disposition: store.DispositionSkippedByCondition,
	}
	if eligible := compare.Eligible(window(nil, []store.StepFile{skipped})); len(eligible) != 0 {
		t.Errorf("Eligible() = %v, want nothing: a Step whose Disposition carries no set concludes about nothing", eligible)
	}
}

func TestEndpoint_IsTheHeadAtThatEndsInstant(t *testing.T) {
	held := series(preview, store.RecordAsset, at(8, 0), at(10, 0), at(12, 0))
	compared := window(nil, nil)

	baseline, stands := compare.Endpoint(compared.Baseline, held)
	if !stands || baseline.Ordinal != 1 {
		t.Errorf("the baseline end is ordinal %d (held %v), want the version standing at 09:02", baseline.Ordinal, stands)
	}
	subject, stands := compare.Endpoint(compared.Subject, held)
	if !stands || subject.Ordinal != 2 {
		t.Errorf("the subject end is ordinal %d (held %v), want the version standing at 11:02 and never the one above it", subject.Ordinal, stands)
	}
}

func TestEndpoint_ASideThatIsNotThereHoldsNoVersion(t *testing.T) {
	if _, stands := compare.Endpoint(compare.Side{}, series(preview, store.RecordAsset, at(8, 0))); stands {
		t.Error("the baseline of a first Run stands at a version; want none, which is what makes every Asset created")
	}
}

func TestChangeRow_AnAssetTheBaselineHeldNoVersionOfIsCreated(t *testing.T) {
	held := series(preview, store.RecordAsset, at(10, 0))
	compared := window(nil, []store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-8801")})
	row := onlyRow(t, compared, []compare.Record{{
		Identity: preview,
		Subject:  compare.End{Held: true, Version: held.Versions[0], Fields: fields("region", "fsn1")},
	}})

	if row.Type != "asset" || row.Change != "created" {
		t.Errorf("the row is %s/%s, want an asset row reading created", row.Type, row.Change)
	}
	if row.FromOrdinal != nil || row.ToOrdinal == nil || *row.ToOrdinal != 1 {
		t.Errorf("the ordinals are %v → %v, want – → 1", row.FromOrdinal, row.ToOrdinal)
	}
	if got, want := strings.Join(row.Cells(), "|"), "created|staging|hetzner-staging|preview-8801|– → 1|region: fsn1"; got != want {
		t.Errorf("the line is %q, want %q", got, want)
	}
}

func TestChangeRow_ARecordChangedAndDestroyedInOneWindowIsOneDestroyedRow(t *testing.T) {
	// Two versions were written inside the window and the row is read from
	// the endpoints, so it is one row and the gap shows in ORDINAL
	// (ADR-0058).
	held := series(preview, store.RecordAsset, at(8, 0), at(10, 0), at(10, 30))
	held.Versions[2].Tombstone = true
	compared := window(nil, []store.StepFile{concluded("retire", "staging", "hetzner-staging", "preview-8801")})

	row := onlyRow(t, compared, []compare.Record{{
		Identity: preview,
		Baseline: compare.End{Held: true, Version: held.Versions[0], Fields: fields("region", "fsn1")},
		Subject:  compare.End{Held: true, Version: held.Versions[2], Fields: fields("region", "fsn1", "server_type", "cx22")},
	}})

	if row.Change != "destroyed" {
		t.Errorf("the row reads %s, want destroyed: one row spanning both versions", row.Change)
	}
	if row.FromOrdinal == nil || *row.FromOrdinal != 1 || row.ToOrdinal == nil || *row.ToOrdinal != 3 {
		t.Errorf("the ordinals are %v → %v, want 1 → 3, which is where the gap shows", row.FromOrdinal, row.ToOrdinal)
	}
	if got, want := row.Cells()[5], "† confirmed 10:30 · region: fsn1 · server_type: cx22"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q — the last known state off the Tombstone itself", got, want)
	}
}

func TestChangeRow_ASeriesATombstoneOpenedIsDestroyedWithAnEmptyFieldsCell(t *testing.T) {
	held := series(preview, store.RecordAsset, at(10, 0))
	held.Versions[0].Tombstone = true
	compared := window(nil, []store.StepFile{concluded("retire", "staging", "hetzner-staging", "preview-8801")})

	row := onlyRow(t, compared, []compare.Record{{
		Identity: preview,
		Subject:  compare.End{Held: true, Version: held.Versions[0]},
	}})
	if row.Change != "destroyed" {
		t.Errorf("the row reads %s, want destroyed: reading absent-then-present as a creation reports the opposite of what happened", row.Change)
	}
	if got, want := row.Cells()[4], "– → 1"; got != want {
		t.Errorf("the ORDINAL cell is %q, want %q — hyper ended a thing it never built", got, want)
	}
	if got, want := row.Cells()[5], "† confirmed 10:00"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q — the marker alone, and hyper never saw what it was", got, want)
	}
	if got, want := wire(t, row), `"fields":{}`; !strings.Contains(got, want) {
		t.Errorf("the row is %s, want %s: the empty mapping is what the empty column reads off", got, want)
	}
}

func TestChangeRow_AnAssetStandingUnchangedAtBothEndsDrawsNoRow(t *testing.T) {
	held := series(preview, store.RecordAsset, at(8, 0))
	compared := window(nil, []store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-8801")})
	rows := changeRows(t, compared, []compare.Record{{
		Identity: preview,
		Baseline: compare.End{Held: true, Version: held.Versions[0]},
		Subject:  compare.End{Held: true, Version: held.Versions[0]},
	}})
	if len(rows) != 0 {
		t.Errorf("the window drew %v, want no row: nothing moved", rows)
	}
}

func TestChangeRow_AnAssetAnotherProcedureMovedInBetweenIsChanged(t *testing.T) {
	// The endpoints are read without asking whose Run wrote them: this
	// surface says *this differs from when we last looked* (ADR-0058).
	held := series(preview, store.RecordAsset, at(8, 0), at(10, 0))
	compared := window([]store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-8801")}, nil)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: preview,
		Baseline: compare.End{Held: true, Version: held.Versions[0], Fields: fields("labels.retire-after", "2026-08-18")},
		Subject:  compare.End{Held: true, Version: held.Versions[1], Fields: fields("labels.retire-after", "2026-08-25")},
	}})
	if row.Change != "changed" {
		t.Errorf("the row reads %s, want changed", row.Change)
	}
	if got, want := row.Cells()[5], "labels.retire-after: 2026-08-18 → 2026-08-25"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q", got, want)
	}
}

var cert = store.Identity{Target: "local", Definition: "uptime", Name: "cert.hyper.dev"}

func TestChangeRow_AnIdentityTheSubjectSawAndTheBaselineDidNotAppeared(t *testing.T) {
	held := series(cert, store.RecordObservation, at(10, 0))
	compared := window(
		[]store.StepFile{concluded("probe", "local", "uptime", "status.hyper.dev")},
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
	)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Subject:  compare.End{Held: true, Version: held.Versions[0], Fields: fields("days_left", "34")},
	}})
	if row.Type != "observation" || row.Change != "appeared" {
		t.Errorf("the row is %s/%s, want an observation row reading appeared", row.Type, row.Change)
	}
	if got, want := row.Cells()[4], "– → 1"; got != want {
		t.Errorf("the ORDINAL cell is %q, want %q: what the side lacks is a view rather than a version", got, want)
	}
}

func TestChangeRow_AppearedBeatsChanged(t *testing.T) {
	// An identity absent from the baseline Run's set, present in the
	// subject's, whose series holds an older version that has since moved:
	// the Disposition-derived name beats the Record-derived one (§8).
	held := series(cert, store.RecordObservation, at(8, 0), at(10, 0))
	compared := window(
		[]store.StepFile{concluded("probe", "local", "uptime", "status.hyper.dev")},
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
	)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: held.Versions[0]},
		Subject:  compare.End{Held: true, Version: held.Versions[1]},
	}})
	if row.Change != "appeared" {
		t.Errorf("the row reads %s, want appeared", row.Change)
	}
	if row.FromOrdinal != nil {
		t.Errorf("the row carries from_ordinal %d; want none — printing the standing ordinal on the side the Procedure could not see invites a reader to difference them", *row.FromOrdinal)
	}
}

func TestChangeRow_AnIdentityTheBaselineSawAndTheSubjectDidNotVanished(t *testing.T) {
	held := series(cert, store.RecordObservation, at(8, 0))
	compared := window(
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
		[]store.StepFile{concluded("probe", "local", "uptime", "status.hyper.dev")},
	)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: held.Versions[0], Fields: fields("days_left", "41")},
		Subject:  compare.End{Held: true, Version: held.Versions[0], Fields: fields("days_left", "41")},
	}})
	if row.Change != "vanished" {
		t.Errorf("the row reads %s, want vanished: a disappearance mints no version at all", row.Change)
	}
	if got, want := row.Cells()[4], "1 → –"; got != want {
		t.Errorf("the ORDINAL cell is %q, want %q", got, want)
	}
	if got, want := row.Cells()[5], "days_left: 41"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q — the baseline end's fields, with no marker", got, want)
	}
	if got := wire(t, row); !strings.Contains(got, `"from_ordinal":1`) || strings.Contains(got, "to_ordinal") {
		t.Errorf("the row is %s, want from_ordinal alone: the members are absent exactly where the column renders –", got)
	}
}

func TestChangeRow_AnIdentityMissingFromAPartialSetDrawsNoRow(t *testing.T) {
	// Where a Step's Disposition carries the path a projection failed on,
	// an identity missing from its set is one hyper did not read rather
	// than one the world removed (§8).
	partial := concluded("probe", "local", "uptime", "status.hyper.dev")
	partial.ProjectionFailedPath = "$.certificate.days_left"

	held := series(cert, store.RecordObservation, at(8, 0))
	for _, side := range []struct {
		name              string
		baseline, subject []store.StepFile
		record            compare.Record
	}{
		{
			name:     "as subject, where it would otherwise render vanished",
			baseline: []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
			subject:  []store.StepFile{partial},
			record: compare.Record{
				Identity: cert,
				Baseline: compare.End{Held: true, Version: held.Versions[0]},
				Subject:  compare.End{Held: true, Version: held.Versions[0]},
			},
		},
		{
			name:     "as baseline, where the same identity returning would render appeared",
			baseline: []store.StepFile{partial},
			subject:  []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
			record: compare.Record{
				Identity: cert,
				Baseline: compare.End{Held: true, Version: held.Versions[0]},
				Subject:  compare.End{Held: true, Version: held.Versions[0]},
			},
		},
	} {
		t.Run(side.name, func(t *testing.T) {
			rows := changeRows(t, window(side.baseline, side.subject), []compare.Record{side.record})
			if len(rows) != 0 {
				t.Errorf("the window drew %v, want no row: a partial set is read for what it holds and never for what it omits", rows)
			}
		})
	}
}

func TestChangeRow_ADispositionCarryingNoSetRemovesNothingFromTheOtherSide(t *testing.T) {
	untouched := store.StepFile{
		StepCode:    store.StepCode{ID: "probe", Target: "local", Definition: "uptime"},
		Disposition: store.DispositionAttemptedWorldUntouched,
	}
	held := series(cert, store.RecordObservation, at(8, 0))
	rows := changeRows(t, window([]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")}, []store.StepFile{untouched}), []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: held.Versions[0]},
		Subject:  compare.End{Held: true, Version: held.Versions[0]},
	}})
	if len(rows) != 0 {
		t.Errorf("the window drew %v, want no row: nothing renders vanished on a Step whose Disposition carries no set", rows)
	}
}

func TestChangeRow_AStepNeverReachedInsideAHaltedRunRemovesNothingEither(t *testing.T) {
	// Inside an entry that did not run to its end an absent record is
	// §12's seventh Disposition, which carries no set like the other three
	// (§7).
	compared := window([]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")}, nil)
	compared.Subject.Entry.Owner = store.OutcomeFile{Outcome: store.OutcomeFailed, EndedAt: at(11, 2)}

	held := series(cert, store.RecordObservation, at(8, 0))
	rows := changeRows(t, compared, []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: held.Versions[0]},
		Subject:  compare.End{Held: true, Version: held.Versions[0]},
	}})
	if len(rows) != 0 {
		t.Errorf("the window drew %v, want no row: a Run that stopped short never reached that Step", rows)
	}
}

func TestChangeRow_ASilenceTakesTheDispositionDerivedNameAndNotTheRecordDerivedOne(t *testing.T) {
	// §8 suppresses the row *where it would otherwise render vanished*, and
	// those two names are all it suppresses: a Record whose endpoints differ
	// still differs from when this Procedure last looked, whoever moved it
	// (ADR-0058).
	partial := concluded("probe", "local", "uptime", "status.hyper.dev")
	partial.ProjectionFailedPath = "$.certificate.days_left"

	moved := series(cert, store.RecordObservation, at(8, 0), at(10, 0))
	row := onlyRow(t, window([]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")}, []store.StepFile{partial}), []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: moved.Versions[0], Fields: fields("days_left", "41")},
		Subject:  compare.End{Held: true, Version: moved.Versions[1], Fields: fields("days_left", "34")},
	}})
	if row.Change != "changed" {
		t.Errorf("the row reads %s, want changed: the silence took the name vanished and not the row", row.Change)
	}
	if got, want := row.Cells()[4], "1 → 2"; got != want {
		t.Errorf("the ORDINAL cell is %q, want %q — both ends stand at a version", got, want)
	}
}

func TestChangeRow_AnAssetReCreatedOverATombstoneIsChangedAndNeverCreated(t *testing.T) {
	// §8 gives the Asset table three names and fixes `created` as *an Asset
	// the baseline end held no version of*. A baseline standing at a
	// Tombstone held one, so a re-creation above it is `changed` — and
	// there is no fourth name for it, a resurrection being two versions of
	// one series like any other (§7).
	held := series(preview, store.RecordAsset, at(8, 0), at(10, 0))
	held.Versions[0].Tombstone = true

	row := onlyRow(t, window(nil, []store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-8801")}), []compare.Record{{
		Identity: preview,
		Baseline: compare.End{Held: true, Version: held.Versions[0], Fields: fields("region", "fsn1")},
		Subject:  compare.End{Held: true, Version: held.Versions[1], Fields: fields("region", "nbg1")},
	}})
	if row.Change != "changed" {
		t.Errorf("the row reads %s, want changed", row.Change)
	}
	if got, want := row.Cells()[5], "region: fsn1 → nbg1"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q — the Tombstone copied the previous Head's fields forward (§7)", got, want)
	}
}

func TestChangeRow_AStepACompletedRunNeverHeldIsAnAbsenceOfSightAndNotASilence(t *testing.T) {
	// A Run that completed reached every Step it declared, so an absent
	// record there is a Step that Run's revision did not have — which is
	// what appeared reports (§8).
	held := series(cert, store.RecordObservation, at(10, 0))
	compared := window([]store.StepFile{concluded("label", "staging", "hetzner-staging", "preview-1")}, []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")})
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Subject:  compare.End{Held: true, Version: held.Versions[0]},
	}})
	if row.Change != "appeared" {
		t.Errorf("the row reads %s, want appeared", row.Change)
	}
}

func TestChangeRow_ARunWhoseEveryStepSkippedDrawsNoRow(t *testing.T) {
	skipped := store.StepFile{
		StepCode:    store.StepCode{ID: "probe", Target: "local", Definition: "uptime"},
		Disposition: store.DispositionSkippedByCondition,
	}
	refused := store.StepFile{
		StepCode:    store.StepCode{ID: "retire", Target: "staging", Definition: "hetzner-staging"},
		Disposition: store.DispositionRefused,
	}
	compared := window(nil, []store.StepFile{skipped, refused})
	if rows := changeRows(t, compared, nil); len(rows) != 0 {
		t.Errorf("the window drew %v, want none: a Refusal reached nothing, so it is not a change", rows)
	}
	if eligible := compare.Eligible(compared); len(eligible) != 0 {
		t.Errorf("Eligible() = %v, want nothing", eligible)
	}
}

func TestChangeRows_SortByTargetThenDefinitionThenName(t *testing.T) {
	var records []compare.Record
	var steps []store.StepFile
	for _, id := range []store.Identity{
		{Target: "staging", Definition: "hetzner-staging", Name: "preview-2"},
		{Target: "local", Definition: "uptime", Name: "b"},
		{Target: "local", Definition: "heartbeat", Name: "z"},
		{Target: "staging", Definition: "hetzner-staging", Name: "preview-1"},
	} {
		held := series(id, store.RecordObservation, at(10, 0))
		records = append(records, compare.Record{Identity: id, Subject: compare.End{Held: true, Version: held.Versions[0]}})
		steps = append(steps, concluded("probe", id.Target, id.Definition, id.Name))
	}

	var order []string
	for _, row := range changeRows(t, window(nil, steps), records) {
		order = append(order, row.Target+"/"+row.Definition+"/"+row.Name)
	}
	want := []string{"local/heartbeat/z", "local/uptime/b", "staging/hetzner-staging/preview-1", "staging/hetzner-staging/preview-2"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("the rows are %v, want %v — the columns read left to right", order, want)
	}
}

func TestChangeRows_AssetsRenderBeforeObservations(t *testing.T) {
	// Two tables and one derivation: the Asset rows are the first table's
	// and the Observation rows the second's, whatever the identity order
	// between them is (§8).
	asset := store.Identity{Target: "zzz", Definition: "hetzner", Name: "preview-1"}
	observation := store.Identity{Target: "aaa", Definition: "uptime", Name: "cert"}
	records := []compare.Record{
		{Identity: observation, Subject: compare.End{Held: true, Version: series(observation, store.RecordObservation, at(10, 0)).Versions[0]}},
		{Identity: asset, Subject: compare.End{Held: true, Version: series(asset, store.RecordAsset, at(10, 0)).Versions[0]}},
	}
	compared := window(nil, []store.StepFile{
		concluded("label", "zzz", "hetzner", "preview-1"),
		concluded("probe", "aaa", "uptime", "cert"),
	})

	rows := changeRows(t, compared, records)
	if len(rows) != 2 || rows[0].Type != "asset" || rows[1].Type != "observation" {
		t.Errorf("the rows are %v, want the asset row first", rows)
	}
}

func TestFields_AValueRendersWholeOrRendersChanged(t *testing.T) {
	compared := window(
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
	)
	moved := series(cert, store.RecordObservation, at(8, 0), at(10, 0))

	for _, held := range []struct {
		name       string
		from, to   store.Value
		cell, wire string
	}{
		{name: "a scalar over the budget", from: store.String(strings.Repeat("a", 121)), to: store.String("short"), cell: "stdout: changed"},
		{name: "a scalar carrying a newline", from: store.String("one\ntwo"), to: store.String("three"), cell: "stdout: changed"},
		{name: "anything nested", from: store.Array{store.String("a")}, to: store.Array{store.String("b")}, cell: "stdout: changed", wire: `"stdout":[["a"],["b"]]`},
	} {
		t.Run(held.name, func(t *testing.T) {
			row := onlyRow(t, compared, []compare.Record{{
				Identity: cert,
				Baseline: compare.End{Held: true, Version: moved.Versions[0], Fields: store.Mapping{"stdout": held.from}},
				Subject:  compare.End{Held: true, Version: moved.Versions[1], Fields: store.Mapping{"stdout": held.to}},
			}})
			if got := row.Cells()[5]; got != held.cell {
				t.Errorf("the FIELDS cell is %q, want %q — there is no truncated form", got, held.cell)
			}
			if held.wire != "" && !strings.Contains(wire(t, row), held.wire) {
				t.Errorf("the row is %s, want %s: --json carries every value whole regardless", wire(t, row), held.wire)
			}
		})
	}
}

func TestFields_AOneSidedRowsUnrenderableFieldIsItsBarePath(t *testing.T) {
	held := series(cert, store.RecordObservation, at(10, 0))
	compared := window(nil, []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")})
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Subject: compare.End{Held: true, Version: held.Versions[0], Fields: store.Mapping{
			"days_left": store.String("34"),
			"stdout":    store.String(strings.Repeat("a", 121)),
		}},
	}})
	if got, want := row.Cells()[5], "days_left: 34 · stdout"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q — `changed` would be false on a one-sided row", got, want)
	}
}

func TestFields_TheBudgetIsAStatedConstantAndAValueAtItRendersWhole(t *testing.T) {
	held := series(cert, store.RecordObservation, at(10, 0))
	compared := window(nil, []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")})
	exactly := strings.Repeat("a", 120)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Subject:  compare.End{Held: true, Version: held.Versions[0], Fields: store.Mapping{"stdout": store.String(exactly)}},
	}})
	if got, want := row.Cells()[5], "stdout: "+exactly; got != want {
		t.Errorf("the FIELDS cell is %q, want the value whole at the budget itself", got)
	}
}

func TestFields_SortByCodePointWithNoCap(t *testing.T) {
	held := series(cert, store.RecordObservation, at(10, 0))
	compared := window(nil, []store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")})
	projected := store.Mapping{}
	for _, name := range []string{"z", "a", "M", "b"} {
		projected[name] = store.String(name)
	}
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Subject:  compare.End{Held: true, Version: held.Versions[0], Fields: projected},
	}})
	if got, want := row.Cells()[5], "M: M · a: a · b: b · z: z"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q", got, want)
	}
}

func TestFields_AChangedRowRendersTheFieldsThatMovedAndNoOthers(t *testing.T) {
	compared := window(
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
	)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: series(cert, store.RecordObservation, at(8, 0)).Versions[0], Fields: fields("host", "cert.hyper.dev", "days_left", "41")},
		Subject:  compare.End{Held: true, Version: series(cert, store.RecordObservation, at(8, 0), at(10, 0)).Versions[1], Fields: fields("host", "cert.hyper.dev", "days_left", "34")},
	}})
	if got, want := row.Cells()[5], "days_left: 41 → 34"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q", got, want)
	}
	if got, want := wire(t, row), `"fields":{"days_left":["41","34"]}`; !strings.Contains(got, want) {
		t.Errorf("the row is %s, want %s", got, want)
	}
}

func TestFields_AFieldOneEndDidNotCarryHasMovedAndItsSideIsNothing(t *testing.T) {
	compared := window(
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
		[]store.StepFile{concluded("probe", "local", "uptime", "cert.hyper.dev")},
	)
	row := onlyRow(t, compared, []compare.Record{{
		Identity: cert,
		Baseline: compare.End{Held: true, Version: series(cert, store.RecordObservation, at(8, 0)).Versions[0]},
		Subject:  compare.End{Held: true, Version: series(cert, store.RecordObservation, at(8, 0), at(10, 0)).Versions[1], Fields: fields("days_left", "34")},
	}})
	if got, want := row.Cells()[5], "days_left: – → 34"; got != want {
		t.Errorf("the FIELDS cell is %q, want %q", got, want)
	}
	if got, want := wire(t, row), `"fields":{"days_left":[null,"34"]}`; !strings.Contains(got, want) {
		t.Errorf("the row is %s, want %s — a two-element array has no key to omit", got, want)
	}
}

func TestChangeRow_TwoRenderingsOfOneWindowAreByteIdentical(t *testing.T) {
	var records []compare.Record
	var steps []store.StepFile
	for _, name := range []string{"preview-3", "preview-1", "preview-2"} {
		id := store.Identity{Target: "staging", Definition: "hetzner-staging", Name: name}
		held := series(id, store.RecordAsset, at(10, 0))
		records = append(records, compare.Record{
			Identity: id,
			Subject:  compare.End{Held: true, Version: held.Versions[0], Fields: fields("region", "fsn1", "server_type", "cx22")},
		})
		steps = append(steps, concluded("label", "staging", "hetzner-staging", name))
	}
	compared := window(nil, steps)

	first, second := streamed(t, compared, records), streamed(t, compared, records)
	if first != second {
		t.Errorf("two renderings of one window differ:\n%s\n%s", first, second)
	}
}

// streamed is one window's whole stream as bytes, which is what two renderings
// being byte-identical is asserted over.
func streamed(t *testing.T, compared compare.Window, records []compare.Record) string {
	t.Helper()

	var out strings.Builder
	for _, row := range compare.Rows(compared, records, compare.Code{}) {
		out.WriteString(wire(t, row))
	}
	return out.String()
}

// A ChangeRow is a row of the stream like any other, which is what puts it on
// the page and on the wire through one path (ADR-0026).
var _ render.Row = compare.ChangeRow{}
