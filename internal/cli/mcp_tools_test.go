package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// The two surfaces held to one list, and the schemas held to their bytes (§9,
// issue #204).
//
// tree_test.go one file over transcribes §9's sixteen commands and holds the
// compiled-in tree against them. This is the same reading where the second
// surface is: the thirteen tools are §9's sixteen less three, the three are
// absent by decision rather than by omission, and what a client writes its
// calls against is checked in.

// listing is `tools/list` as a client receives it, over a server assembled the
// way the binary assembles one.
//
// The process is empty and the facts carry nothing but a version, which is the
// whole of what a listing needs: no tool runs, no repository resolves and no
// gate is reached, so a fixture repository here would be an input nothing
// reads (mcp.Server.Tools).
func listing(t *testing.T) []mcp.Declared {
	t.Helper()

	listed, err := cli.MCPServer(cli.Process{}, version.Facts{Version: defaultVersion}).Tools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 {
		t.Fatal("tools/list answered nothing; every rule below would be held over an empty set")
	}
	return listed
}

// commandsBehindTheTools is the command each of the thirteen tools carries: the
// tool's own name, with the one exception §9 states resolved through the table
// golden_mcp_test.go already keeps for the corpus's pairings.
//
// It is that table rather than a second one because *a tool is named for the
// command it carries* is one fact, and two lists of the exceptions to it would
// be two lists to keep in step.
func commandsBehindTheTools(t *testing.T) []string {
	t.Helper()

	var carried []string
	for _, tool := range listing(t) {
		name := tool.Name
		if command, differs := commands[name]; differs {
			name = command
		}
		carried = append(carried, name)
	}
	slices.Sort(carried)
	return carried
}

// absentFromTheToolSet is the three of §9's sixteen that get no tool, and one
// line puts all three on the far side of the boundary: **an agent may read the
// record and add to it, and may not create it, prune it, or bring anything new
// into the repository.**
//
// `install` is the single point at which third-party data enters the
// repository, and the review moment is what an agent installing an Extension
// costs — the claim that a hostile Extension reaches nothing you did not grant
// survives it, and the human between acquisition and effect does not (§9,
// ADR-0004). `store` creates the record and `compact` removes from it
// permanently, and `compact` is the one command that would let an agent prune
// the account it is itself held to (§7, §9).
//
// It is transcribed from §9 rather than derived from what happens to be
// unimplemented, which is the whole difference this fence is for: an absence
// nobody wrote down is one a fourteenth tool closes by accident.
var absentFromTheToolSet = []string{"compact", "install", "store"}

// TestToolSet_IsSectionNinesSixteenLessThree holds the second surface against
// the first: the commands the thirteen tools carry are exactly §9's tree less
// the three above, and the three are named rather than merely missing.
//
// **The subtraction is the assertion.** A test listing thirteen names would
// pass the day a fourteenth tool carried `compact`, because nothing in it would
// be about the tree; holding the two transcriptions against each other is what
// makes a tool for an absent command fail here rather than pass quietly.
func TestToolSet_IsSectionNinesSixteenLessThree(t *testing.T) {
	reachable := slices.DeleteFunc(cli.Tree(), func(command string) bool {
		return slices.Contains(absentFromTheToolSet, command)
	})
	slices.Sort(reachable)

	if got := commandsBehindTheTools(t); !slices.Equal(got, reachable) {
		t.Errorf("the tools carry %q,\n         want §9's tree less %q: %q", got, absentFromTheToolSet, reachable)
	}
}

// TestToolSet_TheFiveCommandsWithNoToolHaveNone is the same absence stated the
// way §9 states it — by name, one command at a time — and it reaches the two
// the subtraction above cannot.
//
// **The list it derives holds six spellings and §9 names five**, and the sixth
// is not a slip: `store` is the verb `store init` hangs off, and a tool named
// for the bare verb would be the same crossing under a shorter name. §9's five
// are the acts; this is the vocabulary they can be typed in.
//
// `install`, `store init` and `compact` are the three inside the tree.
// `version` and `completions` are the two outside it, and they are absent for a
// different reason worth keeping distinct: *a client writes no completion
// script*, and *the version of the binary that would act is the version of the
// server the client started*, which the handshake already carries (§9,
// ADR-0088).
//
// **`store init` is why this is a fence over spellings and not over names.**
// The command is a verb and a sub-verb, and a tool for it would be spelled
// `store_init` — a name §9's tree holds nothing equal to, so a subtraction over
// the tree alone would never see it. What is compared is the tool's name with
// its underscores read back as the spaces §9's tree writes, which is the same
// transformation §9 states for an argument's name read the other way round.
func TestToolSet_TheFiveCommandsWithNoToolHaveNone(t *testing.T) {
	// Derived from the two compiled-in lists rather than typed out, so that
	// a second `store` sub-verb or a fourth command outside the tree joins
	// this fence by existing — one entry each, spelled as a command line
	// spells it. `mcp` is the one outside the tree that is not a command
	// with no tool: it is the invocation that starts the server, so a tool
	// for it would be the surface offering to start itself.
	absent := slices.Clone(absentFromTheToolSet)
	for _, verb := range cli.StoreSubVerbs() {
		absent = append(absent, "store "+verb)
	}
	for _, outside := range cli.OutsideTree() {
		if outside != "mcp" {
			absent = append(absent, outside)
		}
	}
	slices.Sort(absent)

	// And held against §9's own sentence, which is the half a derivation
	// cannot state: these six spellings and no others are what the
	// specification puts on the far side of the line.
	if want := []string{"compact", "completions", "install", "store", "store init", "version"}; !slices.Equal(absent, want) {
		t.Fatalf("the commands with no tool derive as %q, want §9's %q", absent, want)
	}

	for _, tool := range listing(t) {
		spelled := strings.ReplaceAll(tool.Name, "_", " ")
		if slices.Contains(absent, spelled) {
			t.Errorf("the tool set carries %q, which is %q; §9 puts it on the far side of the line an agent may not cross", tool.Name, spelled)
		}
	}
}

// toolsGolden is where the published schemas are checked in: one file under the
// second surface's own corpus, beside the cases that call the tools it
// describes.
const toolsGolden = "testdata/mcp/tools.golden"

// TestToolSet_PublishesTheSchemasItsGoldenHolds is the fence the tool set needs
// that no case can give it: **a schema is the contract an agent writes its
// calls against**, and a schema that drifts between two releases is the one way
// this surface can break a caller without any answer changing (§9, issue #204).
//
// A `call` case holds what one call answered; nothing in the corpus holds what
// a client is *told it may ask for*, so an argument silently widened from an
// enum to a bare string, a required member made optional, or an output member
// dropped from a closed object would pass every case that does not happen to
// exercise it. This is the one golden about the surface rather than about an
// answer, and it is regenerated behind the corpus's own `-update` — one flag
// serves them all, so a run that regenerates the corpus regenerates this too
// (golden_test.go).
//
// It holds the **descriptions** beside the schemas, and that is deliberate
// rather than incidental: a description is what an agent chooses a tool by, so
// a reworded one is a change to how this surface is reached even though no
// schema moved.
func TestToolSet_PublishesTheSchemasItsGoldenHolds(t *testing.T) {
	rendered := renderToolListing(t, listing(t))
	if *update {
		compareRendering(t, toolsGolden, rendered)
		return
	}
	if held := readFile(t, toolsGolden); held != rendered {
		reportSchemaDrift(t, held, rendered)
	}
}

// reportSchemaDrift is why this golden is not compared through
// compareRendering: the file is two thousand lines, and the whole of it quoted
// twice is a failure nobody reads and everybody regenerates. What a reader
// needs is **which tool moved and where**, so the two renderings are walked
// line by line and each difference is reported under the `=== ` header above
// it.
//
// The comparison is still byte for byte — the lines are the whole file — and
// the regeneration is still compareRendering's behind the corpus's one flag.
// This is the reporting and nothing else.
func reportSchemaDrift(t *testing.T, held, rendered string) {
	t.Helper()

	was, is := strings.Split(held, "\n"), strings.Split(rendered, "\n")
	tool := "(before the first tool)"
	var reported int
	for i := 0; i < max(len(was), len(is)); i++ {
		before, after := lineAt(was, i), lineAt(is, i)
		if name, opens := strings.CutPrefix(after, "=== "); opens {
			tool = name
		}
		if before == after {
			continue
		}
		reported++
		if reported > schemaDriftReported {
			t.Errorf("%s: %d further lines are left to compare and the reporting stops here; regenerate it with `go test ./internal/cli -update` and read the diff", toolsGolden, max(len(was), len(is))-i)
			return
		}
		t.Errorf("%s: %s, line %d:\n held:      %q\n publishes: %q", toolsGolden, tool, i+1, before, after)
	}
}

// schemaDriftReported is how many differing lines are worth printing before the
// rest is better read as a diff. A schema that moved reports as a handful of
// lines; a tool added or removed reports as everything after it, which is a
// count rather than a list.
const schemaDriftReported = 10

// lineAt is one line of a rendering, and a marker where that rendering has no
// such line — the two sides of this comparison being different lengths whenever
// a tool arrives or leaves.
func lineAt(lines []string, i int) string {
	if i >= len(lines) {
		return "(no line)"
	}
	return lines[i]
}

// renderToolListing is how the corpus holds a tool listing: each tool opened by
// name, its description on the line beneath, and its two schemas indented.
//
// **The order is the wire's**, which is name order — the SDK holds its tools
// keyed by name and pages them sorted. §9's group order is the table's own and
// is held where the table is (internal/mcp's tool_set_test.go); what belongs in
// a golden of the published surface is the order a client actually reads.
//
// The schemas are indented rather than written as they arrived, because what
// this file is for is being read in a diff: a one-line schema whose fourth
// nested property changed reports as one line replaced, which says a schema
// moved and not which part of it did. Indenting is the golden's own choice and
// changes no byte of what crossed — the keys stay in the order tools.go wrote
// them, which is the property the whole file rests on (mcp.Declared).
func renderToolListing(t *testing.T, listed []mcp.Declared) string {
	t.Helper()

	var page strings.Builder
	for _, tool := range listed {
		fmt.Fprintf(&page, "=== %s\n%s\n", tool.Name, tool.Description)
		fmt.Fprintf(&page, "--- input\n%s\n", indented(t, tool.Input))
		fmt.Fprintf(&page, "--- output\n%s\n", indented(t, tool.Output))
	}
	return page.String()
}

// indented is one schema laid out over lines, two spaces to the level. It is
// json.Indent and nothing else: the elision of insignificant whitespace is what
// makes the layout the golden's rather than the raw string's in tools.go, and
// the key order is untouched.
func indented(t *testing.T, schema json.RawMessage) string {
	t.Helper()

	var laid bytes.Buffer
	if err := json.Indent(&laid, schema, "", "  "); err != nil {
		t.Fatalf("a published schema is not one JSON value: %v", err)
	}
	return laid.String()
}

// TestToolSet_EveryPublishedSchemaIsOneClosedObject reads what is **checked
// in** and holds it to the two things §9 fixes about every schema this surface
// publishes: it is one JSON value, and it is a closed object.
//
// It reads the golden rather than the listing on the corpus's own footing — a
// fence reading the golden notices a hand edit, and one reading the server
// would agree with the server by construction. The same rule holds over the
// declarations one package over, where a violation is caught before it is ever
// written down (internal/mcp's tool_set_test.go); here it is caught in the one
// file a reader is invited to read.
//
// The closed object is what makes *no tool takes an override argument of any
// kind, under any name* a thing a client can check rather than a promise.
func TestToolSet_EveryPublishedSchemaIsOneClosedObject(t *testing.T) {
	held := readToolsGolden(t)

	published := make([]string, 0, len(held))
	for name, schemas := range held {
		published = append(published, name)
		for half, schema := range map[string]string{"input": schemas.input, "output": schemas.output} {
			var read struct {
				Type                 string `json:"type"`
				AdditionalProperties *bool  `json:"additionalProperties"`
			}
			if err := json.Unmarshal([]byte(schema), &read); err != nil {
				t.Errorf("%s's published %s schema is not one JSON value: %v", name, half, err)
				continue
			}
			if read.Type != "object" || read.AdditionalProperties == nil || *read.AdditionalProperties {
				t.Errorf("%s's published %s schema is not a closed object:\n%s", name, half, schema)
			}
		}
	}

	listed := make([]string, 0, len(held))
	for _, tool := range listing(t) {
		listed = append(listed, tool.Name)
	}
	slices.Sort(published)
	slices.Sort(listed)
	if !slices.Equal(published, listed) {
		t.Errorf("%s holds schemas for %q and tools/list answers %q", toolsGolden, published, listed)
	}
}

// publishedSchemas is one tool's two schemas as the golden holds them.
type publishedSchemas struct{ input, output string }

// readToolsGolden reads the checked-in listing back into the tools it names.
// It is the layout renderToolListing writes, parsed: `=== ` opens a tool, the
// line beneath it is the description, and `--- ` opens each of the two schemas.
func readToolsGolden(t *testing.T) map[string]publishedSchemas {
	t.Helper()

	held := map[string]publishedSchemas{}
	for _, block := range strings.Split(readFile(t, toolsGolden), "=== ") {
		name, rest, opened := strings.Cut(block, "\n")
		if !opened {
			continue
		}
		_, schemas, split := strings.Cut(rest, "--- input\n")
		if !split {
			t.Errorf("%s: the entry for %s carries no input schema", toolsGolden, name)
			continue
		}
		input, output, both := strings.Cut(schemas, "--- output\n")
		if !both {
			t.Errorf("%s: the entry for %s carries no output schema", toolsGolden, name)
			continue
		}
		held[name] = publishedSchemas{input: strings.TrimSpace(input), output: strings.TrimSpace(output)}
	}
	if len(held) == 0 {
		t.Fatalf("%s names no tool; regenerate it with `go test ./internal/cli -update`", toolsGolden)
	}
	return held
}
