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
	"github.com/TheLoomLabs/hyper/internal/store"
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

// outcomeStub is §8's other terminal row in the shape the command writes it,
// and it stands here for the same reason stubRow does: what these cases
// exercise is the crossing, and a row this file wrote is one no command's
// changes can quietly alter underneath them.
//
// The members are the ones the envelope lifts — §12's triple, the rehearsal
// marker written always, and the Run id absent where no entry was written — in
// the order the command's own row declares them (cli's outcomeRow).
type outcomeStub struct {
	Type      string `json:"type"`
	Outcome   string `json:"outcome"`
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code,omitempty"`
	DryRun    bool   `json:"dry_run"`
	RunID     string `json:"run_id,omitempty"`
}

// Cells is empty: the terminal row's line on the page is §8's terminal line,
// which the command's page writes beneath its table rather than inside it.
func (r outcomeStub) Cells() []string { return nil }

// ran is a Run's terminal row as this file writes one: the outcome, the code
// §12 fixes for it, the rehearsal marker, and the entry it wrote.
func ran(outcome string, code int, dryRun bool, runID string) outcomeStub {
	return outcomeStub{Type: "outcome", Outcome: outcome, Code: code, DryRun: dryRun, RunID: runID}
}

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
	return NewServer("1.4.0", func(call Call) Answer {
		argv = call.Argv
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

// TestCall_ATruncationMarkersHintNamesTheToolsArgumentsAndNotItsCommandsFlags
// is §9's marker on this surface, and the **one member of an answer whose
// wording differs between the two**: the axis and both counts are the command's
// own, and the hint names what a caller of a tool would type — which is an
// argument, there being no flag to type here at all (§9, issue #199).
//
// The marker is `changes`'s, because it is the one whose two spellings are not
// one word: `--kind` is `record_kind` in a flat argument object, where a bare
// `kind` is an Operation's Kind. A surface that rewrote the sentence rather than
// spelling the parameters would have no way to know that.
func TestCall_ATruncationMarkersHintNamesTheToolsArgumentsAndNotItsCommandsFlags(t *testing.T) {
	marker := render.TruncationMarker{
		Axis:     render.AxisIdentity,
		Returned: 50,
		Dropped:  2840,
		Narrows:  render.Narrowing{{Flag: "--target", Argument: "target"}, {Flag: "--kind", Argument: "record_kind"}},
	}
	rows := []render.Row{stubRow{Type: "provider", Name: "alpha"}}
	server, _ := answering(rows, render.NewTruncatedResultRow(marker))

	envelope, err := server.Call(t.Context(), "providers", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"axis":"identity","returned":50,"dropped":2840,"hint":"narrow with ` + "`target`" + ` or ` + "`record_kind`" + `"}`
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
//
// **What is not here is the argument a command refuses**, and its absence is
// the boundary rather than a gap: `outcome: "open"`, `limit: 0` and
// `record_kind: "tombstone"` are well-typed values their commands close a set
// against where the flag is read, so each arrives as a protocol error carrying
// **the command's own sentence** — which is a fixture repository's to drive and
// the corpus's to hold (§9, flags.go, internal/cli/testdata/mcp). A tool that
// refused them here would be holding a second copy of a closed set §12 owns.
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
		{"a Procedure named as the empty string", "runs", `{"procedure":""}`},
		{"a cap that is not a whole number", "runs", `{"limit":1.5}`},
		{"a predicate where a typed parameter belongs", "runs", `{"where":"outcome == failed"}`},
		{"the Run id left off", "run_show", `{}`},
		{"a Run id naming nothing at all", "run_show", `{"run_id":""}`},
		{"an expansion that is not a boolean", "run_show", `{"run_id":"0199206d-4e15-7c30-9b8a-52d9ea01f7b4","expansion":"yes"}`},
		{"a window with one end", "changes", `{"between":["0199206d-4e15-7c30-9b8a-52d9ea01f7b4"]}`},
		{"a window with three", "changes", `{"between":["a","b","c"]}`},
		{"a window naming no end at all", "changes", `{"between":[]}`},
		{"an end of a window naming nothing", "changes", `{"between":["0199206d-4e15-7c30-9b8a-52d9ea01f7b4",""]}`},
		{"the CLI's flag name for it", "changes", `{"kind":"asset"}`},
		{"a Procedure named as the empty string on a tool whose positional is optional", "changes", `{"procedure":""}`},
		{"a Record named as the empty string", "records", `{"name":""}`},
		{"a history that is not a boolean", "records", `{"history":"all"}`},
		{"a Probe with no Operation named", "probe", `{"provider":"uptime"}`},
		{"a Probe naming its Provider as the empty string", "probe", `{"provider":"","operation":"check_http"}`},
		{"an inputs member that is a list", "probe", `{"provider":"uptime","operation":"check_http","inputs":{"host":["a"]}}`},
		{"an inputs member that is a mapping", "probe", `{"provider":"uptime","operation":"check_http","inputs":{"host":{"name":"a"}}}`},
		{"an inputs member that is null", "probe", `{"provider":"uptime","operation":"check_http","inputs":{"host":null}}`},
		{"an input named by the empty string", "probe", `{"provider":"uptime","operation":"check_http","inputs":{"":"a"}}`},
		{"an input whose name carries an =", "probe", `{"provider":"uptime","operation":"check_http","inputs":{"a=b":"c"}}`},
		{"inputs that are not an object at all", "probe", `{"provider":"uptime","operation":"check_http","inputs":"host=a"}`},
		{"a Definition where a Probe takes a Provider and an Operation", "probe", `{"definition":"uptime","operation":"check_http"}`},
		{"a Target on the one tool that binds local and nothing else", "probe", `{"provider":"uptime","operation":"check_http","target":"cloudflare-prod"}`},
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
		{"a listing with no parameter named", "runs", `{}`, []string{"runs"}},
		{"every parameter a listing takes", "runs",
			`{"since":"2026-08-04T09:12:00Z","procedure":"publish-preview","target":"local","outcome":"failed","limit":2}`,
			[]string{"runs", "--since", "2026-08-04T09:12:00Z", "--procedure", "publish-preview", "--target", "local", "--outcome", "failed", "--limit", "2"}},
		{"a name spelled like a flag in a parameter's value", "runs", `{"procedure":"--json"}`, []string{"runs", "--procedure", "--json"}},
		{"one entry read back", "run_show", `{"run_id":"0199206d-4e15-7c30-9b8a-52d9ea01f7b4"}`,
			[]string{"show", "--", "0199206d-4e15-7c30-9b8a-52d9ea01f7b4"}},
		{"one entry read back with its Expansions", "run_show", `{"run_id":"0199206d-4e15-7c30-9b8a-52d9ea01f7b4","expansion":true}`,
			[]string{"show", "--expansion", "--", "0199206d-4e15-7c30-9b8a-52d9ea01f7b4"}},
		{"an expansion asked for as false", "run_show", `{"run_id":"0199206d-4e15-7c30-9b8a-52d9ea01f7b4","expansion":false}`,
			[]string{"show", "--", "0199206d-4e15-7c30-9b8a-52d9ea01f7b4"}},
		{"a Comparison across every Procedure", "changes", `{}`, []string{"changes"}},
		{"a Comparison of one, with the positional last", "changes", `{"procedure":"publish-preview","target":"local"}`,
			[]string{"changes", "--target", "local", "--", "publish-preview"}},
		{"the Record type under the name the CLI spells --kind", "changes", `{"record_kind":"observation"}`,
			[]string{"changes", "--kind", "observation"}},
		{"a window named by its two ends", "changes", `{"between":["019917f2-2c81-7d55-8e3a-1b4c9d70e6a2","01991ea6-b118-7c93-8d41-6b2f7ae05c19"]}`,
			[]string{"changes", "--between", "019917f2-2c81-7d55-8e3a-1b4c9d70e6a2", "01991ea6-b118-7c93-8d41-6b2f7ae05c19"}},
		{"a window named both ways at once, which the command refuses", "changes", `{"since":"2026-08-04T09:12:00Z","between":["a","b"]}`,
			[]string{"changes", "--since", "2026-08-04T09:12:00Z", "--between", "a", "b"}},
		{"the Heads of every Record", "records", `{}`, []string{"records"}},
		{"one Record's history in a window", "records", `{"target":"local","definition":"uptime","name":"status.hyper.dev","history":true,"since":"2026-08-04T09:12:00Z","limit":5}`,
			[]string{"records", "--target", "local", "--definition", "uptime", "--name", "status.hyper.dev", "--history", "--since", "2026-08-04T09:12:00Z", "--limit", "5"}},
		{"a window named without a history, which the command refuses", "records", `{"since":"2026-08-04T09:12:00Z"}`,
			[]string{"records", "--since", "2026-08-04T09:12:00Z"}},
		{"a Run of a Procedure by name", "run", `{"procedure":"publish-preview"}`, []string{"run", "--", "publish-preview"}},
		{"a Run of a Procedure by path", "run", `{"procedure":"procedures/publish-preview.yaml"}`,
			[]string{"run", "--", "procedures/publish-preview.yaml"}},
		{"a Procedure spelled like a flag", "run", `{"procedure":"--dry-run"}`, []string{"run", "--", "--dry-run"}},
		{"a rehearsal", "run", `{"procedure":"publish-preview","dry_run":true}`, []string{"run", "--dry-run", "--", "publish-preview"}},
		{"a rehearsal asked for as false", "run", `{"procedure":"publish-preview","dry_run":false}`, []string{"run", "--", "publish-preview"}},
		{"a sink under the name of the thing it supplies", "run", `{"procedure":"read-session","secret_sink":"/run/secrets/session-token"}`,
			[]string{"run", "--secret-out", "/run/secrets/session-token", "--", "read-session"}},
		{"a rehearsal with a sink, both flags ahead of the positional", "run",
			`{"procedure":"read-session","dry_run":true,"secret_sink":"/run/secrets/session-token"}`,
			[]string{"run", "--dry-run", "--secret-out", "/run/secrets/session-token", "--", "read-session"}},
		{"a Probe of an Operation taking no inputs", "probe", `{"provider":"uptime","operation":"check_http"}`,
			[]string{"probe", "--", "uptime", "check_http"}},
		{"a Probe with its inputs left off entirely", "probe", `{"provider":"uptime","operation":"check_http","inputs":{}}`,
			[]string{"probe", "--", "uptime", "check_http"}},
		{"one input, as the repeated flag its command takes", "probe",
			`{"provider":"uptime","operation":"check_http","inputs":{"host":"status.hyper.dev"}}`,
			[]string{"probe", "--input", "host=status.hyper.dev", "--", "uptime", "check_http"}},
		{"two inputs, by name and not by the order they were written", "probe",
			`{"provider":"metrics","operation":"window","inputs":{"minutes":15,"host":"metrics.hyper.dev"}}`,
			[]string{"probe", "--input", "host=metrics.hyper.dev", "--input", "minutes=15", "--", "metrics", "window"}},
		{"a number spelled as the caller wrote it", "probe",
			`{"provider":"metrics","operation":"window","inputs":{"minutes":1.0}}`,
			[]string{"probe", "--input", "minutes=1.0", "--", "metrics", "window"}},
		{"a boolean input", "probe", `{"provider":"uptime","operation":"check_http","inputs":{"follow":true}}`,
			[]string{"probe", "--input", "follow=true", "--", "uptime", "check_http"}},
		{"a value carrying an = of its own, which the pair splits at its first", "probe",
			`{"provider":"uptime","operation":"check_http","inputs":{"query":"a=b"}}`,
			[]string{"probe", "--input", "query=a=b", "--", "uptime", "check_http"}},
		{"two names spelled like flags, past the --", "probe", `{"provider":"--json","operation":"--limit"}`,
			[]string{"probe", "--", "--json", "--limit"}},
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
		{"the Journal listed", []string{"run", "run"}, false, "2 Runs"},
		{"one entry read back whole", []string{"entry", "provenance", "step", "provenance"}, false, "1 Journal entry, 2 Provenance rows, 1 Step"},
		{"a plural that is not the English s", []string{"entry", "entry"}, false, "2 Journal entries"},
		{"a Comparison", []string{"window", "asset", "observation", "code"}, false, "1 window, 1 Asset, 1 Observation, 1 code fact"},
		{"a projection", []string{"workflow", "workflow"}, false, "2 workflows"},
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
	return NewServer("1.4.0", func(Call) Answer { return answer })
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
// structured half at all** (ADR-0102).
//
// The gate declines before the command opens a row stream, so there is nothing
// for the half to carry and it is absent rather than empty (structuredOf).
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
			if envelope.StructuredContent != nil {
				structured, err := json.Marshal(envelope.StructuredContent)
				if err != nil {
					t.Fatal(err)
				}
				t.Errorf("the structured content is %s, want none: the command opened no row stream, so there is nothing here to conform to the schema this tool published", structured)
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

// TestCall_ReviewsTextBlockIsTheRenderingWholeAndUntouched is `review`'s row of
// §9's asymmetric table: **`review` carries the full rendered review surface**,
// and the whole of it — the gutter, `AUTHORITY`, `FLAGS` — where no other tool
// carries a line of the command's page that the command did not draw as rows.
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

// TestCall_ReviewsPageTravelsInTheStructuredContentAsWell is the same row read
// on the other channel: **`review`'s page is written into `structuredContent`
// as well as into the text block**, and it is the same string rather than a
// second composition (§9, ADR-0100, issue #217).
//
// Why the block alone is not enough is Structured.Rendering's to say. What this
// holds is the composition: one page, both channels, byte for byte.
//
// The rows are asserted here for the reason `check`'s case asserts its block:
// *the rows did not move* is the half of the change the member above cannot
// see.
func TestCall_ReviewsPageTravelsInTheStructuredContentAsWell(t *testing.T) {
	rows := []render.Row{stubRow{Type: "gutter", Name: "read"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Rendering: reviewPage})

	envelope, err := server.Call(t.Context(), "review", json.RawMessage(`{"artefact":"uptime"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := envelope.StructuredContent.Rendering, reviewPage; got != want {
		t.Errorf("the structured content carries %q, want the page whole: %q", got, want)
	}
	if got, want := envelope.StructuredContent.Rendering, envelope.Content[0].Text; got != want {
		t.Errorf("the structured half carries %q and the text block %q; one page is composed once and written twice", got, want)
	}
	if got, want := len(envelope.StructuredContent.Rows), 1; got != want {
		t.Errorf("the envelope carries %d rows, want %d: this adds a channel and moves none", got, want)
	}
}

// TestCall_NoToolButAReviewCarriesARenderingMember is the narrowness of the row
// above, held over the three shapes that could plausibly have claimed it (§9,
// ADR-0100).
//
// A **listing** and a **`check`** carry none because their blocks are composed of
// members the structured half already holds, and a **Refusal** carries none
// because MCP names no structured channel for one at all. The three arguments
// are Structured.Rendering's; what this holds is that the code draws the line
// where they put it.
//
// **The Refusal's arm holds the stronger form of that claim.** Since ADR-0102 a
// guardrail declining a `review` answers no structured half at all, so the
// member is absent along with every other — which is the same argument reaching
// its conclusion rather than a different rule: *MCP names no structured channel
// for an error* is why a Refusal has no `rendering`, and it is why it has no
// half to put one in.
func TestCall_NoToolButAReviewCarriesARenderingMember(t *testing.T) {
	problems := []render.Row{stubRow{Type: "problem", Name: "a"}}
	entries := []render.Row{stubRow{Type: "entry", Name: "a"}}
	refusal := "refused: version-pin-absent\n  hyper.yaml carries no version pin — run: hyper project\n"

	for _, one := range []struct {
		name      string
		tool      string
		arguments string
		answer    Answer
	}{
		{"a listing", "runs", `{}`, Answer{Rows: entries, Terminal: render.NewResultRow(false), Rendering: reviewPage}},
		{"a check reporting problems", "check", `{}`, Answer{Rows: problems, Terminal: render.NewResultRow(false), Rendering: "FILE  LINE\n", Exit: 1}},
		{"a guardrail declining a review", "review", `{"artefact":"uptime"}`, Answer{Exit: 77, Refusal: refusal, Rendering: reviewPage}},
	} {
		t.Run(one.name, func(t *testing.T) {
			envelope, err := returning(one.answer).Call(t.Context(), one.tool, json.RawMessage(one.arguments))
			if err != nil {
				t.Fatal(err)
			}
			if envelope.StructuredContent == nil {
				if one.answer.Exit != 77 {
					t.Errorf("%s answered no structured half; only a guardrail declining answers content alone (ADR-0102)", one.name)
				}
				return
			}
			if got := envelope.StructuredContent.Rendering; got != "" {
				t.Errorf("the structured content carries the rendering %q, want none", got)
			}
			if members := structuredMembers(t, envelope); slices.Contains(members, "rendering") {
				t.Errorf("the structured content is keyed %q; the rendering member is written where the text block is a page and nowhere else", members)
			}
		})
	}
}

// TestCall_ATextBlockIsASummaryLineOnEveryToolTheTableDoesNotName is the first
// row of the same table, held over a tool that answers a rendering of its own
// and still summarises: §9 names four cases and a namespace listing is none of
// them.
//
// The rendering is supplied and deliberately not carried, which is what makes
// this a case about the table rather than about an empty buffer.
//
// `targets` is the exemplar rather than `runs`, which used to stand here and
// now names a row of its own (issue #233): what this holds is the **default**,
// so it has to be asked of a tool that declares nothing.
func TestCall_ATextBlockIsASummaryLineOnEveryToolTheTableDoesNotName(t *testing.T) {
	rows := []render.Row{stubRow{Type: "entry", Name: "a"}, stubRow{Type: "entry", Name: "b"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Rendering: reviewPage})

	envelope, err := server.Call(t.Context(), "targets", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := envelope.Content[0].Text, "2 Journal entries"; got != want {
		t.Errorf("the text block is %q, want the summary line %q", got, want)
	}
	if strings.Contains(envelope.Content[0].Text, "AUTHORITY") {
		t.Error("the text block carries the page; §9 gives the whole rendering to review and to a Refusal, and to nothing else")
	}
}

// TestCall_TheTwoListingsOverTheRecordSayWhereTheRecordIs is §9's fourth row:
// `runs` and `records` carry store.Location beneath the summary line, on every
// answer including the one that found nothing (§9, ADR-0113, issue #233).
//
// **What it is holding is a false statement a session made in front of a
// human.** An agent that had just run two Procedures went looking for the
// account, found no `.hyper/`, no `store/` and a clean `git status`, and
// reported that a clone would get the Procedure and not the history. Nothing it
// was allowed to call would have contradicted it: these two tools render the
// account's content and rendered its location nowhere.
//
// The empty answer is asserted beside the full one because it is the call most
// easily read as *there is no record*, and because it is where the two arms of
// §9's table differ: `check`'s rows stand where the rows are, and this sentence
// stands whether or not there are any.
func TestCall_TheTwoListingsOverTheRecordSayWhereTheRecordIs(t *testing.T) {
	for _, one := range []struct {
		name string
		tool string
		rows []render.Row
		want string
	}{
		{"runs", "runs", []render.Row{stubRow{Type: "entry", Name: "a"}}, "1 Journal entry"},
		{"records", "records", []render.Row{stubRow{Type: "record", Name: "a"}}, "1 Record"},
		{"runs over an empty Store", "runs", nil, "no rows"},
		{"records over an empty Store", "records", nil, "no rows"},
	} {
		t.Run(one.name, func(t *testing.T) {
			server := returning(Answer{Rows: one.rows, Terminal: render.NewResultRow(false)})

			envelope, err := server.Call(t.Context(), one.tool, json.RawMessage(`{}`))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := envelope.Content[0].Text, one.want+"\n\n"+store.Location; got != want {
				t.Errorf("the text block is %q, want the summary line and the record's location beneath it: %q", got, want)
			}
			if !strings.Contains(envelope.Content[0].Text, "travels with a clone") {
				t.Error("the text block never says the record travels with a clone; portability is the half of the fact a session got backwards")
			}
		})
	}
}

// TestCall_ChecksTextBlockCarriesItsRowsBeneathTheSummaryLine is §9's third
// row, and the one an agent walks most: **a `check` that found problems carries
// them in the text block**, file, line, field, `error_code` and message, as the
// command's own renderer drew them (§9, issue #214).
//
// The summary line survives and stands first, because outcome-first is what
// this surface has in place of an exit code, and the rows go beneath it
// untouched — the composition prepends and so trims nothing, where the Refusal
// path trims because it appends.
//
// A repository with problems still answers `isError: true` and still carries
// every row in the structured half: this adds a channel and moves none. The
// keys of that half are asserted here rather than in a case of their own,
// because *the rows did not move* is the half of the change a text-block
// assertion cannot see — a block that carried the rows and a structured half
// that had lost them would satisfy every line above.
func TestCall_ChecksTextBlockCarriesItsRowsBeneathTheSummaryLine(t *testing.T) {
	table := "FILE                      LINE  FIELD    ERROR_CODE   MESSAGE\n" +
		"definitions/probe_d.yaml  4     binding  unknown-key  \"binding\" is not a key the schema at this position admits\n"
	rows := []render.Row{stubRow{Type: "problem", Name: "a"}, stubRow{Type: "problem", Name: "b"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Rendering: table, Exit: 1})

	envelope, err := server.Call(t.Context(), "check", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := envelope.Content[0].Text, "2 problems\n\n"+table; got != want {
		t.Errorf("the text block is %q, want the summary line and the rows beneath it: %q", got, want)
	}
	if !envelope.IsError {
		t.Error("isError is false on a repository with problems; the caller did not get what they asked for")
	}
	if got, want := len(envelope.StructuredContent.Rows), 2; got != want {
		t.Errorf("the envelope carries %d rows, want %d: the text block is a second channel and not a move", got, want)
	}
	if got, want := structuredMembers(t, envelope), []string{"rows", "truncated"}; !slices.Equal(got, want) {
		t.Errorf("the structured content is keyed %q, want %q: no outcome key, no error_code of the envelope's own, and nothing restating the bit", got, want)
	}
}

// structuredMembers is the keys the structured half went out under, sorted.
//
// It reads the **members** and not a search through the bytes: every real
// `check` row carries an `error_code` of its own and a `review`'s page carries
// the words a member name is spelled with, so a search would either pass
// vacuously or fail against a fixture repository. What §9 states is about the
// envelope — a tool that is not a Run carries no `outcome` key, a rendering
// member is written where the text block is a page and nowhere else — and the
// keys are where that is checkable.
func structuredMembers(t *testing.T, envelope Envelope) []string {
	t.Helper()

	structured, err := json.Marshal(envelope.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(structured, &members); err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(maps.Keys(members))
}

// TestCall_ACheckThatFoundNothingCarriesTheSummaryLineAlone is the same row
// read where there are no rows: what goes beneath the line is the row set, so
// an empty one puts nothing there.
//
// That is what keeps `check`'s row a fact about the **rows** rather than about
// the path a command took — the objection §9's `review` row answers by promising
// the page byte for byte on every path `review` answers at all. A clean check's
// page is a sentence about a count and not a table, and a text block that
// carried it would be this surface saying twice what it has already said once.
func TestCall_ACheckThatFoundNothingCarriesTheSummaryLineAlone(t *testing.T) {
	server := returning(Answer{Terminal: render.NewResultRow(false), Rendering: "checked 5 artefacts: no problems found\n"})

	envelope, err := server.Call(t.Context(), "check", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := envelope.Content[0].Text, "no rows"; got != want {
		t.Errorf("the text block is %q, want the summary line alone: %q", got, want)
	}
	if envelope.IsError {
		t.Error("isError is true on a repository with no problems")
	}
}

// TestCall_ACheckCarryingRowsAndNoRenderingIsAFaultInTheServer is
// TestCall_ARenderingToolThatRenderedNothing...'s rule read over `check`'s row:
// where the text block promises the rows and the command rendered none, the
// promise is unkeepable, and a wrong envelope is harder to notice than a
// missing one.
func TestCall_ACheckCarryingRowsAndNoRenderingIsAFaultInTheServer(t *testing.T) {
	rows := []render.Row{stubRow{Type: "problem", Name: "a"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(false), Exit: 1})

	envelope, err := server.Call(t.Context(), "check", json.RawMessage(`{}`))
	if err == nil {
		t.Fatalf("the call answered %+v, want a fault in the server", envelope)
	}
}

// TestCall_ACutResultSaysSoAboveTheRowsItCarries is §9's *a truncated result
// must never look complete*, held where the rows themselves enter the text
// block: the marker rides on the summary line, which stands **first**, so a
// reader meets it before the table rather than after it.
//
// `check` carries no `--limit` and its command cuts nothing, so no fixture
// repository produces this answer; the case is here because the rule is about
// the composition and not about what any command found.
func TestCall_ACutResultSaysSoAboveTheRowsItCarries(t *testing.T) {
	rows := []render.Row{stubRow{Type: "problem", Name: "a"}}
	server := returning(Answer{Rows: rows, Terminal: render.NewResultRow(true), Rendering: "FILE  LINE\n", Exit: 1})

	envelope, err := server.Call(t.Context(), "check", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := strings.Cut(envelope.Content[0].Text, "\n")
	if got, want := line, "1 problem, truncated"; got != want {
		t.Errorf("the first line of the text block is %q, want %q: a truncated result must never read as complete", got, want)
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

// The execution half (§9, issue #200).
//
// The corpus one package over is what says a Run through the tool writes the
// Store the command writes, and it holds every envelope byte for byte. What is
// here is the half no fixture repository produces: the two 75s, which write no
// terminal row at all, and the closure of the one input schema §9 spends a
// paragraph on.

// TestCall_ARunsEnvelopeLiftsTheTripleTheMarkerAndTheEntry is §9's execution
// members, taken from the terminal row and from nowhere else: what the CLI's
// `outcome` row carries and what the envelope carries are one fact rather than
// two that have to agree (ADR-0026).
func TestCall_ARunsEnvelopeLiftsTheTripleTheMarkerAndTheEntry(t *testing.T) {
	for _, one := range []struct {
		name     string
		terminal outcomeStub
		want     Structured
		isError  bool
		text     string
	}{
		{
			name:     "a Run that completed",
			terminal: ran("completed", 0, false, "01991ea6-b118-7c93-8d41-6b2f7ae05c19"),
			want:     Structured{Outcome: "completed", RunID: "01991ea6-b118-7c93-8d41-6b2f7ae05c19", DryRun: no},
			text:     "completed · run 01991ea6-b118-7c93-8d41-6b2f7ae05c19",
		},
		{
			name:     "a rehearsal, which is completed and says so",
			terminal: ran("completed", 0, true, "01991ea6-b118-7c93-8d41-6b2f7ae05c19"),
			want:     Structured{Outcome: "completed", RunID: "01991ea6-b118-7c93-8d41-6b2f7ae05c19", DryRun: yes},
			text:     "completed · dry-run · run 01991ea6-b118-7c93-8d41-6b2f7ae05c19",
		},
		{
			name:     "a Run the world resisted",
			terminal: ran("failed", 1, false, "01991ea6-b118-7c93-8d41-6b2f7ae05c19"),
			want:     Structured{Outcome: "failed", RunID: "01991ea6-b118-7c93-8d41-6b2f7ae05c19", DryRun: no},
			isError:  true,
			text:     "failed · run 01991ea6-b118-7c93-8d41-6b2f7ae05c19",
		},
		{
			name:     "a Run that lost the Store to a push it could not land",
			terminal: ran("failed", 75, false, "01991ea6-b118-7c93-8d41-6b2f7ae05c19"),
			want:     Structured{Outcome: "failed", RunID: "01991ea6-b118-7c93-8d41-6b2f7ae05c19", DryRun: no},
			isError:  true,
			text:     "failed · run 01991ea6-b118-7c93-8d41-6b2f7ae05c19",
		},
		{
			// The two paths that decline before a Run is identified.
			// What is missing is the `run_id` and never the key
			// beside it, exactly as §8 states for the row.
			name:     "a guardrail declining before a Run had an id",
			terminal: ran("refused", 77, false, ""),
			want:     Structured{Outcome: "refused", DryRun: no},
			isError:  true,
			text:     refusalText(reviewPage),
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			// The exit code is the terminal row's own, which is
			// what the command returns: §12's code space is finer
			// than the triple, and the row carries both.
			//
			// The Refusal buffer is left empty on the arm that
			// refuses, which is `run`'s own shape: a Run that a
			// guardrail declined renders the Refusal inside its
			// page, and the page is what the text block carries
			// (envelopeOf, runPage).
			server := NewServer("1.4.0", func(Call) Answer {
				return Answer{Terminal: one.terminal, Rendering: reviewPage, Exit: one.terminal.Code}
			})

			envelope, err := server.Call(t.Context(), "run", json.RawMessage(`{"procedure":"publish-preview"}`))
			if err != nil {
				t.Fatal(err)
			}
			structured := envelope.StructuredContent
			if structured.Outcome != one.want.Outcome {
				t.Errorf("outcome = %q, want %q", structured.Outcome, one.want.Outcome)
			}
			if structured.RunID != one.want.RunID {
				t.Errorf("run_id = %q, want %q", structured.RunID, one.want.RunID)
			}
			if !reflect.DeepEqual(structured.DryRun, one.want.DryRun) {
				t.Errorf("dry_run = %v, want %v", spelled(structured.DryRun), spelled(one.want.DryRun))
			}
			if envelope.IsError != one.isError {
				t.Errorf("isError = %t, want %t", envelope.IsError, one.isError)
			}
			if got := envelope.Content[0].Text; got != one.text {
				t.Errorf("the text block is %q, want %q", got, one.text)
			}
			if got := string(structured.Truncated); got != "null" {
				t.Errorf("truncated = %s, want null — a Run ranges over nothing for a limit to have cut", got)
			}
		})
	}
}

// TestCall_ARunThatWroteNoTerminalRowTakesItsOutcomeFromTheCode is the one path
// where the members are not lifted, and the reason the tool is asked what the
// call named.
//
// A Run that lost the Store to the lock or to the sync at Run start stands
// before `run.json`, so §8 leaves it writing nothing at all and the CLI's exit
// code carries the news on its own. This surface has no exit code, so §12's
// mapping stands in — and `dry_run` comes from the call, because §7 writes it
// wherever the outcome is and a rehearsal told it reached the world is the one
// error that cannot be walked back (executionOf).
func TestCall_ARunThatWroteNoTerminalRowTakesItsOutcomeFromTheCode(t *testing.T) {
	for _, one := range []struct {
		name, arguments string
		dryRun          *bool
		text            string
	}{
		{"a Run that lost the Store", `{"procedure":"publish-preview"}`, no, "failed"},
		{"a rehearsal that lost the Store", `{"procedure":"publish-preview","dry_run":true}`, yes, "failed · dry-run"},
	} {
		t.Run(one.name, func(t *testing.T) {
			server := NewServer("1.4.0", func(Call) Answer {
				// What the command left: a sentence on the
				// narration, and no answer at all.
				return Answer{Narration: "hyper run: another Run holds the Store lock\n", Exit: 75}
			})

			envelope, err := server.Call(t.Context(), "run", json.RawMessage(one.arguments))
			if err != nil {
				t.Fatal(err)
			}
			structured := envelope.StructuredContent
			if structured.Outcome != "failed" {
				t.Errorf("outcome = %q, want failed — §12 maps 75 onto it, and a caller with no exit code reads this instead", structured.Outcome)
			}
			if structured.RunID != "" {
				t.Errorf("run_id = %q, want none — no entry was written", structured.RunID)
			}
			if !reflect.DeepEqual(structured.DryRun, one.dryRun) {
				t.Errorf("dry_run = %v, want %v", spelled(structured.DryRun), spelled(one.dryRun))
			}
			if !envelope.IsError {
				t.Error("isError = false; a caller did not get what they asked for")
			}
			if got := envelope.Content[0].Text; got != one.text {
				t.Errorf("the text block is %q, want %q", got, one.text)
			}
		})
	}
}

// TestCall_NoToolButRunCarriesAnOutcomeKey is §9 read the other way: *a tool
// that is not a Run carries no `outcome` key at all*, and the marker beside it
// rides with the triple or not at all.
//
// It is held over the whole set rather than over a sample, because the shape it
// refuses is a tool acquiring an execution member by accident — the envelope
// composes one structured half for thirteen tools, and the day a second one
// carries an outcome is the day this surface has two accounts of what a Run is.
func TestCall_NoToolButRunCarriesAnOutcomeKey(t *testing.T) {
	var held int
	for _, declaring := range tools {
		if declaring.name == "run" {
			if declaring.executes == nil {
				t.Error("run declares no execution half; it is the one tool of the thirteen whose answer carries §12's triple")
			}
			continue
		}
		if declaring.executes != nil {
			t.Errorf("%s declares an execution half; §9 gives the outcome triple to run and to no other tool", declaring.name)
		}

		server, _ := answering([]render.Row{stubRow{Type: "provider", Name: "uptime"}}, render.NewResultRow(false))
		envelope, err := server.Call(t.Context(), declaring.name, argumentsSatisfying(t, declaring))
		if err != nil {
			// Unreachable against the stub above, which answers
			// every argv cleanly, and stated rather than skipped:
			// a tool that declined here would leave the rule held
			// over less than the set, which is what the count below
			// catches.
			t.Errorf("%s: %v", declaring.name, err)
			continue
		}
		held++
		switch structured := envelope.StructuredContent; {
		case structured.Outcome != "":
			t.Errorf("%s carries outcome %q; a tool that is not a Run carries no outcome key at all", declaring.name, structured.Outcome)
		case structured.DryRun != nil:
			t.Errorf("%s carries dry_run; the marker rides beside the triple and nowhere else", declaring.name)
		case structured.RunID != "":
			t.Errorf("%s names a Run; only the tool that performs one does", declaring.name)
		}
	}
	if held < len(tools)-1 {
		t.Errorf("%d of the %d tools that are not Runs answered an envelope; the rest were passed over and the rule was held over less than the set", held, len(tools)-1)
	}
}

// TestListTools_RunTakesTheThreeArgumentsAndNoBypassUnderAnyName is §9's *no
// tool takes an override argument of any kind, under any name*, held where it
// is holdable: the published schema.
//
// **It is a claim about closure and not about a list of forbidden words.** The
// schema is closed, so the names §9 spends paragraphs refusing — a `definition`,
// a `target`, an `inputs`, a `force` — are refused by there being three
// properties and no fourth, which also refuses the name nobody has thought of
// yet. The three are asserted by name because they are the signature, and a
// fourth appearing is what this case exists to fail on (ADR-0001, ADR-0008).
func TestListTools_RunTakesTheThreeArgumentsAndNoBypassUnderAnyName(t *testing.T) {
	schema := declared(t, runTool.input)

	if !schema.closed() {
		t.Error("run's input schema admits properties it does not state; a schema that admits a member it does not name is one under which an override argument is well-formed")
	}
	if got, want := schema.names(), []string{"dry_run", "procedure", "secret_sink"}; !slices.Equal(got, want) {
		t.Errorf("run takes %q, want %q — the occasion and never authority", got, want)
	}
	if got, want := schema.Required, []string{"procedure"}; !slices.Equal(got, want) {
		t.Errorf("run requires %q, want %q: every Run is a Run of a Procedure, and the other two are the occasion", got, want)
	}
}

// TestListTools_ProbeTakesTwoNamesAndAnInputsObject is §9's signature for the
// second Execution tool, held where it is holdable: the published schema
// (issue #201).
//
// **The closure is the claim and `inputs` is the exception it makes room for.**
// The argument object is closed over four properties, so the `definition`, the
// `target` and the `repo_dir` §9 spends paragraphs refusing are refused by there
// being no fifth. The fourth is `response`, and it names a file rather than a
// reach: a supplied response makes no call, so it widens nothing a Probe could
// otherwise touch (ADR-0108). What `inputs` admits is open, and it has to be: its keys are
// the Operation's, declared in a Manifest this schema has never read, so a
// closed object there would be this surface naming a shape it does not own
// (closedObject).
//
// What it is **not** open about is what a member may hold. Every §12 type is a
// scalar, so the three JSON scalars are named and nothing else is — an `object`
// and an `array` read as nothing at every position a hole fills (ADR-0078), and
// a member that is one of them names a value no input can carry.
func TestListTools_ProbeTakesTwoNamesAndAnInputsObject(t *testing.T) {
	schema := declared(t, probeTool.input)

	if !schema.closed() {
		t.Error("probe's input schema admits properties it does not state; a schema that admits a member it does not name is one under which an override argument is well-formed")
	}
	if got, want := schema.names(), []string{"inputs", "operation", "provider", "response"}; !slices.Equal(got, want) {
		t.Errorf("probe takes %q, want %q — the two positionals, the inputs and the supplied response, and no Definition and no Target", got, want)
	}
	if got, want := schema.Required, []string{"provider", "operation"}; !slices.Equal(got, want) {
		t.Errorf("probe requires %q, want %q: a Probe resolves two names, and an Operation taking no inputs is supplied none", got, want)
	}

	var inputs struct {
		Type          string `json:"type"`
		PropertyNames struct {
			MinLength *int   `json:"minLength"`
			Pattern   string `json:"pattern"`
		} `json:"propertyNames"`
		AdditionalProperties struct {
			Type []string `json:"type"`
		} `json:"additionalProperties"`
	}
	if err := json.Unmarshal(schema.Properties["inputs"], &inputs); err != nil {
		t.Fatal(err)
	}
	if inputs.Type != "object" {
		t.Errorf("inputs is declared %q, want an object keyed by input name", inputs.Type)
	}
	if got, want := inputs.AdditionalProperties.Type, []string{"string", "number", "boolean"}; !slices.Equal(got, want) {
		t.Errorf("an input admits %q, want %q: every type §12 declares is a scalar, and object and array read as nothing anywhere", got, want)
	}
	// **The key is declared too, and it has to be**: the server refuses a
	// name that is empty and one that carries an `=`, and a refusal the
	// schema does not state is this surface declining what it published as
	// well-formed (suppliedInputs).
	if inputs.PropertyNames.MinLength == nil || *inputs.PropertyNames.MinLength != 1 {
		t.Errorf("inputs declares propertyNames minLength %v, want 1: a key that is well-typed and names no input is a malformed call", inputs.PropertyNames.MinLength)
	}
	if got, want := inputs.PropertyNames.Pattern, `^[^=]*$`; got != want {
		t.Errorf("inputs declares propertyNames pattern %q, want %q: an input is spelled as one --input name=value pair, split at its first =", got, want)
	}
}

// TestCall_AnInputNameTheSchemaRefusesIsRefusedByTheServer is the other half of
// the pair above: a `propertyNames` is a claim a client may or may not check,
// and this is the reading that makes it true where it is enforceable (§9,
// readArguments).
//
// Both keys are **well-typed** — `inputs` is an object and its member is a
// string — so nothing but the key itself declines them, which is why the schema
// and the server are asserted together rather than either alone.
func TestCall_AnInputNameTheSchemaRefusesIsRefusedByTheServer(t *testing.T) {
	for _, called := range []struct{ name, arguments string }{
		{"a key that names no input", `{"provider":"uptime","operation":"check_http","inputs":{"":"status.hyper.dev"}}`},
		{"a key no --input pair can address", `{"provider":"uptime","operation":"check_http","inputs":{"a=b":"c"}}`},
	} {
		t.Run(called.name, func(t *testing.T) {
			server, argv := answering(nil, render.NewResultRow(false))

			if envelope, err := server.Call(t.Context(), "probe", json.RawMessage(called.arguments)); err == nil {
				t.Fatalf("the call answered %+v, want a protocol error", envelope)
			}
			if len(*argv) > 0 {
				t.Errorf("the tool built %q; a name the schema refuses reaches no command line", *argv)
			}
		})
	}
}

// TestListTools_ProbeAnswersOneRowAndNoOutcome is the other half of the tool:
// what a Probe returns, and the key it does not carry.
//
// **A Probe has no outcome triple to report**, having written no Record and no
// Journal entry (ADR-0009), so the output schema names the rows and the
// truncation marker and nothing beside them. The `outcome`, `run_id` and
// `dry_run` `run` declares are absent here by the schema being closed over what
// §9 names, which is the same closure that keeps a secret out of a Run's answer.
//
// **The two members whose keys are not this file's are declared open**, and that
// is closedObject's stated exception rather than a hole in it: a projection's
// keys are the Manifest's and a response's are the world's, and a closed object
// over either would be this surface stating a shape it does not own — wrongly,
// since the next Manifest projects a field it has never heard of.
func TestListTools_ProbeAnswersOneRowAndNoOutcome(t *testing.T) {
	schema := declared(t, probeTool.output)

	if !schema.closed() {
		t.Error("probe's output schema admits members it does not state")
	}
	if got, want := schema.names(), []string{"rows", "truncated"}; !slices.Equal(got, want) {
		t.Errorf("probe answers %q, want %q — a Probe is no Run, and carries no outcome key at all", got, want)
	}
}

// TestListTools_ProjectTakesNoArgumentsAtAll is §9's signature for the one
// Lifecycle tool, held where it is holdable: the published schema (issue #203).
//
// **The empty closed object is the whole claim.** `project` is repo-wide and
// all-or-nothing — per-Procedure projection would let two Procedures pin
// different versions against one Store — so there is nothing here for an
// argument to name, and the closure is what says so to a client rather than
// leaving `project({"procedure": "…"})` to be quietly ignored.
func TestListTools_ProjectTakesNoArgumentsAtAll(t *testing.T) {
	schema := declared(t, projectTool.input)

	if !schema.closed() {
		t.Error("project's input schema admits properties it does not state; a schema that admits a member it does not name is one under which an override argument is well-formed")
	}
	if got := schema.names(); len(got) != 0 {
		t.Errorf("project takes %q, want nothing: projection is repo-wide and all-or-nothing, and there is no per-Procedure projection for an argument to name", got)
	}
	if got := schema.Required; len(got) != 0 {
		t.Errorf("project requires %q, want nothing", got)
	}
}

// TestListTools_ProjectAnswersWorkflowRowsAndNoOutcome is the other half of the
// tool: what it answers, and the keys it does not carry (§9, §10, issue #203).
//
// **`project` is not a Run**, so the output schema names the rows and the
// truncation marker and nothing beside them — no `outcome`, no `run_id`, no
// `dry_run` — which the closure is what makes checkable rather than promised.
// It is the same shape `probe` answers, for a different reason: a Probe has no
// outcome to report because it wrote nothing, and this has none because writing
// a file is not a Run.
func TestListTools_ProjectAnswersWorkflowRowsAndNoOutcome(t *testing.T) {
	schema := declared(t, projectTool.output)

	if !schema.closed() {
		t.Error("project's output schema admits members it does not state")
	}
	if got, want := schema.names(), []string{"rows", "truncated"}; !slices.Equal(got, want) {
		t.Errorf("project answers %q, want %q — it is no Run, and carries no outcome key at all", got, want)
	}
}

// TestListTools_ACadenceCrossesAsTheGlossParts is §10's rule held over the two
// rows that carry a Cadence: **the parts, and never the
// composed phrase-and-rate line** (§8, §9, §10, ADR-0063, issue #203).
//
// Wherever a Cadence renders, the gloss renders with it and there is no surface
// exempt — so a rule that is total is one no consumer may hold a second copy
// of, and two schemas spelling the group apart would be that copy arriving on
// the wire. There is one fragment behind both (cadenceGlossMembers), and this is
// what says the two rows still reach it.
//
// **The composed line is what it refuses by name.** The page stacks the three
// under one another and a review's header joins them with `·`; how they are
// arranged is the surface's and what they are is not, so a member carrying an
// arrangement would be a page's layout landing in a machine contract.
func TestListTools_ACadenceCrossesAsTheGlossParts(t *testing.T) {
	for _, carrying := range []struct {
		row     string
		members map[string]json.RawMessage
	}{
		{"project's workflow row", rowMembers(t, projectTool.output, "workflow")},
		{"review's artefact row", rowMembers(t, reviewTool.output, "artefact")},
	} {
		t.Run(carrying.row, func(t *testing.T) {
			for _, part := range []string{"cadence", "phrase", "rate"} {
				if _, declares := carrying.members[part]; !declares {
					t.Errorf("%s declares no %s; the gloss crosses in its three parts wherever a Cadence does", carrying.row, part)
				}
			}
			for _, composed := range []string{"gloss", "cadence_gloss", "cadence_line", "rate_text"} {
				if _, declares := carrying.members[composed]; declares {
					t.Errorf("%s declares %s; how the parts are arranged is the surface's, and a composed line is a page's layout in a machine contract", carrying.row, composed)
				}
			}
		})
	}
}

// rowMembers is the properties of one row type inside a tool's output schema,
// found by the `type` const that discriminates it.
//
// It walks the schema rather than being handed a fragment, because what a fence
// reads should be **what a client receives**: a fragment asserted directly would
// agree with itself whether or not any row still composed it in.
func rowMembers(t *testing.T, output json.RawMessage, discriminator string) map[string]json.RawMessage {
	t.Helper()

	var schema struct {
		Properties struct {
			Rows struct {
				Items json.RawMessage `json:"items"`
			} `json:"rows"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(output, &schema); err != nil {
		t.Fatal(err)
	}

	// A row set is one object schema or a `oneOf` over several, which is the
	// difference between a tool answering one row type and a tool answering
	// the five a Journal entry read back carries.
	var alternatives struct {
		OneOf []json.RawMessage `json:"oneOf"`
	}
	if err := json.Unmarshal(schema.Properties.Rows.Items, &alternatives); err != nil {
		t.Fatal(err)
	}
	candidates := alternatives.OneOf
	if len(candidates) == 0 {
		candidates = []json.RawMessage{schema.Properties.Rows.Items}
	}

	for _, candidate := range candidates {
		var row struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(candidate, &row); err != nil {
			t.Fatal(err)
		}
		var named struct {
			Const string `json:"const"`
		}
		if err := json.Unmarshal(row.Properties["type"], &named); err != nil {
			continue
		}
		if named.Const == discriminator {
			return row.Properties
		}
	}
	t.Fatalf("no row in the schema is discriminated %q", discriminator)
	return nil
}

// TestListTools_RunsOutputSchemaStatesTheExecutionMembersAndNoSecret is the
// other half of the sink's guarantee, and the half a corpus case cannot state:
// **returning the secret in the tool result is not one of the sink's forms**.
//
// A golden can only say that no secret appeared in the one answer it drove. The
// schema says that none can: the structured half is closed over the members §9
// names, so there is no key for a generated credential to arrive under — which
// is what keeps it out of an agent's context and out of whatever transcript that
// agent writes (ADR-0007).
func TestListTools_RunsOutputSchemaStatesTheExecutionMembersAndNoSecret(t *testing.T) {
	schema := declared(t, runTool.output)

	if !schema.closed() {
		t.Error("run's output schema admits members it does not state; a member the schema does not name is a member this surface does not write")
	}
	if got, want := schema.names(), []string{"dry_run", "outcome", "rows", "run_id", "truncated"}; !slices.Equal(got, want) {
		t.Errorf("run answers %q, want %q — §12's triple, the marker, the entry, the rows, and nothing a secret could ride on", got, want)
	}
	if got, want := slices.Sorted(slices.Values(schema.Required)), []string{"dry_run", "outcome", "rows", "truncated"}; !slices.Equal(got, want) {
		t.Errorf("run requires %q, want %q: run_id is the one member absent where no entry was written", got, want)
	}
}

// yes and no are the two values of §7's one exception to the absence rule, as
// the envelope carries it: a pointer, because `false` and *this tool is not a
// Run* are two different answers (Structured.DryRun).
var (
	yes = pointerTo(true)
	no  = pointerTo(false)
)

// pointerTo is a value's address where the language has no literal for one. It
// is here because the third state a case has to be able to state is the
// **absence**, which a bare `false` cannot.
func pointerTo[T any](value T) *T { return &value }

// declaredSchema is one of this surface's schemas read for what it declares:
// whether it is closed, which names it requires, and which properties it
// admits.
//
// It is one type rather than the same anonymous struct at each reader, for the
// reason objectSchema above is one function: what a schema declares is asked
// three ways in this file, and three spellings of the read are three chances
// for one of them to check something slightly different.
type declaredSchema struct {
	AdditionalProperties *bool                      `json:"additionalProperties"`
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
}

// declared reads one schema this package publishes. It takes the raw bytes
// rather than the SDK's value because these are the tool table's own, checked
// in as the JSON a client receives (closedObject).
func declared(t *testing.T, schema json.RawMessage) declaredSchema {
	t.Helper()

	var read declaredSchema
	if err := json.Unmarshal(schema, &read); err != nil {
		t.Fatal(err)
	}
	return read
}

// closed answers objectSchema's question of a raw schema: `additionalProperties`
// stated and false, which is what makes *no tool takes an override argument of
// any kind, under any name* a thing a client can check rather than a promise.
func (s declaredSchema) closed() bool {
	return s.AdditionalProperties != nil && !*s.AdditionalProperties
}

// names is every property the schema admits, sorted — the signature, in the one
// order a comparison can be written against.
func (s declaredSchema) names() []string { return slices.Sorted(maps.Keys(s.Properties)) }

// spelled is a dry_run marker in an error message: `true`, `false`, or the
// absence, which is the third state and the one a bare `%v` on a pointer would
// print as an address.
func spelled(dryRun *bool) string {
	if dryRun == nil {
		return "absent"
	}
	return strconv.FormatBool(*dryRun)
}

// argumentsSatisfying is arguments enough for one tool to build its command
// line: every required property of its schema, filled with a name.
//
// It is derived from the schema rather than tabulated per tool, so that a tool
// landing tomorrow is held to the rule above without this file being edited.
func argumentsSatisfying(t *testing.T, declaring tool) json.RawMessage {
	t.Helper()

	schema := declared(t, declaring.input)
	filled := map[string]any{}
	for _, name := range schema.Required {
		var property struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(schema.Properties[name], &property); err != nil {
			continue
		}
		switch property.Type {
		case "array":
			filled[name] = []string{"definitions/uptime.yaml"}
		default:
			filled[name] = "uptime"
		}
	}
	arguments, err := json.Marshal(filled)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return arguments
}

// TestCall_NothingIsSentBetweenCalls is §9's *hyper never speaks first* held
// over the gap the single-call driver cannot reach: **two calls on one
// session**, with what arrived read off the wire in the order it arrived (§9,
// ADR-0021, ADR-0092, issue #202).
//
// The progress notifications above belong to the call that is in flight and
// stop when it does, so what this asserts is the shape of the whole recording:
// the boundaries of the first call, then the boundaries of the second, and
// **nothing at all in between or after** — no logging message, no
// server-initiated request, no notification outside a call in flight.
//
// The dispatch narrates two boundaries per call whether or not anybody is
// watching, so the silence between them is the surface's decision and not an
// absence of anything to say.
func TestCall_NothingIsSentBetweenCalls(t *testing.T) {
	narrating := NewServer("1.4.0", func(call Call) Answer {
		if call.Progress != nil {
			call.Progress(1, 2, "first")
			call.Progress(2, 2, "second")
		}
		return Answer{Terminal: render.NewResultRow(false), Rendering: reviewPage}
	})

	// The session Call and Tools are made over, stood here rather than
	// driven through either: what this case is about is **two calls sharing
	// one session**, which no door that makes one call can reach.
	client, arriving, done, err := narrating.paired(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	// Two calls, and the second under a token of its own: a notification
	// carrying the first call's token after the first call returned would be
	// the server still talking about a call that is over.
	for _, token := range []string{"watching-the-first", "watching-the-second"} {
		params := &sdk.CallToolParams{Name: "probe", Arguments: json.RawMessage(`{"provider":"uptime","operation":"check_http"}`)}
		params.SetProgressToken(token)
		if _, err := client.CallTool(t.Context(), params); err != nil {
			t.Fatalf("call under %s: %s", token, err)
		}
	}

	seen := arriving.watching()
	if len(seen.Unasked) != 0 {
		t.Errorf("the server sent %v unasked across two calls; it has no logging channel and initiates nothing", seen.Unasked)
	}
	// Four boundaries and not one more, in the two calls' own order: the
	// notifications a call sends stop when it does, so a fifth would be one
	// belonging to no call in flight.
	if len(seen.Progress) != 4 {
		t.Fatalf("two calls of two Steps each sent %d notifications, want 4: %+v", len(seen.Progress), seen.Progress)
	}
	for at, want := range []string{"watching-the-first", "watching-the-first", "watching-the-second", "watching-the-second"} {
		if got := seen.Progress[at].Token; got != want {
			t.Errorf("notification %d carries token %v, want %q — a notification belongs to the call in flight", at+1, got, want)
		}
	}
}
