package mcp

import (
	"strings"
	"testing"
)

// The orientation, held to what §9 fixes about it (issues #209 and #211).
//
// It reaches an agent two ways — the `initialize` handshake carries it, and
// `hyper project` writes it to `AGENTS.md` where a repository has none — and
// these cases are about the text rather than about either channel, which is why
// they stand here beside it (ADR-0093, ADR-0095).
//
// **These cases are about the claims, not about the prose.** What the text says
// is a paragraph anybody may rewrite; what it must *state* is a list §9 writes
// down, because each member of it is something an agent cannot learn from the
// tool set at all — the tool descriptions arrive with a call in mind, and every
// one of these is a fact about the surface above them. A case per claim is what
// keeps a rewrite that dropped one from landing silently.
//
// The worked example is asserted one package over, where a `check` is
// reachable: what the orientation teaches is held to checking clean rather than
// to looking right (internal/cli/instructions_test.go).

// TestInitialize_CarriesTheOrientationInTheHandshake is the field itself.
//
// It is the one mechanism that reaches an agent **before its first tool call**,
// with no file in the user's repository and no setup beyond the client config
// they already need — and it is a field of the answer to a request the client
// made, so nothing about it is `hyper` speaking first (ADR-0021, ADR-0093).
func TestInitialize_CarriesTheOrientationInTheHandshake(t *testing.T) {
	server, _ := answering(nil, nil)
	carried := connected(t, server).InitializeResult().Instructions

	if carried == "" {
		t.Fatal("the handshake carries no instructions; an agent's first tool call is then its first sight of hyper")
	}
	if carried != Instructions("1.4.0") {
		t.Error("the handshake carries a text the package does not state; the orientation is one text, at the server's own version")
	}
}

// TestInstructions_NameTheFiveArtefacts is the first thing the tool set cannot
// teach. `operation` returns a Manifest verbatim and so teaches one of the
// five; nothing anywhere teaches the other four, and an agent that does not
// know a Definition exists authors a config file instead (§9).
func TestInstructions_NameTheFiveArtefacts(t *testing.T) {
	carried := Instructions("1.4.0")

	for _, artefact := range []string{"Manifest", "Target declaration", "Definition", "Procedure", "Repository declaration"} {
		if !strings.Contains(carried, artefact) {
			t.Errorf("the orientation never names the %s; §2 counts five artefacts and an agent authors four of them", artefact)
		}
	}
}

// TestInstructions_StateTheLoop is the second: author with your own file tools,
// `check`, repair, `review`, hand it to the human, `run`. An agent that does not
// know the loop shells out or invents a config file, which is the failure this
// text exists to prevent.
func TestInstructions_StateTheLoop(t *testing.T) {
	carried := Instructions("1.4.0")

	for _, step := range []string{"`check`", "`review`", "`run`"} {
		if !strings.Contains(carried, step) {
			t.Errorf("the orientation never names %s; the loop is the thing an agent has no other way to learn", step)
		}
	}
	// The one step of the loop that is not a tool call, and the one an
	// agent skips by default: nothing here authors an artefact, and a
	// human reads it before it runs (§9, ADR-0001).
	if !strings.Contains(unwrapped(carried), "you write what is reviewed") {
		t.Error("the orientation never says the agent authors with its own file tools; no hyper command writes a reviewed artefact")
	}
}

// TestInstructions_SayWhereTheRecordLives is the fact a session got backwards
// in front of a human (§9, ADR-0113, issue #233).
//
// The clause stood in the text already — *a Store that is append-only and
// travels in the repository* — inside the paragraph about the `--response` file
// and why not to author a throwaway Operation to look at a body. An agent asked
// what its repository's account amounted to read the whole file at its fifth
// call, went looking for the account in the working tree, found no `.hyper/`,
// no `store/` and a clean `git status`, and reported that a clone would get the
// Procedure and not the history. It is stated now in the paragraph about
// **reading the record back**, which is the paragraph an agent is in when it
// asks the question.
//
// Portability is asserted in its own right because it is the half that was
// wrong: *there is a record* and *the record survives leaving this machine* are
// two claims, and the session got the first one right.
func TestInstructions_SayWhereTheRecordLives(t *testing.T) {
	carried := unwrapped(Instructions("1.4.0"))

	if !strings.Contains(carried, "hyper-store") {
		t.Error("the orientation never names the branch the record is on; no surface an agent may call names it either, and the working tree shows nothing")
	}
	if !strings.Contains(carried, "travels with a clone") {
		t.Error("the orientation never says the record travels with a clone; that is the claim a session got backwards, and existence is not portability")
	}

	// Where it is said is the whole of what this ticket was about, so the
	// paragraph is what the case reads: the sentence stands with the four
	// commands that read the record back, and not beside the `--response`
	// file it used to.
	reading := strings.Index(carried, "Read the record back with")
	located := strings.Index(carried, "The record is a branch in this repository")
	if reading < 0 || located < reading {
		t.Error("the fact does not stand in the paragraph about reading the record back; a fact stated in the wrong paragraph is a fact the reader does not meet")
	}
}

// TestInstructions_PutTheThreeCommandsOutOfReachAndSayWhy is the third,
// and the one with teeth. `install`, `store init` and `compact` are the human's
// **deliberately**, and an agent that does not know that runs them — which is
// the exact bypass their absence from the tool set exists to prevent (§9).
func TestInstructions_PutTheThreeCommandsOutOfReachAndSayWhy(t *testing.T) {
	carried := Instructions("1.4.0")

	for _, absent := range []string{"install", "store init", "compact"} {
		if !strings.Contains(carried, absent) {
			t.Errorf("the orientation never names %q; a command out of reach without a reason is one an agent reaches for", absent)
		}
	}
	// §9's own line, and the whole reason all three sit on the far side:
	// an agent may read the record and add to it, and may not create it,
	// prune it, or bring anything new into the repository.
	if !strings.Contains(unwrapped(carried), "may not create it, prune it, or bring anything new into the repository") {
		t.Error("the orientation states no reason for the three; the line is what makes shelling out legible as a bypass rather than as a workaround")
	}
}

// TestInstructions_SayARefusalIsFinal is the fourth. §8's rendering already
// says a verbatim retry refuses identically, and agents retry anyway — so the
// fact is stated where it is read before the first call rather than only in the
// Refusal that is already being ignored.
func TestInstructions_SayARefusalIsFinal(t *testing.T) {
	carried := Instructions("1.4.0")

	if !strings.Contains(unwrapped(carried), "The same call retried refuses identically") {
		t.Error("the orientation never says a Refusal is final; the same call retried refuses identically, and an agent that does not know that retries")
	}
}

// TestInstructions_TellTheAgentToOfferASectionWhereAnAGENTSFileStands is the
// sixth, and it is the one the transcripts argued for rather than §9 (issues
// #209 and #211).
//
// **No harness delivers this text unconditionally.** Codex carries it in a
// `tool_search_output`, so a model that reaches for a tool without searching
// never sees it; a Claude Code session spent six calls running `strings` over
// the binary and copied the Manifest it dug out. So `project` writes the text
// to `AGENTS.md`, which every harness reads up front, and the orientation's own
// paragraph about the file is what is left over once it does.
//
// **What is left over is one case.** `project` never overwrites, so a
// repository that already holds an `AGENTS.md` for its own reasons gets
// nothing — and that is the repository where the agent offers. The paragraph
// used to say *offer to write one*, which was the fallback for an orientation
// that had not arrived, gated on the orientation arriving; what it says now
// fires where the file that would have carried it could not be written
// (ADR-0095).
func TestInstructions_TellTheAgentToOfferASectionWhereAnAGENTSFileStands(t *testing.T) {
	carried := unwrapped(Instructions("1.4.0"))

	if !strings.Contains(carried, "AGENTS.md") {
		t.Error("the orientation never names AGENTS.md; the one mechanism every harness reads up front goes unmentioned")
	}
	// **Offer, and a section rather than a file.** The file is `project`'s
	// to write and only where the repository holds none; an agent that
	// wrote one itself would be widening `hyper`'s surface by way of a
	// sentence in its prose, and one that offered to *write* a file that
	// already stands is offering an act with no effect.
	if !strings.Contains(carried, "offer to add a section to it") {
		t.Error("the orientation does not say to *offer a section*; the file itself is `project`'s to write, and only where none stands")
	}
	// It is not one of the five, and an agent that thought it was would be
	// authoring authority where there is none (§2, ADR-0093).
	if !strings.Contains(carried, "not a reviewed artefact") {
		t.Error("the orientation does not say an AGENTS.md carries no authority; the five artefacts are the five")
	}
}

// TestInstructions_AreWordedForAReaderOnEitherSurface is the constraint the
// second channel put on every sentence in the text (ADR-0095, issue #211).
//
// The same bytes reach an agent through the handshake and through the
// `AGENTS.md` `project` writes, and the second reader may hold nothing but a
// terminal. So the text names **commands**, which both surfaces have, rather
// than tools, which only one does — and the three commands out of reach are
// *the human's* rather than *absent from this surface*, because absent-from-here
// is read as permission by the reader holding the other one.
func TestInstructions_AreWordedForAReaderOnEitherSurface(t *testing.T) {
	carried := unwrapped(Instructions("1.4.0"))

	// The one tool whose name differs from its command's, and therefore the
	// one spelling that would be a name in a client's namespace and nothing
	// at all on a command line (§9).
	if strings.Contains(carried, "run_show") {
		t.Error("the orientation names a tool rather than a command; `run_show` names nothing on the surface half its readers are standing on")
	}
	// The absence is not what puts the three out of reach — a terminal has
	// all sixteen — so the text may not rest the guardrail on it.
	if !strings.Contains(carried, "not yours to type into a terminal either") {
		t.Error("the orientation puts the three out of reach as absent tools; an agent holding a terminal reads that as permission")
	}
}

// TestInstructions_CarryTheRepositoryDeclarationAtTheServersOwnVersion is why
// the text is a function rather than a constant.
//
// The Repository declaration pins **which version of `hyper` may act here**,
// and the version that would act is the version of the server the client
// started (§9, ADR-0020). An orientation carrying a constant would teach every
// agent to author a pin that Refuses the gate on every repository but one.
func TestInstructions_CarryTheRepositoryDeclarationAtTheServersOwnVersion(t *testing.T) {
	if got := Instructions("2.0.0"); !strings.Contains(got, "version: 2.0.0") {
		t.Error("the orientation pins a version that is not the server's own; the pin it teaches would then Refuse the gate")
	}
	if got := Instructions("1.4.0"); strings.Contains(got, "version: 2.0.0") {
		t.Error("the orientation carries a version no caller asked for")
	}
}

// unwrapped is the text as a reader takes it in, with every run of whitespace
// one space.
//
// The claims above are sentences, and a sentence in a wrapped markdown document
// carries a newline wherever the margin fell. Asserting on the raw text would
// make a case fail when a paragraph was re-flowed and pass when the claim was
// deleted and re-wrapped, which is the wrong way round for both.
func unwrapped(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
