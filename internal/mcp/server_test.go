package mcp

import (
	"encoding/json"
	"reflect"
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

// answering is a server whose every tool answers the same rows, whatever argv
// it was handed, with the argv recorded for the cases that are about it.
func answering(rows []render.Row, terminal render.Row) (*Server, *[]string) {
	var argv []string
	return NewServer("1.4.0", func(line []string) Answer {
		argv = line
		return Answer{Rows: rows, Terminal: terminal}
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
// It is here rather than in the corpus because neither tool this milestone
// builds can answer no rows — `providers` always finds the built-in and
// `provider` always writes its Manifest's header row — so the rule has no
// fixture repository to be driven from and would otherwise be held by nothing.
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
func TestCall_AnArgumentTheSchemaDoesNotAdmitIsAProtocolError(t *testing.T) {
	for _, called := range []struct{ name, tool, arguments string }{
		{"a member no schema declares", "providers", `{"limit":10}`},
		{"an override under a name of its own", "provider", `{"name":"shell","repo_dir":"/elsewhere"}`},
		{"a member of the wrong type", "provider", `{"name":7}`},
		{"a required member left off", "provider", `{}`},
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
