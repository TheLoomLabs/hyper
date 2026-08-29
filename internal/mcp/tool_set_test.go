package mcp

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// The tool set as a closed list, held where the list is (§9, issue #204).
//
// This is tree_test.go's own shape reaching the second surface: *a surface the
// specification fixes, asserted as a list*. The transcription below is §9's
// table typed out from the specification rather than read from tools.go, so
// that what follows is a check on the set and not a restatement of it — the day
// a milestone adds a fourteenth tool, the same edit that adds it has to add it
// here, which is what makes a fourteenth a thing someone decided rather than a
// thing that happened.

// sectionNineTools is §9's tool table, transcribed from the specification's own
// groups and in their order: Discovery, the repository, Authoring, Execution,
// Inspection, Lifecycle. Thirteen tools, each named for the command it carries,
// with `run_show` the one name that differs from its command's.
var sectionNineTools = []string{
	"providers", "provider", "operation",
	"targets",
	"check", "review",
	"run", "probe",
	"runs", "run_show", "changes", "records",
	"project",
}

// TestTools_AreSectionNinesThirteen holds the tool set against the
// specification it transcribes, in the two places it exists: the compiled-in
// table, and what a client receives from `tools/list`.
//
// **The order asserted is the table's**, and that is the honest half of §9's
// sentence rather than a weakening of it. The set a client receives is the same
// thirteen in **name** order — the SDK holds its tools keyed by name and pages
// them sorted — so there is no wire ordering for a group order to be checked
// against, and asserting one would be asserting the library's. What §9's groups
// order is the place a fourteenth would be written, which is this table
// (tools.go, Server.Tools).
//
// The two halves are one case because they are one claim asked at its two ends:
// a table §9 does not hold, or a table that reached the wire short, are both
// *the tool set is not §9's*, and a reader chasing that sentence should find it
// answered once.
func TestTools_AreSectionNinesThirteen(t *testing.T) {
	declared := make([]string, 0, len(tools))
	for _, held := range tools {
		declared = append(declared, held.name)
	}

	if got, want := declared, sectionNineTools; !slices.Equal(got, want) {
		t.Errorf("the tool set is %q,\n           want §9's %q", got, want)
	}
	if len(tools) != 13 {
		t.Errorf("the tool set is %d tools, want the thirteen §9 states", len(tools))
	}
	seen := make(map[string]bool, len(tools))
	for _, held := range tools {
		if seen[held.name] {
			t.Errorf("%q is declared twice in the tool set", held.name)
		}
		seen[held.name] = true
	}

	server, _ := answering(nil, render.NewResultRow(false))
	listed, err := server.Tools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	published := make([]string, 0, len(listed))
	for _, held := range listed {
		published = append(published, held.Name)
	}
	slices.Sort(published)
	want := slices.Sorted(slices.Values(sectionNineTools))
	if !slices.Equal(published, want) {
		t.Errorf("tools/list answers %q,\n                want §9's %q", published, want)
	}
}

// TestListTools_PublishesEverySchemaAsItWasWritten is what makes the corpus's
// schema golden a reading of the wire rather than of this package: the two
// schemas cross as the bytes they were declared as, so a golden holding them
// holds the keys in the order tools.go wrote them (Server.Tools).
//
// A schema decoded into a map and re-encoded would still be the same schema and
// would no longer be the same bytes, which is exactly the drift a checked-in
// golden is for.
func TestListTools_PublishesEverySchemaAsItWasWritten(t *testing.T) {
	server, _ := answering(nil, render.NewResultRow(false))
	listed, err := server.Tools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	declared := map[string]tool{}
	for _, held := range tools {
		declared[held.name] = held
	}
	for _, published := range listed {
		wrote, found := declared[published.Name]
		if !found {
			t.Errorf("tools/list answers %q, which the table does not declare", published.Name)
			continue
		}
		if got, want := compactSchema(t, published.Input), compactSchema(t, wrote.input); got != want {
			t.Errorf("%s publishes the input schema\n  %s\nand declares\n  %s", published.Name, got, want)
		}
		if got, want := compactSchema(t, published.Output), compactSchema(t, wrote.output); got != want {
			t.Errorf("%s publishes the output schema\n  %s\nand declares\n  %s", published.Name, got, want)
		}
	}
}

// compactSchema is one schema with its insignificant whitespace gone and its
// key order kept — the one comparison that says two schemas are the same bytes
// without saying they were laid out the same way. The table writes its schemas
// as indented raw strings and the wire carries them compact, which is a
// difference in the framing and not in the schema.
func compactSchema(t *testing.T, schema json.RawMessage) string {
	t.Helper()

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, schema); err != nil {
		t.Fatalf("a schema is not one JSON value: %v", err)
	}
	return compacted.String()
}

// TestToolSet_TheRenderingMemberIsDeclaredWhereTheTextBlockIsAPage is what
// keeps §9's two channels from drifting apart on the tool that has a page to
// lose (§9, ADR-0100, issue #217).
//
// **The rule is keyed on the case and not on the tool's name.** A tool whose
// text block is the command's page carries that page in `structuredContent` as
// well (Structured.Rendering), so a tool granted `wholeRendering` and no member
// here would be one whose whole promise lives on one channel. `review` is the
// only one today; the next one arrives with the member or arrives failing.
//
// It is **required** where it is declared, on the footing `truncated` is
// required on every one of the thirteen: an output schema states what the tool
// answers. That footing is ADR-0102's rather than the one issue #217 stated it
// on — a guardrail declining no longer stands *outside* the schemas, it answers
// no structured half at all, so there is no half for a required member to be
// missing from and the schema can state the answer without qualification
// (structuredOf).
func TestToolSet_TheRenderingMemberIsDeclaredWhereTheTextBlockIsAPage(t *testing.T) {
	for _, held := range tools {
		t.Run(held.name, func(t *testing.T) {
			schema := declared(t, held.output)
			_, declares := schema.Properties["rendering"]

			if want := held.text == wholeRendering; declares != want {
				if want {
					t.Errorf("%s's text block is the command's page and its output schema declares no rendering member; the page would reach a structured-only reader on neither channel", held.name)
				} else {
					t.Errorf("%s's text block is not a page and its output schema declares a rendering member; a summary line is composed of members the structured half already carries", held.name)
				}
			}
			if declares && !slices.Contains(schema.Required, "rendering") {
				t.Errorf("%s declares the rendering member and does not require it; what the tool answers is what the schema states", held.name)
			}
		})
	}
}
