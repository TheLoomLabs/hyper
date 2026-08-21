package cli_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/cli"
)

// The two decisions cli.Child makes, asserted rather than assumed (issues #134,
// #142).
//
// They landed before their first caller did, and they moved here with it: the
// launcher sits in this package because the corpus drives it, so the criterion
// and the value the binary wires are now one file apart rather than one module.
// The process group is decided here and re-decided nowhere, which is what makes
// the shell Capability's deadline reach everything the argv started.

// TestChild_StartsInItsOwnProcessGroup is the criterion the shell Capability
// stands on: the child leads a process group of its own rather than joining
// hyper's, so a deadline can reach everything the argv started and not only the
// process hyper waited on (§5, §6).
func TestChild_StartsInItsOwnProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cmd := cli.Child(ctx, []string{sleepBinary(t), "30"})
	if err := cmd.Start(); err != nil {
		t.Fatalf("the child would not start: %v", err)
	}
	// Cancelling is what stops it, which is the other decision doing its
	// work: the wait answers the cancellation rather than an exit status.
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	group, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("the child's process group could not be read: %v", err)
	}
	if group != cmd.Process.Pid {
		t.Errorf("the child's process group is %d, want its own pid %d", group, cmd.Process.Pid)
	}
	if group == syscall.Getpgrp() {
		t.Errorf("the child is in hyper's own process group %d, want a group of its own", group)
	}
}

// TestChild_ACancellationThatFindsNothingToKillIsNotAFault is the race the
// negated-pid kill opens and the ESRCH arm closes: a child that finished
// between the deadline expiring and the signal being sent.
//
// os/exec surfaces any error from Cancel out of Wait as *exec: canceling Cmd*,
// so an unmapped ESRCH would reach the Capability as a child that could not be
// started — the one shape §12's response object reserves for an argv that never
// ran at all. The child here is run to completion and reaped first, which is
// that race with the timing taken out of it.
func TestChild_ACancellationThatFindsNothingToKillIsNotAFault(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cmd := cli.Child(ctx, []string{sleepBinary(t), "0"})
	if err := cmd.Run(); err != nil {
		t.Fatalf("the child would not run: %v", err)
	}

	if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("cancelling a finished child answered %v, want os.ErrProcessDone", err)
	}
}

// sleepBinary is a child that does nothing for as long as it is told to, which
// is the whole of what these two cases need one for. It is resolved rather than
// spelled as a path: what is under test is the launch, and a case that failed
// because /bin/sleep sat somewhere else would be reporting the wrong thing.
func sleepBinary(t *testing.T) string {
	t.Helper()

	path, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("this platform has no sleep to launch (%v); the probe cannot be built here", err)
	}
	return path
}

// TestChild_ADeadlineKillsTheWholeGroup is the criterion the process group
// exists for, driven through the Capability that declares the deadline: a
// command's own children do not outlive the deadline that bounded it (§6,
// issue #142).
//
// The child here starts a grandchild that would outlive it by half a minute and
// writes its pid down, then waits. What the deadline kills is the group and not
// the leader, so the grandchild goes with it — and it is a grandchild rather
// than a second child because that is the ordinary shape of a command, a script
// that forks being the usual case rather than the exotic one.
//
// SIGKILL with no grace period is what the immediacy stands for: nothing here
// waits for the child to tidy up, and nothing may, the wait being a guessed
// constant on a Provider that knows nothing whatever about the command.
//
// The pid reaches the case through a file rather than through the response
// object, and that is the rule under test one step over: nothing is recorded
// for a member that reached its deadline, so the killed command's stdout is not
// there to read.
func TestChild_ADeadlineKillsTheWholeGroup(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skipf("this platform has no sleep to launch (%v); the group cannot be driven here", err)
	}

	root := t.TempDir()
	forking := filepath.Join(root, "forks")
	if err := os.WriteFile(forking, []byte("#!/bin/sh\nsleep 30 &\nprintf '%s' \"$!\" > grandchild\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()

	// The invoking environment whole: the script resolves `sleep` through
	// it, and there is no credential slot in a test process for the
	// composition to withhold. root is the directory the child stands in,
	// which is where it writes the pid and where this reads it.
	began := time.Now()
	object, err := capability.Command{Argv: []string{forking}}.
		Perform(ctx, cli.Child, root, capability.Inherited(os.Environ(), nil))
	// **The deadline is when the command stops, and not when its children
	// decide to.** A kill that reached the leader alone would leave the
	// grandchild holding the pipe the child's stdout was read through, so
	// `hyper` would wait out the whole thirty seconds the command was never
	// granted — which is the same fact the pid below states, observed from
	// the other end.
	if waited := time.Since(began); waited > 5*time.Second {
		t.Errorf("the command was bounded at 250ms and took %s to stop", waited)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the command answered %v, want the deadline it was bounded by", err)
	}
	if len(object) != 1 {
		t.Errorf("the killed command recorded %s, and nothing is recorded for a member that reached its deadline", object)
	}

	grandchild := pidWritten(t, filepath.Join(root, "grandchild"))
	// The pid is read off a group that has already been signalled, so what
	// is waited for is the reaping rather than the kill: a process that has
	// died and not yet been collected still answers a signal of 0.
	for waited := 0; waited < 200; waited++ {
		if err := syscall.Kill(grandchild, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Errorf("the command's own child %d outlived the deadline that bounded it", grandchild)
}

// pidWritten is the pid the forking script wrote down, and a failed case where
// it never got that far.
func pidWritten(t *testing.T, path string) int {
	t.Helper()

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the child wrote down no grandchild: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(written)))
	if err != nil {
		t.Fatalf("the child wrote %q rather than a pid: %v", written, err)
	}
	return pid
}
