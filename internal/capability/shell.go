package capability

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

// The `shell` half of this package: the argv exec'd, and the four-member
// response object §12 closes over what the child did (§3, §5, issue #142).
//
// It is the shallower of the two Capabilities by a long way, and that is the
// point rather than an accident of scope. `hyper`'s own `shell` Provider is the
// only one that may declare this Capability (§11, ADR-0039) and it knows
// nothing whatever about the command, so there is no request block to read: the
// argv **is** the Operation input named `command`, arriving in a Step's `args:`,
// and what this package adds to it is the four decisions no artefact makes —
// the process group, the deadline, the directory and the environment.

// The four members of the shell response object, in §12's own order, which is
// the order they are assembled and rendered in. They are named here for the
// reason the http object's five are: the built-in Provider's projection writes
// $.exit_code, so a second spelling of one of them is a path that silently
// resolves to nothing.
const (
	MemberCommand  = "command"
	MemberExitCode = "exit_code"
	MemberStdout   = "stdout"
	MemberStderr   = "stderr"
)

// Exec is how a child process is started: the read cli.Process threads, and the
// one this package is handed rather than reaches for — `http`'s Dial one
// Capability over.
//
// It answers the child that argv names, ready to run, carrying the two launch
// decisions that belong to the process rather than to a Capability: the child
// starts in its own process group, and cancelling ctx kills that whole group
// with SIGKILL and no grace period (§5, §6). What this package sets on the
// answer is everything the Capability decides — the directory, the environment,
// the streams — and never a process attribute, which is what keeps the process
// group decided in one place.
type Exec func(ctx context.Context, argv []string) *exec.Cmd

// Environment is what a `shell` Operation's child inherits: the invoking
// environment with every credential-slot variable in the repository removed
// (§3, §11).
//
// It is a type of its own rather than a []string so that the removal cannot be
// skipped by a caller holding the process's own environment: what it carries is
// unexported, so the only way to fill one is Inherited below. That is ADR-0007's
// rule at this position — a credential is suppressed by the position it occupies
// rather than by every caller remembering to.
//
// Its **zero value is not an empty environment**, and Perform will not start a
// child under one. os/exec reads a nil Env as *inherit the parent's*, and the
// parent's is the environment `hyper` was invoked in, credential slots
// included — so *nobody composed one* and *the composition came out empty* are
// two states this must be able to tell apart.
type Environment struct{ composed []string }

// Composed says this Environment was filled by Inherited rather than left at
// its zero value. It is what Perform reads before it starts anything.
func (e Environment) Composed() bool { return e.composed != nil }

// Inherited composes that environment: environ as the process holds it, less
// every variable in withheld.
//
// The withheld set is **every** credential-slot variable any Target declaration
// in the repository names, not only those a Run resolved, so it is decided
// offline and does not turn on which Steps a Run reached (§11). `hyper` knows
// those names by position (§3), which is the same knowledge that lets it
// suppress a credential rather than scan for one, used here to keep the
// credentials it resolved out of a process it cannot describe.
//
// Everything else is the command's, and `hyper` neither reads it nor records
// it; §13 states what that costs. What is *not* here is an authored `env:`,
// which would route a secret through an argument list — the working directory,
// stdin and the environment are fixed rather than authored, each because a key
// whose only legal content is the one `hyper` requires is a second spelling
// that can only ever disagree with the first (§3).
func Inherited(environ []string, withheld []string) Environment {
	removed := make(map[string]bool, len(withheld))
	for _, name := range withheld {
		if name != "" {
			removed[name] = true
		}
	}

	// Non-nil always, empty included: a nil Env is os/exec's word for
	// *inherit the parent's*, which is the one thing this composition
	// exists to prevent.
	inherited := make([]string, 0, len(environ))
	for _, variable := range environ {
		name, _, named := strings.Cut(variable, "=")
		if !named || removed[name] {
			continue
		}
		inherited = append(inherited, variable)
	}
	return Environment{composed: inherited}
}

// Command is one shell Operation's request, filled: the argv as it will be
// exec'd, first word first.
//
// It is the whole of the request. There is no method, no path and no body,
// because a `shell:` block carries no keys at all — the block's key *is* the
// Capability, and the words are the Step's rather than the Manifest's (§3).
type Command struct{ Argv []string }

// Text is §12's `command` member: the argv as run, JSON-encoded on one line.
//
// JSON rather than a joining rule because it must be injective — `[echo, "a
// b"]` and `[echo, a, b]` are two commands and must be two identities, and a
// joining rule silently makes them one series that `record-identity-collision`
// could never catch, the two names being genuinely equal.
//
// It is a fact about the call rather than about the answer, which is what lets
// a `read` record a command that never started: it is `host`'s member argument
// one Capability over, present because an Operation whose answer carries no
// identity of its own has nowhere else to project one from.
func (c Command) Text() string {
	encoded, err := CompactJSON(c.Argv)
	if err != nil {
		// Unreachable: a list of Go strings always encodes, invalid
		// UTF-8 included, which encoding/json writes as the replacement
		// character rather than refusing.
		return "[]"
	}
	return string(encoded)
}

// Perform execs the argv and assembles the response object §12 closes at four
// members. root is the repository root, which is the working directory every
// child runs in — fixed rather than authored, so that a laptop and a runner
// agree without a line saying so (§3).
//
// **Nothing stands between the artefact and the process.** There is no shell
// here, so a pipe, a redirection, a glob and an `&&` reach the child as literal
// argv words and join §13's limits (ADR-0051).
//
// The object is always usable: where the command could not be started at all —
// no such binary, not executable — it is `command` and nothing else, which is
// the no-answer case one Capability over and what a `read` records as the
// answer it is (§6, §12, ADR-0050). The error beside it says what went wrong
// and is narration's alone; no member of the object says it, that being the
// catch-all bucket ADR-0017 closed.
//
// **A non-zero exit is an answer and never an error.** The code is recorded and
// nothing stands beside it, so a check script whose exit status *is* the
// finding is describable without a second declaration saying what success means
// (§6, ADR-0050). The one error this answers past a start that failed is the
// **deadline**, which is `hyper` stopping rather than the command answering.
func (c Command) Perform(ctx context.Context, start Exec, root string, environment Environment) (Object, error) {
	object := Object{{Name: MemberCommand, Value: c.Text()}}

	if !environment.Composed() {
		// Unreachable: a Run composes one before Step 1. It is refused
		// rather than defaulted because the default os/exec would take
		// is the environment `hyper` was invoked in — the one place a
		// command it cannot describe could read a credential out of
		// (§11, ADR-0007).
		return object, errors.New("the child's environment was never composed, and no child inherits hyper's own")
	}

	child := start(ctx, c.Argv)
	child.Dir = root
	// stdin is empty and is not authorable anywhere: os/exec reads a nil
	// Stdin as the null device, which is a command handed nothing rather
	// than a command handed hyper's own terminal (§3).
	child.Stdin = nil
	child.Env = environment.composed
	var stdout, stderr bytes.Buffer
	child.Stdout, child.Stderr = &stdout, &stderr

	if err := child.Start(); err != nil {
		// The child never ran, so there is no exit code, no stdout and
		// no stderr — three members absent together, which is the one
		// shape §12 reserves for an argv that never became a process.
		return object, err
	}

	waited := child.Wait()

	// The deadline, read off the context rather than off what Wait made of
	// the kill: the Operation declared it, so it is the one stopping this
	// answers as itself. Its whole process group has been killed with
	// SIGKILL and no grace period by then — the group rather than the
	// process, so a command's own children do not outlive the deadline that
	// bounded it (§6, cli.Child).
	//
	// Nothing is recorded for a member that reached it. On a `read` the Step
	// fails, which is `hyper` stopping rather than the world answering
	// nothing (§6, ADR-0050).
	if deadline := ctx.Err(); errors.Is(deadline, context.DeadlineExceeded) {
		return object, deadline
	}
	if child.ProcessState == nil {
		// Wait answered before the process did, which is not a state
		// os/exec reaches by any path this package takes. It is read for
		// rather than asserted because a nil dereference is a worse
		// answer than the no-answer object above.
		return object, waited
	}

	// ExitCode answers -1 for a child a signal ended, which is a value no
	// exit status can carry and so tells itself apart from every code a
	// command chose. The one signal `hyper` sends is the deadline's, and
	// that returned above.
	object = append(object,
		Member{Name: MemberExitCode, Value: child.ProcessState.ExitCode()},
		// Text, and never parsed (ADR-0052). A command that answers in
		// JSON is recorded as the string it printed, and
		// $.stdout.result.id is not a path — which is the `opaque` trait
		// arriving in the projection, `hyper` being unable to describe
		// what a command does and parsing its output being a description
		// of exactly that (§3, §12).
		Member{Name: MemberStdout, Value: stdout.String()},
		Member{Name: MemberStderr, Value: stderr.String()},
	)
	return object, nil
}
