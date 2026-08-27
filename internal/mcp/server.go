// Package mcp is hyper's second surface: the MCP server §9 states, started by
// `hyper mcp` and speaking JSON-RPC over the stdio the client handed the
// process (§9, ADR-0088, issue #195).
//
// **It is the one package that imports the SDK, and that is the whole of why it
// exists.** The SDK owns the transport, the handshake, `tools/list` and its
// paging, the notification plumbing and the JSON-RPC framing — every part of
// this surface that is the protocol's rather than hyper's. What it may never
// decide is what an answer *says*: it does not compose the text block, set
// `isError`, shape `structuredContent` or validate an output. No `render`,
// `run`, `store` or `cli` type is expressed in its terms — nothing here hands
// it a domain value to infer a schema from or to marshal on hyper's behalf — so
// the day it is replaced is a day one package changes.
//
// **What crosses the boundary is a Call and an Answer.** A tool declares
// its arguments, builds the command line its command would have received, and
// hands it to the dispatch it was constructed with; the dispatch runs the
// command behind hyper's own destination and answers the rows. A tool holds no
// command logic, so there is no second place for a guardrail to be skipped, a
// Refusal to be reworded or a row to be reshaped: *ergonomics is the whole of
// the difference between the two* (§9), and making it the whole of the
// difference in the code is what keeps it true.
//
// The Call is the argv and the two facts about the call itself that an argv
// cannot carry, both of them the protocol's rather than hyper's: the context
// the call is alive for, which the SDK cancels when the client cancels the
// request, and where a Step boundary goes, which is a notification where the
// client attached a progress token and nothing anywhere else (§9, ADR-0021,
// ADR-0092, issue #202).
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// name is what the server calls itself at `initialize`, and it is the binary's
// own name because the server *is* the same binary: one process per client,
// started by `hyper mcp`, dying with it (§9, ADR-0088).
const name = "hyper"

// Dispatch runs one hyper command line and answers what it produced. It is a
// function rather than an interface with one method because that is the whole
// of what this package needs of the tool behind it: the Call goes in, the rows
// come out, and nothing here knows which command ran or what it read.
//
// It is what keeps the import one-directional. The dispatch lives in
// internal/cli, where the commands and their destination are; this package is
// handed one and never reaches for it, so the SDK stays on this side of the
// boundary and the command surface stays on the other.
type Dispatch func(call Call) Answer

// Call is one tool call as the dispatch behind it receives it: the
// command line the tool built, the context the call is alive for, and where the
// Steps of a Run go as it performs them.
//
// It is one value rather than three parameters because two of the three are
// facts about the **call** rather than about the command line, and the day a
// third such fact lands is a day no signature moves. Neither is expressed in
// the SDK's terms: what crosses is a context and a function, so the package on
// the other side still cannot name a frame, a session or a notification (§9).
type Call struct {
	// Context is the handler's own, and the whole of what this surface has
	// in place of a signal. The client cancels a call by cancelling the
	// request; the SDK cancels this; and the drain §6 states reads it where
	// the next Step would start — the Step in flight finishes, no further
	// Step starts, and the Run closes its own entry `failed` (§6, §9,
	// ADR-0015, ADR-0092).
	//
	// It is never the parent of anything a command performs. A Step whose
	// context was this one would be a Step that stopped mid-call, which is
	// the ambiguity the drain exists to avoid.
	Context context.Context
	// Argv is the command line the tool built, exactly as the command would
	// have received it on the terminal.
	Argv []string
	// Progress is one Step boundary reaching the client, and it is **nil
	// where the client supplied no progress token** (Progress below). A
	// dispatch handed nil is one whose Run nobody is watching, which is a
	// state the engine already has a reading of.
	Progress Progress
}

// Progress is where a Run's Step boundaries go on this surface: the Step's
// position, how many the Run holds, and its authored id — the three §9's
// narration carries, at the same boundary §7 writes a Journal entry at.
//
// It is narration, so it carries no machine contract and no row of its own: a
// caller reads what happened off the envelope when the call returns, and this
// is what stands between now and then on a surface with no scrollback.
//
// **It is nil where the client supplied no progress token**, and that is the
// rule this type exists to carry rather than a detail of who builds one. A
// progress notification exists to be correlated with the request that is
// proceeding, so without a token there is nothing to attach one to and a server
// that sent one anyway would be speaking unasked, which is the one thing this
// server never does (§9, ADR-0021, ADR-0092). Nil rather than a function that
// drops what it is handed is what puts the decision in one place: the surface
// decides whether anybody is watching, and the Run behind it narrates or does
// not.
//
// The Run naming itself before its first Step is the other event §9's narration
// carries, and this type has no member for it: it sends nothing here (ADR-0047,
// ADR-0092).
type Progress func(position, of int, step string)

// Answer is one command's answer as a value: the rows, the terminal row that
// ends them, the rendering the command's page produced, and the exit code it
// returned.
//
// It is the destination's retained state crossing the boundary unflattened —
// the rows as **values** and not as a stream of bytes — which is the property
// milestone 11 was prefactored for: the rows a page is written from and the
// rows an envelope carries are one list (destination.go, ADR-0026, issue #194).
//
// Rendering is the page the command wrote into the destination's buffer. Most
// tools do not put it on the wire — an ordinary return's text block is one
// summary line (§9) — and it is filled on every answer anyway, because it is
// what `review`'s text block *is*: the second row of §9's asymmetric table is a
// tool handing back the whole rendering, and a buffer that only some answers
// filled would be one that tool had to ask twice for (envelope.go, issue #198).
//
// **Refusal and Narration are what a surface with no exit code needs the exit
// code's two other halves for** (§9, issue #196). A command that Refuses
// renders §8's Refusal where the CLI would have written it on stderr, and that
// rendering *is* the text block on this surface, exactly where the command
// exits `77`. A command that reports a usage error writes one human sentence
// where the CLI would have written it on stderr, and that sentence is the
// message of the protocol error a malformed call comes back as — so an agent
// reads the sentence a person would have read. Neither is narration this
// surface forwards: what is not read by the mapping is dropped, a tool's
// narration going nowhere (destination.go).
type Answer struct {
	Rows      []render.Row
	Terminal  render.Row
	Rendering string
	Refusal   string
	Narration string
	Exit      int
}

// A Server is the MCP server over one dispatch: the tool set, the version it
// announces itself at, and the way in to both.
//
// It is constructed once and reached two ways — [Server.Serve] over the
// process's stdio, and [Server.Call] over the SDK's in-memory transports — so
// that what the corpus drives and what a client starts are one server with one
// tool set. A second assembly for the tests would be a corpus asserting a
// server nobody runs.
type Server struct {
	// version is the binary's own, and it is announced at `initialize` as
	// the server's: the version of the binary that would act is the version
	// of the server the client started, which is why the three commands
	// outside the tree get no tool of their own (§9). It is **the same
	// string the pin gate compares**, threaded from the same version.Facts
	// the dispatch behind it gates on.
	version  string
	dispatch Dispatch
}

// NewServer is the server over one dispatch, at one version.
//
// The tool set is not a parameter. It is this package's own table — a tool is a
// schema and an argv (tools.go) — and a server assembled with a different one
// would be a client's view of hyper that hyper never stated.
func NewServer(version string, dispatch Dispatch) *Server {
	return &Server{version: version, dispatch: dispatch}
}

// Serve runs the server on the process's stdin and stdout until the client goes
// away, and answers whatever ended it.
//
// **Nothing here holds a writer onto stdout.** The transport does, and it
// reaches os.Stdout itself: this is the one place §9's *stdout is the answer* is
// not true of the process — a stray write would corrupt a frame rather than
// merely mislead a reader — and what makes that structural is that there is
// nothing for a command to write to even by mistake. The destination behind
// every tool retains rows, a buffer and io.Discard, and no stream at all
// (destination.go, issue #194).
//
// It listens on no port and outlives nothing. `Server.Run` returns when the
// session ends, which is when the client that started this process closes its
// end — *one process per client, dying with it* (§9, §13, ADR-0088).
func (s *Server) Serve(ctx context.Context) error {
	return s.server().Run(ctx, &sdk.StdioTransport{})
}

// server is the SDK server with hyper's tools registered on it, and it is built
// where it is used rather than held on the value: the two ways in are two
// sessions, and one SDK value shared between them would be shared state this
// package neither wants nor can see into.
//
// **The tools are registered through the low-level Server.AddTool**, not the
// generic top-level AddTool, and the reason is a distinction §9 rests on. The
// generic form validates arguments for you and, on a violation, returns a
// CallToolResult with IsError: true — a domain answer. §9 requires the
// opposite: *an argument violating a schema* is a malformed call and therefore
// **a protocol error**, JSON-RPC errors being reserved for malformed calls and
// a Refusal being an answer to a well-formed one. The low-level form hands the
// handler raw arguments and treats a returned error as a protocol error, which
// is the contract this surface needs.
//
// The cost is hand-written schemas and hand-written unmarshalling, both of
// which are wanted anyway: §9's arguments are closed sets and enums that
// inference over a Go type would widen, and *an `outputSchema` is declared once
// and for every call of the tool*.
//
// **Tools only: no resources and no prompts** (§9). The capabilities are stated
// rather than inferred, because what the SDK infers is not what §9 says: its
// default is a `logging` capability hyper does not have — *hyper never speaks
// first*, and it *has no logging channel* (ADR-0021) — and its inferred `tools`
// carries `listChanged`, which promises a notification for a tool set that is
// fixed at startup and never changes. Naming the capability with nothing beside
// it is the whole of what this server advertises.
func (s *Server) server() *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{Name: name, Version: s.version},
		&sdk.ServerOptions{Capabilities: &sdk.ServerCapabilities{Tools: &sdk.ToolCapabilities{}}},
	)
	for _, t := range tools {
		server.AddTool(t.declaration(), s.handler(t))
	}
	return server
}

// handler is one tool's handler: read the arguments, build the argv, run the
// command, shape the envelope.
//
// Every step that can fail before the command runs answers an **error**, which
// the low-level registration above turns into a JSON-RPC error: an argument
// that will not unmarshal or that the schema does not admit is a malformed
// call, and §9 reserves the protocol's errors for exactly those (§9, ADR-0060).
//
// **What the command exited with is read in one place and not here**: the
// mapping §9 fixes from an exit code to an envelope is envelopeOf's, stated
// once and reached by every tool through this one handler (envelope.go, issue
// #196). Two of its arms answer an error rather than an envelope — the usage
// error a malformed call arrives as, and a code the mapping does not hold —
// and both travel the way an argument error travels, because both are the same
// thing to the protocol.
//
// The one thing about the tool the mapping is told is which row of §9's
// text-block table it is: *any ordinary return* or **`review`**. That is a
// property of the tool rather than of the answer, so it crosses from the table
// where the tool is declared rather than being read off a rendering here
// (tools.go, issue #198).
//
// The error is answered unwrapped, which is the one asymmetry in this function.
// An argument never reached a command, so the tool names itself; a message the
// command wrote is the sentence a person would have read, and a tool name
// pasted in front of it would be this surface editorialising over prose that is
// already addressed to a caller (§9, ADR-0026).
func (s *Server) handler(t tool) sdk.ToolHandler {
	return func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		argv, err := t.argv(request.Params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}

		// What the tool can say about the call before it runs, and it is
		// one tool's: `run` carries §12's triple, and the envelope needs
		// the rehearsal marker beside it on the one path where the
		// command writes no row carrying its own (tools.go, executionOf).
		//
		// **A tool with no execution half sends nil**, which is how the
		// envelope is told that this answer carries no `outcome` key at
		// all: the table is where a tool declares it, and this is that
		// declaration crossing rather than a second copy of it.
		var called *execution
		if t.executes != nil {
			carried, err := t.executes(request.Params.Arguments)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", t.name, err)
			}
			called = &carried
		}

		answered := s.dispatch(Call{
			Context:  ctx,
			Argv:     argv,
			Progress: progressTo(ctx, request),
		})
		envelope, err := envelopeOf(answered, t.rendersInFull, called)
		if err != nil {
			return nil, err
		}
		return envelope.result(), nil
	}
}

// progressTo is this call's Progress: the token read off the request, and
// **nil where the client supplied none** — the rule Progress states, made true
// at the one place a token can be read (§9, ADR-0021).
//
// **What a failed send does is nothing.** A notification the transport could not
// carry is narration that did not arrive, and narration carries no machine
// contract: a Run that stopped because its progress line could not be written
// would be a Run whose effects turned on whether anybody was reading. The
// answer the caller gets is the envelope, and where the client has gone away it
// gets no delivery at all — which is §6's account and not this function's to
// improve on (§9).
func progressTo(ctx context.Context, request *sdk.CallToolRequest) Progress {
	token := request.Params.GetProgressToken()
	if token == nil {
		return nil
	}
	return func(position, of int, step string) {
		// The error is dropped, which is the paragraph above as one line:
		// narration that did not arrive is narration.
		_ = request.Session.NotifyProgress(ctx, &sdk.ProgressNotificationParams{
			ProgressToken: token,
			Progress:      float64(position),
			Total:         float64(of),
			Message:       step,
		})
	}
}

// Call drives one tool call against this server and answers the envelope that
// came back, read off the wire.
//
// **The call is real; only the client is in-process** — the principle
// golden_serve_test.go already states for the TLS fixture. The transports are a
// net.Pipe, so the handshake, the framing, the `tools/call` round trip and the
// JSON of every row are the wire's; what is a fixture is that the client is a
// goroutine rather than an editor.
//
// **The envelope is read from the recorded frame and not from the client's
// decoded value**, and that is not a shortcut but the only reading that can be
// checked. §8 fixes that a row's `type` is its first key and the rest follow
// declaration order; the SDK decodes `structuredContent` into `any`, which is a
// map, and a map re-encoded has its keys in whatever order the encoder chooses.
// A corpus holding that would be holding the harness's ordering rather than
// hyper's. So the client-side stream is teed as it is read and the answer is
// taken from the bytes that arrived.
//
// It is the door the golden corpus drives a `call` case through, and it is here
// rather than in the harness for the reason the tool set is not a parameter: a
// harness that stood its own client would be one importing the SDK, and the
// SDK is reachable from this package and no other.
//
// **It attaches no progress token**, which is what makes it the door that also
// says what a call with none gets: nothing but its envelope. A driver that
// wants the notifications supplies a token through Watched below.
func (s *Server) Call(ctx context.Context, tool string, arguments json.RawMessage) (Envelope, error) {
	envelope, _, err := s.Watched(ctx, tool, arguments, nil)
	return envelope, err
}

// Watched is Call under a progress token, and it answers **everything the
// client saw** beside the envelope: the progress notifications the server sent
// during the call, and the method of anything else it sent unasked (§9,
// ADR-0021, issue #202).
//
// token is the progress token this call carries, and **nil is a call that
// supplies none** — which is Call above, and the reason the two are one
// function: *a notification is sent where the client supplied a token and
// nowhere else* is a claim about the difference between two calls, and a driver
// that reached one of them through a second assembly of the server would be
// comparing two servers rather than two calls.
//
// What arrives is read off the recorded frames for the reason the envelope is:
// what a driver holds should be the messages hyper sent, in the order they
// arrived on the wire, rather than a client-side handler's account of them.
func (s *Server) Watched(ctx context.Context, tool string, arguments json.RawMessage, token any) (Envelope, Watching, error) {
	client, arriving, done, err := s.paired(ctx)
	if err != nil {
		return Envelope{}, Watching{}, err
	}
	defer done()

	params := &sdk.CallToolParams{Name: tool, Arguments: arguments}
	if token != nil {
		params.SetProgressToken(token)
	}
	if _, err := client.CallTool(ctx, params); err != nil {
		// **The protocol error is read off the frame too**, for the reason
		// the envelope is: what the client raises is the wire's message
		// wrapped in a sentence of its own — `calling "tools/call": …` —
		// and what a case holds should be the message hyper sent. The
		// client's own error stands only where no frame carries one at
		// all, which is a transport that failed before an answer arrived
		// (issue #196).
		if failed := arriving.failure(); failed != nil {
			return Envelope{}, arriving.watching(), failed
		}
		return Envelope{}, arriving.watching(), err
	}
	envelope, err := arriving.envelope()
	return envelope, arriving.watching(), err
}

// Tools is `tools/list` as a client receives it: the thirteen §9 states, each
// carrying the two schemas it publishes, read off the wire (§9, issue #204).
//
// It is the third door onto this server and it exists for the reason the first
// two do — the SDK is reachable from this package and no other, so a driver
// that wanted the listing had nowhere to stand. What it is for is the corpus's
// schema golden: **a schema is the contract an agent writes its calls
// against**, and a schema that drifts between two releases is the one way this
// surface can break a caller without any answer changing
// (internal/cli/mcp_tools_test.go).
//
// **The listing is read off the recorded frame**, for the reason Call's
// envelope is and with more riding on it: a schema's keys are its author's
// order, and the SDK decodes a published schema into `any`, which is a map. A
// golden holding a re-encoded map would hold the encoder's ordering and would
// change under a Go release rather than under an edit to this package.
//
// **The order is the wire's and not the table's.** The SDK holds its tools in a
// set keyed by name and pages them in name order, so what a client receives is
// the thirteen alphabetically; §9's group order is the table's own, and the
// fence that holds it holds it there (tools.go, tool_set_test.go).
func (s *Server) Tools(ctx context.Context) ([]Declared, error) {
	client, arriving, done, err := s.paired(ctx)
	if err != nil {
		return nil, err
	}
	defer done()

	if _, err := client.ListTools(ctx, nil); err != nil {
		if failed := arriving.failure(); failed != nil {
			return nil, failed
		}
		return nil, err
	}
	return arriving.listed()
}

// Declared is one tool as `tools/list` publishes it: what it is called, what a
// client is told it is for, and the two schemas.
//
// The schemas stay raw, which is the whole point of the type. What a caller
// holds is the bytes the server sent, keys in the order this package wrote
// them, rather than a decoded shape re-encoded on the way back out.
type Declared struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Input       json.RawMessage `json:"inputSchema"`
	Output      json.RawMessage `json:"outputSchema"`
}

// paired is a client session against this server over a net.Pipe, with
// everything the client reads recorded on the way past, and the one way to end
// both sessions.
//
// It is one function rather than the same fifteen lines at each door because
// the ordering below is load-bearing and is the kind of thing a second copy
// gets subtly wrong.
//
// **The server session is closed last and it waits**, which is what makes a
// cancelled call assertable at all: the handler is still draining its Run when
// the client has gone, and a driver that read the Store branch before the
// closer returned would be reading it mid-Run (mcp_cancelled_test.go).
func (s *Server) paired(ctx context.Context) (client *sdk.ClientSession, arriving *frames, done func(), err error) {
	serverSide, clientSide := net.Pipe()
	arriving = &frames{}

	session, err := s.server().Connect(ctx, &sdk.IOTransport{Reader: serverSide, Writer: serverSide}, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	client, err = sdk.NewClient(&sdk.Implementation{Name: "hyper-in-process-client", Version: s.version}, nil).
		Connect(ctx, &sdk.IOTransport{Reader: arriving.tee(clientSide), Writer: clientSide}, nil)
	if err != nil {
		session.Close()
		return nil, nil, nil, err
	}
	return client, arriving, func() { client.Close(); session.Close() }, nil
}

// Watching is what a client saw beside one call's envelope.
//
// The two members are §9's two claims about this server's own voice, in a shape
// a driver can hold: what it sent because it was asked to, and what it sent
// unasked — which is nothing, on every path, always (ADR-0021).
type Watching struct {
	// Progress is one member per `notifications/progress`, in arrival order.
	Progress []Boundary
	// Unasked is the method of every other message the server sent: a
	// logging notification, a server-initiated request, anything at all. It
	// is empty on every path — *hyper never speaks first* — and it is
	// collected rather than assumed so that a driver holds the claim rather
	// than restating it (§9, ADR-0021).
	Unasked []string
}

// Boundary is one progress notification as it arrived: the token it was
// correlated with, the Step's position, how many the Run holds, and the Step's
// authored id.
//
// It is this package's own shape and not the SDK's, for the reason every value
// crossing the boundary is: a driver outside this package holds what hyper
// said, not the type the transport said it in.
type Boundary struct {
	Token    any
	Position int
	Of       int
	Step     string
}

// frames records the newline-delimited JSON the client read, so that the
// envelope can be taken from the bytes that arrived rather than from a decoded
// value (Call above).
//
// It is guarded because the SDK reads on a goroutine of its own: the recording
// is written there and read here, after the call it belongs to has returned.
type frames struct {
	mu       sync.Mutex
	recorded bytes.Buffer
}

// tee wraps the client's end of the pipe so that everything the client reads is
// recorded on the way past. The pipe is closed through the wrapper, the SDK
// closing its transport's reader when the session ends.
func (f *frames) tee(conn net.Conn) io.ReadCloser {
	return readCloser{Reader: io.TeeReader(conn, f), Closer: conn}
}

// Write is the tee's sink, and it is where the mutex is taken.
func (f *frames) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.recorded.Write(p)
}

// envelope is the last answer among the recorded frames, read into hyper's own
// shape.
func (f *frames) envelope() (Envelope, error) {
	answered, _, err := f.answered()
	switch {
	case err != nil:
		return Envelope{}, err
	case answered == nil:
		return Envelope{}, errors.New("no frame the client read carries a result")
	}
	return envelopeFrom(answered)
}

// listed is the tool listing among the recorded frames, read into this
// package's own shape with every schema's bytes as they arrived.
//
// **A cursor is a fault rather than a page to follow.** §9 fixes the set at
// thirteen and the SDK's page is far larger, so a listing that arrived in two
// halves is this surface answering something other than its own table — and a
// caller silently following the cursor would turn that into a golden nobody
// reads twice (§9, tools.go).
func (f *frames) listed() ([]Declared, error) {
	answered, _, err := f.answered()
	switch {
	case err != nil:
		return nil, err
	case answered == nil:
		return nil, errors.New("no frame the client read carries a result")
	}

	var listing struct {
		Tools      []Declared `json:"tools"`
		NextCursor string     `json:"nextCursor"`
	}
	if err := json.Unmarshal(answered, &listing); err != nil {
		return nil, fmt.Errorf("the tool listing the client read is not one: %w", err)
	}
	if listing.NextCursor != "" {
		return nil, fmt.Errorf("tools/list answered %d tools and a cursor; the set is fixed at startup and arrives whole", len(listing.Tools))
	}
	return listing.Tools, nil
}

// failure is the message of the last JSON-RPC error among the recorded frames,
// and nil where no frame carries one. It is the half of a call that has no
// envelope at all: a malformed call, and a fault in the server (§9).
//
// The message alone is read, and not the code beside it. What a client shows a
// person is the message — hyper's own sentence, composed by the command that
// declined — where the code is the SDK's mapping of a handler error onto
// JSON-RPC's own set and is not a number this surface chooses.
func (f *frames) failure() error {
	_, failed, err := f.answered()
	if err != nil || failed == nil {
		return nil
	}
	return errors.New(failed.Message)
}

// answered is the last result and the last error among the recorded frames.
//
// The **last** of either is the tool call's: a session sends one response to
// `initialize` and one to the `tools/call` that follows it, and the
// notifications between them carry neither. Call makes exactly one tool call,
// so there is no third.
func (f *frames) answered() (json.RawMessage, *wireError, error) {
	var result json.RawMessage
	var failed *wireError
	for _, line := range f.complete() {
		var frame struct {
			Result json.RawMessage `json:"result"`
			Error  *wireError      `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			return nil, nil, fmt.Errorf("a frame the client read is not JSON: %w", err)
		}
		if len(frame.Result) > 0 {
			result = frame.Result
		}
		if frame.Error != nil {
			failed = frame.Error
		}
	}
	return result, failed, nil
}

// complete is the recording as the frames that are whole, which is the one
// reading of the buffer both questions above are asked through.
//
// SplitAfter rather than Split, so that a **complete** line is the test: the
// transport is newline-delimited JSON, and what the recording ends in where the
// SDK's reader stopped mid-frame is a fragment rather than a frame. Reading one
// as JSON would report a fault in the server for what is only a buffer
// boundary.
//
// It takes the mutex, the recording being written on the SDK's own reading
// goroutine, and answers a slice rather than yielding: a caller holding the
// lock while it decodes is a caller the reader waits behind.
func (f *frames) complete() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	var whole []string
	for _, line := range strings.SplitAfter(f.recorded.String(), "\n") {
		if !strings.HasSuffix(line, "\n") || strings.TrimSpace(line) == "" {
			continue
		}
		whole = append(whole, line)
	}
	return whole
}

// watching is every message the server sent of its own accord among the
// recorded frames: the progress notifications, and the method of anything else.
//
// **A frame carrying a `method` is the server speaking**, which is the whole of
// the test. A response carries an id and a result or an error and never a
// method, so the split needs nothing about the session's state: what is left
// after the two responses of a `Call` is exactly what this surface sent because
// it was asked to, and what §9 says is never there at all (ADR-0021).
//
// The `initialize` handshake sends no notification either way, so a driver that
// found one here found it because a tool call produced it.
func (f *frames) watching() Watching {
	var seen Watching
	for _, line := range f.complete() {
		var frame struct {
			Method string `json:"method"`
			Params struct {
				ProgressToken any     `json:"progressToken"`
				Progress      float64 `json:"progress"`
				Total         float64 `json:"total"`
				Message       string  `json:"message"`
			} `json:"params"`
		}
		if err := json.Unmarshal([]byte(line), &frame); err != nil || frame.Method == "" {
			// A frame that will not read is answered by the two
			// readings above, which say so as an error. This one
			// reports what arrived and never why a frame did not.
			continue
		}
		if frame.Method != notificationProgress {
			seen.Unasked = append(seen.Unasked, frame.Method)
			continue
		}
		seen.Progress = append(seen.Progress, Boundary{
			Token:    frame.Params.ProgressToken,
			Position: int(frame.Params.Progress),
			Of:       int(frame.Params.Total),
			Step:     frame.Params.Message,
		})
	}
	return seen
}

// notificationProgress is the one method this server ever sends, spelled here
// rather than reached for out of the SDK: what the recording is read against is
// the wire's own name for the message, and a constant taken from the library
// that composed it would agree with whatever the library happened to send.
const notificationProgress = "notifications/progress"

// wireError is a JSON-RPC error object as it arrived, read for the one member
// a caller of this surface acts on.
type wireError struct {
	Message string `json:"message"`
}

// readCloser is a reader and a closer that are not the same value: the tee
// above reads, and the connection beneath it closes.
type readCloser struct {
	io.Reader
	io.Closer
}
