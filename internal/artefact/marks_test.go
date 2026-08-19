package artefact

import (
	"slices"
	"testing"
)

// TestReadDefinitionMarks_MarksTheThreeClaimsBesideTheirOwnLines is §8's roster
// on a Definition: what it claims, what it names for `destroy` and which
// Targets it may bind, each anchored to the line that makes the claim (issue
// #122).
func TestReadDefinitionMarks_MarksTheThreeClaimsBesideTheirOwnLines(t *testing.T) {
	marks := ReadDefinitionMarks(parse(t, `kind: definition
definition: things
provider: things
kinds: [mutate, destroy]
destroy: [end_thing]
targets: [staging, production]
`))

	for _, c := range []struct {
		what   string
		mark   KeyMark
		line   int
		values []string
	}{
		{"kinds", marks.Kinds, 4, []string{"mutate", "destroy"}},
		{"destroy", marks.Destroy, 5, []string{"end_thing"}},
		{"targets", marks.Targets, 6, []string{"staging", "production"}},
	} {
		if c.mark.Line != c.line {
			t.Errorf("%s: is marked beside line %d, want line %d", c.what, c.mark.Line, c.line)
		}
		if !slices.Equal(c.mark.Values, c.values) {
			t.Errorf("%s: marks %v, want %v — the artefact's own order", c.what, c.mark.Values, c.values)
		}
	}
}

// TestReadDefinitionMarks_AKeyTheDefinitionNeverWroteHasNoLine is the absence
// this whole roster is read under: a Definition naming no `destroy` Operation
// has no `destroy:` line, so there is no line to mark and therefore no cell —
// which is different from a line rendering a blank one (§7, §8).
func TestReadDefinitionMarks_AKeyTheDefinitionNeverWroteHasNoLine(t *testing.T) {
	marks := ReadDefinitionMarks(parse(t, `kind: definition
definition: things-observed
provider: things
kinds: [read]
targets: [staging]
`))

	if marks.Destroy.Line != 0 || len(marks.Destroy.Values) != 0 {
		t.Errorf("destroy: is marked %v beside line %d, want no line and nothing derived", marks.Destroy.Values, marks.Destroy.Line)
	}
}

// TestReadTargetDeclarationMarks_MarksEveryGrantAndEveryCredentialSlot is §8's
// roster on a Target declaration: the Kinds it accepts, the Capabilities and
// the hosts it grants, and the environment variable each credential slot
// resolves from — the variable's name and never its value (§7, ADR-0007).
func TestReadTargetDeclarationMarks_MarksEveryGrantAndEveryCredentialSlot(t *testing.T) {
	marks := ReadTargetDeclarationMarks(parse(t, `kind: target-declaration
target: staging
class: things
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [api.things.dev, api.things.eu]
auth:
  username: {env: THINGS_USER}
  password: {env: THINGS_PASSWORD}
`))

	if got, want := marks.Kinds.Values, []string{"read", "mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("kinds: marks %v, want %v", got, want)
	}
	if got, want := marks.Capabilities.Values, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities: marks %v, want %v", got, want)
	}
	if got, want := marks.Hosts.Values, []string{"api.things.dev", "api.things.eu"}; !slices.Equal(got, want) {
		t.Errorf("hosts: marks %v, want %v", got, want)
	}

	want := []KeyMark{{Line: 8, Values: []string{"THINGS_USER"}}, {Line: 9, Values: []string{"THINGS_PASSWORD"}}}
	if got := marks.Credentials; !slices.EqualFunc(got, want, sameKeyMark) {
		t.Errorf("the credential slots mark %v, want %v — one per slot, in the mapping's own order", got, want)
	}
}

// TestReadTargetDeclarationMarks_ASlotThatNamesNoVariableIsMarkedWithNothing is
// ADR-0064 at the one line where the roster and the check disagree about what
// is wrong: the slot is a line of the file and still renders, and a slot whose
// value is not the mapping §4 fixes has nothing derived to mark —
// `credential-slot-malformed` is `check`'s to report.
func TestReadTargetDeclarationMarks_ASlotThatNamesNoVariableIsMarkedWithNothing(t *testing.T) {
	marks := ReadTargetDeclarationMarks(parse(t, `kind: target-declaration
target: staging
class: things
kinds: [read]
capabilities: [shell]
auth:
  token: THINGS_API_TOKEN
`))

	want := []KeyMark{{Line: 7}}
	if got := marks.Credentials; !slices.EqualFunc(got, want, sameKeyMark) {
		t.Errorf("the credential slots mark %v, want %v — the slot's line, and nothing derived", got, want)
	}
}

// TestReadTargetDeclarationMarks_TheOptInIsMarkedOnlyWhereItIsGranted is what
// the opt-in's mark says: a declaration writing `opaque-destroy: false` admits
// exactly what one writing nothing admits, and a mark there would name a grant
// that was not made (§4, §8).
func TestReadTargetDeclarationMarks_TheOptInIsMarkedOnlyWhereItIsGranted(t *testing.T) {
	declaration := func(optIn string) string {
		return `kind: target-declaration
target: local
class: local
kinds: [read, destroy]
capabilities: [shell]
` + optIn
	}

	if got := ReadTargetDeclarationMarks(parse(t, declaration("opaque-destroy: true\n"))).OpaqueDestroy; got != 6 {
		t.Errorf("the granted opt-in is marked beside line %d, want line 6", got)
	}
	for _, written := range []string{"opaque-destroy: false\n", ""} {
		if got := ReadTargetDeclarationMarks(parse(t, declaration(written))).OpaqueDestroy; got != 0 {
			t.Errorf("a declaration writing %q marks line %d, want no mark at all", written, got)
		}
	}
}

// TestReadManifestMarks_MarksTheSchemeTheCapabilitiesAndEveryOperation is §8's
// roster on a Manifest, and the Operations are the half of it a reviewer cannot
// read off the lines beside them: an Operation's Kind is declared and its
// opacity is declared nowhere at all (§12).
func TestReadManifestMarks_MarksTheSchemeTheCapabilitiesAndEveryOperation(t *testing.T) {
	marks := ReadManifestMarks(parse(t, manifestStatingRepeatabilityByOmission))

	if got, want := marks.Auth.Line, 6; got != want {
		t.Errorf("auth: is marked beside line %d, want line %d", got, want)
	}
	if got, want := marks.Auth.Values, []string{"header"}; !slices.Equal(got, want) {
		t.Errorf("auth: marks %v, want %v — which of §12's two schemes this Manifest names", got, want)
	}
	if got, want := marks.Capabilities.Values, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities: marks %v, want %v", got, want)
	}

	want := []OperationMark{
		{Line: 9, Kind: "read", Repeatability: "repeatable"},
		{Line: 21, Kind: "destroy", Repeatability: "run-once"},
	}
	if got := marks.Operations; !slices.Equal(got, want) {
		t.Errorf("the Operations mark %v, want %v", got, want)
	}
}

// manifestStatingRepeatabilityByOmission is a Manifest whose two Operations
// state their Repeatability by omission: a `read` that declares none is
// repeatable and a `destroy` that declares none is run-once, which is the word
// no artefact may write (§12, ADR-0037).
const manifestStatingRepeatabilityByOmission = `kind: provider
provider: things
schema-version: 1
class: things
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  list_things:
    kind: read
    http:
      method: GET
      host: "{from-target}"
      path: /things
    input:
      type: object
      properties: {}
    record:
      identity: $.id
      fields: {id: $.id}
  end_thing:
    kind: destroy
    http:
      method: DELETE
      host: "{from-target}"
      path: /things/{thing_id}
    input:
      type: object
      properties:
        thing_id: {type: string}
`

// TestReadManifestMarks_AnOpaqueRequestIsMarkedAndAnAuthlessManifestIsNot holds
// the two absences this roster keeps apart. Opacity is read off the request
// block, no artefact declaring it; and a Manifest with no `auth:` has no line
// to mark, which is not the `none` a row renders for a Provider that sends no
// credential (§9, §12, §13).
func TestReadManifestMarks_AnOpaqueRequestIsMarkedAndAnAuthlessManifestIsNot(t *testing.T) {
	marks := ReadManifestMarks(parse(t, `kind: provider
provider: commands
schema-version: 1
class: local
capabilities: [shell]
operations:
  run:
    kind: mutate
    repeatability: repeatable
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
`))

	if marks.Auth.Line != 0 || len(marks.Auth.Values) != 0 {
		t.Errorf("auth: marks %v beside line %d, want no line and nothing derived", marks.Auth.Values, marks.Auth.Line)
	}
	want := []OperationMark{{Line: 7, Kind: "mutate", Repeatability: "repeatable", Opaque: true}}
	if got := marks.Operations; !slices.Equal(got, want) {
		t.Errorf("the Operation marks %v, want %v", got, want)
	}
}

// TestReadRepositoryDeclarationMarks_MarksThePinAndTheRetention is §8's roster
// on the artefact that governs every Run: the version pin, and the retention
// policy that bounds Compaction. A repository declaring no `retention:` has no
// line to mark (§3, §8, §11).
func TestReadRepositoryDeclarationMarks_MarksThePinAndTheRetention(t *testing.T) {
	marks := ReadRepositoryDeclarationMarks(parse(t, `kind: repository-declaration
version: 1.4.0
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
retention: 90d
`))

	if got, want := marks.Version, (KeyMark{2, []string{"1.4.0"}}); !sameKeyMark(got, want) {
		t.Errorf("version: marks %v, want %v", got, want)
	}
	if got, want := marks.Retention, (KeyMark{4, []string{"90d"}}); !sameKeyMark(got, want) {
		t.Errorf("retention: marks %v, want %v", got, want)
	}

	unbounded := ReadRepositoryDeclarationMarks(parse(t, `kind: repository-declaration
version: 1.4.0
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
`))
	if unbounded.Retention.Line != 0 {
		t.Errorf("a repository with no retention: marks line %d, want no mark at all", unbounded.Retention.Line)
	}
}

// sameKeyMark compares two marks by the line they anchor to and the values they
// carry, which is the whole of a KeyMark.
func sameKeyMark(a, b KeyMark) bool {
	return a.Line == b.Line && slices.Equal(a.Values, b.Values)
}
