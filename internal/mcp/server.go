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
// Rendering is the page the command wrote into the destination's buffer. This
// ticket's two tools do not put it on the wire — an ordinary return's text
// block is one summary line (§9) — and it is carried anyway because it is what
// `review`'s full rendering and a Refusal's are, and a buffer that only some
// answers filled would be one a later tool had to ask twice for.
type Answer struct {
	Rows      []render.Row
	Terminal  render.Row
	Rendering string
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
// A command that did not exit clean is the one path this ticket leaves
// unshaped, and half of it is already where §9 puts it: a usage error — a name
// that satisfies every schema and resolves to nothing — is a malformed call
// there too, and arrives as the protocol error the CLI spends exit `2` to draw.
// What is not yet shaped is the **Refusal**, which §9 answers with `isError:
// true` and the rendering in full, exactly where the command exits `77`; issue
// #196 builds it, and until then such an exit is reported as a fault in the
// server rather than dressed as a domain answer this ticket has not built. A
// wrong envelope is harder to notice than a missing one.
func (s *Server) handler(t tool) sdk.ToolHandler {
	return func(_ context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		argv, err := t.argv(request.Params.Arguments)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}

		answered := s.dispatch(argv)
		if answered.Exit != 0 {
			return nil, fmt.Errorf("%s: the command exited %d; the paths that decline are issue #196's", t.name, answered.Exit)
		}

		envelope, err := envelopeOf(answered)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
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
//
// The **last** result is the tool call's: a session sends one response to
// `initialize` and one to the `tools/call` that follows it, and the
// notifications between them carry no result at all. Call makes exactly one
// tool call, so there is no third.
func (f *frames) envelope() (Envelope, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var answered json.RawMessage
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
		}
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			return Envelope{}, fmt.Errorf("a frame the client read is not JSON: %w", err)
		}
		if len(frame.Result) > 0 {
			answered = frame.Result
		}
	}
	if answered == nil {
		return Envelope{}, errors.New("no frame the client read carries a result")
	}
	return envelopeFrom(answered)
}

// readCloser is a reader and a closer that are not the same value: the tee
// above reads, and the connection beneath it closes.
type readCloser struct {
	io.Reader
	io.Closer
}
