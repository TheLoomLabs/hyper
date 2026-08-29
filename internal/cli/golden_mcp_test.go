package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
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
// its envelope.golden — or against its error.golden, where the call was
// malformed and came back as a protocol error rather than as an answer. Both
// are compared byte for byte and behind the same one flag every other golden
// is.
//
// The server is assembled the way the binary assembles it — cli.MCPServer over
// the case's process and facts — so the version it announces, the repository
// each tool resolves and the gate each one passes are the case's own inputs and
// not the harness's.
//
// **Which of the two goldens a case holds is decided by what came back and not
// by what the case declares**, which is the reading every other golden already
// gets: a case states its inputs and the corpus holds the answer. A case that
// changed sides is caught by the fence rather than by this driver, a stale
// golden left beside a new one being the failure a comparison cannot see
// (TestGoldenCorpora_ACallCaseHoldsOneGoldenAndNotTheOther).
func (c goldenCase) compareEnvelope(t *testing.T, run goldenRun) {
	t.Helper()

	envelope, err := cli.MCPServer(c.process(t, run), c.facts(t)).
		Call(t.Context(), c.call.Tool, c.call.Arguments)
	if err != nil {
		// The message and nothing else: what a JSON-RPC error carries to a
		// caller is the sentence the command wrote where a person would
		// have read it on stderr, and the code beside it is the SDK's own
		// mapping of a handler error rather than a number hyper chooses
		// (§9, mcp.Server.Call).
		compareRendering(t, filepath.Join(c.dir, "error.golden"), err.Error()+"\n")
		return
	}
	compareRendering(t, filepath.Join(c.dir, "envelope.golden"), renderEnvelope(t, envelope))
}

// answersAProtocolError says whether the case holds an error.golden: a call §9
// answers with a JSON-RPC error rather than with an envelope — an argument the
// schema does not admit, and a positional that matches nothing (§9, issue
// #196).
//
// It reads the disk rather than a member of the case, for the reason every
// other input is read off the disk: a corpus is directories, and what a case
// holds is what is in it.
func (c goldenCase) answersAProtocolError() bool {
	return isFile(filepath.Join(c.dir, "error.golden"))
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

	// **The half is absent where the command opened no row stream**, and the
	// corpus writes that absence as the wire carries it: the key is not
	// there at all, which is MCP's own shape for a tool that declined and is
	// the whole of what ADR-0102 changed. A golden writing `null` or an
	// empty object would be holding the shape that decision removed.
	structured := envelope.StructuredContent
	if structured == nil {
		fmt.Fprintf(&page, "  ],\n  \"isError\": %t\n}\n", envelope.IsError)
		return page.String()
	}
	page.WriteString("  ],\n  \"structuredContent\": {\n")

	// The execution members, written first and written only where the
	// answer carries them: §9 puts them ahead of the rows in the envelope
	// it states, and a tool that is not a Run carries no `outcome` key at
	// all (§9, issue #200). `dry_run` is written wherever `outcome` is,
	// `false` included, which is why the corpus writes it off the pointer
	// rather than off its value.
	if structured.Outcome != "" {
		fmt.Fprintf(&page, "    \"outcome\": %q,\n", structured.Outcome)
		if structured.RunID != "" {
			fmt.Fprintf(&page, "    \"run_id\": %q,\n", structured.RunID)
		}
		if structured.DryRun != nil {
			fmt.Fprintf(&page, "    \"dry_run\": %t,\n", *structured.DryRun)
		}
	}

	// The page, where the tool's text block is one: it stands above the rows
	// in the envelope for the reason the summary line stands above them in
	// the block, and the corpus writes it where the wire writes it (§9,
	// ADR-0100, issue #217). It is compacted like a content block rather
	// than expanded, so the two channels carrying one page are two lines a
	// reader can put beside each other.
	if structured.Rendering != "" {
		fmt.Fprintf(&page, "    \"rendering\": %s,\n", compact(t, structured.Rendering))
	}

	if len(structured.Rows) == 0 {
		page.WriteString("    \"rows\": [],\n")
	} else {
		page.WriteString("    \"rows\": [\n")
		for i, row := range structured.Rows {
			page.WriteString("      " + string(row) + separator(i, len(structured.Rows)) + "\n")
		}
		page.WriteString("    ],\n")
	}

	fmt.Fprintf(&page, "    \"truncated\": %s\n  },\n", structured.Truncated)
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
		if c.call == nil || c.answersAProtocolError() {
			continue
		}
		read++

		var held struct {
			Content           []json.RawMessage `json:"content"`
			StructuredContent *struct {
				Outcome   string            `json:"outcome"`
				RunID     string            `json:"run_id"`
				DryRun    *bool             `json:"dry_run"`
				Rendering string            `json:"rendering"`
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
		// **The content block is held first, and it is held on every
		// case.** It is the one member of §9's envelope that is never
		// absent, and on the path below it is the *whole* envelope — a
		// tool that declined answers `content` and the bit alone
		// (ADR-0102), so a case that lost its block there would have
		// lost the entire Refusal and there would be no structured half
		// left to notice it by.
		if len(held.Content) == 0 {
			t.Errorf("case %s: its envelope carries no content block", c.name)
			continue
		}
		// **The structured half is absent or it is whole**, and the
		// absence is a shape rather than a member gone missing: MCP's
		// own error mechanism is `isError` and the `content` beside it,
		// and it names no structured channel at all. The rules below are
		// about members that are not there to hold; what the corpus
		// keeps is that a half arriving *half*-composed still fails,
		// `truncated` included.
		if held.StructuredContent == nil {
			if held.IsError == nil || !*held.IsError {
				t.Errorf("case %s: an ordinary return carries no structured half; only a tool that declined answers content alone", c.name)
			}
			continue
		}
		switch {
		case len(held.StructuredContent.Truncated) == 0:
			t.Errorf("case %s: its envelope carries no truncated member", c.name)
		case held.IsError == nil:
			t.Errorf("case %s: its envelope carries no isError; the bit is written whichever it is", c.name)
		// §7's one exception to the absence rule, held over the surface
		// that carries it: `dry_run` is written wherever `outcome` is,
		// the bare `false` included, and a golden holding the one
		// without the other would state an envelope whose reader cannot
		// tell a rehearsal from a Run that reached the world (§9, issue
		// #200).
		case held.StructuredContent.Outcome != "" && held.StructuredContent.DryRun == nil:
			t.Errorf("case %s: its envelope carries outcome %q and no dry_run; the marker is written wherever the outcome is", c.name, held.StructuredContent.Outcome)
		case held.StructuredContent.Outcome == "" && held.StructuredContent.DryRun != nil:
			t.Errorf("case %s: its envelope carries dry_run and no outcome; the marker rides beside the triple and nowhere else", c.name)
		case held.StructuredContent.Outcome == "" && held.StructuredContent.RunID != "":
			t.Errorf("case %s: its envelope names a Run and carries no outcome; a tool that is not a Run carries neither", c.name)
		}
	}
	if read == 0 {
		t.Fatal("no case under testdata/ holds a call; the rule was held over nothing")
	}
}

// TestGoldenCorpora_AToolLeavesTheBranchItsCommandLeaves is the claim this
// surface is really about, held as a comparison of bytes: *ergonomics is the
// whole of the difference between the two* (§9, issues #200 and #201).
//
// A tool builds the command line its command would have received and hands it
// to the same dispatch, so what a call leaves on the Store branch and what the
// same invocation leaves through the argv are one thing. On `run` that is the
// Journal entry, the Records and the Provenance a Run writes; on `probe` it is
// **nothing at all**, a Probe writing no Record and no Journal entry (ADR-0009)
// — and a store.golden is how the corpus says either without taking the tool's
// word for it.
func TestGoldenCorpora_AToolLeavesTheBranchItsCommandLeaves(t *testing.T) {
	pairedAgainstTheCommandCorpus(t, pairing{
		golden:  "store.golden",
		left:    "left a branch",
		nothing: "no case under testdata/mcp/ is paired with the command corpus; the claim that one invocation reaches one branch through two doors is held over nothing",
	})
}

// TestGoldenCorpora_AToolWritesTheTreeItsCommandWrites is the same claim over
// the one place the other half of it lands: the **working tree** (§10, issue
// #203).
//
// `project` is the tool that writes files, and a store.golden has nothing to
// say about it — the projection is workflows and two scalars in a reviewed
// artefact, and none of that is a Store branch. What says a call wrote what its
// command writes is the tree.golden the corpus already asks every projecting
// case for: the two text streams say what a command *reported*, and only the
// tree says what it *did* (testdata/project/README.md, issue #177).
//
// It is what makes the exemption assertable through this door too. A call
// against a repository whose pin is absent, and one against a repository whose
// pin this binary is not, each leave a tree — and holding those bytes to the
// argv twin's is how the corpus says the tool passed no gate the command passes
// either, without taking the dispatch's word for it (§11, ADR-0020).
func TestGoldenCorpora_AToolWritesTheTreeItsCommandWrites(t *testing.T) {
	pairedAgainstTheCommandCorpus(t, pairing{
		golden:  "tree.golden",
		left:    "wrote a tree",
		nothing: "no call case writes the working tree; the claim that one invocation reaches one tree through two doors is held over nothing",
	})
}

// TestGoldenCorpora_AToolPushesWhatItsCommandPushes is the third and last place
// an invocation lands, and the one that carries the record off the machine:
// **the remote** (§7, issue #204).
//
// A store.golden says what a Run left on the local branch and a tree.golden
// what it wrote in the working tree; neither says whether the entry reached
// `origin`, which is the half of §7's transport a second clone actually reads.
// A Run pushes after each effect and once at the close, and a surface that did
// that differently would be a surface whose Runs are invisible to everyone but
// the machine that ran them.
//
// It is the milestone's closing check read over its own last axis: the two
// surfaces write one Store, and one Store is the branch, the tree and the
// remote together (ADR-0006, ADR-0026).
func TestGoldenCorpora_AToolPushesWhatItsCommandPushes(t *testing.T) {
	pairedAgainstTheCommandCorpus(t, pairing{
		golden:  "remote.golden",
		left:    "pushed a branch",
		nothing: "no call case pushes to a remote; the claim that one invocation reaches one remote through two doors is held over nothing",
	})
}

// pairedAgainstTheCommandCorpus holds one golden of every call case against the
// file of the same name in the case one directory up, and is the two fences
// above as one reading.
//
// **The pairing is by name and the name is the command corpus's**, which is what
// makes both fences self-maintaining: a case under `testdata/mcp/<tool>/` named
// for a case under the corpus of the command that tool carries is asserted
// against it, and one named for nothing — every `usage-` case, which reaches no
// invocation at all — carries neither golden and is passed over. A twin renamed
// on one side and not the other stops being compared, which the count is what
// catches.
//
// It takes one value rather than three strings because two of the three are
// prose and only their position said which: a mid-sentence verb phrase and a
// whole t.Fatal message read alike at a call site, and naming them is what
// stops the two being passed the wrong way round.
func pairedAgainstTheCommandCorpus(t *testing.T, held pairing) {
	t.Helper()

	var paired int
	walkTestdata(t, held.golden, func(dir string) {
		name := filepath.ToSlash(strings.TrimPrefix(dir, "testdata"+string(filepath.Separator)))
		twin, under := twinOf(name)
		if !under {
			return
		}
		beside := filepath.Join(twin, held.golden)
		if !isFile(beside) {
			t.Errorf("case %s holds a %s and %s holds none; a call is named for the invocation through the command it is the same invocation as", name, held.golden, twin)
			return
		}
		paired++
		if through, command := readFile(t, filepath.Join(dir, held.golden)), readFile(t, beside); through != command {
			t.Errorf("case %s %s its argv twin did not:\n through the tool:    %q\n through the command: %q", name, held.left, through, command)
		}
	})
	if paired == 0 {
		t.Fatal(held.nothing)
	}
}

// pairing is one of the two fences above as a value: which golden is compared,
// and the two sentences that differ between them.
type pairing struct {
	// golden is the file compared, and the file whose presence finds the
	// cases to compare: a case holding one opts into the pairing by holding
	// it, which is how both fences stay self-maintaining.
	golden string
	// left is what the golden is a record of, as the middle of the one
	// sentence that reports a difference — `case X left a branch its argv
	// twin did not`.
	left string
	// nothing is what the fence says where it compared nothing at all,
	// which is a rule held over an empty corpus and therefore a failure
	// rather than a pass.
	nothing string
}

// TestGoldenCorpora_EveryErrorGoldenIsDrivenBySomething is
// TestGoldenCorpora_EveryEnvelopeGoldenIsDrivenBySomething for the half of this
// surface that answers no envelope at all, and it exists for the same reason: a
// case whose `call` went missing would stop being driven and its golden would
// sit there green and unexercised.
func TestGoldenCorpora_EveryErrorGoldenIsDrivenBySomething(t *testing.T) {
	driven := map[string]bool{}
	for _, c := range goldenCases(t) {
		if c.call != nil {
			driven[c.dir] = true
		}
	}

	var held int
	walkTestdata(t, "error.golden", func(dir string) {
		held++
		if !driven[dir] {
			t.Errorf("%s holds an error.golden and is not a case the harness drives; it needs a call", dir)
		}
	})
	if held == 0 {
		t.Fatal("no case under testdata/ holds an error.golden; the malformed half of §9's surface is covered by nothing")
	}
}

// TestGoldenCorpora_ACallCaseHoldsOneGoldenAndNotTheOther is the fence the
// driver needs, and the failure it is for is the quiet one: a case that came to
// answer a protocol error where it used to answer an envelope would have its
// error.golden written under -update and leave the envelope.golden beside it,
// checked in, unexercised, and stating an answer this surface no longer gives.
//
// **A domain outcome is never a protocol error, and the corpus says which a
// case is by which file it holds** (§9). Holding both would be a case claiming
// both at once, and holding neither is a call driven against nothing.
func TestGoldenCorpora_ACallCaseHoldsOneGoldenAndNotTheOther(t *testing.T) {
	for _, c := range goldenCases(t) {
		if c.call == nil {
			continue
		}
		envelope := isFile(filepath.Join(c.dir, "envelope.golden"))
		switch {
		case envelope && c.answersAProtocolError():
			t.Errorf("case %s holds both an envelope.golden and an error.golden; a call is answered one way or the other", c.name)
		case !envelope && !c.answersAProtocolError():
			t.Errorf("case %s holds neither an envelope.golden nor an error.golden; its call is driven against nothing", c.name)
		}
	}
}

// TestGoldenCorpora_WhatDeclinesInAnEnvelopeIsWhatTheCLIWroteOnStderr is
// ADR-0026 held over the second half of §9's mapping, the way the row fence
// below holds it over the first: a Refusal and a usage error are one rendering
// apiece, and the two surfaces carry the same one (§9, issue #196).
//
// **What it does not hold is an ordinary return**, and that is the rule rather
// than an omission: what a twin wrote beside an answer is narration — a
// truncation line, a warning — and narration is what this surface drops, the
// envelope saying in both halves what the CLI said in the line beside its table
// (§9, destination.go). So the pairing is over the calls that **declined**, and
// each half of that has a pairing of its own.
//
// A **Refusal** is paired against the whole corpus, exactly as a row is: §8's
// rendering opens `refused: <error_code>` in both its forms, so the renderings
// the CLI writes are collectable, and the claim is that a text block that opens
// one of them opens it **whole**. What follows it is the retry sentence this
// surface adds, which is the only place the protocol leaves for saying that a
// verbatim retry refuses identically (ADR-0001, refusal.go). Pairing by the
// corpus rather than by directory is what lets a case filed with the
// invocations it is contrasted with be held too — `exemption/provider` is a
// tool against the repository `exemption/check` Refuses on, and it is not under
// `mcp/` to have a twin one directory up (testdata/exemption).
//
// A **usage error** is paired by name, because there is nothing in a sentence
// to collect it by: `mcp/<tool>/<case>` against `<command>/<case>`, one fixture
// driven two ways. The message is the sentence itself, byte for byte — an agent
// reads what a person would have read.
func TestGoldenCorpora_WhatDeclinesInAnEnvelopeIsWhatTheCLIWroteOnStderr(t *testing.T) {
	refused := refusalsWritten(t)

	var compared int
	for _, c := range goldenCases(t) {
		if c.call == nil {
			continue
		}

		if c.answersAProtocolError() {
			twin, paired := twinOf(c.name)
			if !paired {
				continue
			}
			wrote := strings.TrimRight(readFile(t, filepath.Join(twin, "stderr.golden")), "\n")
			if wrote == "" {
				continue
			}
			compared++
			if got := strings.TrimRight(readFile(t, filepath.Join(c.dir, "error.golden")), "\n"); got != wrote {
				t.Errorf("case %s: the protocol error carries\n  %q\nand %s wrote\n  %q", c.name, got, twin, wrote)
			}
			continue
		}

		text := textBlock(t, c.dir)
		if !strings.HasPrefix(text, refusalOpening) {
			continue
		}
		compared++
		if !slices.ContainsFunc(refused, func(rendering string) bool { return strings.HasPrefix(text, rendering) }) {
			t.Errorf("case %s: its text block is\n  %q\nand no case in the corpus writes that Refusal on stderr", c.name, text)
		}
	}
	if compared == 0 {
		t.Fatal("no call case declined against a case that wrote on stderr; the rule was held over nothing")
	}
}

// refusalOpening is what every §8 Refusal rendering begins with, in both of its
// forms: the word and the code that declined (refusal.go). It is what tells a
// Refusal from the narration a command writes beside an answer, on stderr and
// in a text block alike.
const refusalOpening = "refused: "

// refusalsWritten is every Refusal the CLI writes, as the corpus holds it: the
// stderr golden of each case that Refused, trimmed of its trailing newline.
//
// It is collected off the disk rather than composed, which is the same reading
// the row fence takes: what a case checked in is what the command wrote, and a
// rendering assembled here would be this file's account of §8 rather than the
// corpus's.
func refusalsWritten(t *testing.T) []string {
	t.Helper()

	var refused []string
	walkTestdata(t, "stderr.golden", func(dir string) {
		if wrote := strings.TrimRight(readFile(t, filepath.Join(dir, "stderr.golden")), "\n"); strings.HasPrefix(wrote, refusalOpening) {
			refused = append(refused, wrote)
		}
	})
	if len(refused) == 0 {
		t.Fatal("no case under testdata/ writes a Refusal on stderr; the pairing has nothing to hold against")
	}
	return refused
}

// textBlock is the text of a case's first content block, read out of its
// checked-in envelope.golden.
//
// It is the one member of an envelope a fence has to decode rather than
// compare: the block is a JSON string carrying a rendering whose own newlines
// are escaped, and what is compared against it is those newlines as they stand.
// The shape is spelled here rather than taken from internal/mcp because a fence
// reading the corpus reads what is **checked in** — a shape borrowed from the
// package it is fencing would agree with that package by construction.
func textBlock(t *testing.T, dir string) string {
	t.Helper()

	held := readEnvelope(t, dir)
	if len(held.Content) == 0 {
		t.Fatalf("%s: its envelope carries no content block", dir)
	}
	return held.Content[0].Text
}

// heldEnvelope is a checked-in envelope as the fences read it back: the text of
// each content block, the machine half, and the bit.
//
// **The shape is spelled here rather than taken from internal/mcp**, and it is
// one shape rather than one per fence. The first half is textBlock's own
// reason: a fence reading the corpus reads what is checked in, and a shape
// borrowed from the package it is fencing would agree with that package by
// construction. The second is that three fences asking one file three
// questions were three readers of one format, which is three places for a
// member to be renamed in.
//
// It is not the shape
// TestGoldenCorpora_EveryEnvelopeGoldenIsOneJSONValue reads, and that one stays
// its own: what it asserts is that the golden carries **these members and no
// others**, which is a decoder configured to refuse an unknown field rather
// than a value to read one out of.
type heldEnvelope struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	// StructuredContent is a pointer because its absence is a shape the
	// corpus holds: a tool that declined before it opened a row stream
	// answers `content` and the bit alone (ADR-0102), and a value here
	// would read that as an empty half rather than as none.
	StructuredContent *structuredHalf `json:"structuredContent"`
	IsError           bool            `json:"isError"`
}

// readEnvelope reads one case's checked-in envelope as JSON rather than by
// line, for the reason the fences around it read what is checked in: the
// rendering's layout is the corpus's own choice and the answer is what it
// holds, so a fence that depended on the indent would be a fence on
// renderEnvelope.
func readEnvelope(t *testing.T, dir string) heldEnvelope {
	t.Helper()

	var held heldEnvelope
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(dir, "envelope.golden"))), &held); err != nil {
		t.Fatalf("%s: its envelope.golden is not one JSON value: %v", dir, err)
	}
	return held
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
//
// **One identity can have more than one rendering across the corpus, and the
// pairing is against all of them** (issue #197). A row's identity is what names
// it within its type, and two cases can name one row and state different facts
// about it: `targets` computes credential presence when it runs, so
// `cloudflare-prod` is one row with `present: true` in a case whose `env` sets
// the variable and `present: false` in one that does not. Keeping a single
// rendering per identity would silently pair an envelope against whichever case
// the walk read last. So the claim held is that the envelope's row **is one of
// the renderings the stream writes** for that identity, byte-compatible in
// values and identical in key order — which is the strongest thing a
// corpus-wide pairing can say, and says it soundly.
func TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites(t *testing.T) {
	streamed := map[string][]string{}
	for _, c := range goldenCases(t) {
		if !c.opensARowStream() {
			continue
		}
		for _, line := range strings.Split(strings.TrimSuffix(readFile(t, filepath.Join(c.dir, "stdout.golden")), "\n"), "\n") {
			if key := rowIdentity(line); key != "" && !slices.Contains(streamed[key], line) {
				streamed[key] = append(streamed[key], line)
			}
		}
	}

	var compared int
	for _, c := range goldenCases(t) {
		if c.call == nil || c.answersAProtocolError() {
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
			carried := decodedRow(t, line)
			matched := slices.IndexFunc(written, func(w string) bool { return reflect.DeepEqual(decodedRow(t, w), carried) })
			if matched < 0 {
				states := make([]map[string]any, 0, len(written))
				for _, w := range written {
					states = append(states, decodedRow(t, w))
				}
				t.Errorf("case %s carries the row\n  %v\nand no --json stream writes it; the stream writes\n  %v", c.name, carried, states)
				continue
			}
			if got, want := keysInOrder(t, line), keysInOrder(t, written[matched]); !slices.Equal(got, want) {
				t.Errorf("case %s carries a %s row keyed %q; the --json stream writes it keyed %q", c.name, key, got, want)
			}
		}
	}
	if compared == 0 {
		t.Fatal("no row appears in both an envelope.golden and a --json stdout.golden; the rule was held over nothing")
	}
}

// TestGoldenCorpora_AnEnvelopeIsTheStreamLessItsTerminalRow is this milestone's
// most valuable assertion and it is one comparison: **where a case supplies
// both an argv and a call, the rows in `envelope.golden` are the same rows, in
// the same order, that `stdout.golden` carries under `--json`, less the
// terminal row.** One list, two surfaces, checked — which is the whole of
// ADR-0026 stated as a test rather than as a comment (§9, issue #204).
//
// It is the fence above read the other way round and it is a different claim.
// TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites pairs by a row's
// own identity across the whole corpus, so it says *this row is a row the
// stream writes somewhere*; a tool that dropped a row, doubled one or answered
// them in another order would pass it whole. This one holds one fixture's two
// renderings against each other **in order and in full**, which is the only
// reading that can say the two surfaces answer one list.
//
// **The pairing is `mcp/<tool>/<case>` against `<command>/<case>-json`**, the
// `-json` twin being how every corpus here holds a command's second rendering,
// and against `<command>/<case>` where that case is itself a stream. A call
// case whose twin writes a page and opens no stream at all is passed over:
// there is no row list on the other side to compare with, which is a `-json`
// twin to write rather than a rule to weaken.
//
// **A twin that opened a stream and wrote nothing into it is a comparison and
// not a gap.** That is what a Refusal does — the rendering goes where a person
// reads it and stdout stays empty — and what the two surfaces say there is that
// neither carried a row, which the fence holds rather than skips.
//
// **The truncation marker's `hint` is the one stated exception.** It names the
// arguments a tool call can type where the terminal names its flags — *the only
// wording an answer changes between the two surfaces* — so the marker is
// compared member by member with `hint` set aside, and the exception is
// asserted rather than tolerated: where one surface carries a hint the other
// carries one too (§9, render.TruncationMarker.InArguments).
func TestGoldenCorpora_AnEnvelopeIsTheStreamLessItsTerminalRow(t *testing.T) {
	streams := map[string]goldenCase{}
	for _, c := range goldenCases(t) {
		if c.opensARowStream() {
			streams[c.dir] = c
		}
	}

	var compared int
	var unpaired []string
	for _, c := range goldenCases(t) {
		if c.call == nil || c.answersAProtocolError() {
			continue
		}
		twin, paired := twinOf(c.name)
		if !paired {
			continue
		}
		stream, driven := streams[twin+"-json"]
		if !driven {
			if stream, driven = streams[twin]; !driven {
				if isDir(twin) {
					unpaired = append(unpaired, c.name)
				}
				continue
			}
		}
		compared++

		carried := envelopeRows(t, c.dir)
		golden := filepath.Join(stream.dir, "stdout.golden")
		if !isFile(golden) {
			// readFile answers the empty string for a file that is not
			// there, and the branch below reads an empty stream as a
			// Refusal — so a twin whose golden went missing would be a
			// pairing that quietly stopped asserting the rows.
			t.Errorf("case %s is paired against %s, which holds no stdout.golden", c.name, stream.name)
			continue
		}
		if page := readFile(t, golden); page == "" {
			// **A twin that wrote no stream at all is a Refusal**,
			// which writes its rendering where a person reads it and
			// leaves stdout empty on both surfaces (§8, §9). There is
			// no terminal row to drop and nothing to lift out of one,
			// so what the two say is that neither carried a row.
			if len(carried) != 0 {
				t.Errorf("case %s carries %d rows and %s wrote no stream at all; a Refusal answers rows on neither surface", c.name, len(carried), stream.name)
			}
			// **Two shapes say *there was no stream*, and which one
			// a case answers is which command it carries.** A tool
			// that is not a Run answers no structured half at all
			// (ADR-0102); `run` answers one either way, §8 putting
			// it on the `outcome` side on every path a Run was
			// attempted, and there the member is `null`. What is
			// wrong is a boolean, which would claim a stream was
			// complete that was never a stream.
			switch got := readEnvelope(t, c.dir).StructuredContent.truncated(); got {
			case "", "null":
			default:
				t.Errorf("case %s carries truncated %s and %s wrote no stream at all; there was no result set for a cap to have cut", c.name, got, stream.name)
			}
			continue
		}

		written := strings.Split(strings.TrimSuffix(readFile(t, golden), "\n"), "\n")
		terminal := written[len(written)-1]
		if !endsAStream(terminal) {
			t.Errorf("case %s: %s does not end in a terminal row; its last line is %s", c.name, stream.name, terminal)
			continue
		}
		written = written[:len(written)-1]

		if len(carried) != len(written) {
			t.Errorf("case %s carries %d rows and %s writes %d:\n through the tool:    %v\n through the command: %v", c.name, len(carried), stream.name, len(written), carried, written)
			continue
		}
		for i := range carried {
			// The values and the key order, and not the bytes: the
			// JSON-RPC frame is the SDK's and its encoder escapes
			// `<`, `>` and `&` where render.MarshalRow does not,
			// which is a difference in the framing and not in the
			// row (the fence above).
			if got, want := decodedRow(t, carried[i]), decodedRow(t, written[i]); !reflect.DeepEqual(got, want) {
				t.Errorf("case %s: its row %d is\n  %v\nand %s writes\n  %v", c.name, i+1, got, stream.name, want)
				continue
			}
			if got, want := keysInOrder(t, carried[i]), keysInOrder(t, written[i]); !slices.Equal(got, want) {
				t.Errorf("case %s: its row %d is keyed %q and %s writes it keyed %q", c.name, i+1, got, stream.name, want)
			}
		}
		compareTruncation(t, c, stream, terminal)
	}
	if compared == 0 {
		t.Fatal("no call case was paired against a --json stream of the same fixture; the rule was held over nothing")
	}
	// **What the fence could not reach is named rather than dropped
	// quietly.** A case whose argv twin exists and was never given a `--json`
	// rendering is a pairing waiting for a twin to be written, and a
	// coverage bound nobody can see reads as *everything is covered*. It is
	// reported rather than failed because writing those twins is a corpus
	// edit and not this rule lapsing.
	if len(unpaired) > 0 {
		t.Logf("%d of %d call cases have an argv twin that writes a page and no --json stream, so their rows are held against nothing; each is a `-json` twin waiting to be written: %q", len(unpaired), compared+len(unpaired), unpaired)
	}
}

// compareTruncation is the terminal fact half of the fence above: what the
// envelope lifted out of the terminal row, held against what the terminal row
// carried, with the `hint` set aside.
//
// A Run's terminal row carries no truncation at all — §8 makes the type the
// discriminator, and a Run reports what it did rather than ranging over a
// namespace — so what the envelope carries there is `null`, and that is
// asserted rather than skipped (§8, §9, mcp's truncatedOf).
func compareTruncation(t *testing.T, c, stream goldenCase, terminal string) {
	t.Helper()

	lifted := readEnvelope(t, c.dir).StructuredContent.truncated()
	carried := memberOf(t, terminal, "truncated")
	if carried == "" {
		if lifted != "null" {
			t.Errorf("case %s carries truncated %s and %s ends in a row carrying none; a Run has no result set for a cap to have cut", c.name, lifted, stream.name)
		}
		return
	}

	held, wrote := marker(t, lifted), marker(t, carried)
	if held == nil || wrote == nil {
		if lifted != carried {
			t.Errorf("case %s carries truncated %s and %s writes %s", c.name, lifted, stream.name, carried)
		}
		return
	}
	compareHints(t, c, stream, held["hint"], wrote["hint"])
	delete(held, "hint")
	delete(wrote, "hint")
	if !reflect.DeepEqual(held, wrote) {
		t.Errorf("case %s carries the marker %v and %s writes %v; the hint is the only member that differs between the two surfaces", c.name, held, stream.name, wrote)
	}
}

// compareHints is the exception stated rather than tolerated. Setting the
// member aside and comparing the rest would let the exception be used the wrong
// way round: a hint that regressed to naming `--since` would point a caller at
// a flag no schema declares, and a fence that only deleted the key would pass
// it (§9, render.Narrowing).
//
// So what is held is the difference itself. The hint is written on both
// surfaces or on neither; the terminal's names **flags** and opens `--`; and
// the envelope's names the tool's **arguments** and carries no flag spelling
// anywhere in it.
func compareHints(t *testing.T, c, stream goldenCase, lifted, written any) {
	t.Helper()

	if (lifted == nil) != (written == nil) {
		t.Errorf("case %s carries the hint %v and %s writes %v; the hint is written on both surfaces or on neither", c.name, lifted, stream.name, written)
		return
	}
	if lifted == nil {
		return
	}

	tool, terminal := lifted.(string), written.(string)
	if strings.Contains(tool, flagOpening) {
		t.Errorf("case %s carries the hint %q; a tool's hint names its arguments, and nothing on this surface takes a flag", c.name, tool)
	}
	if !strings.Contains(terminal, flagOpening) {
		t.Errorf("%s writes the hint %q; the terminal's names the flags a person types, which is what the exception is an exception to", stream.name, terminal)
	}
}

// flagOpening is what a flag is spelled with on the command line and nowhere on
// the second surface. It is the whole of the difference the truncation marker's
// two hints carry, so it is what the comparison above is made of.
const flagOpening = "--"

// marker is one `truncated` member as an object, and nil where it is not one:
// the member is the marker on a cut result and a bare boolean or `null`
// otherwise, and only the first has members to set a hint aside from.
func marker(t *testing.T, encoded string) map[string]any {
	t.Helper()

	if !strings.HasPrefix(encoded, "{") {
		return nil
	}
	var held map[string]any
	if err := json.Unmarshal([]byte(encoded), &held); err != nil {
		t.Fatalf("%s is not one JSON value: %v", encoded, err)
	}
	return held
}

// memberOf is one member of one row as it was written, and the empty string
// where the row carries none. It reads the raw value rather than a decoded one
// so that what is compared is what crossed.
func memberOf(t *testing.T, row, name string) string {
	t.Helper()

	var held map[string]json.RawMessage
	if err := json.Unmarshal([]byte(row), &held); err != nil {
		t.Fatalf("%s is not one JSON value: %v", row, err)
	}
	return string(held[name])
}

// endsAStream says whether one line is §8's terminal row: a `result` or an
// `outcome`, which are the two §8 states and the two an envelope does not carry
// among its rows.
func endsAStream(line string) bool {
	var row struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return false
	}
	return row.Type == "result" || row.Type == "outcome"
}

// envelopeRows is a case's checked-in envelope rows, each as the one line the
// corpus holds it on.
func envelopeRows(t *testing.T, dir string) []string {
	t.Helper()

	held := readEnvelope(t, dir).StructuredContent
	if held == nil {
		return nil
	}
	lines := make([]string, 0, len(held.Rows))
	for _, row := range held.Rows {
		lines = append(lines, string(row))
	}
	return lines
}

// structuredHalf is the machine half of a checked-in envelope: §8's rows as the
// corpus holds them, the terminal fact the envelope lifted beside them, and the
// page on the one tool whose block is one (ADR-0100).
type structuredHalf struct {
	Outcome   string            `json:"outcome"`
	Rendering string            `json:"rendering"`
	Rows      []json.RawMessage `json:"rows"`
	Truncated json.RawMessage   `json:"truncated"`
}

// truncated is the terminal fact the envelope lifted, raw. It answers the empty
// string where the envelope carries no structured half at all, which is a third
// answer beside §9's shapes and is the one every caller has to tell apart: a
// tool that declined opened no stream, so there is nothing for the member to
// have been lifted out of (ADR-0102).
func (s *structuredHalf) truncated() string {
	if s == nil {
		return ""
	}
	return string(s.Truncated)
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
// and **the members that name a row within that type** — which is each row
// type's own lookup key and not a fixed pair.
//
// The enumeration grows with the row types, and it has to: a key that named
// less would be a fence quietly holding one row against another (issue #197).
// A Manifest header row carries no name, so two Manifests would otherwise be
// one row seen twice; an `operation_detail` row carries neither name nor
// digest, so every Operation in the corpus would collapse onto one key; and the
// Authoring pair's rows carry none of the three, so every `gutter` line
// anywhere would be one key and an envelope's would be held against whichever
// was read last (issue #198). What names each:
//
//	provider, target, manifest, operation   name, digest
//	operation_detail                        source
//	probe_result                            provider, operation
//	artefact                                kind, path
//	gutter                                  line
//	authority                               definition, target
//	flag                                    flag, cites_line
//	problem                                 file, line, error_code
//	run                                     id
//	entry                                   run_id
//	step, provenance                        step
//	refusal                                 error_code, step, file, line, field
//	remediation                             file, line, field
//	window                                  procedure
//	asset, observation                      target, definition, name
//	code                                    fact, subject
//	record                                  key, ordinal, run_id
//
// A `problem`'s three are exactly the key problem.Compare orders on, which is
// the fence borrowing the command's own answer to *which problem is this*
// rather than inventing a second one. The `source` is the declaring lines
// themselves, which is a long key and the honest one: that row has no shorter
// member saying **which** Operation it is about.
//
// **A `probe_result` is named by the two positionals its call resolved** (issue
// #201). It carries no `name` and no `digest` — its members are a Provider, an
// Operation, a projection and a response — so without them every Probe in the
// corpus is one key, and a `metrics window` envelope would be held against the
// `uptime check_http` streams as readily as against its own.
//
// The three `uptime check_http` cases **do** share one key, and that is the
// enumeration working rather than falling short: they are three renderings of
// one identity — a `503`, a `200`, and a host that answered nothing — and what
// the fence holds is that an envelope's row is one of the renderings the stream
// writes, which is why each of them has a `--json` twin written for it
// (testdata/probe/README.md).
//
// **The Inspection four's rows are what made `step` a member** (issue #199).
// Two Steps of one Run against one Definition and one Target differ in their
// position and in nothing else this list held, so a `run_show` envelope's
// second Step was being held against its first — which is exactly the quiet
// pairing the enumeration exists to prevent. `provenance` takes the same member
// for the same reason, that being the split the wire already states between its
// two scopes (ADR-0043), and a `record` takes its identity and its ordinal,
// a history being many versions of one key.
//
// The members are read into one flat struct rather than switched on per type,
// because a key is the members a row happens to carry: a row type that carries
// none of them beyond its `type` is one this list has not caught up with, which
// is a fence to widen rather than a case to special-case.
//
// It is the empty string for a line that is not a row at all — a brace, a text
// block, and the terminal row a stream carries and an envelope does not.
func rowIdentity(line string) string {
	var row struct {
		Type       string `json:"type"`
		Name       string `json:"name"`
		Digest     string `json:"digest"`
		Source     string `json:"source"`
		Provider   string `json:"provider"`
		Operation  string `json:"operation"`
		Kind       string `json:"kind"`
		Path       string `json:"path"`
		Line       int    `json:"line"`
		Definition string `json:"definition"`
		Target     string `json:"target"`
		Flag       string `json:"flag"`
		CitesLine  int    `json:"cites_line"`
		File       string `json:"file"`
		ErrorCode  string `json:"error_code"`
		ID         string `json:"id"`
		RunID      string `json:"run_id"`
		Step       int    `json:"step"`
		Field      string `json:"field"`
		Procedure  string `json:"procedure"`
		Fact       string `json:"fact"`
		Subject    string `json:"subject"`
		Ordinal    int    `json:"ordinal"`
		// Key is a Record's identity, which is the one key here that is
		// nested: a Record is identified by three names together, and
		// three siblings of `ordinal` would be three names that happen
		// to be adjacent (§2, §9).
		Key struct {
			Target     string `json:"target"`
			Definition string `json:"definition"`
			Name       string `json:"name"`
		} `json:"key"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return ""
	}
	switch row.Type {
	case "", "result", "outcome", "text":
		return ""
	}
	return strings.Join([]string{
		row.Type, row.Name, row.Digest, row.Source, row.Provider, row.Operation, row.Kind, row.Path,
		strconv.Itoa(row.Line), row.Definition, row.Target, row.Flag,
		strconv.Itoa(row.CitesLine), row.File, row.ErrorCode,
		row.ID, row.RunID, strconv.Itoa(row.Step), row.Field,
		row.Procedure, row.Fact, row.Subject, strconv.Itoa(row.Ordinal),
		row.Key.Target, row.Key.Definition, row.Key.Name,
	}, "\x00")
}

// twinOf is the case one directory up that a call case is paired against:
// `mcp/<tool>/<case>` against `<command>/<case>`, one fixture repository driven
// two ways. It answers false for a case that is not under `mcp/` at all.
//
// A tool is named for the command it carries, so the two directory names are
// the same word — with one exception, and that exception is why this is a
// function rather than a `strings.CutPrefix` at each pairing. §9 names
// `run_show` differently from its command: a client holds every server's tools
// in one flat namespace, where a bare `show` names nothing. A pairing that
// looked under `testdata/run_show/` would find no twin, hold the case against
// nothing, and pass — which is the failure a fence is least able to notice
// (issue #199).
func twinOf(name string) (string, bool) {
	named, under := strings.CutPrefix(name, "mcp/")
	if !under {
		return "", false
	}
	tool, held, _ := strings.Cut(named, "/")
	if command, differs := commands[tool]; differs {
		tool = command
	}
	return filepath.Join("testdata", tool, filepath.FromSlash(held)), true
}

// commands is every tool whose name is not its command's, which §9 states as
// exactly one: the rest are spelled alike and are not listed, a table of
// identities being a table to keep in step for nothing.
var commands = map[string]string{"run_show": "show"}

// TestGoldenCorpora_AReviewsTextBlockIsWhatTheCLIWroteOnStdout is §9's
// text-block table held where `review`'s row is: **`review` carries the full
// rendered review surface**, and what makes that checkable is that the two
// surfaces render one page (§9, ADR-0026, issue #198).
//
// It is the stdout half of the pairing
// TestGoldenCorpora_WhatDeclinesInAnEnvelopeIsWhatTheCLIWroteOnStderr makes
// over the calls that decline, and it is pairing **by name** for that fence's
// own reason: there is nothing in a rendering to collect it by, so
// `mcp/review/<case>` is held against `review/<case>` — one fixture repository,
// one artefact, driven two ways.
//
// **It holds over `review` alone because `review` alone hands back the page and
// nothing else.** `check` is in the table too and is fenced beneath this one,
// where the assertion is the same fact in a different shape: its block is a
// summary line the CLI does not write, and then the page (issue #214).
//
// **It holds the same page against `structuredContent.rendering`**, which is
// the one member of the envelope that carries a rendering rather than a fact
// (ADR-0100, issue #217). The two channels are one string composed once, so
// what the fence adds is a third reading of one page and not a second
// assertion: the CLI's stdout, the text block, and the structured member all
// have to be the same bytes or the case fails.
//
// **A `review` a guardrail declined is not this row of the table**, and it is
// skipped rather than excused. §9's table is keyed on the tool for every path
// the tool *answers at all*, and a Refusal is the table's own fourth row: the
// command writes it where a person reads it and leaves stdout silent, so there
// is no page on either surface to hold the other against. What holds those
// cases is the stderr pairing above, which collects a Refusal by its opening
// across the whole corpus.
func TestGoldenCorpora_AReviewsTextBlockIsWhatTheCLIWroteOnStdout(t *testing.T) {
	var compared int
	for _, c := range goldenCases(t) {
		if c.call == nil || c.call.Tool != "review" || c.answersAProtocolError() {
			continue
		}
		if strings.HasPrefix(textBlock(t, c.dir), refusalOpening) {
			continue
		}
		twin, paired := twinOf(c.name)
		if !paired {
			continue
		}
		wrote := readFile(t, filepath.Join(twin, "stdout.golden"))
		if wrote == "" {
			t.Errorf("case %s has no twin at %s writing the page on stdout; the text block is held against nothing", c.name, twin)
			continue
		}
		compared++
		if got := textBlock(t, c.dir); got != wrote {
			t.Errorf("case %s: its text block is\n  %q\nand %s writes\n  %q", c.name, got, twin, wrote)
		}
		if got := readEnvelope(t, c.dir).StructuredContent.Rendering; got != wrote {
			t.Errorf("case %s: its structured content carries\n  %q\nand %s writes\n  %q; the page travels on both channels", c.name, got, twin, wrote)
		}
	}
	if compared == 0 {
		t.Fatal("no review case was paired against the page its command writes; the rule was held over nothing")
	}
}

// TestGoldenCorpora_AChecksTextBlockIsItsSummaryLineAndThenWhatTheCLIWroteOnStdout
// is the same fence over `check`'s row of the table: **its rows travel in the
// text block**, drawn by the one renderer both surfaces answer through, so the
// block is a line of this surface's own and then the command's page byte for
// byte (§9, ADR-0026, ADR-0097, issue #214).
//
// **It is total over the check cases rather than a pairing where one happens to
// exist**, which is the difference between this and the fence above it. A case
// with no rows has no page to hold — a clean `check` writes a sentence about a
// count, and §9 gives the block the summary line alone — so what is asserted
// there is that the block is that one line and carries no table at all. The two
// arms together are the rule: *the summary line always, the rows beneath it
// wherever there are rows*, with no case falling through the middle.
//
// **A twin that opens a row stream is not the twin**, and it is skipped by
// asking the twin rather than by naming the case: `--json` writes §8's NDJSON
// to stdout, which is the same rows in the other of the two forms one renderer
// produces, and holding a table against a stream would be this fence asserting
// that the two forms are one string.
func TestGoldenCorpora_AChecksTextBlockIsItsSummaryLineAndThenWhatTheCLIWroteOnStdout(t *testing.T) {
	cases := goldenCases(t)
	named := make(map[string]goldenCase, len(cases))
	for _, c := range cases {
		named[c.name] = c
	}

	var compared int
	for _, c := range cases {
		if c.call == nil || c.call.Tool != "check" || c.answersAProtocolError() {
			continue
		}
		block := textBlock(t, c.dir)
		if len(readEnvelope(t, c.dir).StructuredContent.Rows) == 0 {
			if strings.Contains(block, "\n") {
				t.Errorf("case %s found nothing and its text block is\n  %q\nwant the summary line alone: what goes beneath the line is the row set", c.name, block)
			}
			continue
		}
		twin, paired := twinOf(c.name)
		if !paired {
			continue
		}
		if pair, held := named[strings.TrimPrefix(twin, "testdata"+string(filepath.Separator))]; held && pair.opensARowStream() {
			continue
		}
		wrote := readFile(t, filepath.Join(twin, "stdout.golden"))
		if wrote == "" {
			continue
		}
		compared++
		line, table, cut := strings.Cut(block, "\n\n")
		if !cut || table != wrote {
			t.Errorf("case %s: its text block is\n  %q\nwant a summary line, a blank line, and then what %s writes:\n  %q", c.name, block, twin, wrote)
		}
		if strings.Contains(line, "\n") {
			t.Errorf("case %s: what stands above its table is %q, want one line", c.name, line)
		}
	}
	if compared == 0 {
		t.Fatal("no check case that found rows was paired against the page its command writes; the rule was held over nothing")
	}
}

// TestGoldenCorpora_TheAuthoringToolsAreDrivenWithNothingButARepository is §9's
// own sentence about the pair held as a property of every case that drives
// them: `check` and `review` *reach nothing* — no credential resolves, no
// network is touched, and nothing is invoked — so **both answer with no
// credential present in the environment at all**.
//
// The whole environment is asserted rather than a list of names that look like
// credentials, because a list is a guess about spelling and this is not: what a
// case may set is the variable that fixes the repository, and a case that
// needed a second one would be a case claiming one of the two tools reads
// something. That is the claim worth catching, whatever the variable is called.
//
// It is a fence rather than a reading of the tools because the tools have
// nothing to read: neither resolves a credential today, and what would change
// that is a repository fixture written to need one.
func TestGoldenCorpora_TheAuthoringToolsAreDrivenWithNothingButARepository(t *testing.T) {
	var read int
	for _, c := range goldenCases(t) {
		if c.call == nil || (c.call.Tool != "check" && c.call.Tool != "review") {
			continue
		}
		read++
		for name := range c.variables(t) {
			if name != "HYPER_REPO_DIR" {
				t.Errorf("case %s sets %s; the two tools that reach nothing answer with no credential in the environment at all", c.name, name)
			}
		}
	}
	if read == 0 {
		t.Fatal("no case under testdata/mcp/ drives check or review; the rule was held over nothing")
	}
}

// TestGoldenCorpora_ACheckCallNamesItsPathsAgainstTheRepository is an absence
// held as a rule: a call that names a path carries **no `wd`**, because the
// path is read against the repository root and not against the directory the
// client started the server in (ADR-0089, issue #205).
//
// It is a fence rather than a line in the README because nothing else would
// notice it lapsing — a case that pinned its working directory to the
// repository would pass exactly as it does now, and the corpus would quietly
// stop driving the case the decision is about. What these cases run from is a
// directory that is not the repository at all, which is the one thing a client
// cannot state and this argument therefore cannot depend on.
func TestGoldenCorpora_ACheckCallNamesItsPathsAgainstTheRepository(t *testing.T) {
	var named int
	for _, c := range goldenCases(t) {
		if c.call == nil || c.call.Tool != "check" {
			continue
		}
		var arguments struct {
			Paths []string `json:"paths"`
		}
		if len(c.call.Arguments) > 0 {
			if err := json.Unmarshal(c.call.Arguments, &arguments); err != nil {
				t.Fatalf("case %s: its call's arguments are not check's: %v", c.name, err)
			}
		}
		if len(arguments.Paths) == 0 {
			continue
		}
		named++
		if isFile(filepath.Join(c.dir, "wd")) {
			t.Errorf("case %s names a path and carries a wd; a path is read against the repository root, and a case stating a working directory drives the one case where the two agree", c.name)
		}
	}
	if named == 0 {
		t.Fatal("no call case names a path for check; the rule was held over nothing")
	}
}
