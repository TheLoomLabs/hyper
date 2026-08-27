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
// **What crosses the boundary is an argv and an Answer.** A tool declares its
// arguments, builds the command line its command would have received, and hands
// it to the dispatch it was constructed with; the dispatch runs the command
// behind hyper's own destination and answers the rows. A tool holds no command
// logic, so there is no second place for a guardrail to be skipped, a Refusal
// to be reworded or a row to be reshaped: *ergonomics is the whole of the
// difference between the two* (§9), and making it the whole of the difference
// in the code is what keeps it true.
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
// of what this package needs of the tool behind it: the argv goes in, the rows
// come out, and nothing here knows which command ran or what it read.
//
// It is what keeps the import one-directional. The dispatch lives in
// internal/cli, where the commands and their destination are; this package is
// handed one and never reaches for it, so the SDK stays on this side of the
// boundary and the command surface stays on the other.
type Dispatch func(argv []string) Answer

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
	return func(_ context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
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

		envelope, err := envelopeOf(s.dispatch(argv), t.rendersInFull, called)
		if err != nil {
			return nil, err
		}
		return envelope.result(), nil
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
func (s *Server) Call(ctx context.Context, tool string, arguments json.RawMessage) (Envelope, error) {
	serverSide, clientSide := net.Pipe()
	arriving := &frames{}

	session, err := s.server().Connect(ctx, &sdk.IOTransport{Reader: serverSide, Writer: serverSide}, nil)
	if err != nil {
		return Envelope{}, err
	}
	defer session.Close()

	client, err := sdk.NewClient(&sdk.Implementation{Name: "hyper-in-process-client", Version: s.version}, nil).
		Connect(ctx, &sdk.IOTransport{Reader: arriving.tee(clientSide), Writer: clientSide}, nil)
	if err != nil {
		return Envelope{}, err
	}
	defer client.Close()

	if _, err := client.CallTool(ctx, &sdk.CallToolParams{Name: tool, Arguments: arguments}); err != nil {
		// **The protocol error is read off the frame too**, for the reason
		// the envelope is: what the client raises is the wire's message
		// wrapped in a sentence of its own — `calling "tools/call": …` —
		// and what a case holds should be the message hyper sent. The
		// client's own error stands only where no frame carries one at
		// all, which is a transport that failed before an answer arrived
		// (issue #196).
		if failed := arriving.failure(); failed != nil {
			return Envelope{}, failed
		}
		return Envelope{}, err
	}
	return arriving.envelope()
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
	f.mu.Lock()
	defer f.mu.Unlock()

	var result json.RawMessage
	var failed *wireError
	// SplitAfter rather than Split, so that a **complete** line is the test:
	// the transport is newline-delimited JSON, and what the recording ends in
	// where the SDK's reader stopped mid-frame is a fragment rather than a
	// frame. Reading one as JSON would report a fault in the server for what
	// is only a buffer boundary.
	for _, line := range strings.SplitAfter(f.recorded.String(), "\n") {
		if !strings.HasSuffix(line, "\n") || strings.TrimSpace(line) == "" {
			continue
		}
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
