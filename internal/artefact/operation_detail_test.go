package artefact

import (
	"reflect"
	"slices"
	"testing"
)

// derivedFactsManifest is a Manifest carrying one Operation per case the
// derived block has to answer for: a paginated `read` over a series with a
// declared concurrency limit, a `mutate` under each of the two Repeatability
// values it may declare and one under neither, a `read` declaring none, and a
// `destroy` whose request is http: — the non-opaque one, the built-in shell
// Provider carrying the other.
const derivedFactsManifest = `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    repeatability: repeatable
    deadline: 30s
    concurrency: 4
    patterns:
      pagination:
        cursor: {from: $.body.cursor, into: {query: cursor}}
      retry: {attempts: 3}
    http: {method: GET, host: "{from-target}", path: /widgets}
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id}
  poll_widget:
    kind: read
    repeatability: repeatable
    deadline: 10s
    patterns:
      retry: {attempts: 3}
      polling:
        interval: 5s
        until:
          - field: status
            equals: ready
      pagination:
        page: {from: 1, into: {query: page}}
    http: {method: GET, host: "{from-target}", path: "/widgets/{id}/status"}
    record:
      over: $.body.result
      identity: $.id
      fields: {status: $.status}
  get_widget:
    kind: read
    deadline: 2m
    http: {method: GET, host: "{from-target}", path: "/widgets/{id}"}
    record:
      identity: $.body.id
      fields: {id: $.body.id}
  create_widget:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 1h
    http: {method: POST, host: "{from-target}", path: /widgets}
    record:
      identity: "{name}"
      fields: {id: $.body.id}
  rotate_widget:
    kind: mutate
    deadline: 1d
    http: {method: POST, host: "{from-target}", path: "/widgets/{id}/rotate"}
    record:
      identity: "{id}"
      fields: {id: $.body.id}
  delete_widget:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http: {method: DELETE, host: "{from-target}", path: "/widgets/{id}"}
`

// detail is the derived block for one Operation of the Manifest above.
func detail(t *testing.T, name string) OperationDetail {
	t.Helper()
	return ReadOperationDetail(parse(t, derivedFactsManifest), name)
}

// TestReadOperationDetail_TheCapabilityIsTheRequestBlocksOwnKey is the
// derivation capability-mismatch already runs, read rather than inferred: an
// Operation uses exactly one Capability, and the block's key is it (§12).
func TestReadOperationDetail_TheCapabilityIsTheRequestBlocksOwnKey(t *testing.T) {
	if got, want := detail(t, "list_widgets").Capabilities, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities = %v, want %v", got, want)
	}

	shell := ReadOperationDetail(BuiltinShellProviderRoot(), "destroy")
	if got, want := shell.Capabilities, []string{"shell"}; !slices.Equal(got, want) {
		t.Errorf("the built-in shell destroy's capabilities = %v, want %v", got, want)
	}
}

// TestReadOperationDetail_BoundHasThreeMembersAndNotTwo is §5's fact about the
// most severe Operation the tool runs, stated as the three states it has: a
// `destroy` Step's Bound is mandatory, an opaque `destroy` Step is the one Step
// that carries none and writing one there is refused, and a `read` and a
// `mutate` have nothing for a Bound to guard.
func TestReadOperationDetail_BoundHasThreeMembersAndNotTwo(t *testing.T) {
	if got := detail(t, "delete_widget").Bound; got != boundMandatory {
		t.Errorf("a non-opaque destroy's bound = %q, want %q", got, boundMandatory)
	}
	if got := ReadOperationDetail(BuiltinShellProviderRoot(), "destroy").Bound; got != boundIllegal {
		t.Errorf("an opaque destroy's bound = %q, want %q", got, boundIllegal)
	}
	for _, name := range []string{"list_widgets", "create_widget"} {
		if got := detail(t, name).Bound; got != boundNone {
			t.Errorf("%s's bound = %q, want %q", name, got, boundNone)
		}
	}
}

// TestReadOperationDetail_PatternsResolvedAreTheMembersTheOperationDeclares is
// §12's closed three-member set read off the Operation's own patterns: block,
// in §12's own order, and empty rather than absent where it declares none — a
// list nothing is in is a fact a reader reads (§9).
func TestReadOperationDetail_PatternsResolvedAreTheMembersTheOperationDeclares(t *testing.T) {
	if got, want := detail(t, "list_widgets").PatternsResolved, []string{"pagination", "retry"}; !slices.Equal(got, want) {
		t.Errorf("patterns_resolved = %v, want %v", got, want)
	}
	if got, want := detail(t, "poll_widget").PatternsResolved, []string{"pagination", "polling", "retry"}; !slices.Equal(got, want) {
		t.Errorf("patterns_resolved = %v, want %v — §12's order, and not the order the Operation happened to write them in", got, want)
	}
	got := detail(t, "create_widget").PatternsResolved
	if len(got) != 0 {
		t.Errorf("patterns_resolved = %v, want it empty where the Operation declares no Pattern", got)
	}
	if got == nil {
		t.Error("patterns_resolved is nil; an Operation declaring no Pattern resolves to an empty list, not to no list")
	}
}

// TestReadOperationDetail_TheRecordPairIsCardinalityAndTheIdentityVerbatim:
// series where record: carries an over:, one where it does not, and the
// identity: scalar exactly as it was written — a template hole and a response
// path alike, this reader stating what the artefact stated (§3, §9).
func TestReadOperationDetail_TheRecordPairIsCardinalityAndTheIdentityVerbatim(t *testing.T) {
	series := detail(t, "list_widgets")
	if got, want := series.RecordCardinality, recordSeries; got != want {
		t.Errorf("record_cardinality = %q, want %q", got, want)
	}
	if got, want := series.RecordIdentity, "$.id"; got != want {
		t.Errorf("record_identity = %q, want %q", got, want)
	}

	one := detail(t, "create_widget")
	if got, want := one.RecordCardinality, recordOne; got != want {
		t.Errorf("record_cardinality = %q, want %q", got, want)
	}
	if got, want := one.RecordIdentity, "{name}"; got != want {
		t.Errorf("record_identity = %q, want the template hole %q verbatim", got, want)
	}
}

// TestReadOperationDetail_ADestroyCarriesNeitherRecordMember: a destroy
// declares no record: at all, what it writes being a Tombstone under the series
// its Expansion acted on, so both members are absent together rather than
// written empty (§3, ADR-0037).
func TestReadOperationDetail_ADestroyCarriesNeitherRecordMember(t *testing.T) {
	for _, got := range []OperationDetail{detail(t, "delete_widget"), ReadOperationDetail(BuiltinShellProviderRoot(), "destroy")} {
		if got.RecordCardinality != "" || got.RecordIdentity != "" {
			t.Errorf("a destroy's record pair = (%q, %q), want both absent", got.RecordCardinality, got.RecordIdentity)
		}
	}
}

// TestReadOperationDetail_RepeatabilityIsTheEffectiveValue is the fact with no
// spelling in the source at all: run-once is what an effectful Operation
// declaring no repeatability: is, and §12 gives it no keyword to author, so a
// caller who scanned the Manifest for it would find nothing (§12, ADR-0037).
func TestReadOperationDetail_RepeatabilityIsTheEffectiveValue(t *testing.T) {
	for name, want := range map[string]string{
		"list_widgets":  "repeatable",
		"create_widget": "skip-if-recorded",
		"rotate_widget": "run-once",
		"get_widget":    "repeatable",
		"delete_widget": "repeatable",
	} {
		if got := detail(t, name).Repeatability; got != want {
			t.Errorf("%s's repeatability = %q, want %q", name, got, want)
		}
	}
	if got, want := ReadOperationDetail(BuiltinShellProviderRoot(), "destroy_once").Repeatability, "run-once"; got != want {
		t.Errorf("the built-in destroy_once's repeatability = %q, want %q", got, want)
	}
}

// TestReadOperationDetail_TheDeadlineIsTheAuthoredSpellingAndItsSeconds: §9
// fixed the wire name and its unit with it, and the authored spelling is kept
// beside it because that is what the source the block stands next to says.
func TestReadOperationDetail_TheDeadlineIsTheAuthoredSpellingAndItsSeconds(t *testing.T) {
	for name, want := range map[string]int{
		"list_widgets":  30,
		"get_widget":    120,
		"create_widget": 3600,
		"rotate_widget": 86400,
	} {
		got := detail(t, name).DeadlineSeconds
		if got == nil {
			t.Errorf("%s's deadline_seconds is absent, want %d", name, want)
			continue
		}
		if *got != want {
			t.Errorf("%s's deadline_seconds = %d, want %d", name, *got, want)
		}
	}
	if got, want := detail(t, "get_widget").Deadline, "2m"; got != want {
		t.Errorf("deadline = %q, want the authored spelling %q", got, want)
	}
}

// TestReadOperationDetail_TheConcurrencyLimitIsEffectiveAndAlwaysPresent is
// ADR-0045 on the wire: the declared limit where a read declares one, and 1
// everywhere else — a read that omits the key, and every mutate and destroy,
// whose Expansion is serial and which may not declare it at all. A caller
// asking *how many at once* gets a number for every Operation.
func TestReadOperationDetail_TheConcurrencyLimitIsEffectiveAndAlwaysPresent(t *testing.T) {
	for name, want := range map[string]int{
		"list_widgets":  4,
		"get_widget":    1,
		"create_widget": 1,
		"rotate_widget": 1,
		"delete_widget": 1,
	} {
		if got := detail(t, name).ConcurrencyLimit; got != want {
			t.Errorf("%s's concurrency_limit = %d, want %d", name, got, want)
		}
	}
	for _, name := range []string{"read", "mutate", "destroy", "destroy_once"} {
		if got := ReadOperationDetail(BuiltinShellProviderRoot(), name).ConcurrencyLimit; got != 1 {
			t.Errorf("the built-in %s's concurrency_limit = %d, want 1", name, got)
		}
	}
}

// TestReadOperationDetail_AnExplicitOneAndAnOmittedKeyReadAlike: 1 is an
// ordinary member of an integer's value set, so an author who established that
// an API refuses concurrency may write it — and what they wrote means what the
// omission means (ADR-0045).
func TestReadOperationDetail_AnExplicitOneAndAnOmittedKeyReadAlike(t *testing.T) {
	explicit := ReadOperationDetail(parse(t, `kind: provider
provider: widget
operations:
  poll_widget:
    kind: read
    repeatability: repeatable
    deadline: 10s
    patterns:
      retry: {attempts: 3}
      polling:
        interval: 5s
        until:
          - field: status
            equals: ready
      pagination:
        page: {from: 1, into: {query: page}}
    http: {method: GET, host: "{from-target}", path: "/widgets/{id}/status"}
    record:
      over: $.body.result
      identity: $.id
      fields: {status: $.status}
  get_widget:
    kind: read
    deadline: 2m
    concurrency: 1
    http: {method: GET, host: "{from-target}", path: "/widgets/{id}"}
    record:
      identity: $.body.id
      fields: {id: $.body.id}
`), "get_widget")

	if !reflect.DeepEqual(explicit, detail(t, "get_widget")) {
		t.Errorf("an explicit concurrency: 1 read as %+v and an omitted key as %+v", explicit, detail(t, "get_widget"))
	}
}

// TestReadOperationDetail_AManifestHyperCannotReadStatesWhatItCan is the drop
// rule at this reader: what it cannot read has no value to report, and what is
// wrong with the Manifest is check's to name (ADR-0064). The one member with no
// absence at all is the concurrency limit, which is 1 for every Operation
// including one whose Kind nothing legible declared.
func TestReadOperationDetail_AManifestHyperCannotReadStatesWhatItCan(t *testing.T) {
	got := ReadOperationDetail(parse(t, `kind: provider
provider: widget
operations:
  unreadable:
    kind: sideways
    deadline: soon
    http: {method: GET, host: "{from-target}", path: /widgets}
    shell: {}
`), "unreadable")

	if len(got.Capabilities) != 0 {
		t.Errorf("capabilities = %v; an Operation naming both request blocks names none unambiguously", got.Capabilities)
	}
	if got.Bound != "" {
		t.Errorf("bound = %q; there is no declared Kind to read the fact off", got.Bound)
	}
	if got.Repeatability != "" {
		t.Errorf("repeatability = %q; there is no Kind whose default to fall back to", got.Repeatability)
	}
	if got.DeadlineSeconds != nil {
		t.Errorf("deadline_seconds = %d; `soon` is not a duration to convert", *got.DeadlineSeconds)
	}
	if got.ConcurrencyLimit != 1 {
		t.Errorf("concurrency_limit = %d, want 1; the limit is present on every Operation without exception", got.ConcurrencyLimit)
	}
}

// TestReadOperationDetail_ANameThatIsNoKeyReadsNothing is the nothing-to-read
// case of the same rule. Resolution is the surface's and has already happened
// where the Operation's own source came from, so this answers the empty detail
// rather than a second account of a name matching nothing (§9, ADR-0060).
func TestReadOperationDetail_ANameThatIsNoKeyReadsNothing(t *testing.T) {
	got := detail(t, "list_widget")
	if len(got.Capabilities) != 0 || got.Bound != "" || got.Repeatability != "" || got.Deadline != "" {
		t.Errorf("ReadOperationDetail read %+v off a name that is no key of operations:", got)
	}
}

// TestReadOperationDetail_TheBuiltInsSixOperationsExerciseTheDerivedSets is
// §12's own fixture read as one: hyper ships six Operations, Kind crossed with
// the Repeatability values each Kind may declare, and between them they carry
// every Bound member an opaque Provider can reach and every Repeatability value
// there is — including the one no artefact may write (§12, ADR-0039).
func TestReadOperationDetail_TheBuiltInsSixOperationsExerciseTheDerivedSets(t *testing.T) {
	root := BuiltinShellProviderRoot()
	want := map[string]struct{ bound, repeatability string }{
		"read":                    {boundNone, "repeatable"},
		"mutate":                  {boundNone, "repeatable"},
		"mutate_once":             {boundNone, "run-once"},
		"mutate_skip_if_recorded": {boundNone, "skip-if-recorded"},
		"destroy":                 {boundIllegal, "repeatable"},
		"destroy_once":            {boundIllegal, "run-once"},
	}

	for name, want := range want {
		got := ReadOperationDetail(root, name)
		if got.Bound != want.bound || got.Repeatability != want.repeatability {
			t.Errorf("%s derived (%q, %q), want (%q, %q)", name, got.Bound, got.Repeatability, want.bound, want.repeatability)
		}
		if len(got.Capabilities) != 1 || got.Capabilities[0] != "shell" {
			t.Errorf("%s's capabilities = %v, want exactly [shell]", name, got.Capabilities)
		}
		if got.ConcurrencyLimit != 1 {
			t.Errorf("%s's concurrency_limit = %d, want 1 — hyper is the Provider author here and knows nothing about the command", name, got.ConcurrencyLimit)
		}
	}
	if len(want) != len(topLevelFields(root, "operations")["operations"].Content)/2 {
		t.Error("the built-in declares an Operation this case does not name; the six are the whole of it")
	}
}
