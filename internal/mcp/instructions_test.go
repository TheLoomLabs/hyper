package mcp

import (
	"strings"
	"testing"
)

// The orientation the handshake carries, held to what §9 fixes about it (issue
// #209).
//
// **These cases are about the claims, not about the prose.** What the text says
// is a paragraph anybody may rewrite; what it must *state* is a list §9 writes
// down, because each member of it is something an agent cannot learn from the
// tool set at all — the tool descriptions arrive with a call in mind, and every
// one of these is a fact about the surface above them. A case per claim is what
// keeps a rewrite that dropped one from landing silently.
//
// The worked example is asserted one package over, where a `check` is
// reachable: what the handshake teaches is held to checking clean rather than
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
		t.Error("the orientation never says the agent authors with its own file tools; no tool here writes a reviewed artefact")
	}
}

// TestInstructions_PutTheThreeAbsentCommandsOutOfReachAndSayWhy is the third,
// and the one with teeth. `install`, `store init` and `compact` have no tool
// **deliberately**, and an agent that does not know that shells out to them —
// which is the exact bypass the absence exists to prevent (§9).
func TestInstructions_PutTheThreeAbsentCommandsOutOfReachAndSayWhy(t *testing.T) {
	carried := Instructions("1.4.0")

	for _, absent := range []string{"install", "store init", "compact"} {
		if !strings.Contains(carried, absent) {
			t.Errorf("the orientation never names %q; a command absent without a reason is one an agent shells out to", absent)
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
