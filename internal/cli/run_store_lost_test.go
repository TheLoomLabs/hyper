package cli_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The three ways a Run loses the Store, where a golden cannot hold them (§6,
// §7, issue #138). Everything else about them is in the corpus, and each of
// these says why it could not be.
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
// commit, are checked in and compared like any other case's.
//
// Everything else is the corpus's own machinery. Each case is a directory under
// testdata/run/, materialised the way TestGolden materialises one, so what
// differs between these runs and the corpus's is the one fact each is about.

// TestStoreLost_ARunThatCannotTakeTheLockIsFailedAtSeventyFive is contention, and
// what it is not. The Run is `failed` at 75, no `error_code` is rendered
// anywhere, stdout is silent, and no Journal entry was written — a Run that
// cannot take the lock has no branch to write one on (§6, §7, §12, ADR-0061).
func TestStoreLost_ARunThatCannotTakeTheLockIsFailedAtSeventyFive(t *testing.T) {
	c := theTracerBullet(t)
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

// TestStoreLost_ASecondReadOnlyRunProceeds is the other half of the same
// sentence, and the reason the modes exist: a monitoring Run is not starved
// behind another monitoring Run. The shared lock is held throughout and the
// command completes.
func TestStoreLost_ASecondReadOnlyRunProceeds(t *testing.T) {
	c := theTracerBullet(t)
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

// theTracerBullet is the corpus case these two drive, read off its own
// directory exactly as TestGolden reads it. Nothing about it is special to the
// lock — it is the Run this milestone already asserts end to end, and that is
// the point: what these cases change is one fact about the machine.
func theTracerBullet(t *testing.T) goldenCase {
	t.Helper()

	dir := filepath.Join("testdata", "run", "the-tracer-bullet")
	return goldenCase{
		dir:  dir,
		name: "run/the-tracer-bullet",
		argv: readArgv(t, filepath.Join(dir, "argv")),
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
	dir := filepath.Join("testdata", "run", "a-push-rejected-three-times")
	c := goldenCase{dir: dir, name: "run/a-push-rejected-three-times", argv: []string{"run", "watch-status"}}
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
	invocation.compareBranches(t, dir)
}
