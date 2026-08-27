package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
)

// **Progress at the Step boundary, and the whole of what a client hears from
// this server** (§9, ADR-0021, ADR-0092, issue #202).
//
// A Run on a surface with no scrollback needs something to watch while it
// works: the outcome arrives only when the call returns, so a silent twenty
// minutes is otherwise indistinguishable from a hang. What stands in its place
// is §9's second narration event — one `notifications/progress` per Step
// boundary, at the boundary §7 writes a Journal entry at — and nothing else at
// all.
//
// The cases here drive a corpus case TestGolden already knows, twice: once with
// a progress token attached and once without, which is the difference the rule
// is about. What they hold is everything the client read off the wire, so *what
// arrived* and *what did not* are one assertion rather than two.

// watchedBoundaries is a corpus case driven as one tool call under the progress
// token the case supplies, and everything the client saw beside its envelope.
//
// token is the progress token attached to the call, and **nil is a call that
// attaches none** — the two are one function for the reason the server's two
// doors are: what the rule turns on is the difference between two calls, and a
// driver reaching one of them through a second assembly would be comparing two
// servers instead.
func watchedBoundaries(t *testing.T, name string, token any) (mcp.Envelope, mcp.Watching) {
	t.Helper()

	c := corpusCase(t, name)
	invocation := c.invocation(t)
	envelope, watched, err := cli.MCPServer(c.process(t, invocation), c.facts(t)).
		Watched(t.Context(), c.call.Tool, c.call.Arguments, token)
	if err != nil {
		t.Fatalf("case %s: %s", name, err)
	}
	return envelope, watched
}

// TestProgress_AStepBoundaryIsANotification is the first half of §9's *Long
// Runs*: a multi-Step Run sends one notification per Step boundary, carrying
// the position, the total and the Step.
//
// The Run it drives is the three-Step Procedure whose second and third Steps
// **skip**, which is the case worth driving rather than a Run of three Steps
// that all ran: a boundary is where the next Step would start and is reached
// whatever that Step then does, so a narration that only counted the Steps that
// touched the world would go quiet exactly where a client most needs it not to.
func TestProgress_AStepBoundaryIsANotification(t *testing.T) {
	const token = "watching-the-run"
	envelope, watched := watchedBoundaries(t, "mcp/run/a-skip-propagates", token)

	if envelope.StructuredContent.Outcome != "completed" {
		t.Fatalf("the Run did not complete: %s", envelope.StructuredContent.Outcome)
	}
	if len(watched.Progress) != 3 {
		t.Fatalf("the Run of three Steps sent %d progress notifications, want one per Step boundary: %+v",
			len(watched.Progress), watched.Progress)
	}
	// **The call is synchronous**, which this is what it looks like from the
	// client's side: the last boundary is the last Step's, and the answer
	// that follows it is the finished Run — a Disposition per Step — rather
	// than a handle to come back for (§9, ADR-0092).
	if steps := stepRows(t, envelope); steps != len(watched.Progress) {
		t.Errorf("the call answered %d step rows after %d boundaries; what returns is the Run, not a handle to poll",
			steps, len(watched.Progress))
	}

	// The position, the total and the Step, in the Run's own order — which
	// is the Step table's order and the one the Journal names its files by
	// (§8, §12).
	for at, want := range []struct {
		position int
		of       int
		step     string
	}{{1, 3, "probe"}, {2, 3, "middle"}, {3, 3, "last"}} {
		got := watched.Progress[at]
		if got.Position != want.position || got.Of != want.of || got.Step != want.step {
			t.Errorf("notification %d is %d/%d %q, want %d/%d %q",
				at+1, got.Position, got.Of, got.Step, want.position, want.of, want.step)
		}
		// **The token is what makes it a notification about this call.**
		// Progress belongs to the call that is in flight and stops when
		// it does, and the token is the protocol's whole account of
		// which call that is (§9, ADR-0021).
		if got.Token != token {
			t.Errorf("notification %d carries token %v, want the one this call attached, %q", at+1, got.Token, token)
		}
	}
}

// TestProgress_ARunNamingItselfSendsNothing is `Began` on this surface, held by
// counting: §9's narration is two events and only one of them is a message
// here.
//
// On the CLI the first event exists because the terminal line is not always
// reached, and the Run that dies before it is the one Run whose identity its own
// output would otherwise never carry (ADR-0047). Here the id arrives in the
// summary line and in `run_id`, and a client that gives up gets no delivery at
// all — so a notification naming the id would be narration with no reader on
// the one path it was invented for.
//
// What holds it is that the **first** notification is the first Step's boundary
// and the count is the Steps: a `Began` that sent anything would either arrive
// before `1/3` or push the count to four.
func TestProgress_ARunNamingItselfSendsNothing(t *testing.T) {
	envelope, watched := watchedBoundaries(t, "mcp/run/a-skip-propagates", "watching-the-run")

	if len(watched.Progress) == 0 {
		t.Fatal("the Run sent no progress at all; there is nothing here to say Began is silent")
	}
	if first := watched.Progress[0]; first.Position != 1 {
		t.Errorf("the first thing the client heard is %d/%d %q, want the first Step's boundary",
			first.Position, first.Of, first.Step)
	}
	// One per Step and not one more: the Run wrote three `step` rows, and
	// the count of boundaries is the count of rows (§8).
	if steps := stepRows(t, envelope); len(watched.Progress) != steps {
		t.Errorf("the Run wrote %d step rows and sent %d notifications; the boundary is the Step and nothing else is narrated",
			steps, len(watched.Progress))
	}
	// The other half of the sentence: the id reaches this caller in the
	// summary line **and** in `run_id`, which is what makes a notification
	// naming it narration nobody needed (§8, §9).
	id := envelope.StructuredContent.RunID
	if id == "" {
		t.Fatal("the envelope names no Run; the id reaches this caller in run_id or nowhere")
	}
	if len(envelope.Content) == 0 || !strings.Contains(envelope.Content[0].Text, id) {
		t.Errorf("the summary line does not name the Run: %+v", envelope.Content)
	}
}

// TestProgress_NoTokenIsNoNotification is the rule §9 and the protocol state
// together: **a notification is sent where the client supplied a progress token
// and nowhere else.**
//
// Without a token there is nothing for a notification to be correlated with, so
// sending one anyway would be the server speaking unasked — which is the one
// thing this server never does (ADR-0021). It is the same Run as the case above
// with the one input taken away, so what the pair holds is the difference the
// token makes and nothing else about the two calls.
func TestProgress_NoTokenIsNoNotification(t *testing.T) {
	envelope, watched := watchedBoundaries(t, "mcp/run/a-skip-propagates", nil)

	if envelope.StructuredContent.Outcome != "completed" {
		t.Fatalf("the Run did not complete: %s", envelope.StructuredContent.Outcome)
	}
	if len(watched.Progress) != 0 {
		t.Errorf("a call that attached no progress token was sent %d notifications: %+v",
			len(watched.Progress), watched.Progress)
	}
}

// TestProgress_TheServerSendsNothingElseAtAll is *hyper never speaks first*,
// held over both calls (§9, ADR-0021).
//
// The server has no logging channel, it initiates no message between calls, and
// there is no server-initiated request of any kind: no sampling, because a tool
// that decided anything by asking a model would have moved authority off the
// reviewed artefact and onto a prompt, and no elicitation, because an
// elicitation is a prompt and no surface prompts (ADR-0015).
//
// What the driver holds is every frame the client read that carried a method at
// all — a response carries an id and a result and never one — so what is left
// after the progress notifications is the set §9 says is empty. It is collected
// rather than assumed, which is the difference between a claim and a fence.
func TestProgress_TheServerSendsNothingElseAtAll(t *testing.T) {
	for _, one := range []struct {
		name  string
		token any
	}{
		{"under a progress token", "watching-the-run"},
		{"with no progress token", nil},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, watched := watchedBoundaries(t, "mcp/run/a-skip-propagates", one.token)
			if len(watched.Unasked) != 0 {
				t.Errorf("the server sent %v unasked; it has no logging channel and initiates nothing", watched.Unasked)
			}
		})
	}
}

// stepRows is how many `step` rows an envelope carries, which is one per Step
// that reached a Disposition (§8).
func stepRows(t *testing.T, envelope mcp.Envelope) int {
	t.Helper()

	var rows int
	for _, row := range envelope.StructuredContent.Rows {
		var typed struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(row, &typed); err != nil {
			t.Fatal(err)
		}
		if typed.Type == "step" {
			rows++
		}
	}
	return rows
}
