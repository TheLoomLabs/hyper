package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TheLoomLabs/hyper/internal/exit"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
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
	// discriminator, one bit not carrying three states (§9). Which exit code
	// sets it is the mapping's to say, and it is written whichever it is.
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
	// Outcome is §12's triple, and it is `run`'s alone among the thirteen:
	// a tool that is not a Run carries no `outcome` key at all, and this is
	// the member that is absent there (§9, issue #200). It is **not**
	// restated by any row and it is not `isError` — one bit does not carry
	// three states.
	Outcome string `json:"outcome,omitempty"`
	// RunID is the entry the Run wrote, whole (ADR-0047), and absent
	// exactly where no entry was written: the version pin gate and the
	// bootstrap `store-absent`, the two paths that decline before a Run is
	// identified, and the 75s that stand before `run.json` (§8, §9).
	RunID string `json:"run_id,omitempty"`
	// DryRun rides beside Outcome and is written **wherever Outcome is**,
	// the bare `false` included — §7's one exception to the absence rule
	// holding on this surface for the reason it holds in the Store: what a
	// reader that takes its absence for `false` gets wrong is unrecoverable.
	//
	// It is a pointer for that reason and no other. `false` and *this tool
	// is not a Run* are two different answers, and a bare bool could only
	// say one of them.
	DryRun *bool `json:"dry_run,omitempty"`
	// Rows is §8's row set unchanged: one object per row, carrying the same
	// `type` discriminator, served as an array rather than a line stream. It
	// is `[]` where the command found nothing rather than absent, which is
	// why it is built with a length and never left nil.
	Rows []json.RawMessage `json:"rows"`
	// Truncated is what the terminal row carried, bare.
	Truncated json.RawMessage `json:"truncated"`
}

// execution is what a call says about itself **before its command runs**: that
// the tool carries §12's triple at all, and — where it does — whether the
// invocation named a rehearsal (§9, issue #200).
//
// It exists for one path, and the path is the reason it cannot be read off the
// answer. A Run that lost the Store to the lock or to the sync at Run start
// stands before `run.json` and writes no terminal row at all, so there is
// nothing there to lift an outcome from; §12 still fixes what `75` carries, and
// this is the other operand that arm needs — `dry_run` is the invocation's, and
// a path that guessed it `false` would tell a caller their rehearsal reached
// the world (run.go, reportLockFault).
//
// **It travels as a pointer, and the nil is the whole of *this tool is not a
// Run*.** A tool declares an execution half or it does not (tools.go), and a
// bit inside this value restating that would be one fact in two places — the
// day they disagree, a listing acquires an outcome.
type execution struct {
	// dryRun is what this call named, and it is read only where the command
	// wrote no row carrying its own answer.
	dryRun bool
}

// outcomeMembers is the envelope's execution half as one value: the three
// members above, in the shape the terminal `outcome` row already writes them.
//
// It is decoded straight from that row rather than composed beside it, which is
// what makes the envelope's `outcome` and the CLI's `outcome` row one fact:
// the members are lifted, not recomputed, and a Run that reported one thing on
// the terminal cannot report another here (§9, ADR-0026).
//
// **Two of the row's members stay behind, and neither is an oversight.** The
// exit code is the fact this surface does not have — §9 states the envelope
// without one, and outcomeSummary says what stands in its place. `error_code`
// is the head of the Refusal's array, derived on the row and stored nowhere,
// and a `refusal` row carries it per member: lifting it would put a second
// representation of the array's first member beside the array, which is the
// thing the CLI declines to store for the same reason (§7, §8, cli's
// outcomeRow). Where the array is empty the key is absent there too — the pin
// gate and the bootstrap `store-absent` write no rows on either surface, and
// the check that declined is named in the Refusal rendering the text block
// carries whole, which is where §9 puts a Refusal's identity on every one of
// the thirteen.
type outcomeMembers struct {
	Outcome string `json:"outcome"`
	RunID   string `json:"run_id"`
	DryRun  *bool  `json:"dry_run"`
}

// outcomeType is the terminal row a Run writes, by its own discriminator. §8
// fixes two terminal rows and makes the type itself the discriminator — a Run
// emits `outcome` and everything else emits `result` — so this is the whole of
// what tells the two apart on the wire.
const outcomeType = "outcome"

// envelopeOf is §9's mapping in full: the exit code one command returned, and
// the envelope this surface answers with (§9, issue #196).
//
// **It is stated once and reached by every tool**, through the one handler
// behind all of them (server.go). A surface with no exit code has to say four
// things the exit code said, and §9 states thirteen tools: a tool reading the
// code for itself would be one of thirteen readings of §12's closed set where
// the surface has exactly one. The table, in full:
//
//	 0   the answer; isError: false
//	 1   the answer; isError: true — a Run the world resisted, or a command
//	     reporting problems it found
//	 2   no envelope: a JSON-RPC error carrying what the command wrote
//	75   the answer; isError: true — the Store lost, and the one code a
//	     caller may retry
//	77   the answer; isError: true — a Refusal rendered in full as text
//
// **The structured half is composed the same way whichever arm is taken**, and
// what the exit code decides is the text block and the bit. That is not a
// shortcut: a command that Refuses may still have written a terminal row — §8
// puts `run` on the `outcome` side on every path on which a Run was attempted,
// the two that decline before a Run is identified included — and an arm that
// composed an empty structured half for `77` would drop the one fact §9 moves
// up into it.
//
// **The exit code decides which text block, and the tool decides what an
// ordinary one carries**, which is the second half of §9's asymmetric table
// arriving here: `77` is the Refusal rendered whole and everything above it is
// answerText's, over a bit the tool set supplies (tools.go, issue #198).
//
// **`130` and `143` are unreachable**: the server installs no signal watch, so
// a command answering one is a fault in the server rather than an envelope this
// surface knows how to compose. It travels as a protocol error, which is where
// §9 puts a fault in the server, and it is stated rather than dressed as a
// domain answer — a wrong envelope is harder to notice than a missing one.
func envelopeOf(answered Answer, rendersInFull bool, called *execution) (Envelope, error) {
	structured, kinds, err := structuredOf(answered, called)
	if err != nil {
		return Envelope{}, err
	}

	switch answered.Exit {
	case exit.Clean, exit.Problems, exit.StoreLost:
		// The three codes that answer, and one arm because they answer
		// **one shape**: the structured half composed the same way, the
		// text block composed the same way — answerText's, over the
		// tool rather than over the code — and the bit set on the two
		// that mean *you did not get what you asked for*. `isError` is
		// true of a failure as much as of a Refusal, and nothing in the
		// structured content restates it (§9).
		//
		// **`1` and `75` are one piece of news on this surface**, and
		// deliberately: §12 maps both onto `failed`, and what §9 gives
		// a caller to act on is the triple rather than the code. What
		// separates *retry me* from *a verbatim retry refuses
		// identically* is `failed` standing where `refused` would have
		// — an act is required past a `77` and time is all that is
		// required past a `75` — and that distinction survives the loss
		// of the code intact (§9, §12, ADR-0061, issue #200).
		text, err := answerText(answered, rendersInFull, structured, kinds)
		if err != nil {
			return Envelope{}, err
		}
		return Envelope{
			Content:           []TextBlock{{Type: "text", Text: text}},
			StructuredContent: structured,
			IsError:           answered.Exit != exit.Clean,
		}, nil

	case exit.Refused:
		// **The Refusal as the command rendered it, wherever the command
		// rendered it.** Most of §9's sixteen render one where the CLI
		// writes it on stderr and leave stdout silent, and `answered.Refusal`
		// is that buffer. `run` is the exception and it is not a second
		// rendering: a Run that a guardrail declined renders the Refusal
		// *inside its page*, beneath the Step table and above §8's terminal
		// line, so what the command wrote is the page and the page is what
		// the text block carries — Step table, caret excerpt, `=` notes,
		// `EDIT ONE OF` and all (§8, §9, runPage, issue #200).
		//
		// The pin gate and the bootstrap `store-absent` take the first arm
		// even on `run`: both decline through the shared helper that renders
		// a Refusal, and the page they leave behind is §8's terminal line
		// alone (gate.go, reportRunStoreFault).
		rendering := answered.Refusal
		if rendering == "" {
			rendering = answered.Rendering
		}
		if rendering == "" {
			return Envelope{}, fmt.Errorf("the command exited %d and rendered no Refusal, which is a fault in the server: the rendering is the entire way past (ADR-0001)", answered.Exit)
		}
		return Envelope{
			Content:           []TextBlock{{Type: "text", Text: refusalText(rendering)}},
			StructuredContent: structured,
			IsError:           true,
		}, nil

	case exit.Usage:
		// **A domain outcome is never a protocol error, and a usage
		// error is not a domain outcome.** A positional that matches
		// nothing satisfies every schema and still names nothing, and
		// returning it as an answer would give it `isError: true` with
		// no `outcome` key — which is exactly the shape a guardrail
		// declining already returns, on the one surface with no exit
		// code to tell the two apart (§9, ADR-0060).
		//
		// The message is what the command wrote where the CLI writes a
		// human sentence, so an agent reads the sentence a person would
		// have read. An exit code with nothing beside it is a command
		// that declined and said why to nobody, which is this
		// repository's bug rather than a caller's.
		if written := strings.TrimRight(answered.Narration, "\n"); written != "" {
			return Envelope{}, errors.New(written)
		}
		return Envelope{}, fmt.Errorf("the command exited %d and wrote nothing to say why, which is a fault in the server", answered.Exit)

	default:
		return Envelope{}, fmt.Errorf("the command exited %d, which is not a code §9's mapping holds: this server installs no signal watch, and §12 closes the set at seven", answered.Exit)
	}
}

// structuredOf is the machine half of the envelope, composed from what the
// command produced and from nothing the exit code says.
//
// The rows are marshalled by render's own writer, so a row in an envelope and
// the same row on the `--json` stream are the same bytes: the two surfaces
// cannot state different things because there is one row set and one encoder
// behind both (ADR-0026).
func structuredOf(answered Answer, called *execution) (Structured, []string, error) {
	rows, kinds, err := marshalRows(answered.Rows)
	if err != nil {
		return Structured{}, nil, err
	}
	// The terminal row, encoded and discriminated **once**. What it carries
	// goes two ways from here — the truncation marker into `truncated`, the
	// execution members into the three keys beside it — and §8 fixes that
	// the type is what tells the two terminal rows apart, so a second
	// reading would be the same row asked the same question twice.
	terminal, kind, err := discriminate(answered.Terminal)
	if err != nil {
		return Structured{}, nil, err
	}
	truncated, err := truncatedOf(answered.Terminal, terminal, kind)
	if err != nil {
		return Structured{}, nil, err
	}
	structured := Structured{Rows: rows, Truncated: truncated}

	carried, carries, err := executionOf(terminal, kind, called, answered.Exit)
	if err != nil {
		return Structured{}, nil, err
	}
	if carries {
		structured.Outcome, structured.RunID, structured.DryRun = carried.Outcome, carried.RunID, carried.DryRun
	}
	return structured, kinds, nil
}

// executionOf is the envelope's execution members: §12's triple, the rehearsal
// marker written beside it, and the Run id where an entry was written — and
// whether this answer carries them at all (§9, issue #200).
//
// **They are lifted from the terminal `outcome` row wherever the command wrote
// one**, which is every path on which a Run was attempted and the two that
// decline before a Run is identified — the version pin gate and the bootstrap
// `store-absent`, where what is missing is the `run_id` and never the key
// beside it (§8). Lifting rather than recomposing is what makes the envelope's
// `outcome` and the CLI's `outcome` row one fact rather than two that have to
// agree (ADR-0026).
//
// **The one path with no row to lift from is a Run that lost the Store before
// it was attempted**: the lock, and the sync at Run start. Both stand before
// `run.json`, so §8 leaves them writing nothing at all and the CLI's exit code
// carries the news on its own — and this surface has no exit code, so an
// envelope composed the same way would answer `isError: true` with nothing in
// it to say which of the triple happened. §12 fixes what each code carries, so
// the outcome is read from the code and `dry_run` from the call that named it
// (execution, run.go).
//
// A row that carries no `dry_run` is a fault in the server rather than an
// absent marker: §7's one exception to the absence rule is written always, so
// a row without it is a row this repository composed wrongly, and reading its
// silence as `false` is the one reading §7 forbids.
func executionOf(terminal json.RawMessage, kind string, called *execution, code int) (outcomeMembers, bool, error) {
	if kind == outcomeType {
		var carried outcomeMembers
		if err := json.Unmarshal(terminal, &carried); err != nil {
			return outcomeMembers{}, false, err
		}
		if carried.DryRun == nil {
			return outcomeMembers{}, false, fmt.Errorf("the terminal row %s carries no dry_run member, which is written always (§7)", terminal)
		}
		return carried, true, nil
	}
	if called == nil {
		return outcomeMembers{}, false, nil
	}
	outcome, mapped := outcomeFor(code)
	if !mapped {
		return outcomeMembers{}, false, nil
	}
	dryRun := called.dryRun
	return outcomeMembers{Outcome: outcome, DryRun: &dryRun}, true, nil
}

// outcomeFor is §12's exit code read for the outcome it carries — *seven
// members, one per way an invocation can end, each carrying the outcome of the
// triple it maps onto* — and whether the code is one that carries one at all.
//
// **It is the same table cli's exitFor writes in the other direction**, and the
// two are not one function because they are not one question: exitFor maps an
// answer and a signal onto the finer code space, and this maps a code back onto
// the coarser triple. What holds them together is §12, which states both.
//
// It is reached on the one path above and states the whole table anyway,
// because a partial reading is one that answers *no outcome* on a path nobody
// foresaw. `2` is the member that carries none: no Run began, and a usage error
// never reaches an envelope on this surface in any case (§12, ADR-0060). The
// two signals are unreachable here for the reason envelopeOf states — this
// server installs no signal watch — and they are mapped rather than left out,
// a code that arrives being better answered than guessed at.
func outcomeFor(code int) (string, bool) {
	switch code {
	case exit.Clean:
		return string(store.OutcomeCompleted), true
	case exit.Refused:
		return string(store.OutcomeRefused), true
	case exit.Problems, exit.StoreLost, exit.Interrupted, exit.Terminated:
		return string(store.OutcomeFailed), true
	}
	return "", false
}

// answerText is the text block of an ordinary return, which is the first two
// rows of §9's asymmetric table: *any ordinary return* carries one summary
// line, and **`review`** carries the full rendered review surface (§9).
//
// **The table is keyed on the tool and not on what the tool found**, which is
// why the bit comes in from the tool set rather than being read off the answer
// here (tools.go). A tool that renders in full renders on every path that
// answers at all: `review` against an artefact that will not load writes
// `check`'s rows and `check`'s table, and the text block is that table — what
// the block promises is *what the command wrote to stdout*, byte for byte, and
// a reading that swapped in a summary line on one of the command's own paths
// would break the promise exactly where an agent is least able to check it.
//
// The rendering goes over **untouched**, trailing newline and all. It is the
// page as the command's own writer produced it (destination.go), and a text
// block trimmed here would be a rendering this surface had edited — which is
// the one thing §9's *the rendering is the whole of it* forbids. The Refusal
// path trims because it appends a sentence; this appends nothing.
//
// A rendering that is empty on a tool that renders in full is a fault in the
// server rather than an empty answer: the page is the entire content of the
// block, so there is nothing left to say. It is stated rather than carried,
// for the reason the Refusal arm states its own — a wrong envelope is harder
// to notice than a missing one.
func answerText(answered Answer, rendersInFull bool, structured Structured, kinds []string) (string, error) {
	if !rendersInFull {
		// **Outcome first, and on `run` that means the outcome instead of
		// the counts**: §8's terminal line arriving here as a sentence
		// (§9, issue #200). It is selected on the answer rather than
		// declared per tool for summary's own reason — a tool is a schema
		// and an argv — and what selects it is the one thing that is true
		// of a Run's answer and of nothing else: it carries the triple.
		if structured.Outcome != "" {
			return outcomeSummary(structured), nil
		}
		return summary(kinds, wasCut(structured.Truncated)), nil
	}
	if answered.Rendering == "" {
		return "", fmt.Errorf("the command exited %d and rendered no page, which is a fault in the server: this tool's text block is the rendering and nothing else (§9)", answered.Exit)
	}
	return answered.Rendering, nil
}

// retrySentence is what §9 requires **every** Refusal's text block to end with,
// and it is load-bearing rather than manners: `isError: true` conventionally
// invites a retry, and this surface has no exit code `77` with which to say
// otherwise (§9, ADR-0001). The rendering is the only place the protocol leaves
// for saying it.
//
// It names no artefact of its own and points at the rendering above it, because
// that is where the artefacts are named: §8's `EDIT ONE OF` table where the
// Refusal cites a coordinate, and the remedy the message states where it is a
// fact about the invocation — `hyper project`, `hyper store init`, a binary the
// declaration pins. A sentence that composed a second list would be a second
// account of a remedy the check already knows (refusal.go).
const retrySentence = "a verbatim retry refuses identically: no argument to this tool, and no flag anywhere, " +
	"overrides a guardrail (ADR-0001) — the way past is the edit or the command named above."

// refusalText is §9's third text block: **the full rendered Refusal**, and the
// retry sentence after it.
//
// The rendering arrives whole and is not touched. It is §8's, composed where
// every other surface's Refusal is composed, and this surface reads it rather
// than restating it: with no bypass anywhere the Refusal rendering is the
// entire remediation path, and a second composition here is where the two would
// come to say different things (ADR-0001, ADR-0026).
func refusalText(rendering string) string {
	return strings.TrimRight(rendering, "\n") + "\n\n" + retrySentence
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
		line, kind, err := discriminate(row)
		if err != nil {
			return nil, nil, err
		}
		encoded = append(encoded, line)
		kinds = append(kinds, kind)
	}
	return encoded, kinds, nil
}

// discriminate is one row on the wire and the `type` it went out under, and it
// is the **one** reading of a row this package makes: §8 fixes that a row's
// `type` is its first key, so the encoded row is where that fact is true, and
// every question this file asks about which row it has is asked here.
//
// A nil row is no bytes and no type, which is the shape a command that opened
// no row stream at all leaves behind: a usage error, and a guardrail declining
// from a command that is not a Run (§9, ADR-0060).
func discriminate(row render.Row) (json.RawMessage, string, error) {
	if row == nil {
		return nil, "", nil
	}
	encoded, err := render.MarshalRow(row)
	if err != nil {
		return nil, "", err
	}
	var read struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(encoded, &read); err != nil {
		return nil, "", err
	}
	return encoded, read.Type, nil
}

// truncatedOf is the terminal row's `truncated` member, lifted whole — except
// for the marker's hint, which is the one member of an answer §9 spells
// differently here (issue #199).
//
// **The lift is the rule and the hint is the exception, and the exception is
// §9's own.** Which of §9's three shapes a command wrote is a decision
// render.Truncation already made and already knows how to write, and a reader
// here that switched on the value would be a second implementation of that
// choice; so the bare boolean is carried across untouched, as is whatever §9
// adds to the member next. What the marker carries is a **remedy**, and a
// remedy naming `--since` on a surface where nothing takes a flag would point an
// agent at an argument no schema declares.
//
// It is spelled rather than rewritten. The marker holds the parameters that
// narrow its axis and each surface names them in its own words
// (render.Narrowing), so this asks the marker for its second spelling rather
// than editing the sentence the first one composed — a rewrite here would be
// this package holding an opinion about which flags a command has.
func truncatedOf(terminal render.Row, encoded json.RawMessage, kind string) (json.RawMessage, error) {
	// **No terminal row is `null` and not an error**, and the path that
	// reaches it is a command that opened no row stream at all: a usage
	// error, and a guardrail declining from a command that is not a Run
	// (§9, ADR-0060). There is nothing there to have been cut, so there is
	// nothing for `false` to claim was complete — the member says the
	// truncation marker or nothing, and nothing is what this is.
	if terminal == nil {
		return json.RawMessage("null"), nil
	}
	// The marker as a value, through the door render.ResultRow opens for
	// it: what crossed the boundary is a row, and re-decoding the member in
	// order to re-spell one word of it would be reading back what the
	// terminal row had just encoded.
	if result, is := terminal.(render.ResultRow); is {
		if marker, cut := result.Marker(); cut {
			return json.Marshal(marker.InArguments())
		}
	}
	// **A Run's terminal row has no truncation to carry**, and that is the
	// second way this member is `null`. §8 states two terminal rows and makes
	// the type the discriminator: everything that ranges over something ends
	// in `result`, carrying the marker, and a Run ends in `outcome`, carrying
	// what a Run did. A Run reports what it just did rather than ranging over
	// a namespace, so there is no result set for a cap to have cut — and
	// `false` here would be this surface claiming a stream was complete that
	// was never a stream (§8, §9, issue #200).
	if kind == outcomeType {
		return json.RawMessage("null"), nil
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

// wasCut answers whether a result was truncated, from the member itself: the
// bare true and the marker object are the two shapes §9 admits for a cut
// result, and `false` and `null` are the two ways of carrying none — the first
// a stream that returned everything, the second a command that opened no stream
// at all (truncatedOf). Those four are the whole of what the member can be.
//
// It reads the JSON rather than the value it came from because the value is not
// in hand here — what crossed the boundary is the member — and because the
// question this answers is the one a consumer asks of the same bytes.
func wasCut(truncated json.RawMessage) bool {
	switch string(bytes.TrimSpace(truncated)) {
	case "false", "null":
		return false
	}
	return true
}

// outcomeSummary is a Run's text block: §8's terminal line, arriving here as a
// sentence (§9, issue #200).
//
// **It names the outcome first and the Run id after it**, which is the whole of
// what a client with no exit code and no scrollback needs from it: an entry the
// envelope does not name is one an agent can reach only by asking which Run was
// the last, on a Store two environments write. Where no entry was written there
// is nothing to name, and the line says what happened and stops.
//
// **The exit code is the one member of §8's line this does not compose.** The
// terminal compensates for the outcome arriving last by carrying a code, and
// this surface has no code to carry: what §12's `75` says — *past this lies
// time* — is said here by `failed` standing where `refused` would have, which
// is the distinction a caller acts on (§9, §12, ADR-0061).
//
// That is a fact about **this line** and not about the envelope. A Refusal's
// text block is the command's page forwarded whole, and the page ends in §8's
// terminal line with its code and its `hyper show` pointer in it — because the
// rendering goes over untouched, which is the rule an edit here would break
// (refusalText).
//
// **The rehearsal marker travels**, and it is the one part of the line that is
// load-bearing rather than informative: without it the sentence a Run that
// reached the world writes and the sentence a rehearsal writes are the same
// bytes, on the one path where the difference is the whole point (§7).
//
// It is composed from the lifted members and not from a second reading of the
// answer, so the sentence and the three keys beside it cannot disagree about
// what happened (structuredOf).
func outcomeSummary(structured Structured) string {
	parts := []string{structured.Outcome}
	if structured.DryRun != nil && *structured.DryRun {
		parts = append(parts, "dry-run")
	}
	if structured.RunID != "" {
		parts = append(parts, "run "+structured.RunID)
	}
	return strings.Join(parts, " · ")
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
		counts = append(counts, fmt.Sprintf("%d %s", counted[kind], nounFor(kind, counted[kind])))
	}
	line := strings.Join(counts, ", ")
	if truncated {
		line += ", truncated"
	}
	return line
}

// noun is one row type's word in the two forms a count needs it in. The plural
// is written only where it is not the singular with an `s` on it, which is one
// of the two exceptions this table is a table for: `Journal entries` is not
// `Journal entrys`, and a rule over the discriminator would have no way to know
// that.
type noun struct {
	singular string
	plural   string
}

// nouns is what each of §8's row types is called in prose, spelled as
// CONTEXT.md spells it. It is a table rather than a rule over the discriminator
// because the discriminators are `snake_case` machine names and the glossary's
// words are not derivable from them.
//
// A type with no entry reads as its own discriminator. That is legible, wrong
// enough to be noticed by whoever adds the row type, and better than either
// alternative: a fabricated noun would put a word in the glossary's mouth, and
// a refusal would fail a call over the prose beside an answer that is otherwise
// correct.
var nouns = map[string]noun{
	"provider":  {singular: "Provider"},
	"manifest":  {singular: "Manifest"},
	"operation": {singular: "Operation"},
	// `operation_detail` is one Operation seen up close, and the glossary has
	// one word for the thing however it is rendered: a caller who asked
	// about one Operation is told they got one, in the noun they asked in.
	// The row type's own name is the wire's discriminator and not a second
	// noun (§8, issue #197).
	"operation_detail": {singular: "Operation"},
	"target":           {singular: "Target"},
	// `problem`, `remediation`, `window` and `code` are spelled in lower
	// case, and the entries are here rather than left to the fallthrough on
	// purpose: the glossary has no term for any of them, so the
	// discriminator already *is* the English word — or, for a code fact, two
	// of them — and a row type reading as its own name by accident and by
	// decision should not look the same to whoever reads this table next
	// (§12, CONTEXT.md).
	"problem":     {singular: "problem"},
	"remediation": {singular: "remediation"},
	"window":      {singular: "window"},
	"code":        {singular: "code fact"},
	// The Inspection four's own (issue #199). `entry` is a **Journal**
	// entry, which is what the glossary calls one and what tells it from
	// the Step entries a Procedure holds.
	//
	// `provenance` is the **row** and not the thing, which is the second
	// exception above and the one worth reading twice: the glossary's
	// Provenance is *the record of which code produced something*, a mass
	// noun with no plural in it, and what a Run answers three of is rows —
	// the Run-wide scope and one per Step file written (§7, ADR-0043). So
	// the noun names what is counted rather than coining a plural for a
	// word the glossary does not pluralise, which is the same discipline
	// the rest of this table keeps by not inventing one at all.
	"run":         {singular: "Run"},
	"entry":       {singular: "Journal entry", plural: "Journal entries"},
	"refusal":     {singular: "Refusal"},
	"provenance":  {singular: "Provenance row", plural: "Provenance rows"},
	"step":        {singular: "Step"},
	"asset":       {singular: "Asset"},
	"observation": {singular: "Observation"},
	"record":      {singular: "Record"},
}

// nounFor is one row type in prose, pluralised by count: the plural its entry
// states, or the English `s` where it states none.
func nounFor(kind string, count int) string {
	spelled, named := nouns[kind]
	if !named {
		spelled = noun{singular: kind}
	}
	switch {
	case count == 1:
		return spelled.singular
	case spelled.plural != "":
		return spelled.plural
	}
	return spelled.singular + "s"
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
