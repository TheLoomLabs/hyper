package cli

import (
	"bytes"
	"io"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// destination is where a command's answer goes, and in which form.
//
// It is one value rather than the three parameters it replaces — a stdout, a
// stderr and a `--json` — because those three can only ever say *the CLI's
// streams*, and one function stood between every command's answer and the bytes
// it became. writeAnswer received everything an envelope needs — the rows as
// values, the terminal row as a value, and the command's own page renderer —
// and turned them into a stream immediately, so a caller wanting the rows *as
// values* with the page as text beside them had nowhere to stand (§9, ADR-0026,
// issue #194).
//
// Two implementations stand behind it. The CLI's streams write exactly what the
// three parameters wrote; the MCP server's `collected` writes nowhere at all and
// retains the rows as values, which is the milestone this was prefactored for
// (issue #195). What makes the second possible is that the rows a page is
// written from and the rows an envelope carries are one list.
//
// **Nothing a command decides moves behind it.** Which rows, in which order,
// under what heading, with which terminal row is the command's and stays the
// command's; what is here is the path from an answer to wherever it goes.
//
// **A helper takes this value where it can Refuse, and a plain io.Writer where
// it can only narrate**, which is the rule that keeps the seam legible at forty
// call sites: `gateOnVersionPin`, `refuse`, `refuseProblems`, `declineFetch`
// and `reportReadStoreFault` render a Refusal and take the destination;
// `resolveRepoRoot`, `syncForReading`, `nextValue` and `reportLockFault` write
// a human sentence and take the writer `narrate` answers. A signature says
// which of the two a helper is.
type destination interface {
	// answer is the command's answer: the rows, the terminal row that ends
	// them, and the page they render as where the destination wants a page.
	//
	// The page is the command's own, because a page's columns, its heading
	// and the line that stands where there are no rows are facts about that
	// command rather than about any renderer. What is shared is the path
	// from rows to wherever they go, not the layout.
	answer(rows []render.Row, terminal render.Row, page func(io.Writer, []render.Row) error) error

	// refusal is §8's Refusal rendering, which does not pass through the
	// answer at all: the CLI writes it on stderr in both forms with stdout
	// left silent, a Refusal not being a row (gate.go). It takes the members
	// as values for the answer's own reason — a second surface carries the
	// same Refusal as text, and the two must be one reading of one array.
	//
	// The form is carried beside them because which of §8's two renderings a
	// Refusal takes is what the check had to point at, and that is the
	// caller's fact rather than one the members can be read for
	// (refusalForm).
	refusal(form refusalForm, members []refusalRow) error

	// narrate is where narration goes: a Run naming itself and its Steps,
	// the warning a read that tolerated a failed sync writes, a truncation
	// line, and the human rendering of a usage error. None of it is the
	// answer, and none of it is a row.
	narrate() io.Writer

	// form **answers** the destination the caller asked for rather than
	// setting anything on this one, which is what lets a destination decline
	// the request by answering itself.
	//
	// It is the one place `--json` is read: the flag names a form, a form is
	// a property of where an answer is going, and a command that read the
	// flag for itself would be a command free to disagree with its own
	// destination. It is asked once, by the shared parser — `--json` is a
	// flag of the surface that has a command line, and a surface carrying
	// every answer as structure already has nothing to switch (flags.go).
	form(wire bool) destination
}

// streams is the CLI's destination: stdout for the answer, stderr for
// everything else, and the form `--json` chose between.
//
// **stdout is the answer and nothing else ever goes there** (§9). That rule is
// the whole reason the two writers were never interchangeable, and it is stated
// here now rather than at each of the nineteen call sites that used to hold
// both.
type streams struct {
	stdout, stderr io.Writer
	asJSON         bool
}

// Streams is that value, assembled from the two writers a process hands the
// tool. Main assembles its own out of the argv's own streams and threads it
// down; this is the door for a caller outside the package that drives one
// command rather than the dispatch, which is what every case reaching an entry
// point directly does.
//
// **It answers the interface and not the struct**, so what a caller outside can
// do with a destination is exactly what a command can do with one: hand it an
// answer, hand it a Refusal, narrate. That asymmetry — an exported constructor
// over an unexported type — is the point rather than an oversight: a
// destination is `hyper`'s to implement, both the one that exists and the one
// the milestone after this adds, and neither is a shape a caller supplies. What
// is exported is the ability to obtain the CLI's, which is what an entry point
// driven directly needs and all it needs.
//
// The form is not set here — the flag that names one is read by the shared
// parser, which is the only reader of it there is (flags.go).
func Streams(stdout, stderr io.Writer) destination {
	return streams{stdout: stdout, stderr: stderr}
}

// answer writes the `--json` row stream, terminated by the terminal row, or the
// command's own page. Both are written from one list of rows, which is the
// whole of ADR-0026 — the two surfaces cannot state different things because
// there is one set of rows behind them.
func (s streams) answer(rows []render.Row, terminal render.Row, page func(io.Writer, []render.Row) error) error {
	if s.asJSON {
		return render.WriteJSON(s.stdout, rows, terminal)
	}
	return page(s.stdout, rows)
}

// refusal writes §8's Refusal on stderr, whichever form the answer would have
// taken: stdout is left silent because a Refusal is not a row, so `--json`
// opens no stream to carry one (§9, gate.go).
func (s streams) refusal(form refusalForm, members []refusalRow) error {
	return writeRefusalMembers(s.stderr, form, members)
}

// narrate is stderr, which is where everything that is not the answer goes.
func (s streams) narrate() io.Writer { return s.stderr }

// form is the CLI's streams in the form `--json` named: the wire where it was
// given, the page where it was not.
func (s streams) form(asJSON bool) destination {
	s.asJSON = asJSON
	return s
}

// collected is the second destination, and the one destination.go above was
// prefactored for: where a tool's answer goes on the MCP surface (§9, issue
// #195).
//
// **It holds no writer onto the process's stdout, and that is the whole point
// of it.** The server's stdout belongs to the protocol — a stray write there
// would corrupt a frame rather than merely mislead a reader — so the structural
// guarantee is not a rule a command is asked to keep but the absence of
// anything to write to: what this retains is the rows and the terminal row as
// **values**, and three buffers for the prose. A command behind it can no more
// reach the wire than it can reach the network.
//
// It retains the rows rather than a rendering of them because that is what the
// envelope carries: §8's row set served as an array, one object per row (§9).
// The page is rendered into a buffer beside them because a Refusal's full
// rendering and `review`'s are what §9's text block carries on two of its three
// paths, and a destination that only sometimes rendered would be one the tool
// had to ask twice.
//
// **The three buffers are three destinations and not one**, because the mapping
// that reads them back tells them apart: the page is what `review`'s text block
// carries, the Refusal rendering is what a declining guardrail's carries, and
// the narration is the message a malformed call's protocol error carries (§9,
// mcp.Answer, issue #196). One buffer would hand a Run that Refuses its own
// terminal row's page and its Refusal concatenated, and §9 asks for the second
// alone.
type collected struct {
	rows     []render.Row
	terminal render.Row
	// rendering is the command's page and refusalRendering is §8's
	// Refusal. The second is spelled long because the method that writes
	// it is `refusal`, and a type cannot hold a field and a method under
	// one name.
	rendering        bytes.Buffer
	refusalRendering bytes.Buffer
	// narration is everything the CLI would have written on stderr, kept
	// against the one arm of the mapping that reads it (narrate).
	narration bytes.Buffer
}

// answer retains the answer whole: the rows and the terminal row as values, and
// the command's own page in the buffer.
//
// The page is written here for the same reason the CLI's destination writes one
// — the rows a page is rendered from and the rows an envelope carries are one
// list (ADR-0026) — and it goes to a buffer rather than to a stream because
// there is no stream. A page that will not render is an error like any other
// and travels back the way writeAnswer already sends one.
func (c *collected) answer(rows []render.Row, terminal render.Row, page func(io.Writer, []render.Row) error) error {
	c.rows, c.terminal = rows, terminal
	return page(&c.rendering, rows)
}

// refusal renders §8's Refusal into a buffer of its own.
//
// It is kept rather than written anywhere because on this surface a Refusal
// *is* the answer's text: §9 puts the full rendering in the `text` block, with
// `isError: true` beside it, exactly where the command exits 77 — and with no
// bypass anywhere that rendering is the entire way past (ADR-0001, mcp's
// envelopeOf).
//
// It is not the page's buffer, and the command that says why is `run`: a Run
// that Refuses writes its own terminal row through answer as well, §8 putting
// it on the `outcome` side on every path on which a Run was attempted. Both
// renderings would then be in one buffer, in whatever order the command
// happened to produce them, and the text block asks for one of them.
func (c *collected) refusal(form refusalForm, members []refusalRow) error {
	return writeRefusalMembers(&c.refusalRendering, form, members)
}

// narrate is a buffer, and **nothing in an envelope is ever composed from it**:
// narration is §9's own reading of a Run naming its Steps, a warning beside a
// tolerated sync failure, a truncation line — *none of it is the answer, and
// none of it is a row*. The CLI has a second stream for it and this surface has
// none, so what is not read below is dropped.
//
// What is read below is one thing: **the human rendering of a usage error**,
// which is the message the protocol error a malformed call comes back as
// carries, so that an agent reads the sentence a person would have read (§9,
// ADR-0060, mcp's envelopeOf). It was io.Discard until that sentence had a
// reader; a usage error whose narration went nowhere would arrive as a
// protocol error with nothing in it but a number.
//
// The rest of what a caller would have read on stderr is not lost where it
// matters: a truncated result says so in the envelope, in both halves, and a
// Refusal is rendered in full into the buffer above. Progress narration is
// §9's notification at a Step boundary, which arrives with the tool that has
// Steps to narrate.
func (c *collected) narrate() io.Writer { return &c.narration }

// form answers this destination whichever form was named, which is what
// destination.form exists to let a destination do: **decline the request by
// answering itself**.
//
// There is nothing here for `--json` to switch. A surface carrying every answer
// as structure has already made the choice the flag names, and the flag itself
// is unreachable — a tool builds the command line its command would have
// received and no tool takes a global (§9, flags.go). Answering itself is
// therefore the truthful reply rather than a shortcut: the form is not the
// caller's to set here.
func (c *collected) form(bool) destination { return c }
