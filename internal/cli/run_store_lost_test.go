package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The three ways a Run loses the Store, where a golden cannot hold them (§6,
// §7, issues #138, #148). Everything else about them is in the corpus, and each
// of these says why it could not be.
//
// **The lock** is not a directory of files. It is a lock a *live* process
// holds, which is the whole reason a crash cannot leave one behind
// (internal/store/lock.go), so the two cases below take it in the test process
// and drive the command against the same repository — which is exactly the
// shape a second `hyper` on the same machine is in.
//
// **The exhausted push** renders git's own account of the rejection, and that
// account names the bare repository by path — a temp directory, different on
// every run of the suite. So its streams are asserted by what they say rather
// than byte for byte, and the two branch goldens, which name no path and no
// commit, are checked in and compared like any other case's. **An effectful
// Run's sync is the same shape one moment earlier** — it is the push of that
// Run's own `run.json` (§7, ADR-0083) — so the two cases that drive it are here
// beside it for the same reason.
//
// Everything else is the corpus's own machinery. Each case is a directory under
// testdata/run/, materialised the way TestGolden materialises one, so what
// differs between these runs and the corpus's is the one fact each is about.

// TestStoreLost_ARunThatCannotTakeTheLockIsFailedAtSeventyFive is contention, and
// what it is not. The Run is `failed` at 75, no `error_code` is rendered
// anywhere, stdout is silent, and no Journal entry was written — a Run that
// cannot take the lock has no branch to write one on (§6, §7, §12, ADR-0061).
func TestStoreLost_ARunThatCannotTakeTheLockIsFailedAtSeventyFive(t *testing.T) {
	c := corpusCase(t, "run/the-tracer-bullet")
	invocation := c.invocation(t)

	held, err := store.Acquire(invocation.wd, store.Exclusive)
	if err != nil {
		t.Fatalf("the effectful Run's lock: %v", err)
	}
	defer held.Release()

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

	if exit != 75 {
		t.Errorf("exit = %d, want 75 — a Run that lost the Store to the lock", exit)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout carries %q, want silence", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "another Run holds") {
		t.Errorf("stderr is %q, want it to name the Run that holds the lock", got)
	}
	if got := stderr.String(); strings.Contains(got, "refused") || strings.Contains(got, "error_code") {
		t.Errorf("stderr is %q; contention is not a Refusal and carries no error_code", got)
	}
	if branch := invocation.fixture.render(t, invocation.fixture.root); !strings.Contains(branch, "STORE.md") || strings.Contains(branch, "run.json") {
		t.Errorf("the branch holds %q; a Run that never took the lock wrote no entry", branch)
	}
}

// TestStoreLost_AnEffectfulRunTakesTheExclusiveLock is the mode read off the
// other end. The **shared** lock is held in this process — a monitoring Run,
// alive — and an effectful Run asks for the exclusive one and cannot have it:
// contention, `failed` at 75, and no `error_code` anywhere (§6, §12, ADR-0061).
//
// It is what proves the mode rather than the lock: the pair above holds the
// exclusive lock against a `read` Run, which contends whichever mode that Run
// asked for, and this one contends only because the Run asked for exclusive.
func TestStoreLost_AnEffectfulRunTakesTheExclusiveLock(t *testing.T) {
	c := corpusCase(t, "run/a-mutate-step-lands-an-asset")
	invocation := c.invocation(t)

	held, err := store.Acquire(invocation.wd, store.Shared)
	if err != nil {
		t.Fatalf("the read-only Run's lock: %v", err)
	}
	defer held.Release()

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

	if exit != 75 {
		t.Errorf("exit = %d, want 75 — an effectful Run that lost the Store to the lock", exit)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout carries %q, want silence", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "another Run holds") {
		t.Errorf("stderr is %q, want it to name the Run that holds the lock", got)
	}
	if got := stderr.String(); strings.Contains(got, "refused") || strings.Contains(got, "error_code") {
		t.Errorf("stderr is %q; contention is not a Refusal and carries no error_code", got)
	}
	if branch := invocation.fixture.render(t, invocation.fixture.root); strings.Contains(branch, "run.json") {
		t.Errorf("the branch holds %q; a Run that never took the lock wrote no entry", branch)
	}
}

// TestStoreLost_ASecondReadOnlyRunProceeds is the other half of the same
// sentence, and the reason the modes exist: a monitoring Run is not starved
// behind another monitoring Run. The shared lock is held throughout and the
// command completes.
func TestStoreLost_ASecondReadOnlyRunProceeds(t *testing.T) {
	c := corpusCase(t, "run/the-tracer-bullet")
	invocation := c.invocation(t)

	held, err := store.Acquire(invocation.wd, store.Shared)
	if err != nil {
		t.Fatalf("the first read-only Run's lock: %v", err)
	}
	defer held.Release()

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

	if exit != 0 {
		t.Errorf("exit = %d, want 0; stderr: %s", exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "completed") {
		t.Errorf("stdout is %q, want a Run that completed", stdout.String())
	}
}

// TestStoreLost_APushRejectedThreeTimesIsFailedAtSeventyFive is the third way,
// and the one the outcome triple has to be read carefully for. The Run did its
// work — the Step ran, the Observation was written, the entry was closed — and
// then the remote moved under three pushes running. That is `failed` at 75 and
// not at 1: it is a Run that lost the Store, beside the lock and the sync at
// Run start, rather than the world resisting the work (§7, §12, ADR-0061).
//
// What it leaves behind is the half §7 promises and the half the goldens hold:
// every commit stands on the local branch and none of them reached the remote,
// so what this Run wrote goes out with the next Run that pushes.
func TestStoreLost_APushRejectedThreeTimesIsFailedAtSeventyFive(t *testing.T) {
	c := corpusCase(t, "run/a-push-rejected-three-times", "run", "watch-status")
	invocation := c.invocation(t)

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

	if exit != 75 {
		t.Errorf("exit = %d, want 75 — a Run that lost the Store to a push it could not land", exit)
	}
	if want := "failed · exit 75 · run 01991f31-6dc3-7e4f-b051-728394051627"; !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout is %q, want a terminal line reading %q", stdout.String(), want)
	}
	if got := stdout.String(); !strings.Contains(got, "1     status  read  ran          1") {
		t.Errorf("stdout is %q; the Step ran and the table says so — the push is not the work", got)
	}
	if got := stdout.String(); strings.Contains(got, "error_code") || strings.Contains(got, "refusal") {
		t.Errorf("stdout is %q; a Run that lost the Store is no Refusal and carries no error_code", got)
	}
	if got := stderr.String(); !strings.Contains(got, "rejected three times running") {
		t.Errorf("stderr is %q, want git's own account of the rejection", got)
	}

	// The branches, byte for byte: what the Run wrote stands locally, and
	// origin holds only what it held before.
	invocation.compareBranches(t, c.dir)
}

// TestStoreLost_AnEffectfulRunThatCouldNotSyncIsFailedAtSeventyFive is the
// third way, on the arm milestone 5 could not reach: **an effectful Run's sync
// is the push of its own open Journal entry** (§7, ADR-0083), and a Run that
// could not complete it lost the Store before it touched the world.
//
// The fixture leaves origin's push URL working and points its fetch URL at
// nothing, which is the one shape that separates a sync a Run could not
// complete from a push it could not complete. The read-only half of the same
// sentence is a golden — `a-read-only-sync-in-the-same-fixture`, which tolerates
// the failure and proceeds against the branch this clone holds — and this half
// cannot be, git's account of the unreachable remote naming a temp directory.
//
// **No Step ran.** The sync is before the gates and long before Step 1, so what
// this asserts beside the code is that nothing reached the world: the Run has
// no id, the branch holds no entry, and stdout is silent.
func TestStoreLost_AnEffectfulRunThatCouldNotSyncIsFailedAtSeventyFive(t *testing.T) {
	c := corpusCase(t, "run/an-effectful-sync-that-could-not-reach-the-remote", "run", "publish-preview")
	invocation := c.invocation(t)

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

	if exit != 75 {
		t.Errorf("exit = %d, want 75 — an effectful Run that lost the Store to its sync", exit)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout carries %q, want silence", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "the Store could not be reached") {
		t.Errorf("stderr is %q, want the condition named", got)
	}
	if got := stderr.String(); strings.Contains(got, "refused") || strings.Contains(got, "error_code") {
		t.Errorf("stderr is %q; a Run that lost the Store is no Refusal and carries no error_code", got)
	}
	if got := stderr.String(); strings.Contains(got, "reads the branch this clone holds") {
		t.Errorf("stderr is %q; that line is the read-only arm's, and an effectful Run does not proceed", got)
	}

	invocation.compareBranches(t, c.dir)
}

// TestStoreLost_AnEffectfulRunWhoseEntryDidNotReachTheRemoteIsFailedAtSeventyFive
// is the push half of that same sync. The fetch lands, `run.json` goes down,
// and the push that sends it is rejected three times running by a remote that
// moved under each attempt — so the Run is `failed` at 75 on the code §12 fixes
// for a Run that lost the Store, and it never reaches a gate, let alone a Step
// (§7, ADR-0083, run.ErrSyncFailed).
//
// **The entry stands locally and is closed.** A Run that lost the Store still
// finishes on its own terms: `run.json` and `outcome.json` are both on the local
// branch and neither reached the remote, which is what the two branch goldens
// hold.
func TestStoreLost_AnEffectfulRunWhoseEntryDidNotReachTheRemoteIsFailedAtSeventyFive(t *testing.T) {
	c := corpusCase(t, "run/an-effectful-entry-that-did-not-reach-the-remote", "run", "publish-preview")
	invocation := c.invocation(t)

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

	if exit != 75 {
		t.Errorf("exit = %d, want 75 — an effectful Run whose entry never reached the remote", exit)
	}
	if want := "failed \u00b7 exit 75 \u00b7 run 01991ea6-b118-7c93-8d41-6b2f7ae05d0b"; !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout is %q, want a terminal line reading %q", stdout.String(), want)
	}
	if got := stdout.String(); strings.Contains(got, "mutate") {
		t.Errorf("stdout is %q; the sync is before the gates, so no Step was reached and no Step table is rendered", got)
	}
	if got := stderr.String(); !strings.Contains(got, "the Store could not be synced") {
		t.Errorf("stderr is %q, want the condition named", got)
	}
	if got := stderr.String(); !strings.Contains(got, "rejected three times running") {
		t.Errorf("stderr is %q, want git's own account of the rejection beneath it", got)
	}

	invocation.compareBranches(t, c.dir)
}

// The same three ways, through the second door (§9, issue #200).
//
// **The 75s are the one place the two surfaces cannot say the same thing**, and
// these are what say what this one says instead. §8 puts `run` on the `outcome`
// side on every path a Run was attempted on and on the two that decline before
// one is identified — and the lock and the sync at Run start are neither, so
// the command writes nothing to stdout and lets the exit code carry the news.
// The MCP surface has no exit code. So the envelope carries §12's reading of
// the code it would have returned: `failed`, with no `run_id`, no rows, and
// `isError: true` — which is the whole of what a caller needs to know it may
// retry (§12, mcp's executionOf, ADR-0061).
//
// They are here rather than in the corpus for the reason their argv twins above
// are: a lock is held by a live process and git's account of an unreachable
// remote names a temp directory, and neither is a directory of files.

// TestStoreLost_ALockContendedThroughTheToolIsFailedWithNoRunID is contention
// read through the tool: the envelope names the outcome the exit code would
// have carried, names no Run, and carries the bit that says the caller did not
// get what they asked for.
func TestStoreLost_ALockContendedThroughTheToolIsFailedWithNoRunID(t *testing.T) {
	c := corpusCase(t, "run/the-tracer-bullet")
	invocation := c.invocation(t)

	held, err := store.Acquire(invocation.wd, store.Exclusive)
	if err != nil {
		t.Fatalf("the effectful Run's lock: %v", err)
	}
	defer held.Release()

	envelope := callRun(t, c, invocation, `{"procedure": "watch-status"}`)

	assertRunLost(t, envelope, "")
	if branch := invocation.fixture.render(t, invocation.fixture.root); strings.Contains(branch, "run.json") {
		t.Errorf("the branch holds %q; a Run that never took the lock wrote no entry, whichever door it came through", branch)
	}
}

// TestStoreLost_ARehearsalContendedThroughTheToolStillSaysItWasARehearsal is
// the reason the envelope is told what the call named rather than reading it
// off a row: there is no row here to read it off.
//
// §7's one exception to the absence rule is that `dry_run` is written always,
// `false` included, because what a reader that takes its absence for `false`
// gets wrong is unrecoverable — and a surface that answered `false` to a
// rehearsal that lost the lock would be telling a caller their rehearsal
// reached the world (mcp's execution).
func TestStoreLost_ARehearsalContendedThroughTheToolStillSaysItWasARehearsal(t *testing.T) {
	c := corpusCase(t, "run/the-tracer-bullet")
	invocation := c.invocation(t)

	held, err := store.Acquire(invocation.wd, store.Exclusive)
	if err != nil {
		t.Fatalf("the effectful Run's lock: %v", err)
	}
	defer held.Release()

	envelope := callRun(t, c, invocation, `{"procedure": "watch-status", "dry_run": true}`)

	assertRunLost(t, envelope, "")
	if dryRun := envelope.StructuredContent.DryRun; dryRun == nil || !*dryRun {
		t.Errorf("dry_run is %v, want true — the invocation named a rehearsal and no row survived to say so", dryRun)
	}
	if want := "failed · dry-run"; !strings.Contains(envelope.Content[0].Text, want) {
		t.Errorf("the text block is %q, want it to open %q", envelope.Content[0].Text, want)
	}
}

// TestStoreLost_ASyncThroughTheToolIsFailedWithNoRunID is the sync at Run start,
// which stands before `run.json` exactly as the lock does and answers the same
// shape: nothing was attempted, so nothing is named.
func TestStoreLost_ASyncThroughTheToolIsFailedWithNoRunID(t *testing.T) {
	c := corpusCase(t, "run/an-effectful-sync-that-could-not-reach-the-remote", "run", "publish-preview")
	invocation := c.invocation(t)

	assertRunLost(t, callRun(t, c, invocation, `{"procedure": "publish-preview"}`), "")
}

// TestStoreLost_APushRejectedThroughTheToolNamesTheRunItWrote is the third way
// and the one that differs: the Run did its work and closed its entry, and only
// then did the remote move under three pushes running. An entry was written, so
// the envelope names it — which is the whole of §9's *an entry the envelope does
// not name is one an agent can reach only by asking which Run was the last*.
func TestStoreLost_APushRejectedThroughTheToolNamesTheRunItWrote(t *testing.T) {
	c := corpusCase(t, "run/a-push-rejected-three-times", "run", "watch-status")
	invocation := c.invocation(t)

	envelope := callRun(t, c, invocation, `{"procedure": "watch-status"}`)

	assertRunLost(t, envelope, "01991f31-6dc3-7e4f-b051-728394051627")
	if len(envelope.StructuredContent.Rows) == 0 {
		t.Error("the envelope carries no rows; the Step ran and the rows say so — the push is not the work")
	}
}

// callRun drives one `run` call against the server the binary assembles, over
// the case's own process and build facts, and answers the envelope.
//
// It is the corpus's own door (cli.MCPServer) rather than a second assembly of
// a server, for the reason golden_mcp_test.go states: what these drive should be
// the server a client starts.
func callRun(t *testing.T, c goldenCase, invocation goldenRun, arguments string) mcp.Envelope {
	t.Helper()

	envelope, err := cli.MCPServer(c.process(t, invocation), c.facts(t)).
		Call(t.Context(), "run", json.RawMessage(arguments))
	if err != nil {
		t.Fatalf("run(%s): %v — a Run that lost the Store is an answer and never a protocol error", arguments, err)
	}
	return envelope
}

// assertRunLost holds an envelope to what §12's `75` carries on a surface with
// no exit code: `failed`, the bit set, and the Run named only where an entry was
// written.
//
// **It also holds the half to the schema `run` published**, which is the one
// place MCP's *servers MUST provide structured results that conform to this
// schema* can be held over these three paths at all: they need a contended
// repository rather than a fixture, so no corpus case drives them and
// TestToolSet_EveryAnswerConformsToTheSchemaItsToolPublished never sees them.
// They are also the only `isError: true` answers on this surface that still
// carry a structured half — a Run that lost the Store before `run.json` wrote
// no terminal row and carries §12's triple anyway (ADR-0102, mcp's executionOf).
func assertRunLost(t *testing.T, envelope mcp.Envelope, runID string) {
	t.Helper()

	conformsToItsSchema(t, "run", envelope)

	structured := envelope.StructuredContent
	if structured.Outcome != "failed" {
		t.Errorf("outcome = %q, want %q — a Run that lost the Store is no Refusal", structured.Outcome, "failed")
	}
	if !envelope.IsError {
		t.Error("isError = false; a caller did not get what they asked for")
	}
	if structured.RunID != runID {
		t.Errorf("run_id = %q, want %q", structured.RunID, runID)
	}
	if structured.DryRun == nil {
		t.Error("dry_run is absent; it is written wherever the outcome is, the bare false included (§7)")
	}
	if got := string(structured.Truncated); got != "null" {
		t.Errorf("truncated = %s, want null — a Run ranges over nothing for a limit to have cut", got)
	}
	if len(envelope.Content) != 1 || !strings.HasPrefix(envelope.Content[0].Text, "failed") {
		t.Errorf("the text block is %v, want one line opening with the outcome", envelope.Content)
	}
}
