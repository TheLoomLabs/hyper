package run

import (
	"errors"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **An effectful `shell` Operation completes on `0` alone** (§6, §7, §12,
// ADR-0050, ADR-0062, issue #156).
//
// It is a unit test for once_test.go's reason: the answer is a pure function of
// one Kind met by one response object, where the corpus would need a fixture
// per cell to say the same twelve things — and two of the cells it holds, the
// `destroy` told `1` and the `destroy` whose child never started, are the whole
// of the asymmetry with `http` that this ticket is about.

// commandAnswered is the response object a child that ran leaves: the argv as
// run, and the code it exited with (§12, capability.Perform).
func commandAnswered(command string, code int) capability.Object {
	return capability.Object{
		{Name: capability.MemberCommand, Value: command},
		{Name: capability.MemberExitCode, Value: code},
		{Name: capability.MemberStdout, Value: ""},
		{Name: capability.MemberStderr, Value: ""},
	}
}

// commandNeverStarted is the object a child that could not be started at all
// leaves: `command` and nothing else, three members absent together (§12).
func commandNeverStarted(command string) capability.Object {
	return capability.Object{{Name: capability.MemberCommand, Value: command}}
}

// shellStep is a Step binding one of the built-in Provider's Operations: the
// Kind it declares and the Capability its request is, which are the whole of
// what judging reads off a binding.
func shellStep(kind string) binding {
	return binding{operation: artefact.OperationInfo{Kind: kind, IsShell: true}}
}

// TestJudged_AnEffectfulShellStepCompletesOnZeroAlone walks the Kinds crossed
// with what a command can answer with.
//
// **There is no `404` here and that is the point.** A status code is a
// protocol's shared vocabulary; an exit code is the command's own, so no value
// completes a `destroy` that would not complete a `mutate` — and the trap the
// `404` exists to avoid is closed by the `over:` selector instead (§6).
func TestJudged_AnEffectfulShellStepCompletesOnZeroAlone(t *testing.T) {
	const argv = `["rm","-rf","/srv/app/releases/r41"]`

	for _, tc := range []struct {
		name        string
		kind        string
		response    capability.Object
		halts       bool
		disposition store.Disposition
		answered    store.Answered
	}{{
		name:     "a read is judged by nothing, whatever it exited with",
		kind:     "read",
		response: commandAnswered(argv, 3),
	}, {
		name:     "a read whose child never started is judged by nothing either",
		kind:     "read",
		response: commandNeverStarted(argv),
	}, {
		name:     "a mutate completes on 0",
		kind:     "mutate",
		response: commandAnswered(argv, 0),
	}, {
		name:     "a destroy completes on 0",
		kind:     "destroy",
		response: commandAnswered(argv, 0),
	}, {
		name:        "a mutate halts on a nonzero exit, and is ran",
		kind:        "mutate",
		response:    commandAnswered(argv, 1),
		halts:       true,
		disposition: store.DispositionRan,
		answered:    store.ShellAnswer{Command: argv, ExitCode: store.Arrived(1)},
	}, {
		// The asymmetry, stated as the cell it is: `404` completes an
		// `http` destroy and there is no exit code anywhere that
		// completes this one.
		name:        "a destroy halts on a nonzero exit like any other effectful Step",
		kind:        "destroy",
		response:    commandAnswered(argv, 1),
		halts:       true,
		disposition: store.DispositionRan,
		answered:    store.ShellAnswer{Command: argv, ExitCode: store.Arrived(1)},
	}, {
		// A child a signal ended is os/exec's own -1, which is a value
		// no exit status can carry — and it is not 0, so it halts.
		name:        "a child a signal ended halts on the -1 it is recorded as",
		kind:        "mutate",
		response:    commandAnswered(argv, -1),
		halts:       true,
		disposition: store.DispositionRan,
		answered:    store.ShellAnswer{Command: argv, ExitCode: store.Arrived(-1)},
	}, {
		name:        "a mutate whose child never started is attempted, world untouched",
		kind:        "mutate",
		response:    commandNeverStarted(argv),
		halts:       true,
		disposition: store.DispositionAttemptedWorldUntouched,
		answered:    store.ShellAnswer{Command: argv},
	}, {
		name:        "a destroy whose child never started is attempted, world untouched",
		kind:        "destroy",
		response:    commandNeverStarted(argv),
		halts:       true,
		disposition: store.DispositionAttemptedWorldUntouched,
		answered:    store.ShellAnswer{Command: argv},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			completed, fault := shellStep(tc.kind).judged(sequenced{}, tc.response)

			// **Nothing completes a `shell` Step on anything but the
			// ordinary answer**, there being no `404` for a command
			// to answer with — so the arm of `answered` that a
			// completing call writes is unreachable here (§7).
			if completed != nil {
				t.Errorf("a %s answered %+v, and a shell Step completes on 0 alone", tc.kind, completed)
			}
			if !tc.halts {
				if fault != nil {
					t.Fatalf("judged answered %v, want the ordinary answer", fault)
				}
				return
			}
			if fault == nil {
				t.Fatal("judged answered nothing, want a halt")
			}

			var effect effectFault
			if !errors.As(fault, &effect) {
				t.Fatalf("judged answered %v, which carries no Disposition and no answered", fault)
			}
			if effect.disposition != tc.disposition {
				t.Errorf("the Step is %s, want %s", effect.disposition, tc.disposition)
			}
			if effect.answered != tc.answered {
				t.Errorf("answered is %+v, want %+v", effect.answered, tc.answered)
			}
		})
	}
}

// TestCompletesOn_IsTheThresholdOfTheStepItIsRenderedFor holds the sentence a
// halted Step renders against the Step it renders it for: a message naming the
// Kind and then stating another Kind's threshold is the one thing a reader
// cannot recover from (§6, effect.go).
func TestCompletesOn_IsTheThresholdOfTheStepItIsRenderedFor(t *testing.T) {
	for _, tc := range []struct {
		bound binding
		want  string
	}{
		{binding{operation: artefact.OperationInfo{Kind: "mutate"}}, "2xx alone"},
		{binding{operation: artefact.OperationInfo{Kind: "destroy"}}, "2xx and on 404 besides"},
		// Under `shell` it is `0` on **both** Kinds: there is no `404`
		// for a command to answer with, so the threshold does not move
		// with the Kind the way the `http` one does.
		{shellStep("mutate"), "0 alone"},
		{shellStep("destroy"), "0 alone"},
	} {
		if held := tc.bound.completesOn(); held != tc.want {
			t.Errorf("a %s %s Step completes on %q, want %q",
				map[bool]string{true: "shell", false: "http"}[tc.bound.operation.IsShell],
				tc.bound.operation.Kind, held, tc.want)
		}
	}
}

// **A deadline reached on an effectful `shell` Step is *attempted, outcome
// unknown*, and it carries no `answered` at all** (§6, §12, issue #156).
//
// The Disposition is the Step's **Kind's** and never its Capability's, which is
// what this holds: the child's whole process group has been killed with SIGKILL
// and no grace period by the time the halt is built, and *the group died* is not
// *the effect did not happen* — a command killed half-way through may have
// changed the machine already. What the kill itself does is
// [capability.Command.Perform](../capability/shell.go)'s and
// [cli.Child](../cli/child.go)'s, and no golden can drive it: the built-in
// Provider declares one hour on every Operation and no repository may edit it.
func TestHaltedByDeadline_AnEffectfulShellStepIsAttemptedOutcomeUnknown(t *testing.T) {
	for _, tc := range []struct {
		kind string
		want store.Disposition
	}{
		// A `read`'s deadline fails the Step and it is *ran*: the call
		// went out under a bound an artefact declared, and nothing is in
		// doubt that a Disposition could carry.
		{"read", store.DispositionRan},
		{"mutate", store.DispositionAttemptedOutcomeUnknown},
		{"destroy", store.DispositionAttemptedOutcomeUnknown},
	} {
		halted := shellStep(tc.kind).haltedByDeadline("the deadline was reached and %s was killed", `["rm","-rf","/srv"]`)
		if held := dispositionAfter(halted); held != tc.want {
			t.Errorf("a %s shell Step that reached its deadline is %s, want %s", tc.kind, held, tc.want)
		}
		// There is no answer to name, and `answered` is the key that
		// says what one was — so a deadline writes none whatever the
		// Capability (§7).
		if named := answeredBy(halted); named != nil {
			t.Errorf("a %s shell Step that reached its deadline answers %+v, want nothing", tc.kind, named)
		}
	}
}
