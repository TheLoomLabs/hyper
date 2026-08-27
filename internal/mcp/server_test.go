package mcp

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// The surface, held to what §9 fixes about it (issue #195).
//
// The corpus one package over is what says a tool answers its command's rows:
// a `call` case drives this server against a fixture repository and holds the
// whole envelope byte for byte (internal/cli/golden_mcp_test.go). What is here
// is the half a golden cannot state — the handshake, the tool listing, and the
// two shapes an envelope takes that no fixture repository produces — and each
// case is about the protocol rather than about any command.

// stubRow is a row of a shape no command builds: enough of §8's contract to be
// one — a `type` first, members in declaration order — and nothing else. It is
// here so that the envelope's composition is exercised against a row this file
// wrote rather than against a command's, which is what keeps these cases about
// the surface.
type stubRow struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (r stubRow) Cells() []string { return []string{r.Name} }

// reviewPage is a rendering standing in for §8's review surface: a header line,
// the gutter beside a source line, and the two blocks beneath it. It is this
// file's own rather than a command's, for stubRow's reason — what is exercised
// is the crossing, not what any command renders — and it ends in a newline
// because a page does.
const reviewPage = "  DEFINITION  │  definitions/uptime.yaml\n" +
	"  read        │ kinds: [read]\n\n" +
	"  AUTHORITY   assembled from definitions/ and targets/\n\n" +
	"  FLAGS   index into the gutter above\n"

// answering is a server whose every tool answers the same rows, whatever argv
// it was handed, with the argv recorded for the cases that are about it.
//
// It fills the rendering as well, because every command fills one: the
// destination behind a tool writes the command's page into a buffer on every
// answer, and a helper that left it empty would make a tool §9's table names a
// fault in the server on cases that are about something else entirely
// (destination.go, answerText).
func answering(rows []render.Row, terminal render.Row) (*Server, *[]string) {
	var argv []string
	return NewServer("1.4.0", func(line []string) Answer {
		argv = line
		return Answer{Rows: rows, Terminal: terminal, Rendering: reviewPage}
	}), &argv
}

// connected is a client session against a server, over the in-memory
// transports Call uses: the same real handshake, with the session left open so
// a case can ask it what `initialize` answered.
func connected(t *testing.T, s *Server) *sdk.ClientSession {
	t.Helper()

	serverSide, clientSide := sdk.NewInMemoryTransports()
	session, err := s.server().Connect(t.Context(), serverSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	client, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).
		Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// TestInitialize_AnnouncesHyperAtTheBinarysOwnVersion is the handshake: the
// server is `hyper` at the version of the binary that would act, which is the
// same string the pin gate compares — *the version of the binary that would act
// is the version of the server the client started* (§9, ADR-0088).
func TestInitialize_AnnouncesHyperAtTheBinarysOwnVersion(t *testing.T) {
	server, _ := answering(nil, render.NewResultRow(false))
	announced := connected(t, server).InitializeResult().ServerInfo

	if announced.Name != "hyper" {
		t.Errorf("the server announces %q, want hyper", announced.Name)
	}
	if announced.Version != "1.4.0" {
		t.Errorf("the server announces version %q, want the binary's own", announced.Version)
	}
}

// TestInitialize_AdvertisesTheToolsCapabilityAndNoOther is §9's *tools only: no
// resources and no prompts*, held at the one moment a client reads it.
//
// `logging` is the member this case is really for. The SDK's default
// capabilities are `{"logging":{}}`, and hyper has no logging channel: *hyper
// never speaks first* — it initiates no message between calls and has no
// channel to initiate one on (ADR-0021). A capability advertised by default is
// one a client may use, so the default is what this case refuses.
func TestInitialize_AdvertisesTheToolsCapabilityAndNoOther(t *testing.T) {
	server, _ := answering(nil, render.NewResultRow(false))
	advertised := connected(t, server).InitializeResult().Capabilities

	if advertised.Tools == nil {
		t.Error("the server advertises no tools capability; it serves tools and nothing else")
	}
	if advertised.Logging != nil {
		t.Error("the server advertises a logging capability; hyper never speaks first and has no logging channel")
	}
	if advertised.Prompts != nil {
		t.Error("the server advertises a prompts capability; prompts are per-editor glue, which is the thing this tool departs from")
	}
	if advertised.Resources != nil {
		t.Error("the server advertises a resources capability; the read-only half is deliberately not served as resources")
	}
	if advertised.Completions != nil {
		t.Error("the server advertises a completions capability; nothing here completes anything")
	}
}

// TestListTools_AnswersEveryRegisteredToolWithBothSchemas is `tools/list`: the
// tools registered so far, each carrying its input schema and its output
// schema.
//
// The output schema is the half worth naming. §9 states that *an `outputSchema`
// is declared once and for every call of the tool*, which is the whole reason
// the three discovery questions are three tools rather than one taking optional
// arguments — a tool that answered a different shape per call could not declare
// one.
func TestListTools_AnswersEveryRegisteredToolWithBothSchemas(t *testing.T) {
	server, _ := answering(nil, render.NewResultRow(false))
	listed, err := connected(t, server).ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]*sdk.Tool{}
	for _, tool := range listed.Tools {
		got[tool.Name] = tool
	}
	if len(got) != len(tools) {
		t.Errorf("tools/list answered %d tools, want the %d registered", len(got), len(tools))
	}
	for _, registered := range tools {
		tool, listed := got[registered.name]
		if !listed {
			t.Errorf("tools/list does not answer %q", registered.name)
			continue
		}
		if tool.Description == "" {
			t.Errorf("%s carries no description", tool.Name)
		}
		if !objectSchema(t, tool.InputSchema) {
			t.Errorf("%s's input schema is %v, want a closed object", tool.Name, tool.InputSchema)
		}
		if !objectSchema(t, tool.OutputSchema) {
			t.Errorf("%s's output schema is %v, want a closed object", tool.Name, tool.OutputSchema)
		}
	}
}

// objectSchema answers whether a schema as a client received it is the closed
// object every schema this surface publishes is: `additionalProperties: false`
// is what makes *no tool takes an override argument of any kind, under any
// name* a thing a client can check rather than a promise (§9).
func objectSchema(t *testing.T, schema any) bool {
	t.Helper()

	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var read struct {
		Type                 string `json:"type"`
		AdditionalProperties *bool  `json:"additionalProperties"`
	}
	if err := json.Unmarshal(encoded, &read); err != nil {
		t.Fatal(err)
	}
	return read.Type == "object" && read.AdditionalProperties != nil && !*read.AdditionalProperties
}

// TestCall_TheEnvelopeCarriesTheRowsTheTerminalFactAndTheBit is the ordinary
// return: §8's row set as an array, the terminal row's own `truncated` moved up
// beside it, and `isError` written whichever it is (§9).
//
// **No terminal row travels inside `rows`.** An array's end is already its own
// end-of-stream marker, which is the whole reason the member moves.
func TestCall_TheEnvelopeCarriesTheRowsTheTerminalFactAndTheBit(t *testing.T) {
	rows := []render.Row{stubRow{Type: "provider", Name: "alpha"}, stubRow{Type: "provider", Name: "shell"}}
	server, _ := answering(rows, render.NewResultRow(false))

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(envelope.StructuredContent.Rows), 2; got != want {
		t.Fatalf("the envelope carries %d rows, want %d and no terminal row among them", got, want)
	}
	if got, want := string(envelope.StructuredContent.Rows[0]), `{"type":"provider","name":"alpha"}`; got != want {
		t.Errorf("the first row is %s, want %s", got, want)
	}
	if got, want := string(envelope.StructuredContent.Truncated), "false"; got != want {
		t.Errorf("truncated is %s, want %s", got, want)
	}
	if envelope.IsError {
		t.Error("isError is true on an ordinary return")
	}
	if got, want := envelope.Content, []TextBlock{{Type: "text", Text: "2 Providers"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("the text block is %v, want %v", got, want)
	}
}

// TestCall_RowsIsAnEmptyArrayWhereTheCommandFoundNothing is the shape §9 states
// and Go makes easy to get wrong: `rows` is `[]` where the command found
// nothing **rather than absent**, and a nil slice marshals as `null`.
//
// The corpus now drives it too — `targets` against a repository that declares
// none answers `[]` (testdata/mcp/targets/no-targets-directory) — and this case
// stays because it holds the rule where the composition is rather than where a
// fixture happens to reach it: `providers` always finds the built-in and
// `provider` always writes its Manifest's header row, so for two of the four
// tools the shape has no repository to be driven from at all.
func TestCall_RowsIsAnEmptyArrayWhereTheCommandFoundNothing(t *testing.T) {
	server, _ := answering(nil, render.NewResultRow(false))

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(envelope.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), `{"rows":[],"truncated":false}`; got != want {
		t.Errorf("the structured content is %s, want %s", got, want)
	}
}

// TestCall_ACutResultSaysSoInBothHalves is §9's *a truncated result must never
// look complete*, on the one surface with no stderr to say it on: the CLI
// writes a line beside the answer and a tool's narration goes to io.Discard, so
// the text block is the only prose left.
func TestCall_ACutResultSaysSoInBothHalves(t *testing.T) {
	rows := []render.Row{stubRow{Type: "provider", Name: "alpha"}}
	server, _ := answering(rows, render.NewResultRow(true))

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(envelope.StructuredContent.Truncated), "true"; got != want {
		t.Errorf("truncated is %s, want %s", got, want)
	}
	if got := envelope.Content[0].Text; !strings.Contains(got, "truncated") {
		t.Errorf("the text block is %q and does not say the result was cut", got)
	}
}

// TestCall_ATruncationMarkerTravelsWholeAndUnretyped is the other two of §9's
// three shapes for `truncated`, and the reason the member is lifted rather than
// read: the marker object is a shape render.Truncation already knows how to
// write, and a reader here that switched on the value would be a second
// implementation of that choice.
func TestCall_ATruncationMarkerTravelsWholeAndUnretyped(t *testing.T) {
	marker := render.TruncationMarker{
		Axis: render.AxisTime, Returned: 50, Dropped: 2840, Hint: "narrow with `since` or `target`",
	}
	rows := []render.Row{stubRow{Type: "provider", Name: "alpha"}}
	server, _ := answering(rows, render.NewTruncatedResultRow(marker))

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"axis":"time","returned":50,"dropped":2840,"hint":"narrow with ` + "`since`" + ` or ` + "`target`" + `"}`
	if got := string(envelope.StructuredContent.Truncated); got != want {
		t.Errorf("truncated is %s, want %s", got, want)
	}
}

// TestCall_AnArgumentTheSchemaDoesNotAdmitIsAProtocolError is §9's line, and
// the whole reason the tools are registered through the low-level
// `Server.AddTool`: **a domain outcome is never a protocol error, and an
// argument violating a schema is not a domain outcome**. The generic
// registration would answer a `CallToolResult` with `IsError: true` here, which
// is the shape a Refusal has — and this surface has no exit code with which to
// tell the two apart.
//
// **Every case here is refused by hyper's own reading and not by the SDK's**,
// which is the property rather than a gap in the cases: a schema is a claim a
// client may or may not check, and the server is where the claim is made true
// (tools.go). The empty name is the case that says so — the schema's
// `minLength` and the tool's own reading refuse the same argument — and what
// the table catches is an SDK that begins validating and answers a **result**
// where these expect an error, whichever of the two would have refused it.
func TestCall_AnArgumentTheSchemaDoesNotAdmitIsAProtocolError(t *testing.T) {
	for _, called := range []struct{ name, tool, arguments string }{
		{"a member no schema declares", "providers", `{"limit":10}`},
		{"an override under a name of its own", "provider", `{"name":"shell","repo_dir":"/elsewhere"}`},
		{"a member of the wrong type", "provider", `{"name":7}`},
		{"a required member left off", "provider", `{}`},
		{"a name the schema's minLength refuses", "provider", `{"name":""}`},
		{"one of two positionals left off", "operation", `{"provider":"shell"}`},
		{"the second positional named nothing at all", "operation", `{"provider":"shell","operation":""}`},
		{"the first positional named nothing at all", "operation", `{"provider":"","operation":"destroy"}`},
		{"a cap on a tool that takes no arguments", "targets", `{"limit":10}`},
		{"a repair flag on a gate", "check", `{"fix":true}`},
		{"a path list that is not a list", "check", `{"paths":"definitions/a.yaml"}`},
		{"a path member of the wrong type", "check", `{"paths":[7]}`},
		{"a path naming nothing at all", "check", `{"paths":[""]}`},
		{"a second path naming nothing at all", "check", `{"paths":["definitions/a.yaml",""]}`},
		{"the artefact to review left off", "review", `{}`},
		{"an artefact naming nothing at all", "review", `{"artefact":""}`},
	} {
		t.Run(called.name, func(t *testing.T) {
			server, _ := answering(nil, render.NewResultRow(false))

			envelope, err := server.Call(t.Context(), called.tool, json.RawMessage(called.arguments))
			if err == nil {
				t.Fatalf("the call answered %+v, want a protocol error", envelope)
			}
		})
	}
}

// TestCall_TheArgvIsTheCommandLineItsCommandWouldHaveReceived is the whole of
// what a tool is past its schemas: §9 fixes that *ergonomics is the whole of
// the difference between the two*, so a tool builds a command line and hands it
// to the same dispatch, holding no command logic of its own.
//
// The `--` is the case's second half. A Provider name is matched byte-exact
// against a namespace that can hold anything, so a name spelled like a flag
// must arrive as the positional it is: the parser already states that "--" ends
// the flags, which makes this the command line the command would have received
// rather than a guard invented in the tool.
func TestCall_TheArgvIsTheCommandLineItsCommandWouldHaveReceived(t *testing.T) {
	for _, called := range []struct {
		name, tool, arguments string
		want                  []string
	}{
		{"a tool taking no arguments", "providers", `{}`, []string{"providers"}},
		{"a tool taking a name", "provider", `{"name":"shell"}`, []string{"provider", "--", "shell"}},
		{"a name spelled like a flag", "provider", `{"name":"--json"}`, []string{"provider", "--", "--json"}},
		{"a tool taking two names", "operation", `{"provider":"shell","operation":"destroy"}`, []string{"operation", "--", "shell", "destroy"}},
		{"two names spelled like flags", "operation", `{"provider":"--json","operation":"--limit"}`, []string{"operation", "--", "--json", "--limit"}},
		{"a second tool taking no arguments", "targets", `{}`, []string{"targets"}},
		{"a positional list left off", "check", `{}`, []string{"check"}},
		{"a positional list with one member", "check", `{"paths":["definitions/a.yaml"]}`, []string{"check", "--", "definitions/a.yaml"}},
		{"a positional list with two", "check", `{"paths":["definitions/a.yaml","targets/local.yaml"]}`, []string{"check", "--", "definitions/a.yaml", "targets/local.yaml"}},
		{"a path spelled like a flag", "check", `{"paths":["--json"]}`, []string{"check", "--", "--json"}},
		{"a tool taking an artefact by path", "review", `{"artefact":"procedures/deploy.yaml"}`, []string{"review", "--", "procedures/deploy.yaml"}},
		{"a tool taking an artefact by name", "review", `{"artefact":"deploy"}`, []string{"review", "--", "deploy"}},
	} {
		t.Run(called.name, func(t *testing.T) {
			server, argv := answering(nil, render.NewResultRow(false))

			if _, err := server.Call(t.Context(), called.tool, json.RawMessage(called.arguments)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*argv, called.want) {
				t.Errorf("the tool built %q, want %q", *argv, called.want)
			}
		})
	}
}

// TestSummary_CountsTheRowsByTheirOwnDiscriminator is §9's *one summary line,
// outcome first* on the tools that have no outcome: what a listing answered,
// counted by the `type` §8 puts first on every row.
func TestSummary_CountsTheRowsByTheirOwnDiscriminator(t *testing.T) {
	for _, one := range []struct {
		name      string
		kinds     []string
		truncated bool
		want      string
	}{
		{"a listing", []string{"provider", "provider", "provider"}, false, "3 Providers"},
		{"one of a kind", []string{"provider"}, false, "1 Provider"},
		{"a header and the rows beneath it", []string{"manifest", "operation", "operation"}, false, "1 Manifest, 2 Operations"},
		{"one Operation seen up close", []string{"operation_detail"}, false, "1 Operation"},
		{"the repository's grants", []string{"target", "target"}, false, "2 Targets"},
		{"a cut listing", []string{"provider", "provider"}, true, "2 Providers, truncated"},
		{"nothing found", nil, false, "no rows"},
		{"a row type the table does not name", []string{"widget"}, false, "1 widget"},
	} {
		t.Run(one.name, func(t *testing.T) {
			if got := summary(one.kinds, one.truncated); got != one.want {
				t.Errorf("summary = %q, want %q", got, one.want)
			}
		})
	}
}

// The mapping: §9's table from the exit code a command returned to the envelope
// this surface answers with (issue #196).
//
// Every case here drives a server whose dispatch answers one exit code, because
// that is the axis: what a command found is the corpus's to state, and what
// happens to `77` is this file's. Three of the seven codes are reachable from
// the two tools m11.3 built — the corpus drives those against fixture
// repositories — and the rest are reachable from tools their own tickets build,
// so the rule is held here over a dispatch that answers them directly.
//
// **The codes are spelled as numbers and not read off internal/exit**, which is
// the rule TestExitCodes_AreTheClosedSetOfSeven already keeps one package over:
// a case that took its wanted value from the constant under test would agree
// with whatever that constant happened to be. §12 fixes the numbers, and these
// are the numbers.

// returning is a server whose every tool answers this Answer, whatever argv it
// was handed. It stands beside `answering` for the cases that are about the
// exit code rather than about the rows.
func returning(answer Answer) *Server {
	return NewServer("1.4.0", func([]string) Answer { return answer })
}

// TestCall_APositionalThatMatchesNothingIsAProtocolError is §9's third member
// of the malformed set, and the one this surface has in place of exit code `2`:
// `provider("nope")` satisfies every schema and still names nothing.
//
// **The message is what the command wrote to stderr**, so an agent reads the
// sentence a person would have read. Returning it as a domain answer would give
// it `isError: true` with no `outcome` key — which is exactly the shape a
// guardrail declining already returns, and the distinction the CLI half spends
// `2` to draw (§9, ADR-0060).
func TestCall_APositionalThatMatchesNothingIsAProtocolError(t *testing.T) {
	wrote := "hyper provider: no Provider named \"nope\" in this repository's Provider namespace\n" +
		"  hyper providers lists every Provider in it\n"
	server := returning(Answer{Exit: 2, Narration: wrote})

	envelope, err := server.Call(t.Context(), "provider", json.RawMessage(`{"name":"nope"}`))
	if err == nil {
		t.Fatalf("the call answered %+v, want a protocol error", envelope)
	}
	if got, want := err.Error(), strings.TrimRight(wrote, "\n"); got != want {
		t.Errorf("the error carries %q, want the sentence the command wrote: %q", got, want)
	}
}

// TestCall_ACallToANameOutsideTheToolSetIsAProtocolError is the first member of
// §9's malformed set. It is the SDK's own answer rather than hyper's, and it is
// held anyway: what the set says is that *every* malformed call arrives as a
// protocol error, and an SDK that answered an unknown tool with a result
// carrying the bit would put a usage error where a Refusal lives.
func TestCall_ACallToANameOutsideTheToolSetIsAProtocolError(t *testing.T) {
	server := returning(Answer{Terminal: render.NewResultRow(false)})

	envelope, err := server.Call(t.Context(), "run", json.RawMessage(`{}`))
	if err == nil {
		t.Fatalf("the call answered %+v, want a protocol error", envelope)
	}
}

// TestCall_AGuardrailDecliningIsTheRefusalRenderedWhole is §9's `77`: the two
// codes the version pin gate Refuses under are reachable from every tool, and
// what comes back is the whole rendering as `text`, `isError: true`, and **no
// `outcome` key at all** — a tool that is not a Run carries none, and the gate
// is not a Run refusing.
func TestCall_AGuardrailDecliningIsTheRefusalRenderedWhole(t *testing.T) {
	for _, refused := range []struct{ name, rendering string }{
		{"version-pin-absent", "refused: version-pin-absent\n  hyper.yaml carries no version pin — run: hyper project\n"},
		{"version-pin-mismatch", "refused: version-pin-mismatch\n  this binary is 1.4.0; the repository pins 9.9.9 — run: hyper project, or install 9.9.9\n"},
	} {
		t.Run(refused.name, func(t *testing.T) {
			server := returning(Answer{Exit: 77, Refusal: refused.rendering})

			envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
			if err != nil {
				t.Fatalf("a Refusal is an answer to a well-formed call, not a protocol error: %v", err)
			}
			if !envelope.IsError {
				t.Error("isError is false on a Refusal; it means only that you did not get what you asked for")
			}
			if got := envelope.Content[0].Text; !strings.HasPrefix(got, refused.rendering) {
				t.Errorf("the text block is %q, want the whole rendering first", got)
			}
			structured, err := json.Marshal(envelope.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(structured), `{"rows":[],"truncated":null}`; got != want {
				t.Errorf("the structured content is %s, want %s: no rows, no outcome key, and nothing restating the bit", got, want)
			}
		})
	}
}

// TestCall_EveryRefusalSaysAVerbatimRetryRefusesIdentically is load-bearing
// rather than manners: `isError: true` conventionally invites a retry, and this
// surface has no exit code `77` with which to say otherwise (§9, ADR-0001). The
// rendering is the only place the protocol leaves for saying it.
func TestCall_EveryRefusalSaysAVerbatimRetryRefusesIdentically(t *testing.T) {
	server := returning(Answer{Exit: 77, Refusal: "refused: version-pin-absent\n  hyper.yaml carries no version pin — run: hyper project\n"})

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimRight(envelope.Content[0].Text, "\n"); !strings.HasSuffix(got, retrySentence) {
		t.Errorf("the text block ends %q, want it to end with %q", got, retrySentence)
	}
}

// TestCall_AnAnswerTheWorldResistedCarriesTheBitAndTheRows is the other half of
// §9's `isError`: `1` and `75` are answers, not protocol errors, and the rows a
// command found travel exactly as they do on a clean return. Neither code is
// reachable from a Discovery tool — `check` exercises `1` and `run` exercises
// `75`, in their own tickets — which is why the rule is held here.
func TestCall_AnAnswerTheWorldResistedCarriesTheBitAndTheRows(t *testing.T) {
	for _, code := range []struct {
		name string
		exit int
	}{
		{"problems the command found", 1},
		{"the Store lost", 75},
	} {
		t.Run(code.name, func(t *testing.T) {
			rows := []render.Row{stubRow{Type: "provider", Name: "alpha"}}
			server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Exit: code.exit})

			envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
			if err != nil {
				t.Fatalf("exit %d is an answer, not a protocol error: %v", code.exit, err)
			}
			if !envelope.IsError {
				t.Errorf("isError is false on exit %d; you did not get what you asked for", code.exit)
			}
			if got, want := len(envelope.StructuredContent.Rows), 1; got != want {
				t.Errorf("the envelope carries %d rows, want %d", got, want)
			}
			if got, want := envelope.Content[0].Text, "1 Provider"; got != want {
				t.Errorf("the text block is %q, want %q", got, want)
			}
		})
	}
}

// TestCall_AnExitTheMappingDoesNotHoldIsAFaultInTheServer is what is left of
// §12's seven once §9's table has taken five: `130` and `143` are unreachable
// here — the server installs no signal watch — so a command answering one is a
// fault in the server rather than an envelope this surface knows how to
// compose. A wrong envelope is harder to notice than a missing one.
func TestCall_AnExitTheMappingDoesNotHoldIsAFaultInTheServer(t *testing.T) {
	for _, code := range []int{130, 143, 3} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			server := returning(Answer{Terminal: render.NewResultRow(false), Exit: code})

			envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
			if err == nil {
				t.Fatalf("the call answered %+v, want a fault in the server", envelope)
			}
		})
	}
}

// TestCall_AGuardrailDecliningWithNothingRenderedIsAFaultInTheServer is the
// same rule read the other way: `77` is *the Refusal rendered in full*, so an
// answer carrying the code and no rendering is a path through a command that
// left the remediation nowhere — and with no bypass anywhere the rendering is
// the entire way past (ADR-0001).
func TestCall_AGuardrailDecliningWithNothingRenderedIsAFaultInTheServer(t *testing.T) {
	server := returning(Answer{Exit: 77})

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err == nil {
		t.Fatalf("the call answered %+v, want a fault in the server", envelope)
	}
}

// The Authoring pair, and the one asymmetry they bring to the envelope (§9,
// issue #198).
//
// What each of the two tools *finds* is the corpus's to state — a `call` case
// drives them against the check and review corpora's own repositories and holds
// the whole envelope byte for byte (internal/cli/golden_mcp_test.go). What is
// here is §9's text-block table, which is a property of the tool rather than of
// any repository: a fixture can show `review`'s rendering arriving as text, and
// only a dispatch this file writes can show it arriving *instead of* a summary
// line, arriving *after* a Refusal takes precedence over it, and failing where
// there is no rendering at all.

// TestCall_ReviewsTextBlockIsTheRenderingWholeAndUntouched is the second row of
// §9's asymmetric table: **`review` carries the full rendered review surface**,
// and the whole of it — the gutter, `AUTHORITY`, `FLAGS` — where every other
// tool carries one summary line.
//
// The rendering is compared byte for byte, trailing newline included, because
// that is the promise: what the tool hands back is what the command writes to
// stdout, so that an agent can hand a human reviewer the page verbatim before
// asking them to read it (§9, ADR-0026).
func TestCall_ReviewsTextBlockIsTheRenderingWholeAndUntouched(t *testing.T) {
	rows := []render.Row{stubRow{Type: "gutter", Name: "read"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Rendering: reviewPage})

	envelope, err := server.Call(t.Context(), "review", json.RawMessage(`{"artefact":"uptime"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := envelope.Content, []TextBlock{{Type: "text", Text: reviewPage}}; !reflect.DeepEqual(got, want) {
		t.Errorf("the text block is %q, want the rendering whole: %q", got, want)
	}
	if envelope.IsError {
		t.Error("isError is true on a review that rendered; a FLAGS row is a fact about the artefact rather than a problem with it")
	}
	if got, want := len(envelope.StructuredContent.Rows), 1; got != want {
		t.Errorf("the envelope carries %d rows, want %d: the rendering is the text and the rows travel beside it", got, want)
	}
}

// TestCall_ATextBlockIsASummaryLineOnEveryToolTheTableDoesNotName is the first
// row of the same table, held over the tool that would be easiest to widen into
// the second: `check` answers a rendering of its own and still summarises, §9
// naming `review` and nothing else.
//
// The rendering is supplied and deliberately not carried, which is what makes
// this a case about the table rather than about an empty buffer.
func TestCall_ATextBlockIsASummaryLineOnEveryToolTheTableDoesNotName(t *testing.T) {
	rows := []render.Row{stubRow{Type: "problem", Name: "a"}, stubRow{Type: "problem", Name: "b"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Rendering: "FILE  LINE\n", Exit: 1})

	envelope, err := server.Call(t.Context(), "check", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := envelope.Content[0].Text, "2 problems"; got != want {
		t.Errorf("the text block is %q, want the summary line %q", got, want)
	}
	if !envelope.IsError {
		t.Error("isError is false on a repository with problems; the caller did not get what they asked for")
	}
	// The **members of the structured half**, and not a search through its
	// bytes: every real `check` row carries an `error_code` of its own, so a
	// search would either pass vacuously here or fail against a fixture
	// repository. What §9 states is about the envelope — a tool that is not
	// a Run carries no `outcome` key, and the code naming a check that
	// declined belongs to the row rather than to the answer around it.
	structured, err := json.Marshal(envelope.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(structured, &members); err != nil {
		t.Fatal(err)
	}
	if got, want := slices.Sorted(maps.Keys(members)), []string{"rows", "truncated"}; !slices.Equal(got, want) {
		t.Errorf("the structured content is keyed %q, want %q: no outcome key, no error_code of the envelope's own, and nothing restating the bit", got, want)
	}
}

// TestCall_ARefusalTakesPrecedenceOverAToolsOwnRendering is where the two
// asymmetries meet, and the order they meet in: §9's table gives a Refusal its
// own row, so a `review` the version pin gate declines carries **the Refusal**
// and not the page — whatever the destination happened to have in its buffer.
//
// The two renderings are two buffers for exactly this reason (destination.go),
// and the case supplies both so that the arm is choosing rather than finding
// only one.
func TestCall_ARefusalTakesPrecedenceOverAToolsOwnRendering(t *testing.T) {
	refusal := "refused: version-pin-absent\n  hyper.yaml carries no version pin — run: hyper project\n"
	server := returning(Answer{Exit: 77, Refusal: refusal, Rendering: reviewPage})

	envelope, err := server.Call(t.Context(), "review", json.RawMessage(`{"artefact":"uptime"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := envelope.Content[0].Text; !strings.HasPrefix(got, refusal) {
		t.Errorf("the text block is %q, want the Refusal rendered whole first", got)
	}
	if strings.Contains(envelope.Content[0].Text, "AUTHORITY") {
		t.Error("the text block carries the page as well as the Refusal; §9's table gives a Refusal its own row")
	}
}

// TestCall_ARenderingToolThatRenderedNothingIsAFaultInTheServer is the
// rendering half of the rule TestCall_AGuardrailDecliningWithNothingRendered...
// holds for a Refusal: where the text block *is* the rendering, an answer
// carrying none has nothing left to say, and a wrong envelope is harder to
// notice than a missing one.
func TestCall_ARenderingToolThatRenderedNothingIsAFaultInTheServer(t *testing.T) {
	server := returning(Answer{Terminal: render.NewResultRow(false)})

	envelope, err := server.Call(t.Context(), "review", json.RawMessage(`{"artefact":"uptime"}`))
	if err == nil {
		t.Fatalf("the call answered %+v, want a fault in the server", envelope)
	}
}
