package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
)

// The corpus reaches the second surface, and this file is the whole of what
// that costs (issue #195).
//
// **The call is real; only the client is in-process** — the principle
// golden_serve_test.go states for the TLS fixture, read across. A case
// supplying a `call` is driven through the MCP server over the SDK's in-memory
// transports, against the same process value, the same build facts and the same
// fixture repository an argv case is driven against; what a golden holds is the
// envelope that came back off the wire.
//
// **Nothing here stands a client of its own.** The SDK is reachable from
// internal/mcp and no other package, so the harness asks that package to make
// the call and hands it a server the binary would have started
// (cli.MCPServer) — which is what makes the corpus an assertion about the server
// rather than about a second assembly of one.

// toolCall is a case's `call` file: the tool one call names, and the arguments
// it is made with.
//
// The arguments stay raw so that the case says exactly what a client sent,
// down to the member a schema does not admit: reading them into a typed value
// here would be the harness validating the call before the server got a chance
// to, which is the one thing these cases are about.
type toolCall struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

// readCall reads a case's call file. Unknown fields are an error, on the
// corpus's own discipline: a misspelt key would otherwise be a case driving
// something other than what its file says, which is the failure mode a golden
// is least able to notice.
func readCall(t *testing.T, path string) *toolCall {
	t.Helper()

	var call toolCall
	decoder := json.NewDecoder(strings.NewReader(readFile(t, path)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&call); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if call.Tool == "" {
		t.Fatalf("%s: names no tool; a call is a tool and its arguments", path)
	}
	return &call
}

// compareEnvelope drives the case's one call and holds what came back against
// its envelope.golden, byte for byte and behind the same one flag every other
// golden is.
//
// The server is assembled the way the binary assembles it — cli.MCPServer over
// the case's process and facts — so the version it announces, the repository
// each tool resolves and the gate each one passes are the case's own inputs and
// not the harness's.
func (c goldenCase) compareEnvelope(t *testing.T, run goldenRun) {
	t.Helper()

	envelope, err := cli.MCPServer(c.process(t, run), c.facts(t)).
		Call(t.Context(), c.call.Tool, c.call.Arguments)
	if err != nil {
		t.Fatalf("case %s: %v", c.name, err)
	}
	compareRendering(t, filepath.Join(c.dir, "envelope.golden"), renderEnvelope(t, envelope))
}

// renderEnvelope is how the corpus holds an envelope: the envelope itself
// indented, and **each row on one line, compact**.
//
// The rows are written that way rather than expanded because it is the one
// rendering that makes the corpus's central claim readable: a row in an
// envelope.golden is one line, and so is the same row in the stdout.golden of
// the `--json` case beside it, there being one renderer behind both forms
// (ADR-0026, render.MarshalRow). What separates the two lines is the framing's
// own escaping and nothing else, which is what
// TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites holds them to. An
// expanded row would still be the same row and would no longer look like it.
//
// Every member is written whichever it is, `isError: false` included: §9 makes
// that bit the thing a Refusal is told from an ordinary return by, and a
// corpus that dropped it where it is false would hold nothing about the path
// this milestone builds. The wire omits it — MCP's own convention — which is
// exactly why the rendering states it.
func renderEnvelope(t *testing.T, envelope mcp.Envelope) string {
	t.Helper()

	var page strings.Builder
	page.WriteString("{\n  \"content\": [\n")
	for i, block := range envelope.Content {
		page.WriteString("    " + string(compact(t, block)) + separator(i, len(envelope.Content)) + "\n")
	}
	page.WriteString("  ],\n  \"structuredContent\": {\n")

	if len(envelope.StructuredContent.Rows) == 0 {
		page.WriteString("    \"rows\": [],\n")
	} else {
		page.WriteString("    \"rows\": [\n")
		for i, row := range envelope.StructuredContent.Rows {
			page.WriteString("      " + string(row) + separator(i, len(envelope.StructuredContent.Rows)) + "\n")
		}
		page.WriteString("    ],\n")
	}

	fmt.Fprintf(&page, "    \"truncated\": %s\n  },\n", envelope.StructuredContent.Truncated)
	fmt.Fprintf(&page, "  \"isError\": %t\n}\n", envelope.IsError)
	return page.String()
}

// separator is the comma between the members of a JSON list, and nothing after
// the last one.
func separator(i, of int) string {
	if i == of-1 {
		return ""
	}
	return ","
}

// compact is one value as its own line: marshalled, with the HTML escaping off
// that §8's row stream already writes without. A text block quoting a `&` or a
// `<` is one a reader reads back as it was written.
//
// It is not render.MarshalRow, and cannot be: what that answers is a **row**,
// and the only value this needs it for is a content block, which is not one.
// The rows in an envelope never reach here at all — they arrive already
// encoded, and this file writes them down as they came.
func compact(t *testing.T, value any) []byte {
	t.Helper()

	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSuffix(encoded.Bytes(), []byte("\n"))
}

// TestGoldenCorpora_EveryEnvelopeGoldenIsDrivenBySomething is
// TestGoldenCorpora_EveryGoldenTripleIsDrivenBySomething for the second
// surface, and it exists for the same reason: with the harness reading its
// cases off the disk, a case whose `call` went missing would simply stop being
// run and its golden would sit there green and unexercised.
func TestGoldenCorpora_EveryEnvelopeGoldenIsDrivenBySomething(t *testing.T) {
	driven := map[string]bool{}
	for _, c := range goldenCases(t) {
		if c.call != nil {
			driven[c.dir] = true
		}
	}
	if len(driven) == 0 {
		t.Fatal("no case under testdata/ holds a call; the second surface is covered by nothing")
	}

	walkTestdata(t, "envelope.golden", func(dir string) {
		if !driven[dir] {
			t.Errorf("%s holds an envelope.golden and is not a case the harness drives; it needs a call", dir)
		}
	})
}

// TestGoldenCorpora_EveryEnvelopeGoldenIsOneJSONValue is the fence the
// hand-written rendering above needs: renderEnvelope writes the braces, the
// commas and the indentation itself, so a corpus that never parsed what it
// checked in could hold a shape no client could read.
//
// It parses rather than compares, because what the rendering is free to choose
// is the whitespace and nothing else — a golden that is valid JSON and carries
// §9's three members is the envelope, however it was laid out.
func TestGoldenCorpora_EveryEnvelopeGoldenIsOneJSONValue(t *testing.T) {
	var read int
	for _, c := range goldenCases(t) {
		if c.call == nil {
			continue
		}
		read++

		var held struct {
			Content           []json.RawMessage `json:"content"`
			StructuredContent struct {
				Rows      []json.RawMessage `json:"rows"`
				Truncated json.RawMessage   `json:"truncated"`
			} `json:"structuredContent"`
			IsError *bool `json:"isError"`
		}
		golden := readFile(t, filepath.Join(c.dir, "envelope.golden"))
		decoder := json.NewDecoder(strings.NewReader(golden))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&held); err != nil {
			t.Errorf("case %s: its envelope.golden is not one JSON value: %v", c.name, err)
			continue
		}
		switch {
		case len(held.Content) == 0:
			t.Errorf("case %s: its envelope carries no content block", c.name)
		case len(held.StructuredContent.Truncated) == 0:
			t.Errorf("case %s: its envelope carries no truncated member", c.name)
		case held.IsError == nil:
			t.Errorf("case %s: its envelope carries no isError; the bit is written whichever it is", c.name)
		}
	}
	if read == 0 {
		t.Fatal("no case under testdata/ holds a call; the rule was held over nothing")
	}
}

// TestGoldenCorpora_NoCallCaseNamesARepository is §9's own line held over the
// corpus: **no tool takes an override argument of any kind, under any name**.
//
// It is worth a fence rather than a reading of the schemas because the schemas
// are what a fence would be checking against, and a case is where a name would
// actually be typed. A call whose arguments carried a repository would be
// admitted by nothing today and would be the first thing anyone reached for the
// day a tool needed one — which is precisely the day the rule has to hold.
func TestGoldenCorpora_NoCallCaseNamesARepository(t *testing.T) {
	for _, c := range goldenCases(t) {
		if c.call == nil {
			continue
		}
		var named map[string]json.RawMessage
		if err := json.Unmarshal(c.call.Arguments, &named); err != nil {
			t.Errorf("case %s: its arguments are not an object: %v", c.name, err)
			continue
		}
		for argument := range named {
			if strings.Contains(argument, "repo") || strings.Contains(argument, "dir") {
				t.Errorf("case %s calls %s(%s: …); no tool takes an argument naming a repository", c.name, c.call.Tool, argument)
			}
		}
	}
}

// TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites is ADR-0026 held
// where the two surfaces meet: §8's row set is served here as an array rather
// than as a line stream, and *there is one renderer behind both forms, so the
// terminal and this surface cannot drift apart*.
//
// What it holds a row to is its **keys, in order** and its **values**, which is
// exactly what §8 fixes about a row: the `type` is the first key, the rest
// follow declaration order, an absent member is absent rather than null, and
// nothing is abbreviated. A row that reached the envelope through a second
// composition would differ in one of those.
//
// It is not a byte comparison, and the one thing that stops it being one is
// worth naming: the JSON-RPC frame is the SDK's, and the SDK's encoder escapes
// `<`, `>` and `&` as `\u003c` and friends where render.MarshalRow does not.
// That is a difference in the **framing** and not in the row — every JSON
// decoder unescapes it, so a consumer still reads the artefact's own bytes back
// as they were written, which is what render.go's rule is for. Holding the two
// to the same bytes would be holding hyper to a choice the transport made.
//
// It holds only over the rows that appear in both corpora, and says nothing
// about a row only one surface has a case for; what it fences is the claim that
// where both have one, it is one row.
func TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites(t *testing.T) {
	streamed := map[string]string{}
	for _, c := range goldenCases(t) {
		if !c.opensARowStream() {
			continue
		}
		for _, line := range strings.Split(strings.TrimSuffix(readFile(t, filepath.Join(c.dir, "stdout.golden")), "\n"), "\n") {
			if key := rowIdentity(line); key != "" {
				streamed[key] = line
			}
		}
	}

	var compared int
	for _, c := range goldenCases(t) {
		if c.call == nil {
			continue
		}
		for _, line := range strings.Split(readFile(t, filepath.Join(c.dir, "envelope.golden")), "\n") {
			line = strings.TrimSuffix(strings.TrimSpace(line), ",")
			key := rowIdentity(line)
			if key == "" {
				continue
			}
			written, streamedToo := streamed[key]
			if !streamedToo {
				continue
			}
			compared++
			if got, want := keysInOrder(t, line), keysInOrder(t, written); !slices.Equal(got, want) {
				t.Errorf("case %s carries a %s row keyed %q; the --json stream writes it keyed %q", c.name, key, got, want)
			}
			if got, want := decodedRow(t, line), decodedRow(t, written); !reflect.DeepEqual(got, want) {
				t.Errorf("case %s carries the row\n  %v\nand the --json stream writes it\n  %v", c.name, got, want)
			}
		}
	}
	if compared == 0 {
		t.Fatal("no row appears in both an envelope.golden and a --json stdout.golden; the rule was held over nothing")
	}
}

// keysInOrder is one row's keys in the order they were written, which is the
// order §8 fixes: the `type` first, the rest in the row type's own declaration
// order. It reads them off the tokens rather than into a map, a map having no
// order to read.
func keysInOrder(t *testing.T, line string) []string {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(line))
	if opening, err := decoder.Token(); err != nil || opening != json.Delim('{') {
		t.Fatalf("%s does not open an object: %v", line, err)
	}

	var keys []string
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key.(string))
		// The value, whatever it is, read past whole: a nested object or
		// array is one token to this loop and its own keys are not this
		// row's.
		var skipped json.RawMessage
		if err := decoder.Decode(&skipped); err != nil {
			t.Fatal(err)
		}
	}
	return keys
}

// decodedRow is one row's members as values, which is what makes the comparison
// about what the row states rather than about how the frame spelled it.
func decodedRow(t *testing.T, line string) map[string]any {
	t.Helper()

	var row map[string]any
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		t.Fatal(err)
	}
	return row
}

// rowIdentity is what makes two renderings of one row the same row: its `type`,
// and the two members that name a row within a type — the `name` a listing's
// rows carry and the `digest` that identifies the artefact a header row is
// about. Both are needed: a Manifest header row carries no name, so two
// Manifests would otherwise be one row seen twice.
//
// It is the empty string for a line that is not a row at all — a brace, a text
// block, and the terminal row a stream carries and an envelope does not.
func rowIdentity(line string) string {
	var row struct {
		Type   string `json:"type"`
		Name   string `json:"name"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return ""
	}
	switch row.Type {
	case "", "result", "outcome", "text":
		return ""
	}
	return strings.Join([]string{row.Type, row.Name, row.Digest}, "\x00")
}
