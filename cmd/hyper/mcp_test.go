package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// `hyper mcp` against the real binary: a process started the way a client
// starts one, spoken to over its own stdin and stdout (§9, ADR-0088, issue
// #195).
//
// **These cases exist because the transport is the one part of this surface
// nothing else can reach.** internal/mcp drives the server over in-memory
// transports and the golden corpus drives a tool call the same way; both stop
// short of the two claims a client actually depends on — that the process
// speaks the protocol on the streams it was handed, and that it dies when the
// client goes away. Neither is a fact about a package, so neither is assertable
// from one.
//
// **Nothing here imports the SDK.** The frames are newline-delimited JSON
// written by hand, which is what the transport carries, so what these cases
// hold is the wire rather than a client library's reading of it — and the SDK
// stays reachable from internal/mcp and no other package (§9's tool set is one
// dependency's worth of surface, and this file is not the second).

// protocolVersion is the version of MCP a client asks for at `initialize`. It
// is written down rather than derived because that is what a client does: a
// version this server does not speak is a fact about the handshake, and one
// taken from the server would make the handshake agree with itself.
const protocolVersion = "2025-06-18"

// TestMCP_TheProcessSpeaksTheProtocolOnItsOwnStreamsAndDiesWithTheClient is the
// whole invocation, end to end: the client starts the binary, shakes hands,
// lists the tools, calls one, and closes its end.
//
// It is one case rather than five because it is one session, and a session is
// what the claims are about: a server that announced itself and could not
// answer a call, or one that answered and would not die, is a server that
// passed four cases and works for nobody.
func TestMCP_TheProcessSpeaksTheProtocolOnItsOwnStreamsAndDiesWithTheClient(t *testing.T) {
	binary := build(t, "1.4.0")

	// A repository the process is standing in, named the one way a tool can
	// be told about one: the environment the client started it in. **No tool
	// takes an argument naming a repository** (§9), and `hyper mcp` takes no
	// arguments at all, so this variable and the working directory are the
	// whole of what fixes which repository the session acts on.
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "hyper.yaml"), "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")

	client := start(t, binary, repo)

	announced := client.call(t, 1, "initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "hyper-cmd-test", "version": "0"},
	})
	client.notify(t, "notifications/initialized")

	var handshake struct {
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		Capabilities map[string]json.RawMessage `json:"capabilities"`
	}
	decode(t, announced, &handshake)

	// The server is `hyper` at the version of the binary that would act,
	// which is the same string the pin gate compares (§9).
	if handshake.ServerInfo.Name != "hyper" {
		t.Errorf("the server announces %q, want hyper", handshake.ServerInfo.Name)
	}
	if handshake.ServerInfo.Version != "1.4.0" {
		t.Errorf("the server announces version %q, want the version the build stamped", handshake.ServerInfo.Version)
	}
	// Tools only: no resources and no prompts, and no logging channel either
	// — hyper never speaks first (§9, ADR-0021).
	if got := advertised(handshake.Capabilities); !slices.Equal(got, []string{"tools"}) {
		t.Errorf("the server advertises %q, want tools and nothing else", got)
	}

	var listed struct {
		Tools []struct {
			Name         string          `json:"name"`
			InputSchema  json.RawMessage `json:"inputSchema"`
			OutputSchema json.RawMessage `json:"outputSchema"`
		} `json:"tools"`
	}
	decode(t, client.call(t, 2, "tools/list", map[string]any{}), &listed)

	published := map[string]bool{}
	for _, tool := range listed.Tools {
		published[tool.Name] = true
		if len(tool.InputSchema) == 0 || len(tool.OutputSchema) == 0 {
			t.Errorf("%s is listed with input schema %s and output schema %s; a tool publishes both", tool.Name, tool.InputSchema, tool.OutputSchema)
		}
	}
	for _, want := range []string{"providers", "provider", "operation", "targets"} {
		if !published[want] {
			t.Errorf("tools/list does not answer %q", want)
		}
	}

	answered := client.call(t, 3, "tools/call", map[string]any{"name": "providers", "arguments": map[string]any{}})
	var envelope struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent struct {
			Rows      []json.RawMessage `json:"rows"`
			Truncated json.RawMessage   `json:"truncated"`
		} `json:"structuredContent"`
		IsError bool `json:"isError"`
	}
	decode(t, answered, &envelope)

	// The repository the environment named has no providers/ directory, so
	// the answer is the built-in and nothing else — which says the tool
	// resolved *that* repository and not the one the test binary was
	// compiled in.
	if len(envelope.StructuredContent.Rows) != 1 {
		t.Fatalf("the envelope carries %d rows, want the built-in alone: %s", len(envelope.StructuredContent.Rows), answered)
	}
	if got := string(envelope.StructuredContent.Rows[0]); !strings.Contains(got, `"name":"shell"`) {
		t.Errorf("the row is %s, want the built-in shell Provider", got)
	}
	if got := string(envelope.StructuredContent.Truncated); got != "false" {
		t.Errorf("truncated is %s, want false", got)
	}
	if len(envelope.Content) != 1 || envelope.Content[0].Type != "text" {
		t.Errorf("the content is %v, want one text block", envelope.Content)
	}
	if envelope.IsError {
		t.Error("isError is true on an ordinary return")
	}

	// **The server dies with the client.** Nothing listens on a port and
	// nothing outlives the invocation that started it (§9, §13): the client
	// closing its end of the pipe is the whole of the shutdown, and the
	// process exits clean without being signalled.
	client.close(t)
}

// session is a client's end of a `hyper mcp` process: the frames it writes, the
// frames it reads, and the process itself.
type session struct {
	command *exec.Cmd
	writing io.WriteCloser
	reading *bufio.Scanner
}

// start launches the binary as a client would — `hyper mcp` with no arguments,
// standing in the repository, with its two streams wired to pipes — and answers
// the session over it.
//
// stderr is left to the test's own output. The server writes nothing there in
// an ordinary session, and a session that failed is one whose narration a
// reader wants.
func start(t *testing.T, binary, repo string) *session {
	t.Helper()

	command := exec.Command(binary, "mcp")
	command.Dir = repo
	command.Env = append(command.Environ(), "HYPER_REPO_DIR="+repo)

	writing, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	reading, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		writing.Close()
		command.Wait()
	})
	return &session{command: command, writing: writing, reading: bufio.NewScanner(reading)}
}

// call writes one request and reads frames until the answer to it arrives,
// answering the `result` member. A frame carrying an `error` fails the case
// where it stands: every call these cases make is well-formed, so a JSON-RPC
// error is the server declining something it was not asked to decline.
//
// Frames that are not this call's answer are read past. The server sends
// nothing it was not asked for (ADR-0021), so in practice there are none —
// reading past them is what makes that a fact the case can survive rather than
// one it depends on.
func (s *session) call(t *testing.T, id int, method string, params any) json.RawMessage {
	t.Helper()

	s.write(t, map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	for s.reading.Scan() {
		var frame struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(s.reading.Bytes(), &frame); err != nil {
			t.Fatalf("%s: the server wrote a frame that is not JSON: %s", method, s.reading.Bytes())
		}
		if frame.ID == nil || *frame.ID != id {
			continue
		}
		if len(frame.Error) > 0 {
			t.Fatalf("%s answered the error %s", method, frame.Error)
		}
		return frame.Result
	}
	t.Fatalf("%s: the server answered nothing and closed its stream: %v", method, s.reading.Err())
	return nil
}

// notify writes one notification, which carries no id and is answered by
// nothing.
func (s *session) notify(t *testing.T, method string) {
	t.Helper()
	s.write(t, map[string]any{"jsonrpc": "2.0", "method": method})
}

// write puts one frame on the wire: compact JSON and a newline, which is what
// the stdio transport carries.
func (s *session) write(t *testing.T, frame map[string]any) {
	t.Helper()

	encoded, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.writing.Write(append(encoded, '\n')); err != nil {
		t.Fatalf("writing %s: %v", encoded, err)
	}
}

// close is the client going away, and the assertion beside it is that the
// process goes with it: stdin closes, the session ends, and `hyper mcp` exits
// clean of its own accord.
func (s *session) close(t *testing.T) {
	t.Helper()

	if err := s.writing.Close(); err != nil {
		t.Fatal(err)
	}
	err := s.command.Wait()
	var exited *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exited):
		t.Errorf("the server exited %d when its client went away, want 0", exited.ExitCode())
	default:
		t.Fatal(err)
	}
}

// decode reads one frame into the shape a case states for it. Unknown fields
// are passed over: the frame is the protocol's, and the SDK attaches members of
// its own that hyper neither writes nor reads.
func decode(t *testing.T, frame json.RawMessage, into any) {
	t.Helper()

	if err := json.Unmarshal(frame, into); err != nil {
		t.Fatalf("%s: %v", frame, err)
	}
}

// advertised is which capabilities the handshake named, sorted. It is the keys
// and not what each one carries, because that is what §9 fixes: *tools only: no
// resources and no prompts*.
func advertised(capabilities map[string]json.RawMessage) []string {
	named := make([]string, 0, len(capabilities))
	for capability := range capabilities {
		named = append(named, capability)
	}
	slices.Sort(named)
	return named
}
