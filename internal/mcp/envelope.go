package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// The return envelope: the one shape every tool answers with (§9, issue #195).
//
// **hyper composes it and the SDK carries it.** The text block, the rows, the
// truncation and the bit are decided here, out of the answer the command
// produced; what the SDK does with the value is frame it. That division is what
// the low-level registration buys — the generic one would compose an envelope
// of its own out of a Go type — and it is why nothing in this file is expressed
// in the SDK's terms but `result` at the end of it, which is the crossing
// itself. What comes back is read off the wire and not out of an SDK value at
// all (envelopeFrom).

// Envelope is hyper's own reading of what a tool call answered: the shape §9
// fixes, and the shape the corpus holds.
//
// It is a type of this package's rather than the SDK's CallToolResult for two
// reasons that are one reason. The result carries members this surface never
// sets — an input request, a request state, a result type — so a corpus holding
// one would be asserting the SDK's shape rather than hyper's. And `isError` is
// `omitempty` there, so a result that carries the bit as `false` and one that
// does not carry it at all are the same value: §9 makes that bit the thing a
// Refusal is told from an ordinary return by, so it is written **always**, and
// a shape that could drop it is not the shape to state it in.
type Envelope struct {
	// Content is the unstructured half, and on every tool this milestone
	// builds it is one text block. §9's asymmetry — a summary line, a
	// review's full rendering, a Refusal's full rendering — is a fact about
	// which tool answered and which path it took, so the member is a list
	// and the composition is below.
	Content []TextBlock `json:"content"`
	// StructuredContent is the machine half: §8's rows, and the terminal
	// fact moved up beside them.
	StructuredContent Structured `json:"structuredContent"`
	// IsError means only *you did not get what you asked for*, which is true
	// of a Refusal and of a failure alike; it is not the outcome
	// discriminator, one bit not carrying three states (§9). It is false
	// throughout this ticket — the paths that decline are issue #196's — and
	// it is written whichever it is.
	IsError bool `json:"isError"`
}

// TextBlock is one member of `content`: MCP's text content, and the only
// content type this surface produces. hyper answers rows and renderings, and
// neither is an image, an audio clip or a resource link.
type TextBlock struct {
	// Type is the content type, and it is `text` on every block this
	// surface writes. It is carried rather than assumed because it is a
	// discriminator a client reads, and a block that arrived as anything
	// else is a block this surface did not compose (envelopeFrom).
	Type string `json:"type"`
	// Text is what §9's asymmetry decides: one summary line on an ordinary
	// return, `review`'s full rendering, or a Refusal's.
	Text string `json:"text"`
}

// Structured is `structuredContent`: §8's row set as an array, and the terminal
// fact beside it.
//
// **There is no terminal row inside `rows`.** An array's end is already its own
// end-of-stream marker — the thing §8's terminal row exists to be on a line
// stream with no framing — so what the terminal row *carried* moves up here and
// the row itself does not travel (§9).
//
// Both members are carried as raw JSON because both are already decided
// elsewhere and this surface may not re-decide them. A row's key order, its
// absent members and its unabbreviated digests are §8's contract held by the
// row type that declared them (render.Row); `truncated` is the terminal row's
// own member, in whichever of §9's three shapes its command wrote — the bare
// boolean on a namespace listing with no axis to name, and the marker object on
// a command whose parameters can narrow what it cut. Re-typing either here
// would be this package holding a second opinion about a shape it does not own.
type Structured struct {
	// Rows is §8's row set unchanged: one object per row, carrying the same
	// `type` discriminator, served as an array rather than a line stream. It
	// is `[]` where the command found nothing rather than absent, which is
	// why it is built with a length and never left nil.
	Rows []json.RawMessage `json:"rows"`
	// Truncated is what the terminal row carried, bare.
	Truncated json.RawMessage `json:"truncated"`
}

// envelopeOf composes the envelope from one command's answer.
//
// The rows are marshalled by render's own writer, so a row in an envelope and
// the same row on the `--json` stream are the same bytes: the two surfaces
// cannot state different things because there is one row set and one encoder
// behind both (ADR-0026).
func envelopeOf(answered Answer) (Envelope, error) {
	rows, kinds, err := marshalRows(answered.Rows)
	if err != nil {
		return Envelope{}, err
	}
	truncated, err := truncatedOf(answered.Terminal)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		Content:           []TextBlock{{Type: "text", Text: summary(kinds, wasCut(truncated))}},
		StructuredContent: Structured{Rows: rows, Truncated: truncated},
	}, nil
}

// marshalRows is the row set as it goes on the wire, and the discriminators in
// the order they appeared beside it.
//
// The kinds are read back out of the encoded row rather than off the value,
// which is deliberate: §8 fixes that **a row's `type` is its first key**, so the
// encoded row is where that fact is true, and a summary composed from anything
// else would be composed from a fact no consumer can see. render.Row itself
// declares only Cells, there being no method on it that answers what kind of
// row it is — the discriminator is a member of each row type, and the wire is
// where the member is.
func marshalRows(rows []render.Row) ([]json.RawMessage, []string, error) {
	// Never nil: `rows` is `[]` where the command found nothing rather than
	// absent (§9), and a nil slice marshals as null.
	encoded := make([]json.RawMessage, 0, len(rows))
	kinds := make([]string, 0, len(rows))
	for _, row := range rows {
		line, err := render.MarshalRow(row)
		if err != nil {
			return nil, nil, err
		}
		var discriminated struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &discriminated); err != nil {
			return nil, nil, err
		}
		encoded = append(encoded, line)
		kinds = append(kinds, discriminated.Type)
	}
	return encoded, kinds, nil
}

// truncatedOf is the terminal row's `truncated` member, lifted whole.
//
// It is read out of the encoded terminal row for marshalRows' own reason: which
// of §9's three shapes a command wrote is a decision render.Truncation already
// made and already knows how to write, and a reader here that switched on the
// value would be a second implementation of that choice. Lifting the member
// carries the bare boolean and the marker object alike, and carries whatever
// §9 adds to it next without this file changing.
func truncatedOf(terminal render.Row) (json.RawMessage, error) {
	encoded, err := render.MarshalRow(terminal)
	if err != nil {
		return nil, err
	}
	var carried struct {
		Truncated json.RawMessage `json:"truncated"`
	}
	if err := json.Unmarshal(encoded, &carried); err != nil {
		return nil, err
	}
	if carried.Truncated == nil {
		return nil, fmt.Errorf("the terminal row %s carries no truncated member", encoded)
	}
	return carried.Truncated, nil
}

// wasCut answers whether a result was truncated, from the member itself: anything
// that is not the literal `false` is a cut result, the bare true and the marker
// object being the two other shapes §9 admits and both of them meaning one.
//
// It reads the JSON rather than the value it came from because the value is not
// in hand here — what crossed the boundary is the member — and because the
// question this answers is the one a consumer asks of the same bytes.
func wasCut(truncated json.RawMessage) bool {
	return !bytes.Equal(bytes.TrimSpace(truncated), []byte("false"))
}

// summary is the ordinary return's text block: **one summary line, outcome
// first** (§9).
//
// What it counts is the rows, by their own discriminator, in the order the
// kinds first appeared — `3 Providers` for a listing, `1 Manifest, 6
// Operations` for a Manifest and the Operations beneath it. That is composed
// from the answer rather than declared per tool on purpose: a tool is a schema
// and an argv, and a noun it carried would be a tool holding an opinion about
// what its command found. The nouns are §8's row types spelled as the glossary
// spells them, which is a property of the row set and not of any one caller.
//
// **A cut result says so here as well as in the structured content.** §9 is
// flat about it — *a truncated result must never look complete* — and the CLI
// keeps that promise with a line on stderr, which this surface does not have: a
// tool's narration goes to io.Discard, so the text block is the only place left
// for the sentence. It names no counts because the bare boolean carries none:
// a namespace listing has no axis, and inventing a remedy the marker did not
// state would point a caller at a parameter their tool does not take.
func summary(kinds []string, truncated bool) string {
	if len(kinds) == 0 {
		return "no rows"
	}

	counted := map[string]int{}
	var order []string
	for _, kind := range kinds {
		if counted[kind] == 0 {
			order = append(order, kind)
		}
		counted[kind]++
	}

	counts := make([]string, 0, len(order))
	for _, kind := range order {
		counts = append(counts, fmt.Sprintf("%d %s", counted[kind], noun(kind, counted[kind])))
	}
	line := strings.Join(counts, ", ")
	if truncated {
		line += ", truncated"
	}
	return line
}

// nouns is what each of §8's row types is called in prose, singular, spelled as
// CONTEXT.md spells it. It is a table rather than a rule over the discriminator
// because the discriminators are `snake_case` machine names and the glossary's
// words are not derivable from them.
//
// A type with no entry reads as its own discriminator. That is legible, wrong
// enough to be noticed by whoever adds the row type, and better than either
// alternative: a fabricated noun would put a word in the glossary's mouth, and
// a refusal would fail a call over the prose beside an answer that is otherwise
// correct.
var nouns = map[string]string{
	"provider":  "Provider",
	"manifest":  "Manifest",
	"operation": "Operation",
}

// noun is one row type in prose, pluralised by count. The plural is the English
// `s` and nothing cleverer: every noun in the table above takes it, and a row
// type that does not is one whose entry says so.
func noun(kind string, count int) string {
	word, named := nouns[kind]
	if !named {
		word = kind
	}
	if count == 1 {
		return word
	}
	return word + "s"
}

// result is the envelope as the SDK carries it, and it is the whole of the
// crossing: every member is already composed, and nothing is left for the SDK
// to decide.
//
// StructuredContent goes over as the value rather than as bytes because the SDK
// marshals the result whole and a json.RawMessage would arrive as a string;
// what matters is that the shape it marshals is this package's own, declared
// above, and not a domain type it was handed to reflect over.
func (e Envelope) result() *sdk.CallToolResult {
	content := make([]sdk.Content, 0, len(e.Content))
	for _, block := range e.Content {
		content = append(content, &sdk.TextContent{Text: block.Text})
	}
	return &sdk.CallToolResult{
		Content:           content,
		StructuredContent: e.StructuredContent,
		IsError:           e.IsError,
	}
}

// envelopeFrom is the crossing read back: the `result` frame the client read,
// as hyper's own shape.
//
// It is the corpus's half of Call, and it is a **decode and not a second
// composition** — every member is taken from the bytes that arrived, with
// nothing recomputed and nothing re-encoded. That is what preserves §8's key
// order: `rows` is a list of raw messages, so each row is the object the server
// wrote, byte for byte.
//
// **The frame is read in two passes, and the division is the boundary this
// package draws.** The result object is the protocol's — the SDK attaches a
// `_meta` and a result type of its own, and hyper neither writes nor reads
// them — so members it does not state are passed over there. The structured
// content is entirely hyper's composition, so it is read **closed**: a member
// arriving there that this surface does not state is a member that reached the
// wire from somewhere other than envelopeOf, which is exactly the fault a
// corpus is least able to notice by looking at its golden files.
//
// `isError` absent is `isError: false`. The bit is `omitempty` on the wire —
// MCP's own convention, absence meaning the call succeeded — and hyper states
// it always in the shape it publishes, which is the asymmetry this function
// resolves in one direction on purpose.
func envelopeFrom(frame json.RawMessage) (Envelope, error) {
	var result struct {
		Content           []TextBlock     `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
	}
	if err := json.Unmarshal(frame, &result); err != nil {
		return Envelope{}, err
	}
	for _, block := range result.Content {
		if block.Type != "text" {
			return Envelope{}, fmt.Errorf("the envelope carries %q content; this surface answers text", block.Type)
		}
	}

	envelope := Envelope{Content: result.Content, IsError: result.IsError}
	decoder := json.NewDecoder(bytes.NewReader(result.StructuredContent))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope.StructuredContent); err != nil {
		return Envelope{}, err
	}
	if envelope.StructuredContent.Rows == nil {
		envelope.StructuredContent.Rows = []json.RawMessage{}
	}
	return envelope, nil
}
