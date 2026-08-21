package projection_test

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/projection"
)

// response is one response object of every shape a path can reach into: the
// object's own members, a headers mapping whose names carry hyphens, a parsed
// body carrying a null and a nested object, and the tls object beside them.
func response(t *testing.T) capability.Object {
	t.Helper()

	var body any
	if err := json.Unmarshal([]byte(`{"result":{"id":"abc"},"error":null,"count":3}`), &body); err != nil {
		t.Fatal(err)
	}
	return capability.Object{
		{Name: capability.MemberHost, Value: "status.hyper.dev"},
		{Name: capability.MemberStatus, Value: 503},
		{Name: capability.MemberHeaders, Value: map[string]string{"content-type": "text/html"}},
		{Name: capability.MemberBody, Value: body},
		{Name: capability.MemberTLS, Value: capability.Object{
			{Name: capability.MemberDaysLeft, Value: 34},
		}},
	}
}

// TestResolve is §12's grammar over its three productions, and the one
// distinction the whole package exists for: a path that resolved to nothing is
// a different answer from a path that resolved to the null a body carried.
func TestResolve(t *testing.T) {
	for _, c := range []struct {
		name     string
		path     string
		want     any
		resolved bool
	}{
		{"the root is the object itself", "$", nil, true},
		{"a member of the object", "$.host", "status.hyper.dev", true},
		{"a member holding an integer", "$.status", 503, true},
		{"a member of a nested object", "$.tls.days_left", 34, true},
		{"a member of the parsed body", "$.body.count", float64(3), true},
		{"a member two deep in the body", "$.body.result.id", "abc", true},
		{"a bracketed member is the same member", `$["body"]["result"]["id"]`, "abc", true},
		{"a bracketed member reaches a name a dot cannot", `$.headers["content-type"]`, "text/html", true},
		{"resolved to null is a value", "$.body.error", nil, true},
		{"resolved to nothing is not", "$.body.absent", nil, false},
		{"a member the object does not carry", "$.tls.subject", nil, false},
		{"a path reaches no further inside a scalar", "$.host.length", nil, false},
		{"a path with no root resolves to nothing", ".host", nil, false},
		{"a path outside the grammar resolves to nothing", "$..host", nil, false},
		{"an index is outside the grammar", "$.body.result[0]", nil, false},
		{"an iteration is outside the grammar", "$.body[*]", nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			object := response(t)
			value, resolved := projection.Resolve(c.path, object)
			if resolved != c.resolved {
				t.Fatalf("Resolve(%q) resolved = %v, want %v", c.path, resolved, c.resolved)
			}
			if !resolved {
				return
			}
			if c.path == "$" {
				// The root is the object itself, which is the one
				// answer this table cannot spell as a literal.
				if _, isObject := value.(capability.Object); !isObject {
					t.Fatalf("Resolve($) = %T, want the response object", value)
				}
				return
			}
			if value != c.want {
				t.Errorf("Resolve(%q) = %#v, want %#v", c.path, value, c.want)
			}
		})
	}
}

// TestResolve_NothingIsNotNull is the distinction stated on its own, because it
// is the one a plausible resolver collapses: both answers are a Go nil, and
// only the second return tells them apart.
func TestResolve_NothingIsNotNull(t *testing.T) {
	object := response(t)

	null, resolved := projection.Resolve("$.body.error", object)
	if !resolved || null != nil {
		t.Errorf("$.body.error = %#v, %v; want nil, true — the body carried a null", null, resolved)
	}
	nothing, resolved := projection.Resolve("$.body.missing", object)
	if resolved || nothing != nil {
		t.Errorf("$.body.missing = %#v, %v; want nil, false — the body carried no such member", nothing, resolved)
	}
}

// TestProject is the recorded fields off one response, in the Manifest's own
// order, with a path that resolved to nothing contributing no field at all —
// which is the absence a version is written without (§6, §7).
func TestProject(t *testing.T) {
	read := projection.Read(operation(t, `
kind: read
record:
  identity: $.host
  fields:
    host: $.host
    status: $.status
    days_left: $.tls.days_left
    subject: $.tls.subject
`))

	projected := read.Project(response(t))
	want := projection.Fields{
		{Name: "host", Value: "status.hyper.dev"},
		{Name: "status", Value: 503},
		{Name: "days_left", Value: 34},
	}
	if len(projected) != len(want) {
		t.Fatalf("Project = %#v, want %#v", projected, want)
	}
	for i := range want {
		if projected[i] != want[i] {
			t.Errorf("field %d = %#v, want %#v", i, projected[i], want[i])
		}
	}

	// The wire shape is the shape a Record would have held: one compact
	// mapping in the Manifest's own order, which is the order the page
	// beneath it renders (§9, ADR-0026).
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), `{"host":"status.hyper.dev","status":503,"days_left":34}`; got != want {
		t.Errorf("projection on the wire = %s, want %s", got, want)
	}
	empty, err := json.Marshal(projection.Fields(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(empty), "{}"; got != want {
		t.Errorf("a projection that resolved nothing = %s, want %s", got, want)
	}
}

// TestRead_AnOperationProjectingNothing is the reader's absence rule: a destroy
// declares no record: at all, and an Operation nobody declared is read the same
// way (§3, ADR-0037, ADR-0064).
func TestRead_AnOperationProjectingNothing(t *testing.T) {
	for _, source := range []string{
		"kind: destroy\n",
		"kind: read\nrecord:\n  identity: $.host\n",
	} {
		if read := projection.Read(operation(t, source)); len(read.Fields) != 0 {
			t.Errorf("Read(%q) = %#v, want no fields", source, read.Fields)
		}
	}
	if read := projection.Read(nil); len(read.Fields) != 0 {
		t.Errorf("Read(nil) = %#v, want no fields", read.Fields)
	}
}

// TestText is what a projected value reads as on a page: a string as itself,
// and everything else as the JSON it is.
func TestText(t *testing.T) {
	for _, c := range []struct {
		value any
		want  string
	}{
		{"status.hyper.dev", "status.hyper.dev"},
		{503, "503"},
		{34, "34"},
		{nil, "null"},
		{true, "true"},
		{map[string]any{"id": "abc"}, `{"id":"abc"}`},
	} {
		if got := projection.Text(c.value); got != c.want {
			t.Errorf("Text(%#v) = %q, want %q", c.value, got, c.want)
		}
	}
}

// operation parses one Operation's declaration and answers its node, which is
// what Read is handed — the node artefact.OperationNode finds in a Manifest.
func operation(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(source), &root); err != nil {
		t.Fatal(err)
	}
	return root.Content[0]
}

// seriesResponse is one response of an Operation of `series` cardinality: a
// collection under a path, whose members are what identity: and every fields:
// entry root at.
func seriesResponse(t *testing.T, encoded string) capability.Object {
	t.Helper()

	var body any
	if err := json.Unmarshal([]byte(encoded), &body); err != nil {
		t.Fatal(err)
	}
	return capability.Object{
		{Name: capability.MemberHost, Value: "api.hyper.dev"},
		{Name: capability.MemberStatus, Value: 200},
		{Name: capability.MemberBody, Value: body},
	}
}

// TestRead_TheTwoRootsAreOneBlock is §3's `series` cardinality read: `over:`
// names the collection and sits beside the fields: paths that root inside it,
// so one reading of one block answers both roots.
func TestRead_TheTwoRootsAreOneBlock(t *testing.T) {
	read := projection.Read(operation(t, "kind: read\nrecord:\n  identity: $.id\n  over: $.body.records\n  fields:\n    state: $.state\n"))
	if read.Over != "$.body.records" {
		t.Errorf("Over = %q, want $.body.records", read.Over)
	}
	if len(read.Fields) != 1 || read.Fields[0].Path != "$.state" {
		t.Errorf("Fields = %#v, want one entry rooted at a member", read.Fields)
	}
	if one := projection.Read(operation(t, "kind: read\nrecord:\n  identity: $.host\n")); one.Over != "" {
		t.Errorf("Over = %q on an Operation of one cardinality, want the empty string", one.Over)
	}
}

// TestCollection_ACollectionThatWasEmptyIsNotAPathThatWasWrong is the
// distinction §6 halts a Run over: both answer no members, and only one of them
// resolved.
func TestCollection_ACollectionThatWasEmptyIsNotAPathThatWasWrong(t *testing.T) {
	empty := seriesResponse(t, `{"records":[]}`)
	if members, resolved := projection.Collection("$.body.records", empty); !resolved || len(members) != 0 {
		t.Errorf("an empty collection = %#v, resolved %v; want no members and resolved", members, resolved)
	}
	if _, resolved := projection.Collection("$.body.items", empty); resolved {
		t.Error("a path the response does not carry answers resolved")
	}
	// A path that reached a value with nothing inside it is not a second
	// fault: it resolved, and a scalar has no members.
	if members, resolved := projection.Collection("$.status", empty); !resolved || len(members) != 0 {
		t.Errorf("a path reaching a scalar = %#v, resolved %v; want no members and resolved", members, resolved)
	}
}

// TestProject_EveryPathBesideOverRootsAtAMember is the other root: the same
// grammar, the same walk, and `$` naming whatever the collection held.
func TestProject_EveryPathBesideOverRootsAtAMember(t *testing.T) {
	response := seriesResponse(t, `{"records":[{"id":"r1","state":"ready"},{"id":"r2"}]}`)
	members, resolved := projection.Collection("$.body.records", response)
	if !resolved || len(members) != 2 {
		t.Fatalf("Collection = %#v, resolved %v; want two members", members, resolved)
	}

	read := projection.Read(operation(t, "kind: read\nrecord:\n  identity: $.id\n  over: $.body.records\n  fields:\n    state: $.state\n"))
	if got := encoded(t, read.Project(members[0])); got != `{"state":"ready"}` {
		t.Errorf("the first member projected %s, want {\"state\":\"ready\"}", got)
	}
	// The second member carries no `state`, which is the ordinary absence:
	// the field is not written, and nothing about it is an error (§6).
	if got := encoded(t, read.Project(members[1])); got != `{}` {
		t.Errorf("the second member projected %s, want {}", got)
	}
	if name, resolved := projection.Resolve("$.id", members[1]); !resolved || name != "r2" {
		t.Errorf("the second member's identity = %v, resolved %v; want r2", name, resolved)
	}
}

// TestResolve_AShellScalarHasNothingInsideIt is the grammar reaching no
// further inside a scalar than §12 says it does: `$.stdout` answers the text a
// command printed, and no path reaches inside it (ADR-0052).
func TestResolve_AShellScalarHasNothingInsideIt(t *testing.T) {
	shell := capability.Object{
		{Name: capability.MemberCommand, Value: `["status","--json"]`},
		{Name: capability.MemberStdout, Value: `{"result":{"id":"abc"}}`},
	}
	if text, resolved := projection.Resolve("$.stdout", shell); !resolved || text != `{"result":{"id":"abc"}}` {
		t.Errorf("$.stdout = %v, resolved %v; want the string the command printed", text, resolved)
	}
	if _, resolved := projection.Resolve("$.stdout.result", shell); resolved {
		t.Error("$.stdout.result resolved; a path reaches no further inside a scalar than inside any string")
	}
}

// encoded is a projection's fields as the one mapping they render to.
func encoded(t *testing.T, fields projection.Fields) string {
	t.Helper()
	written, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(written)
}
