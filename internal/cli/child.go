package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/TheLoomLabs/hyper/internal/capability"
)

// Child is Process.Exec, which states what the two launch decisions here are
// for. This is how they are made (§5, §6, issue #142).
//
// Setpgid is the process group, and it is set rather than the child being left
// in `hyper`'s own: what a deadline kills is then the whole tree the argv
// started, a command that forks being the ordinary case rather than the exotic
// one. It is also what makes draining true of a `shell` Step at all — in
// `hyper`'s group a terminal's interrupt reaches the child directly and it dies
// at once, so the Step in flight would not finish and the drain would be a
// sentence the implementation contradicts (§6).
//
// Cancel is replaced because exec.CommandContext's own cancellation signals the
// leader alone, which would leave the group Setpgid just made — so the signal
// goes to the negated pid, which is the group, and it is SIGKILL with nothing
// before it. SIGKILL rather than a SIGTERM and a wait, because the wait is a
// guessed constant on a Provider that knows nothing whatever about the command,
// which is the ground `concurrency:` is 1 on (§6, ADR-0045).
//
// argv is exec'd directly with no shell between the artefact and the process,
// and its head is read off it without a question asked: the authoring format
// admits no other shape and `check` refuses a command without a literal head
// long before a Run reaches a Capability (ADR-0051).
//
// It sits in this package rather than in the binary's own main because the
// corpus drives it: a case that launched a child through a stand-in launcher
// would leave the process group and the SIGKILL unchecked, and two spellings of
// the two decisions is where the day comes that the binary and the suite make
// them differently. cmd/hyper wires this value and adds nothing to it.
func Child(ctx context.Context, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); !errors.Is(err, syscall.ESRCH) {
			return err
		}
		// The group went on its own between the deadline expiring and the
		// signal being sent, which is a race and not a fault. os/exec
		// surfaces any other error from Cancel out of Wait as *exec:
		// canceling Cmd*, and a command that merely finished in time would
		// then reach the Capability as a child that could not be started —
		// the one shape §12's response object reserves for an argv that
		// never ran at all. os.ErrProcessDone is what os/exec reads as
		// *there was nothing left to kill*.
		return os.ErrProcessDone
	}
	return cmd
}

// Child satisfies capability.Exec, which is the signature the Capability that
// runs a command is written against. The assertion is here rather than at the
// wiring so that a change to either side is a compile error in the file that
// made it.
var _ capability.Exec = Child
